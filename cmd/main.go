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
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const genericProductType = 999

var (
	version    string
	configFile string
	log        = logrus.New()
)

func init() {
	log.Formatter = &prefixed.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	}

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
		log.Level = logrus.DebugLevel
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
		log.Fatal("Error opening configuration file: " + err.Error())
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		log.Fatal("Error parsing config" + err.Error())
	}

	//////////////////////
	// Netflow decoder //
	//////////////////////

	decoderConfig := decoder.Netflow10DecoderConfig{
		SerialNumberElementID: uint16(config.Decoder.ElementID),
		ProductTypeElementID:  uint16(config.Decoder.DeviceTypeElementID),
		OptionTemplateID:      uint16(config.Decoder.OptionTemplateID),
	}
	nfDecoder := decoder.NewNetflow10Decoder(decoderConfig)

	///////////////////
	// Chef updater //
	///////////////////

	key, err := ioutil.ReadFile(config.Updater.Key)
	if err != nil {
		log.Fatal("Error reading client Key: " + err.Error())
	}

	chefUpdater, err := updater.NewChefUpdater(updater.ChefUpdaterConfig{
		URL:                  config.Updater.URL,
		AccessKey:            string(key),
		Name:                 config.Updater.NodeName,
		SerialNumberPath:     config.Updater.SerialNumberPath,
		SensorUUIDPath:       config.Updater.SensorUUIDPath,
		ObservationIDPath:    config.Updater.ObservationIDPath,
		IPAddressPath:        config.Updater.IPAddressPath,
		BlockedStatusPath:    config.Updater.BlockedStatusPath,
		ProductTypePath:      config.Updater.ProductTypePath,
		OrganizationUUIDPath: config.Updater.OrganizationUUIDPath,
		LicenseUUIDPath:      config.Updater.LicenseUUIDPath,
		DataBagName:          config.Updater.DataBagName,
		DataBagItem:          config.Updater.DataBagItem,
	})
	if err != nil {
		log.Fatal("Error creating Chef API client: " + err.Error())
	}

	err = chefUpdater.FetchNodes()
	if err != nil {
		log.Errorln("Error fetching nodes: " + err.Error())
	}

	fetchSignal :=
		time.NewTicker(time.Duration(config.Updater.FetchInterval) * time.Second)

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
		log.Fatal("Error creating Kafka config: " + err.Error())
	}

	kafkaConsumer, err := consumer.NewKafkaConsumer(consumerConfig)
	if err != nil {
		log.Fatal("Error creating Kafka consumer: " + err.Error())
	}
	defer kafkaConsumer.Close()

	//////////////////////////////////////////////////////////////////////////////
	// Discarded Netflow Processing
	//////////////////////////////////////////////////////////////////////////////

	nfMessages, nfEvents := kafkaConsumer.ConsumeNetflow()

	wg.Add(1)
	log.Infoln("Listening for netflow")
	go func() {
		lastUpdated := make(map[string]time.Time)

		for message := range nfMessages {
			sensor, err := nfDecoder.Decode(message.IP, message.Data)
			if err != nil {
				log.Errorln("Error decoding netflow: " + err.Error())
				continue
			}

			if sensor == nil {
				continue
			}

			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, message.IP)

			if time.Since(lastUpdated[sensor.SerialNumber]) <
				time.Duration(config.Updater.UpdateInterval)*time.Second {
				continue
			}

			lastUpdated[sensor.SerialNumber] = time.Now()

			err = chefUpdater.UpdateNode(
				ip,
				sensor.SerialNumber,
				sensor.ObservationID,
				sensor.ProductType,
			)
			if err != nil {
				log.Warnf("Error updating node [%s | %s]: %s",
					sensor.SerialNumber, ip.String(), err.Error())
				continue
			}

			log.Infof(
				"Updated sensor [IP: %s | SERIAL_NUMBER: %s | OBS. Domain ID: %d]",
				ip.String(), sensor.SerialNumber, sensor.ObservationID)
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for event := range nfEvents {
			log.Debugln(event)
		}
		wg.Done()
	}()

	//////////////////////////////////////////////////////////////////////////////
	// Sensors limits messages
	//////////////////////////////////////////////////////////////////////////////

	limitsMessages, limitsEvents := kafkaConsumer.ConsumeLimits()

	wg.Add(1)
	log.Infoln("Listening for limits messages")
	go func() {
		var lastBlocked time.Time

	receiving:
		for {
			select {
			case <-fetchSignal.C:
				err = chefUpdater.FetchNodes()
				if err != nil {
					log.Errorln("Error fetching nodes: " + err.Error())
					continue
				}

				log.Debugln("Sensors DB updated")

			case message, ok := <-limitsMessages:
				if !ok {
					break receiving
				}

				switch m := message.(type) {
				case consumer.BlockOrganization:
					if time.Since(lastBlocked) <
						time.Duration(config.Updater.UpdateInterval)*time.Second {
						continue receiving
					}

					lastBlocked = time.Now()
					org := string(m)

					errs := chefUpdater.BlockOrganization(org, genericProductType)
					if err != nil {
						for _, err := range errs {
							log.Warnf("Error blocking sensor %s: %s", org, err.Error())
						}
						continue receiving
					}

					log.Infoln("Blocked organization: " + org)

				case consumer.AllowLicense:
					errs := chefUpdater.AllowLicense(m.License)
					if err != nil {
						for _, err := range errs {
							log.Warnf("Error blocking license %s: %s", m.License, err.Error())
						}
						continue receiving
					}

					log.Infoln("Allowed license: " + m.License)

				case consumer.ResetSensors:
					err := chefUpdater.ResetAllSensors()
					if err != nil {
						log.Errorf("Error resetting sensors: %s", err.Error())
						continue receiving
					}

					log.Infof("All sensors has been reset")

				default:
					log.Warnln("Unknown message received")
				}
			}
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for event := range limitsEvents {
			log.Debugln(event)
		}

		wg.Done()
	}()

	//////////////////////////////////////////////////////////////////////////////
	// The End
	//////////////////////////////////////////////////////////////////////////////

	wg.Wait()
	log.Infoln("Bye bye...")
}
