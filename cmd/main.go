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
	"github.com/redBorder/dynamic-sensors-watcher/internal/consumer"
	"github.com/redBorder/dynamic-sensors-watcher/internal/decoder"
	"github.com/redBorder/dynamic-sensors-watcher/internal/updater"
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
		logrus.Fatal(err)
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	//////////////////////
	// Netflow decoder //
	//////////////////////

	decoderConfig := decoder.Netflow10DecoderConfig{
		ElementID: uint16(config.Decoder.ElementID),
	}
	nfDecoder := decoder.NewNetflow10Decoder(decoderConfig)

	///////////////////
	// Chef updater //
	///////////////////

	lastUpdated := make(map[uint32]time.Time)

	key, err := ioutil.ReadFile(config.Updater.Key)
	if err != nil {
		logrus.Fatal(err)
	}

	chefUpdater, err := updater.NewChefUpdater(updater.ChefUpdaterConfig{
		URL:  config.Updater.URL,
		Key:  string(key),
		Name: config.Updater.NodeName,
		Path: config.Updater.Path,
	})
	if err != nil {
		logrus.Fatal(err)
	}

	err = chefUpdater.FetchNodes()

	ticker := time.NewTicker(
		time.Duration(config.Updater.FetchInterval) * time.Second)
	go func() {
		for range ticker.C {
			err = chefUpdater.FetchNodes()
			if err != nil {
				logrus.Warn(err)
			}
		}
	}()

	////////////////////////////
	// Kafka Netflow consumer //
	////////////////////////////

	consumerConfig, err := BootstrapRdKafka(
		config.Broker.Address,
		config.Broker.ConsumerGroup,
		config.Broker.Topics)
	if err != nil {
		logrus.Fatal(err)
	}

	kafkaConsumer, err := consumer.NewKafkaNetflowConsumer(consumerConfig)
	if err != nil {
		logrus.Fatal(err)
	}

	messages, events := kafkaConsumer.Consume()
	defer kafkaConsumer.Close()

	///////////////////////////
	// Kafka Limits consumer //
	///////////////////////////

	// TODO

	//////////////////////////////////////////////////////////////////////////////
	// Discarded Netflow Processing
	//////////////////////////////////////////////////////////////////////////////

	wg.Add(1)
	go func() {
		for message := range messages {
			deviceID, obsID, err := nfDecoder.Decode(message.IP, message.Data)
			if err != nil {
				logrus.Errorln(err)
				continue
			}

			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, message.IP)

			if deviceID == 0 {
				logrus.Debugf("Message without Device ID from: %s", ip.String())
				continue
			}

			err = chefUpdater.UpdateNode(ip, deviceID, obsID)
			if err != nil {
				logrus.Warn("Error: " + err.Error())
				continue
			}

			if time.Since(lastUpdated[deviceID]) <
				time.Duration(config.Updater.UpdateInterval)*time.Second {
				continue
			}

			lastUpdated[deviceID] = time.Now()
			logrus.Infof(
				"Updated sensor [IP: %s | DEVICE_ID: %d | OBS. Domain ID: %d]",
				ip.String(), deviceID, obsID)
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for event := range events {
			logrus.Debugln(event)
		}
		wg.Done()
	}()

	//////////////////////////////////////////////////////////////////////////////
	// Sensors limits messages
	//////////////////////////////////////////////////////////////////////////////

	// TODO

	//////////////////////////////////////////////////////////////////////////////
	// The End
	//////////////////////////////////////////////////////////////////////////////

	wg.Wait()
	logrus.Infoln("Bye bye...")
}
