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
		Topics        []string `yaml:"topics"`
	}
	Decoder struct {
		ElementID int `yaml:"element_id"`
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
