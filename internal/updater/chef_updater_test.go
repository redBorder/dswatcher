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
	"net"
	"testing"

	"github.com/go-chef/chef"
	"github.com/stretchr/testify/assert"
)

var testPEMKey = `
-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0FPqri0cb2JZfXJ/DgYSF6vUp
wmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/3j+skZ6UtW+5u09lHNsj6tQ5
1s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQABAoGAFijko56+qGyN8M0RVyaRAXz++xTqHBLh
3tx4VgMtrQ+WEgCjhoTwo23KMBAuJGSYnRmoBZM3lMfTKevIkAidPExvYCdm5dYq3XToLkkLv5L2
pIIVOFMDG+KESnAFV7l2c+cnzRMW0+b6f8mR1CJzZuxVLL6Q02fvLi55/mbSYxECQQDeAw6fiIQX
GukBI4eMZZt4nscy2o12KyYner3VpoeE+Np2q+Z3pvAMd/aNzQ/W9WaI+NRfcxUJrmfPwIGm63il
AkEAxCL5HQb2bQr4ByorcMWm/hEP2MZzROV73yF41hPsRC9m66KrheO9HPTJuo3/9s5p+sqGxOlF
L0NDt4SkosjgGwJAFklyR1uZ/wPJjj611cdBcztlPdqoxssQGnh85BzCj/u3WqBpE2vjvyyvyI5k
X6zk7S0ljKtt2jny2+00VsBerQJBAJGC1Mg5Oydo5NwD6BiROrPxGo2bpTbu/fhrT8ebHkTz2epl
U9VQQSQzY1oZMVX8i1m5WUTLPz2yLJIBQVdXqhMCQBGoiuSoSjafUhV7i1cEGpb88h5NBYZzWXGZ
37sJ5QsW+sJyoNde3xH8vdXhzU7eT82D6X/scw9RZz+/6rCJ4p0=
-----END RSA PRIVATE KEY-----`

func bootstrapSensorsDB() map[string]*chef.Node {
	nodes := make(map[string]*chef.Node)
	nodes["0"] = &chef.Node{
		NormalAttributes: map[string]interface{}{
			"org": map[string]interface{}{
				"uuid":          "0000",
				"serial_number": "888888",
				"device_id":     "224",
			},
		},
	}
	nodes["1"] = &chef.Node{
		NormalAttributes: map[string]interface{}{
			"org2": map[string]interface{}{
				"uuid":          "1111",
				"serial_number": "777777",
				"device_id":     "123",
			},
		},
	}

	nodes["2"] = &chef.Node{
		NormalAttributes: map[string]interface{}{
			"org": map[string]interface{}{},
		},
	}

	nodes["3"] = &chef.Node{
		NormalAttributes: map[string]interface{}{
			"uuid": "9999",
		},
	}

	return nodes
}

func TestGetKeyFromPath(t *testing.T) {
	path := "lorem/ipsum/dolor/sit"
	key := getKeyFromPath(path)
	assert.Equal(t, "sit", key)
}

func TestFindNode(t *testing.T) {
	nodes := bootstrapSensorsDB()

	node := findNode("org/uuid", "0000", nodes)
	assert.Equal(t, nodes["0"], node)

	node = findNode("org2/uuid", "1111", nodes)
	assert.Equal(t, nodes["1"], node)

	node = findNode("uuid", "9999", nodes)
	assert.Equal(t, nodes["3"], node)

	node = findNode("org/uuid", "1234", nodes)
	assert.Nil(t, node)

	node = findNode("org", "", nodes)
	assert.Nil(t, node)
}

func TestBlockSensors(t *testing.T) {
	chefUpdater, err := NewChefUpdater(ChefUpdaterConfig{
		AccessKey:         testPEMKey,
		Name:              "test",
		SensorUUIDPath:    "org/uuid",
		BlockedStatusPath: "org/blocked",
	})
	assert.NoError(t, err)

	chefUpdater.nodes = bootstrapSensorsDB()

	blocked, err := chefUpdater.BlockSensor("0000")
	assert.NoError(t, err)
	attributes, err := getParent(
		chefUpdater.nodes["0"].NormalAttributes,
		chefUpdater.BlockedStatusPath)
	assert.NoError(t, err)
	assert.True(t, attributes["blocked"].(bool))
	assert.True(t, blocked)

	blocked, err = chefUpdater.BlockSensor("0000")
	assert.NoError(t, err)
	attributes, err = getParent(
		chefUpdater.nodes["0"].NormalAttributes,
		chefUpdater.BlockedStatusPath)
	assert.NoError(t, err)
	assert.True(t, attributes["blocked"].(bool))
	assert.False(t, blocked)

	blocked, err = chefUpdater.BlockSensor("7777")
	assert.Error(t, err)
	assert.False(t, blocked)
}

func TestBlockAllSensors(t *testing.T) {
	chefUpdater, err := NewChefUpdater(ChefUpdaterConfig{
		AccessKey:         testPEMKey,
		Name:              "test",
		SensorUUIDPath:    "org/uuid",
		BlockedStatusPath: "org/blocked",
	})
	assert.NoError(t, err)

	chefUpdater.nodes = bootstrapSensorsDB()

	errs := chefUpdater.BlockAllSensors()
	assert.Equal(t, 2, len(errs))

	var attributes map[string]interface{}
	var ok bool

	attributes, err = getParent(
		chefUpdater.nodes["0"].NormalAttributes,
		chefUpdater.BlockedStatusPath)

	assert.NoError(t, err)
	assert.True(t, attributes["blocked"].(bool))

	attributes, err = getParent(
		chefUpdater.nodes["1"].NormalAttributes,
		chefUpdater.BlockedStatusPath)

	assert.Error(t, err)
	_, ok = attributes["blocked"].(bool)
	assert.False(t, ok)

	attributes, err = getParent(
		chefUpdater.nodes["2"].NormalAttributes,
		chefUpdater.BlockedStatusPath)

	assert.NoError(t, err)
	assert.True(t, attributes["blocked"].(bool))

	attributes, err = getParent(
		chefUpdater.nodes["3"].NormalAttributes,
		chefUpdater.BlockedStatusPath)

	assert.Error(t, err)
	_, ok = attributes["blocked"].(bool)
	assert.False(t, ok)
}

func TestResetSensors(t *testing.T) {
	chefUpdater := &ChefUpdater{
		nodes: bootstrapSensorsDB(),
		ChefUpdaterConfig: ChefUpdaterConfig{
			AccessKey:         testPEMKey,
			Name:              "test",
			SensorUUIDPath:    "org/uuid",
			BlockedStatusPath: "org/blocked",
		},
	}

	attributes0, err := getParent(
		chefUpdater.nodes["0"].NormalAttributes,
		chefUpdater.BlockedStatusPath)
	assert.NoError(t, err)
	attributes2, err := getParent(
		chefUpdater.nodes["2"].NormalAttributes,
		chefUpdater.BlockedStatusPath)
	assert.NoError(t, err)

	attributes0["blocked"] = true
	attributes2["blocked"] = true

	chefUpdater.ResetSensors()

	assert.False(t, attributes0["blocked"].(bool))
	assert.False(t, attributes2["blocked"].(bool))
}

func TestUpdateNode(t *testing.T) {
	chefUpdater := &ChefUpdater{
		nodes: bootstrapSensorsDB(),
		ChefUpdaterConfig: ChefUpdaterConfig{
			AccessKey:        testPEMKey,
			Name:             "test",
			SensorUUIDPath:   "org/uuid",
			ProductTypePath:  "org/device_id",
			SerialNumberPath: "org/serial_number",
			IPAddressPath:    "org/ipaddress",
		},
	}

	address := make(net.IP, 4)
	err := chefUpdater.UpdateNode(address, "888888", 10, 224)
	assert.NoError(t, err)

	attrs, err := getParent(chefUpdater.nodes["0"].NormalAttributes,
		chefUpdater.SensorUUIDPath)
	assert.NoError(t, err)

	ip, ok := attrs["ipaddress"].(string)
	assert.True(t, ok)
	assert.Equal(t, address.String(), ip)
}

func TestUpdateNodeError(t *testing.T) {
	chefUpdater := &ChefUpdater{
		nodes: bootstrapSensorsDB(),
		ChefUpdaterConfig: ChefUpdaterConfig{
			AccessKey:        testPEMKey,
			Name:             "test",
			SensorUUIDPath:   "org2/uuid",
			ProductTypePath:  "org2/device_id",
			SerialNumberPath: "org2/serial_number",
			IPAddressPath:    "org2/ipaddress",
		},
	}

	address := make(net.IP, 4)
	err := chefUpdater.UpdateNode(address, "777777", 10, 224)
	assert.Error(t, err)

	attrs, err := getParent(chefUpdater.nodes["1"].NormalAttributes,
		chefUpdater.SensorUUIDPath)
	assert.NoError(t, err)

	_, ok := attrs["ipaddress"].(string)
	assert.False(t, ok)
}
