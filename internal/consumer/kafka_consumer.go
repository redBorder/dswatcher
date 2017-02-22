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

package consumer

import (
	"errors"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

///////////////
//Interfaces //
///////////////

// RdKafkaConsumer is an interface for rdkafka consumer. Used for mocking
// purposes.
type RdKafkaConsumer interface {
	SubscribeTopics([]string, kafka.RebalanceCb) error
	Events() chan kafka.Event
	Assign(partitions []kafka.TopicPartition) error
	Unassign() error
}

/////////////////////////
// KakfaConsumerConfig //
/////////////////////////

// KakfaConsumerConfig contains the configuration for a Kafka Consumer.
type KakfaConsumerConfig struct {
	RdConsumer RdKafkaConsumer
	Topics     []string
}

///////////////////
// KafkaConsumer //
///////////////////

// KafkaConsumer implements "Consumer" and consumes messages from a Kafka broker
type KafkaConsumer struct {
	*KakfaConsumerConfig
}

// NewKafkaConsumer creates a new instance of a Kafka consumer and subscribes
// to the provided topics
func NewKafkaConsumer(config *KakfaConsumerConfig) (kc *KafkaConsumer, err error) {
	kc = &KafkaConsumer{
		KakfaConsumerConfig: config,
	}

	err = kc.RdConsumer.SubscribeTopics(config.Topics, nil)
	if err != nil {
		return nil, errors.New("Error on subscription to topics: " + err.Error())
	}

	return
}

// Consume receives events from the kafka broker. "messages" channel receives
// actual messages and "info" channel receives notifications
func (kc *KafkaConsumer) Consume() (messages chan []byte, info chan string) {
	messages = make(chan []byte, 100)
	info = make(chan string)

	go func() {
		for ev := range kc.RdConsumer.Events() {
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				kc.RdConsumer.Assign(e.Partitions)
				info <- "Partition assignment ocurred"

			case kafka.RevokedPartitions:
				kc.RdConsumer.Unassign()
				info <- "Partition unassign ocurred"

			case *kafka.Message:
				messages <- e.Value

			case kafka.Error:
				info <- "Error: " + e.String()

			default:
				info <- "Unknown event received"
			}
		}
	}()

	return
}
