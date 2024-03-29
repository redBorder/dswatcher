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

package main

import (
	"errors"

	yaml "gopkg.in/yaml.v2"
)

// DynamicSensorsWatcherConfig contains the main application configuration
type DynamicSensorsWatcherConfig struct {
	Broker struct {
		Address       string   `yaml:"address"`
		ConsumerGroup string   `yaml:"consumer_group"`
		NetflowTopics []string `yaml:"netflow_topics"`
		LimitsTopics  []string `yaml:"limits_topics"`
	}

	Decoder struct {
		ElementID           int `yaml:"element_id"`
		DeviceTypeElementID int `yaml:"product_type_element_id"`
		OptionTemplateID    int `yaml:"option_template_id"`
	}

	Updater struct {
		URL                  string `yaml:"chef_server_url"`
		Key                  string `yaml:"client_key"`
		NodeName             string `yaml:"node_name"`
		SerialNumberPath     string `yaml:"serial_number_path"`
		ObservationIDPath    string `yaml:"observation_id_path"`
		IPAddressPath        string `yaml:"ipaddress_path"`
		SensorUUIDPath       string `yaml:"sensor_uuid_path"`
		BlockedStatusPath    string `yaml:"blocked_status_path"`
		ProductTypePath      string `yaml:"product_type_path"`
		OrganizationUUIDPath string `yaml:"organization_uuid_path"`
		LicenseUUIDPath      string `yaml:"license_uuid_path"`
		DataBagName          string `yaml:"data_bag_name"`
		DataBagItem          string `yaml:"data_bag_item"`
		UpdateInterval       int64  `yaml:"update_interval_s"`
		FetchInterval        int64  `yaml:"fetch_interval_s"`
		SkipSSL              bool   `yaml:"skip_ssl"`
	}
}

// ParseConfig parse a YAML formatted string and returns a
// DynamicSensorsWatcherConfig struct containing the parsed configuration.
func ParseConfig(raw []byte) (DynamicSensorsWatcherConfig, error) {
	config := DynamicSensorsWatcherConfig{}
	err := yaml.Unmarshal(raw, &config)
	if err != nil {
		return config, errors.New("Error: " + err.Error())
	}

	return config, nil
}
