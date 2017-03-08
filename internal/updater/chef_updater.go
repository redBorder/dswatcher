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
	"encoding/json"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
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

	Name  string
	URL   string
	Key   string
	IDKey string
	Path  string
}

// ChefUpdater uses the Chef client API to update a sensor node with an IP
// address.
type ChefUpdater struct {
	nodes map[uint32]chef.Node

	ChefUpdaterConfig
}

// NewChefUpdater creates a new instance of a ChefUpdater.
func NewChefUpdater(config ChefUpdaterConfig) (*ChefUpdater, error) {
	updater := &ChefUpdater{
		nodes:             make(map[uint32]chef.Node),
		ChefUpdaterConfig: config,
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    updater.Name,
		Key:     updater.Key,
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
	keys := strings.Split(cu.Path, "/")
	deviceIDKey := keys[len(keys)-1]

	nodeList, err := cu.client.Nodes.List()
	if err != nil {
		return errors.New("Couldn't list nodes: " + err.Error())
	}

	for node := range nodeList {
		node, err := cu.client.Nodes.Get(node)
		if err != nil {
			return errors.New("Error getting node info: " + err.Error())
		}

		attributes, err := getAttributes(node.NormalAttributes, cu.Path)
		if err != nil {
			return errors.New("Error getting node info: " + err.Error())
		}

		deviceIDStr, ok := attributes[deviceIDKey].(string)
		if !ok {
			continue
		}

		deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
		if err != nil {
			logrus.Warn(err)
			continue
		}

		cu.nodes[uint32(deviceID)] = node
	}

	return nil
}

// UpdateNode gets a list of nodes an look for one with the given address. If a
// node is found will update the deviceID.
// If a node with the given address is not found an error is returned
func (cu *ChefUpdater) UpdateNode(address net.IP, deviceID, obsID uint32) error {
	node := cu.nodes[deviceID]

	attributes, err := getAttributes(node.NormalAttributes, cu.Path)
	if err != nil {
		return errors.New("Updating node: " + err.Error())
	}

	attributes["ipaddress"] = address.String()
	attributes["observation_domain_id"] = strconv.FormatUint(uint64(obsID), 10)

	cu.client.Nodes.Put(node)
	if err != nil {
		return errors.New("Updating node: " + err.Error())
	}

	return nil
}

// getAttributes receives the root object containing all the attributes of the
// node and returns the inner object given a path
func getAttributes(
	attributes map[string]interface{}, path string,
) (map[string]interface{}, error) {
	var ok bool
	keys := strings.Split(path, "/")

	attrs := attributes
	for i := 0; i < len(keys)-1; i++ {
		attrs, ok = attributes[keys[i]].(map[string]interface{})
		if !ok {
			return nil, errors.New("Cannot find key: " + path)
		}
	}

	return attrs, nil
}

func printNode(node chef.Node) error {
	jsonData, err := json.MarshalIndent(node, "", "\t")
	if err != nil {
		return err
	}

	os.Stdout.Write(jsonData)
	os.Stdout.WriteString("\n")

	return nil
}
