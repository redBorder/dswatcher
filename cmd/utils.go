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

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/dynamic-sensors-watcher/internal/consumer"
)

// PrintVersion displays the application version.
func PrintVersion() {
	fmt.Println(version)
}

// BootstrapRdKafka creates a Kafka consumer configuration struct.
func BootstrapRdKafka(
	broker, consumerGroup string,
	topics []string,
	additionalAttributes ...string,
) (config consumer.KakfaConsumerConfig, err error) {
	attributes := &rdkafka.ConfigMap{
		"bootstrap.servers":               broker,
		"group.id":                        consumerGroup,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
	}
	for _, attr := range additionalAttributes {
		attributes.Set(attr)
	}

	rdconsumer, err := rdkafka.NewConsumer(attributes)
	if err != nil {
		return
	}

	config = consumer.KakfaConsumerConfig{
		Topics:     topics,
		RdConsumer: rdconsumer,
	}

	return
}
