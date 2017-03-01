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
	"encoding/binary"
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
	Close() error
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

// KafkaFlowConsumer implements "Consumer" and consumes messages from a Kafka broker
type KafkaFlowConsumer struct {
	terminate chan struct{}

	KakfaConsumerConfig
}

// NewKafkaNetflowConsumer creates a new instance of a Kafka consumer and subscribes
// to the provided topics
func NewKafkaNetflowConsumer(config KakfaConsumerConfig) (kc *KafkaFlowConsumer, err error) {
	kc = &KafkaFlowConsumer{
		terminate:           make(chan struct{}),
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
func (kc *KafkaFlowConsumer) Consume() (messages chan FlowData, info chan string) {
	messages = make(chan FlowData, 100)
	info = make(chan string)

	go func() {
	receiving:
		for {
			select {
			case <-kc.terminate:
				break receiving

			case ev := <-kc.RdConsumer.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					kc.RdConsumer.Assign(e.Partitions)
					info <- "Partition assignment ocurred"

				case kafka.RevokedPartitions:
					kc.RdConsumer.Unassign()
					info <- "Partition unassign ocurred"

				case *kafka.Message:
					if len(e.Key) != 4 {
						info <- "Invalid message key"
						continue
					}
					messages <- FlowData{
						IP:   binary.BigEndian.Uint32(e.Key),
						Data: e.Value,
					}

				case kafka.Error:
					info <- "Error: " + e.String()

				default:
					info <- "Unknown event received"
				}
			}
		}

		kc.RdConsumer.Close()
		info <- "Consumer terminated"
	}()

	return
}

// Close terminates the rdkafka consumer
func (kc *KafkaFlowConsumer) Close() {
	kc.terminate <- struct{}{}
}
