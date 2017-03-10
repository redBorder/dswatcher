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
	"fmt"
	"runtime"

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/dynamic-sensors-watcher/internal/consumer"
)

// PrintVersion displays the application version.
func PrintVersion() {
	_, s := rdkafka.LibraryVersion()
	fmt.Printf("Dynamic Sensors Watcher\t:: %s\n", version)
	fmt.Printf("Go\t\t\t:: %s\n", runtime.Version())
	fmt.Printf("librdkafka\t\t:: %s\n", s)
}

// BootstrapRdKafka creates a Kafka consumer configuration struct.
func BootstrapRdKafka(
	broker, consumerGroup string,
	nfTopics []string,
	limitsTopics []string,
	additionalAttributes ...string,
) (config consumer.KakfaConsumerConfig, err error) {

	nfAttributes := &rdkafka.ConfigMap{
		"bootstrap.servers":               broker,
		"group.id":                        consumerGroup,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
	}
	limitsAttributes := &rdkafka.ConfigMap{
		"bootstrap.servers":               broker,
		"group.id":                        consumerGroup,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
	}
	for _, attr := range additionalAttributes {
		nfAttributes.Set(attr)
		limitsAttributes.Set(attr)
	}

	nfConsumer, err := rdkafka.NewConsumer(nfAttributes)
	if err != nil {
		return
	}
	limitsConsumer, err := rdkafka.NewConsumer(limitsAttributes)
	if err != nil {
		return
	}

	config = consumer.KakfaConsumerConfig{
		NetflowConsumer: nfConsumer,
		NetflowTopics:   nfTopics,

		LimitsConsumer: limitsConsumer,
		LimitsTopics:   limitsTopics,
	}

	return
}
