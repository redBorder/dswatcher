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
	"encoding/binary"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/dswatcher/internal/consumer"
	"github.com/redBorder/dswatcher/internal/decoder"
	"github.com/redBorder/dswatcher/internal/updater"
)

var version string
var configFile string

func init() {
	versionFlag := flag.Bool("version", false, "Show version info")
	debugFlag := flag.Bool("debug", false, "Show debug info")
	configFlag := flag.String("config", "", "Application configuration file")
	flag.Parse()

	if *versionFlag {
		PrintVersion()
		os.Exit(0)
	}

	if len(*configFlag) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *debugFlag {
		logrus.SetLevel(logrus.DebugLevel)
	}

	configFile = *configFlag
}

func main() {
	wg := new(sync.WaitGroup)

	////////////////////
	// Configuration //
	////////////////////

	rawConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.Fatal("Error opening configuration file: " + err.Error())
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		logrus.Fatal("Error parsing config" + err.Error())
	}

	//////////////////////
	// Netflow decoder //
	//////////////////////

	decoderConfig := decoder.Netflow10DecoderConfig{
		ElementID:        uint16(config.Decoder.ElementID),
		OptionTemplateID: uint16(config.Decoder.OptionTemplateID),
	}
	nfDecoder := decoder.NewNetflow10Decoder(decoderConfig)

	///////////////////
	// Chef updater //
	///////////////////

	lastUpdated := make(map[string]time.Time)

	key, err := ioutil.ReadFile(config.Updater.Key)
	if err != nil {
		logrus.Fatal("Error reading client Key: " + err.Error())
	}

	chefUpdater, err := updater.NewChefUpdater(updater.ChefUpdaterConfig{
		URL:               config.Updater.URL,
		AccessKey:         string(key),
		Name:              config.Updater.NodeName,
		SerialNumberPath:  config.Updater.SerialNumberPath,
		SensorUUIDPath:    config.Updater.SensorUUIDPath,
		ObservationIDPath: config.Updater.ObservationIDPath,
		IPAddressPath:     config.Updater.IPAddressPath,
	})
	if err != nil {
		logrus.Fatal("Error creating Chef API client: " + err.Error())
	}

	err = chefUpdater.FetchNodes()
	if err != nil {
		logrus.Errorln("Error fetching nodes: " + err.Error())
	}

	ticker := time.NewTicker(
		time.Duration(config.Updater.FetchInterval) * time.Second)

	go func() {
		for range ticker.C {
			err = chefUpdater.FetchNodes()
			if err != nil {
				logrus.Errorln("Error fetching nodes: " + err.Error())
			}
		}
	}()

	////////////////////
	// Kafka consumer //
	////////////////////

	consumerConfig, err := BootstrapRdKafka(
		config.Broker.Address,
		config.Broker.ConsumerGroup,
		config.Broker.NetflowTopics,
		config.Broker.LimitsTopics,
	)
	if err != nil {
		logrus.Fatal("Error creating Kafka config: " + err.Error())
	}

	kafkaConsumer, err := consumer.NewKafkaConsumer(consumerConfig)
	if err != nil {
		logrus.Fatal("Error creating Kafka consumer: " + err.Error())
	}
	defer kafkaConsumer.Close()

	//////////////////////////////////////////////////////////////////////////////
	// Discarded Netflow Processing
	//////////////////////////////////////////////////////////////////////////////

	nfMessages, nfEvents := kafkaConsumer.ConsumeNetflow()

	wg.Add(1)
	go func() {
		for message := range nfMessages {
			serialNumber, obsID, err := nfDecoder.Decode(message.IP, message.Data)
			if err != nil {
				logrus.Errorln("Error decoding netflow: " + err.Error())
				continue
			}

			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, message.IP)

			if len(serialNumber) == 0 {
				continue
			}

			if time.Since(lastUpdated[serialNumber]) <
				time.Duration(config.Updater.UpdateInterval)*time.Second {
				continue
			}

			lastUpdated[serialNumber] = time.Now()

			err = chefUpdater.UpdateNode(ip, serialNumber, obsID)
			if err != nil {
				logrus.Warnf("Error updating node with serial number %s: %s",
					serialNumber, err.Error())
				continue
			}

			logrus.Infof(
				"Updated sensor [IP: %s | SERIAL_NUMBER: %s | OBS. Domain ID: %d]",
				ip.String(), serialNumber, obsID)
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for event := range nfEvents {
			logrus.Debugln(event)
		}
		wg.Done()
	}()

	//////////////////////////////////////////////////////////////////////////////
	// Sensors limits messages
	//////////////////////////////////////////////////////////////////////////////

	limitsMessages, limitsEvents := kafkaConsumer.ConsumeLimits()

	wg.Add(1)
	go func() {
		for uuid := range limitsMessages {
			blocked, err := chefUpdater.BlockSensor(updater.UUID(uuid))

			if err != nil {
				logrus.Warnf("Error blocking sensor %s: %s", uuid, err.Error())
				continue
			}

			if blocked {
				logrus.Infoln("Blocked UUID: " + uuid)
			}
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for event := range limitsEvents {
			logrus.Debugln(event)
		}

		wg.Done()
	}()

	//////////////////////////////////////////////////////////////////////////////
	// The End
	//////////////////////////////////////////////////////////////////////////////

	wg.Wait()
	logrus.Infoln("Bye bye...")
}
