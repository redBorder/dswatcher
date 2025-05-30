// Service for allowing new sensors to send flow based on a serial number.
// Copyright (C) 2017 ENEO Tecnologia SL
// Author: Diego Fernández Barrear <bigomby@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package updater

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-chef/chef"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

///////////////
//Interfaces //
///////////////

// ChefAPIClient is an interface for a chef api client
type ChefAPIClient interface {
	NewClient(interface{}) interface{}
}

// ChefUpdaterConfig contains the configuration for a ChefUpdater.
type ChefUpdaterConfig struct {
	client *chef.Client

	Name                 string
	URL                  string
	AccessKey            string
	SerialNumberPath     string
	SensorUUIDPath       string
	ObservationIDPath    string
	IPAddressPath        string
	BlockedStatusPath    string
	ProductTypePath      string
	OrganizationUUIDPath string
	LicenseUUIDPath      string
	DataBagName          string
	DataBagItem          string
	SkipSSL              bool
}

// ChefUpdater uses the Chef client API to update a sensor node with an IP
// address.
type ChefUpdater struct {
	nodes map[string]*chef.Node

	ChefUpdaterConfig
}

// NewChefUpdater creates a new instance of a ChefUpdater.
func NewChefUpdater(config ChefUpdaterConfig) (*ChefUpdater, error) {
	updater := &ChefUpdater{
		nodes:             make(map[string]*chef.Node),
		ChefUpdaterConfig: config,
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    updater.Name,
		Key:     updater.AccessKey,
		BaseURL: updater.URL,
		SkipSSL: updater.SkipSSL,
	})
	if err != nil {
		return nil, errors.New("Error creating client: " + err.Error())
	}

	updater.client = client

	return updater, nil
}

func (cu *ChefUpdater) fetchLicenses() error {
	licK := getKeyFromPath(cu.LicenseUUIDPath)

	items, err := cu.client.DataBags.GetItem(cu.DataBagName, cu.DataBagItem)
	if err != nil {
		return errors.New("Couldn't get items from data bag: " + err.Error())
	}

	sensorsIf, ok := items.(map[string]interface{})
	if !ok {
		return errors.New("Couldn't get sensors from data bag")
	}

	sensors, ok := sensorsIf["sensors"].(map[string]interface{})
	if !ok {
		return errors.New("Couldn't get sensors from data bag. Failed assertion to " +
			"\"map[string]interface{}\"")
	}

	for k, v := range sensors {
		if node, ok := cu.nodes[k]; ok {
			attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
			if err != nil {
				return errors.New("Error getting node info: " + err.Error())
			}

			attributes[licK] = v.(map[string]interface{})["license"].(string)
		}
	}

	return nil
}

// FetchNodes updates the internal node database and keep it in memory
func (cu *ChefUpdater) FetchNodes() error {
	nodeList, err := cu.client.Nodes.List()
	if err != nil {
		return errors.New("Couldn't list nodes: " + err.Error())
	}

	for n := range nodeList {
		node, err := cu.client.Nodes.Get(n)
		if err != nil {
			return errors.New("Error getting node info: " + err.Error())
		}

		// Ensure redborder.blocked exists
		cu.ensureBlockedField(n, &node)

		// Get parent of sensor UUID path
		parentAttrs, err := getParent(node.NormalAttributes, cu.SensorUUIDPath)
		if err != nil {
			return fmt.Errorf("Failed to get parent attributes for node %s: %v", n, err)
		}

		sensorKey := getKeyFromPath(cu.SensorUUIDPath)
		sensorUUID, ok := parentAttrs[sensorKey].(string)
		if !ok {
			continue
		}

		cu.nodes[sensorUUID] = &node
	}

	if err := cu.fetchLicenses(); err != nil {
		return errors.New("Error fetching licenses: " + err.Error())
	}

	return nil
}

// UpdateNode gets a list of nodes an look for one with the given address. If a
// node is found will update the deviceID.
// If a node with the given address is not found an error is returned
func (cu *ChefUpdater) UpdateNode(
	address net.IP, serialNumber string, obsID uint32, deviceID uint32) error {
	pType := getKeyFromPath(cu.ProductTypePath)

	var (
		ok                 bool
		nodeProductType    interface{}
		nodeProductTypeStr string
		nodeProductTypeInt uint64
	)

	node := findNode(cu.SerialNumberPath, serialNumber, cu.nodes)
	if node == nil {
		return errors.New("Node not found")
	}

	attributes, err := getParent(node.NormalAttributes, cu.ProductTypePath)
	if err != nil {
		return err
	}

	if nodeProductType, ok = attributes[pType]; !ok {
		nodeProductType = "999"
	}

	if nodeProductTypeStr, ok = nodeProductType.(string); !ok {
		return errors.New("Product Type is not string")
	}

	nodeProductTypeInt, err = strconv.ParseUint(nodeProductTypeStr, 10, 32)
	if err != nil {
		return err
	}

	if uint32(nodeProductTypeInt) != deviceID {
		return errors.New("Product Type for " + address.String() + " does not match")
	}

	ipaddressAttributes, err := getParent(node.NormalAttributes, cu.IPAddressPath)
	if err != nil {
		return err
	}

	observationIDAttributes, err :=
		getParent(node.NormalAttributes, cu.ObservationIDPath)
	if err != nil {
		return err
	}

	ipaddressAttributes[getKeyFromPath(cu.IPAddressPath)] = address.String()
	observationIDAttributes[getKeyFromPath(cu.ObservationIDPath)] =
		strconv.FormatUint(uint64(obsID), 10)

	if cu.client != nil {
		cu.client.Nodes.Put(*node)
	}

	return nil
}

// BlockOrganization iterates a node list and block all sensor belonging to an
// organization.
func (cu *ChefUpdater) BlockOrganization(organization string, productType uint32) []error {
	var errs []error
	blocked := getKeyFromPath(cu.BlockedStatusPath)
	org := getKeyFromPath(cu.OrganizationUUIDPath)
	pType := getKeyFromPath(cu.ProductTypePath)

	for _, node := range cu.nodes {
		log.Infof("Blocking node: %s", node.Name)

		attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if attributes[org] == organization || organization == "*" {
			nodeProductTypeStr, ok := attributes[pType].(string)
			if !ok {
				nodeProductTypeStr = "999"
			}

			nodeProductType, err := strconv.ParseUint(nodeProductTypeStr, 10, 32)
			if err != nil || uint32(nodeProductType) == productType {
				if err != nil {
					errs = append(errs, errors.New("Blocking sensor with unknown product type"))
				}

				attributes[blocked] = true

				if cu.client != nil {
					_, err := cu.client.Nodes.Put(*node)
					if err != nil {
						errs = append(errs, err)
					} else {
						log.Infof("Successfully blocked and updated node %s", node.Name)
					}
				}
			}
		}
	}
	return errs
}

// AllowLicense iterates a node list and unblock all sensors with the given
// license.
func (cu *ChefUpdater) AllowLicense(license string) []error {
	var errs []error
	blocked := getKeyFromPath(cu.BlockedStatusPath)

	for _, node := range cu.nodes {
		attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		attributes[blocked] = false

		if cu.client != nil {
			cu.client.Nodes.Put(*node)
		}
	}

	return errs
}

// ResetAllSensors sets the blocked status to true for all sensors
func (cu *ChefUpdater) ResetAllSensors() error {
	blocked := getKeyFromPath(cu.BlockedStatusPath)

	for _, node := range cu.nodes {
		attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
		if err != nil {
			continue
		}

		attributes[blocked] = true
		if cu.client != nil {
			cu.client.Nodes.Put(*node)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

// getParent receives the root object containing all the attributes of the
// node and returns the inner object given a path
func getParent(root map[string]interface{}, path string) (map[string]interface{}, error) {
	keys := strings.Split(path, "/")
	if len(keys) == 0 {
		return nil, fmt.Errorf("invalid path: %q", path)
	}

	current := root

	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found in path %q", key, path)
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map[string]interface{} at key %q but got %T", key, next)
		}

		current = nextMap
	}

	return current, nil
}

func findNode(keyPath string, value string, nodes map[string]*chef.Node,
) (node *chef.Node) {
	key := getKeyFromPath(keyPath)

	for _, node := range nodes {
		attributes, err := getParent(node.NormalAttributes, keyPath)
		if err != nil {
			continue
		}

		if attributes[key] == value {
			return node
		}
	}

	return nil
}

func getKeyFromPath(path string) string {
	keys := strings.Split(path, "/")
	return keys[len(keys)-1]
}

// Create blocked (redborder -> blocked) field if it doesn't exist in the passed node as parameter
func (cu *ChefUpdater) ensureBlockedField(n string, node *chef.Node) {
	redborderAttrs, ok := node.NormalAttributes["redborder"].(map[string]interface{})
	if !ok || redborderAttrs == nil {
		redborderAttrs = make(map[string]interface{})
		node.NormalAttributes["redborder"] = redborderAttrs
	}

	if _, exists := redborderAttrs["blocked"]; !exists {
		log.Infof("Adding missing 'blocked: false' field to node %s", n)
		redborderAttrs["blocked"] = false

		if _, err := cu.client.Nodes.Put(*node); err != nil {
			log.Errorf("Failed to update node %s: %v", n, err)
		}
	}
}
