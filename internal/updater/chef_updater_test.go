// Service for allowing new sensors to send flow based on a serial number.
// Copyright (C) 2017 ENEO Tecnologia SL
// Author: Diego Fernández Barrera <bigomby@gmail.com>
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
	"testing"

	"github.com/go-chef/chef"
	"github.com/redBorder/dswatcher/internal/updater/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	TestDataBagName = "databag_name"
	TestDataBagItem = "databag_item"

	TestSensorUUIDPath    = "organization/sensor_uuid"
	TestLicenseUUIDPath   = "organization/license_uuid"
	TestBlockedStatusPath = "organization/blocked"
)

/////////////
// Helpers //
/////////////

func BootstrapServices() (NodeService, DataBagService) {
	nodeService := &mocks.NodeService{}
	dataBagService := &mocks.DataBagService{}

	nodeService.On("List").Return(map[string]string{
		"latte_name": "",
		"coffe_name": "",
		"sugar_name": "",
	}, nil)

	nodeService.On("Get", "latte_name").Return(chef.Node{
		NormalAttributes: map[string]interface{}{
			"organization": map[string]interface{}{
				"sensor_uuid": "latte_uuid",
			},
		},
	}, nil)
	nodeService.On("Get", "coffe_name").Return(chef.Node{
		NormalAttributes: map[string]interface{}{
			"organization": map[string]interface{}{
				"sensor_uuid": "coffe_uuid",
			},
		},
	}, nil)
	nodeService.On("Get", "sugar_name").Return(chef.Node{
		NormalAttributes: map[string]interface{}{
			"organization": map[string]interface{}{
				"sensor_uuid": "sugar_uuid",
			},
		},
	}, nil)

	dataBagService.
		On("GetItem", "databag_name", "databag_item").
		Return(map[string]interface{}{
			"sensors": map[string]interface{}{
				"latte_uuid": map[string]interface{}{
					"license": "license1_uuid",
				},
				"coffe_uuid": map[string]interface{}{
					"license": "license1_uuid",
				},
				"sugar_uuid": map[string]interface{}{
					"license": "license2_uuid",
				},
			},
		}, nil)

	return nodeService, dataBagService
}

///////////
// TESTS //
///////////

func TestGetKeyFromPath(t *testing.T) {
	path := "lorem/ipsum/dolor/sit"
	key := getKeyFromPath(path)
	assert.Equal(t, "sit", key)
}

func TestCreateUpdaterWithoutServices(t *testing.T) {
	nodeService := &mocks.NodeService{}
	dataBagService := &mocks.DataBagService{}

	f1 := func() {
		NewChefUpdater(&ChefUpdaterConfig{
			NodeService: nodeService,
		})
	}
	f2 := func() {
		NewChefUpdater(&ChefUpdaterConfig{
			DataBagService: dataBagService,
		})
	}

	assert.Panics(t, f1)
	assert.Panics(t, f2)
}

func TestCreateUpdater(t *testing.T) {
	nodeService := &mocks.NodeService{}
	dataBagService := &mocks.DataBagService{}

	f := func() {
		NewChefUpdater(&ChefUpdaterConfig{
			NodeService:    nodeService,
			DataBagService: dataBagService,
		})
	}

	assert.NotPanics(t, f)
}

func TestFetchNodes(t *testing.T) {
	nodeService, dataBagService := BootstrapServices()

	updater := NewChefUpdater(&ChefUpdaterConfig{
		SensorUUIDPath: TestSensorUUIDPath,
		NodeService:    nodeService,
		DataBagService: dataBagService,
	})

	updater.fetchNodes()

	assert.NotNil(t, updater.nodes["latte_uuid"])
	assert.NotNil(t, updater.nodes["coffe_uuid"])
	assert.NotNil(t, updater.nodes["sugar_uuid"])
}

func TestFetchLicenses(t *testing.T) {
	nodeService, dataBagService := BootstrapServices()

	updater := NewChefUpdater(&ChefUpdaterConfig{
		SensorUUIDPath:    TestSensorUUIDPath,
		DataBagName:       TestDataBagName,
		DataBagItem:       TestDataBagItem,
		LicenseUUIDPath:   TestLicenseUUIDPath,
		BlockedStatusPath: TestBlockedStatusPath,

		NodeService:    nodeService,
		DataBagService: dataBagService,
	})

	updater.nodes = map[string]*chef.Node{
		"latte_uuid": &chef.Node{
			"organization": map[string]interface{}{
				NormalAttributes: make(map[string]interface{}),
			},
		},
		"coffe_uuid": &chef.Node{
			"organization": map[string]interface{}{
				NormalAttributes: make(map[string]interface{}),
			},
		},
		"sugar_uuid": &chef.Node{
			"organization": map[string]interface{}{
				NormalAttributes: make(map[string]interface{}),
			},
		},
	}

	updater.fetchLicenses()

	node1, err := getNodeAttribute(
		updater.nodes["latte_uuid"].NormalAttributes,
		TestLicenseUUIDPath,
	)
	assert.NoError(t, err)
	assert.Equal(t, "license1_uuid", node1)
}

// func TestFindNode(t *testing.T) {
// 	nodes := bootstrapSensorsDB()
//
// 	node := findNode("org/uuid", "0000", nodes)
// 	assert.Equal(t, nodes["0"], node)
//
// 	node = findNode("org2/uuid", "1111", nodes)
// 	assert.Equal(t, nodes["1"], node)
//
// 	node = findNode("uuid", "9999", nodes)
// 	assert.Equal(t, nodes["3"], node)
//
// 	node = findNode("org/uuid", "1234", nodes)
// 	assert.Nil(t, node)
//
// 	node = findNode("org", "", nodes)
// 	assert.Nil(t, node)
// }
//
// func TestBlockOrganization(t *testing.T) {
// 	chefUpdater, err := NewChefUpdater(ChefUpdaterConfig{
// 		AccessKey:            testPEMKey,
// 		Name:                 "test",
// 		SensorUUIDPath:       "org/uuid",
// 		BlockedStatusPath:    "org/blocked",
// 		OrganizationUUIDPath: "org/organization_uuid",
// 		ProductTypePath:      "org/product_type",
// 	})
// 	assert.NoError(t, err)
//
// 	chefUpdater.nodes = bootstrapSensorsDB()
//
// 	var attributes map[string]interface{}
// 	var ok bool
//
// 	attributes, err = getParent(
// 		chefUpdater.nodes["0"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
//
// 	errs := chefUpdater.BlockOrganization("abcde", 123)
// 	assert.Equal(t, 2, len(errs))
//
// 	assert.NoError(t, err)
// 	assert.False(t, attributes["blocked"].(bool))
//
// 	errs = chefUpdater.BlockOrganization("abcde", 999)
// 	assert.Equal(t, 2, len(errs))
//
// 	assert.True(t, attributes["blocked"].(bool))
//
// 	attributes, err = getParent(
// 		chefUpdater.nodes["1"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
//
// 	assert.Error(t, err)
// 	_, ok = attributes["blocked"].(bool)
// 	assert.False(t, ok)
//
// 	attributes, err = getParent(
// 		chefUpdater.nodes["2"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
//
// 	assert.NoError(t, err)
// 	assert.False(t, attributes["blocked"].(bool))
//
// 	attributes, err = getParent(
// 		chefUpdater.nodes["3"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
//
// 	assert.Error(t, err)
// 	_, ok = attributes["blocked"].(bool)
// 	assert.False(t, ok)
// }
//
// func TestResetSensors(t *testing.T) {
// 	chefUpdater := &ChefUpdater{
// 		nodes: bootstrapSensorsDB(),
// 		config: ChefUpdaterConfig{
// 			AccessKey:            testPEMKey,
// 			Name:                 "test",
// 			SensorUUIDPath:       "org/uuid",
// 			BlockedStatusPath:    "org/blocked",
// 			OrganizationUUIDPath: "organization_uuid",
// 		},
// 	}
//
// 	attributes0, err := getParent(
// 		chefUpdater.nodes["0"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
// 	assert.NoError(t, err)
// 	attributes2, err := getParent(
// 		chefUpdater.nodes["2"].NormalAttributes,
// 		chefUpdater.config.BlockedStatusPath)
// 	assert.NoError(t, err)
//
// 	attributes0["blocked"] = true
// 	attributes2["blocked"] = true
//
// 	chefUpdater.ResetSensors("abcde")
//
// 	assert.False(t, attributes0["blocked"].(bool))
// 	assert.True(t, attributes2["blocked"].(bool))
// }
//
// func TestUpdateNode(t *testing.T) {
// 	chefUpdater := &ChefUpdater{
// 		nodes: bootstrapSensorsDB(),
// 		config: ChefUpdaterConfig{
// 			AccessKey:        testPEMKey,
// 			Name:             "test",
// 			SensorUUIDPath:   "org/uuid",
// 			ProductTypePath:  "org/product_type",
// 			SerialNumberPath: "org/serial_number",
// 			IPAddressPath:    "org/ipaddress",
// 		},
// 	}
//
// 	address := make(net.IP, 4)
// 	err := chefUpdater.UpdateNode(address, "888888", 10, 999)
// 	assert.NoError(t, err)
//
// 	attrs, err := getParent(chefUpdater.nodes["0"].NormalAttributes,
// 		chefUpdater.config.SensorUUIDPath)
// 	assert.NoError(t, err)
//
// 	ip, ok := attrs["ipaddress"].(string)
// 	assert.True(t, ok)
// 	assert.Equal(t, address.String(), ip)
// }
//
// func TestUpdateNodeWithLicenses(t *testing.T) {
// 	chefUpdater := &ChefUpdater{
// 		nodes: bootstrapSensorsDB(),
// 		config: ChefUpdaterConfig{
// 			AccessKey:        testPEMKey,
// 			Name:             "test",
// 			SensorUUIDPath:   "org/uuid",
// 			ProductTypePath:  "org/product_type",
// 			SerialNumberPath: "org/serial_number",
// 			IPAddressPath:    "org/ipaddress",
// 		},
// 	}
//
// 	address := make(net.IP, 4)
// 	err := chefUpdater.UpdateNode(address, "888888", 10, 999)
// 	assert.NoError(t, err)
//
// 	attrs, err := getParent(chefUpdater.nodes["0"].NormalAttributes,
// 		chefUpdater.config.SensorUUIDPath)
// 	assert.NoError(t, err)
//
// 	ip, ok := attrs["ipaddress"].(string)
// 	assert.True(t, ok)
// 	assert.Equal(t, address.String(), ip)
// }
//
// func TestUpdateNodeError(t *testing.T) {
// 	chefUpdater := &ChefUpdater{
// 		nodes: bootstrapSensorsDB(),
// 		config: ChefUpdaterConfig{
// 			AccessKey:        testPEMKey,
// 			Name:             "test",
// 			SensorUUIDPath:   "org2/uuid",
// 			ProductTypePath:  "org2/device_id",
// 			SerialNumberPath: "org2/serial_number",
// 			IPAddressPath:    "org2/ipaddress",
// 		},
// 	}
//
// 	address := make(net.IP, 4)
// 	err := chefUpdater.UpdateNode(address, "777777", 10, 224)
// 	assert.Error(t, err)
//
// 	attrs, err := getParent(chefUpdater.nodes["1"].NormalAttributes,
// 		chefUpdater.config.SensorUUIDPath)
// 	assert.NoError(t, err)
//
// 	_, ok := attrs["ipaddress"].(string)
// 	assert.False(t, ok)
// }
