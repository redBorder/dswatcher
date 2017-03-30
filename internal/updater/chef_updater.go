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
	"net"
	"strconv"
	"strings"

	"github.com/go-chef/chef"
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

	Name              string
	URL               string
	AccessKey         string
	SerialNumberPath  string
	SensorUUIDPath    string
	ObservationIDPath string
	IPAddressPath     string
	BlockedStatusPath string
	DeviceIDPath      string
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
	})
	if err != nil {
		return nil, errors.New("Error creating client: " + err.Error())
	}

	updater.client = client

	return updater, nil
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

		attributes, err := getParent(node.NormalAttributes, cu.SerialNumberPath)
		if err != nil {
			return errors.New("Error getting node info: " + err.Error())
		}

		sensorUUID, ok := attributes[getKeyFromPath(cu.SensorUUIDPath)].(string)
		if !ok {
			continue
		}

		cu.nodes[sensorUUID] = &node
	}

	return nil
}

// UpdateNode gets a list of nodes an look for one with the given address. If a
// node is found will update the deviceID.
// If a node with the given address is not found an error is returned
func (cu *ChefUpdater) UpdateNode(
	address net.IP, serialNumber string, obsID uint32, deviceID uint32) error {
	deviceIDKey := getKeyFromPath(cu.DeviceIDPath)

	var (
		ok               bool
		nodeDeviceID     interface{}
		nodeDeviceIDUint int
	)

	node := findNode(cu.SerialNumberPath, serialNumber, cu.nodes)
	if node == nil {
		return errors.New("Node not found")
	}

	deviceIDAttributes, err := getParent(node.NormalAttributes, cu.DeviceIDPath)
	if err != nil {
		return err
	}

	if nodeDeviceID, ok = deviceIDAttributes[deviceIDKey]; !ok {
		return errors.New("Sensor " + serialNumber + " does not have a DeviceID")
	}

	if nodeDeviceIDUint, ok = nodeDeviceID.(int); !ok {
		return errors.New("DeviceID is not integer")
	}

	if nodeDeviceIDUint != int(deviceID) {
		return errors.New("DeviceID for " + address.String() + " does not match")
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

// BlockAllSensors iterates a node list and block all sensor on the list.
func (cu *ChefUpdater) BlockAllSensors() []error {
	var errs []error
	key := getKeyFromPath(cu.BlockedStatusPath)

	for _, node := range cu.nodes {
		attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		attributes[key] = true

		if cu.client != nil {
			cu.client.Nodes.Put(*node)
		}
	}

	return errs
}

// BlockSensor gets a list of nodes an look for one with the given address. If a
// node is found will update the deviceID.
// If a node with the given address is not found an error is returned
func (cu *ChefUpdater) BlockSensor(uuid UUID) (bool, error) {
	key := getKeyFromPath(cu.BlockedStatusPath)

	node := findNode(cu.SensorUUIDPath, string(uuid), cu.nodes)
	if node == nil {
		return false, errors.New("Node not found")
	}

	attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
	if err != nil {
		return false, err
	}

	if blocked, ok :=
		attributes[key].(bool); ok {
		if blocked {
			return false, nil
		}
	}

	attributes[key] = true

	if cu.client != nil {
		cu.client.Nodes.Put(*node)
	}

	return true, nil
}

// ResetSensors sets the blocked status to false for every sensor.
func (cu *ChefUpdater) ResetSensors() error {
	key := getKeyFromPath(cu.BlockedStatusPath)

	for _, node := range cu.nodes {
		attributes, err := getParent(node.NormalAttributes, cu.BlockedStatusPath)
		if err != nil {
			continue
		}

		attributes[key] = false

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
	var ok bool

	attrs := root
	for i, key := range keys {
		if i < len(keys)-1 {
			if attrs, ok = attrs[key].(map[string]interface{}); !ok || attrs == nil {
				return nil, errors.New("Cannot find key: " + path)
			}
		}
	}

	return attrs, nil
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
