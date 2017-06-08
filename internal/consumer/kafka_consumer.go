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
	"encoding/json"
	"errors"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type limitMessage struct {
	Monitor      string `yaml:"monitor"`
	Type         string `yaml:"type"`
	UUID         string `yaml:"uuid"`
	CurrentBytes string `yaml:"current_bytes"`
	Limit        string `yaml:"limit"`
	Timestamp    int64  `yaml:"timestamp"`
}

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
	NetflowConsumer RdKafkaConsumer
	LimitsConsumer  RdKafkaConsumer
	NetflowTopics   []string
	LimitsTopics    []string
}

///////////////////
// KafkaConsumer //
///////////////////

// KafkaConsumer implements "Consumer" and consumes messages from a Kafka broker
type KafkaConsumer struct {
	terminate chan struct{}

	KakfaConsumerConfig
}

// NewKafkaConsumer creates a new instance of a Kafka consumer and subscribes
// to the provided topics
func NewKafkaConsumer(config KakfaConsumerConfig) (kc *KafkaConsumer, err error) {
	kc = &KafkaConsumer{
		terminate:           make(chan struct{}),
		KakfaConsumerConfig: config,
	}

	if kc.NetflowConsumer != nil {
		err = kc.NetflowConsumer.SubscribeTopics(config.NetflowTopics, nil)
		if err != nil {
			return nil, errors.New("Error on subscription to topics: " + err.Error())
		}
	}

	if kc.LimitsConsumer != nil {
		err = kc.LimitsConsumer.SubscribeTopics(config.LimitsTopics, nil)
		if err != nil {
			return nil, errors.New("Error on subscription to topics: " + err.Error())
		}
	}

	return
}

// ConsumeNetflow receives netflow from the kafka broker. "messages" channel
// receives actual messages and "info" channel receives notifications from the
// Kafka broker.
func (kc *KafkaConsumer) ConsumeNetflow() (chan FlowData, chan string) {
	messages := make(chan FlowData)
	inputMessages, info := receiveLoop(kc.NetflowConsumer, kc.terminate)

	go func() {
		for m := range inputMessages {
			if len(m.Key) != 4 {
				info <- "Ignored message: Invalid message key"
				continue
			}
			messages <- FlowData{
				IP:   binary.LittleEndian.Uint32(m.Key),
				Data: m.Value,
			}
		}

		kc.NetflowConsumer.Close()
		close(messages)
		close(kc.terminate)
	}()

	return messages, info
}

// ConsumeLimits receives limits messages from the kafka broker.
// "messages" channel receives actual messages and "info" channel receives
// notifications from the Kafka broker.
func (kc *KafkaConsumer) ConsumeLimits() (chan Message, chan string) {
	messages := make(chan Message)
	inputMessages, info := receiveLoop(kc.LimitsConsumer, kc.terminate)

	go func() {
		for m := range inputMessages {
			var data limitMessage
			err := json.Unmarshal(m.Value, &data)

			if err != nil {
				info <- err.Error()
				continue
			}

			switch data.Type {
			case "limit_reached":
				messages <- UUID(data.UUID)

			case "counters_reset":
				messages <- ResetSignal{
					data.UUID,
				}

			default:
				info <- "Unknown alert received"
			}
		}

		kc.LimitsConsumer.Close()
		close(messages)
		close(kc.terminate)
	}()

	return messages, info
}

// Close terminates the rdkafka consumer
func (kc *KafkaConsumer) Close() {
	kc.terminate <- struct{}{}
	<-kc.terminate
}

func receiveLoop(
	consumer RdKafkaConsumer,
	terminate <-chan struct{},
) (messages chan *kafka.Message, info chan string) {
	messages = make(chan *kafka.Message)
	info = make(chan string)

	go func() {
	receiving:
		for {
			select {
			case <-terminate:
				break receiving

			case ev := <-consumer.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					consumer.Assign(e.Partitions)
					info <- e.String()

				case kafka.RevokedPartitions:
					consumer.Unassign()
					info <- e.String()

				case kafka.Error:
					info <- "Error: " + e.String()

				case *kafka.Message:
					messages <- e

				default:
					info <- e.String()
				}
			}
		}

		close(messages)
		close(info)
	}()

	return messages, info
}
