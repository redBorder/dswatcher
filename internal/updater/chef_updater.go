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
	"strings"

	"github.com/go-chef/chef"
)

// DataBagService is used for obtain information about licenses from chef.
type DataBagService interface {
	GetItem(string, string) (chef.DataBagItem, error)
}

// NodeService is used for obtain node attributes from chef.
type NodeService interface {
	List() (map[string]string, error)
	Get(string) (chef.Node, error)
	Put(chef.Node) (chef.Node, error)
}

// ChefUpdaterConfig contains the configuration for a ChefUpdater.
type ChefUpdaterConfig struct {
	DataBagService DataBagService
	NodeService    NodeService

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
}

// ChefUpdater uses the Chef client API to update sensors node attributes.
type ChefUpdater struct {
	nodes          map[string]*chef.Node
	dataBagService DataBagService
	nodeService    NodeService

	config *ChefUpdaterConfig
}

// NewChefUpdater creates a new instance of a ChefUpdater.
func NewChefUpdater(config *ChefUpdaterConfig) *ChefUpdater {
	if config.NodeService == nil {
		panic("NodeService can't be nil")
	}
	if config.DataBagService == nil {
		panic("DataBagService can't be nil")
	}

	return &ChefUpdater{
		config:         config,
		nodes:          make(map[string]*chef.Node),
		nodeService:    config.NodeService,
		dataBagService: config.DataBagService,
	}
}

/////////////
// Public  //
/////////////

// Fetch gets data from available sensors in chef. Since licenses are not
// stored on nodes it's necessary to fetch them from a data bag.
func (cu *ChefUpdater) Fetch() error {
	var err error

	err = cu.fetchNodes()
	if err != nil {
		return err
	}

	err = cu.fetchLicenses()
	if err != nil {
		return err
	}

	return nil
}

// SetProductType gets a node by its IP address and update the product type. An
// is returned when a node is not found.
//
// This method uses an internal database, so Fetch() should be called
// periodically to keep the internal database synced with the Chef nodes.
// func (cu *ChefUpdater) SetProductType(
// 	address net.IP, serialNumber string, obsID, productType uint32) error {
// 	pType := getKeyFromPath(cu.config.ProductTypePath)
//
// 	var (
// 		ok                 bool
// 		nodeProductType    interface{}
// 		nodeProductTypeStr string
// 		nodeProductTypeInt uint64
// 	)
//
// 	node := findNode(cu.config.SerialNumberPath, serialNumber, cu.nodes)
// 	if node == nil {
// 		return errors.New("Node not found")
// 	}
//
// 	attributes, err := getParent(node.NormalAttributes, cu.config.ProductTypePath)
// 	if err != nil {
// 		return err
// 	}
//
// 	if nodeProductType, ok = attributes[pType]; !ok {
// 		return errors.New(
// 			"Sensor " + serialNumber + " does not have a Product Type",
// 		)
// 	}
//
// 	if nodeProductTypeStr, ok = nodeProductType.(string); !ok {
// 		return errors.New("Product Type is not string")
// 	}
//
// 	nodeProductTypeInt, err = strconv.ParseUint(nodeProductTypeStr, 10, 32)
// 	if err != nil {
// 		return err
// 	}
//
// 	if uint32(nodeProductTypeInt) != productType {
// 		return errors.New(
// 			"Product Type for " + address.String() + " does not match",
// 		)
// 	}
//
// 	ipaddressAttributes, err :=
// 		getParent(node.NormalAttributes, cu.config.IPAddressPath)
// 	if err != nil {
// 		return err
// 	}
//
// 	observationIDAttributes, err :=
// 		getParent(node.NormalAttributes, cu.config.ObservationIDPath)
// 	if err != nil {
// 		return err
// 	}
//
// 	ipaddressAttributes[getKeyFromPath(cu.config.IPAddressPath)] =
// 		address.String()
// 	observationIDAttributes[getKeyFromPath(cu.config.ObservationIDPath)] =
// 		strconv.FormatUint(uint64(obsID), 10)
//
// 	cu.nodeService.Put(*node)
//
// 	return nil
// }

// BlockOrganization iterates a node list and block all sensor belonging to an
// organization.
// func (cu *ChefUpdater) BlockOrganization(
// 	organization string, productType uint32,
// ) []error {
// 	var errs []error
//
// 	blocked := getKeyFromPath(cu.config.BlockedStatusPath)
// 	org := getKeyFromPath(cu.config.OrganizationUUIDPath)
// 	pType := getKeyFromPath(cu.config.ProductTypePath)
//
// 	for _, node := range cu.nodes {
// 		attributes, err :=
// 			getParent(node.NormalAttributes, cu.config.BlockedStatusPath)
// 		if err != nil {
// 			errs = append(errs, err)
// 			continue
// 		}
//
// 		if attributes[org] == organization || organization == "*" {
// 			nodeProductType, err :=
// 				strconv.ParseUint(attributes[pType].(string), 10, 32)
//
// 			if err != nil || uint32(nodeProductType) == productType {
// 				if err != nil {
// 					errs = append(errs, errors.New(
// 						"Blocking sensor with unknown product type"),
// 					)
// 				}
//
// 				attributes[blocked] = true
// 				cu.nodeService.Put(*node)
// 			}
// 		}
// 	}
//
// 	return errs
// }

// BlockLicense iterates a node list and block all sensor belonging to an
// organization.
// func (cu *ChefUpdater) BlockLicense(license string) []error {
// 	var errs []error
// 	blocked := getKeyFromPath(cu.config.BlockedStatusPath)
// 	lic := getKeyFromPath(cu.config.LicenseUUIDPath)
//
// 	for _, node := range cu.nodes {
// 		attributes, err :=
// 			getParent(node.NormalAttributes, cu.config.BlockedStatusPath)
// 		if err != nil {
// 			errs = append(errs, err)
// 			continue
// 		}
//
// 		if attributes[lic] == license {
// 			attributes[blocked] = true
// 			cu.nodeService.Put(*node)
// 		}
// 	}
//
// 	return errs
// }

// ResetSensors sets the blocked status to false for sensors belonging to an
// organization
// func (cu *ChefUpdater) ResetSensors(organization string) error {
// 	blocked := getKeyFromPath(cu.config.BlockedStatusPath)
// 	org := getKeyFromPath(cu.config.OrganizationUUIDPath)
//
// 	for _, node := range cu.nodes {
// 		attributes, err :=
// 			getParent(node.NormalAttributes, cu.config.BlockedStatusPath)
// 		if err != nil {
// 			continue
// 		}
//
// 		if attributes[org] == organization || organization == "*" {
// 			attributes[blocked] = false
// 			cu.nodeService.Put(*node)
// 		}
// 	}
//
// 	return nil
// }

//////////////
// Private  //
//////////////

func (cu *ChefUpdater) fetchLicenses() error {
	items, err :=
		cu.dataBagService.GetItem(cu.config.DataBagName, cu.config.DataBagItem)
	if err != nil {
		return errors.New("Couldn't get items from data bag: " + err.Error())
	}

	sensorsIf, ok := items.(map[string]interface{})
	if !ok {
		return errors.New("Couldn't get sensors from data bag")
	}

	sensors, ok := sensorsIf["sensors"].(map[string]interface{})
	if !ok {
		return errors.New("Couldn't get sensors from data bag. Failed assertion " +
			"to \"map[string]interface{}\"")
	}

	for k, v := range sensors {
		if node, ok := cu.nodes[k]; ok {
			a := v.(map[string]interface{})["license"]
			setNodeAttribute(node.NormalAttributes, cu.config.LicenseUUIDPath, a)
		}
	}

	return nil
}

// FetchNodes updates the internal node database and keep it in memory
func (cu *ChefUpdater) fetchNodes() error {
	nodeList, err := cu.nodeService.List()

	if err != nil {
		return errors.New("Couldn't list nodes: " + err.Error())
	}

	for n := range nodeList {
		node, err := cu.nodeService.Get(n)
		if err != nil {
			return errors.New("Error getting node info: " + err.Error())
		}

		sensorUUID, err :=
			getNodeAttribute(node.NormalAttributes, cu.config.SensorUUIDPath)
		if err != nil {
			return errors.New(
				"Error getting node attribute: " +
					cu.config.SensorUUIDPath + "" + err.Error(),
			)
		}

		if k, ok := sensorUUID.(string); ok {
			cu.nodes[k] = &node
		} else {
			return errors.New(
				"Error getting node " + cu.config.SensorUUIDPath + ": " + err.Error(),
			)
		}
	}

	return nil
}

///////////////
// Functions //
///////////////

// getParent receives the root object containing the attributes of the
// node and returns the inner object given a path
func getNodeAttribute(root map[string]interface{}, path string,
) (interface{}, error) {
	var ok bool

	keys := strings.Split(path, "/")
	attrs := root

	for i, key := range keys {
		if i < len(keys)-1 {
			if attrs, ok = attrs[key].(map[string]interface{}); !ok || attrs == nil {
				return nil, errors.New("Cannot find key: " + path)
			}
		}
	}

	ret, ok := attrs[getKeyFromPath(path)]
	if !ok {
		return nil, errors.New("Cannot get attribute")
	}

	return ret, nil
}

// setNodeAttribute receives the root object containing the attributes of the
// node and set a value for a given key, where the key can be a path like
// "organization/acme/sensor_uuid".
func setNodeAttribute(
	root map[string]interface{}, path string, value interface{},
) error {
	var ok bool

	keys := strings.Split(path, "/")
	attrs := root

	for i, key := range keys {
		if i < len(keys)-1 {
			if attrs, ok = attrs[key].(map[string]interface{}); !ok || attrs == nil {
				return errors.New("Cannot find key: " + path)
			}
		}
	}

	attrs[getKeyFromPath(path)] = value

	return nil
}

// func findNode(keyPath string, value string, nodes map[string]*chef.Node,
// ) (node *chef.Node) {
// 	key := getKeyFromPath(keyPath)
//
// 	for _, node := range nodes {
// 		attributes, err := getParent(node.NormalAttributes, keyPath)
// 		if err != nil {
// 			continue
// 		}
//
// 		if attributes[key] == value {
// 			return node
// 		}
// 	}
//
// 	return nil
// }

func getKeyFromPath(path string) string {
	keys := strings.Split(path, "/")
	return keys[len(keys)-1]
}
