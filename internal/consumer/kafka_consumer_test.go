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
	"testing"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

/////////////////
// RdKafkaMock //
/////////////////

type RdKafkaMock struct {
	mock.Mock
}

func (k *RdKafkaMock) NewConsumer(config *kafka.ConfigMap) (c *RdConsumerMock, err error) {
	args := k.Called(config)
	return args.Get(0).(*RdConsumerMock), args.Error(1)
}

////////////////////
// RdConsumerMock //
////////////////////

type RdConsumerMock struct {
	mock.Mock
}

func (rdkafka *RdConsumerMock) SubscribeTopics(topics []string, cb kafka.RebalanceCb) error {
	args := rdkafka.Called(topics, cb)
	return args.Error(0)
}

func (rdkafka *RdConsumerMock) Assign(partitions []kafka.TopicPartition) error {
	args := rdkafka.Called(partitions)
	return args.Error(0)
}

func (rdkafka *RdConsumerMock) Unassign() error {
	args := rdkafka.Called()
	return args.Error(0)
}

func (rdkafka *RdConsumerMock) Events() chan kafka.Event {
	args := rdkafka.Called()
	return args.Get(0).(chan kafka.Event)
}

func (rdkafka *RdConsumerMock) Close() error {
	args := rdkafka.Called()
	return args.Error(0)
}

///////////////
// TestEvent //
///////////////

type TestEvent struct{}

func (e TestEvent) String() string {
	return "Unknown event"
}

//////////////////
// TestConsumer //
//////////////////

func TestConsumer(t *testing.T) {
	Convey("Given a working consumer", t, func() {
		topics := []string{"test"}
		attributes := &kafka.ConfigMap{}

		rdKafka := new(RdKafkaMock)
		rdConsumer := new(RdConsumerMock)

		rdKafka.
			On("NewConsumer", attributes).
			Return(rdConsumer, nil)
		rdConsumer.
			On("SubscribeTopics", topics, mock.AnythingOfType("kafka.RebalanceCb")).
			Return(nil)

		c, err := rdKafka.NewConsumer(attributes)
		assert.NoError(t, err)
		assert.Equal(t, rdConsumer, c)

		consumer, err := NewKafkaNetflowConsumer(
			KakfaConsumerConfig{
				RdConsumer: rdConsumer,
				Topics:     topics,
			})
		assert.NoError(t, err)
		assert.NotNil(t, consumer)
		assert.NotNil(t, consumer.terminate)

		Convey("When a message is received", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Close").Return(nil)

			events <- &kafka.Message{
				Key:   []byte{0x00, 0x00, 0x00, 0x00},
				Value: []byte("payload"),
			}

			Convey("The message should be consumed", func() {
				messages, _ := consumer.Consume()
				msg := <-messages
				So(msg.Data, ShouldResemble, []byte("payload"))
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})

		Convey("When a message is received without key", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Close").Return(nil)

			events <- &kafka.Message{
				Value: []byte("payload"),
			}

			Convey("An error shoud be received", func() {
				_, info := consumer.Consume()
				msg := <-info
				So(msg, ShouldEqual, "Ignored message: Invalid message key")
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})

		Convey("When a partition is assigned", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Close").Return(nil)

			rdConsumer.
				On("Assign", mock.AnythingOfType("[]kafka.TopicPartition")).
				Return(nil)

			topicName := "test"
			partitions := kafka.AssignedPartitions{
				Partitions: kafka.TopicPartitions{
					kafka.TopicPartition{
						Topic:     &topicName,
						Partition: 46,
						Offset:    1000,
					},
				},
			}
			events <- partitions

			Convey("The assignment should be triggered", func() {
				_, info := consumer.Consume()
				msg := <-info
				So(msg, ShouldEqual, "AssignedPartitions: [test[46]@1000]")
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})

		Convey("When a partition is unassigned", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Unassign").Return(nil)
			rdConsumer.On("Close").Return(nil)

			partitions := kafka.RevokedPartitions{}
			events <- partitions

			Convey("The unassignment should be triggered", func() {
				_, info := consumer.Consume()
				msg := <-info
				So(msg, ShouldEqual, "RevokedPartitions: []")
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})

		Convey("When an error ocurred", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Close").Return(nil)

			events <- kafka.Error{}

			Convey("The error should be reported", func() {
				_, info := consumer.Consume()
				msg := <-info
				So(msg, ShouldEqual, "Error: Success")
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})

		Convey("When an unknown event ocurrs", func() {
			events := make(chan kafka.Event, 1)
			rdConsumer.On("Events").Return(events)
			rdConsumer.On("Close").Return(nil)

			events <- TestEvent{}

			Convey("The event should be reported", func() {
				_, info := consumer.Consume()
				msg := <-info
				So(msg, ShouldEqual, "Unknown event")
				consumer.Close()
				rdConsumer.AssertExpectations(t)
			})
		})
	})
}

func TestConsumerFail(t *testing.T) {
	Convey("Given a configuration without topics", t, func() {
		Convey("When a consumer is created", func() {
			rdKafka := new(RdKafkaMock)
			rdConsumer := new(RdConsumerMock)

			attributes := &kafka.ConfigMap{}
			config := KakfaConsumerConfig{
				RdConsumer: rdConsumer,
				Topics:     []string{},
			}

			rdKafka.
				On("NewConsumer", attributes).
				Return(rdConsumer, nil)
			rdConsumer.
				On("SubscribeTopics", []string{}, mock.AnythingOfType("kafka.RebalanceCb")).
				Return(errors.New("No topics provided"))

			c, err := rdKafka.NewConsumer(attributes)
			assert.NoError(t, err)
			assert.Equal(t, rdConsumer, c)

			Convey("Should fail", func() {
				consumer, err := NewKafkaNetflowConsumer(config)
				So(err, ShouldNotBeNil)
				So(consumer, ShouldBeNil)
			})
		})
	})
}
