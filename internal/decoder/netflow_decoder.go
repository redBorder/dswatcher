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

package decoder

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/tehmaze/netflow"
	"github.com/tehmaze/netflow/ipfix"
	"github.com/tehmaze/netflow/session"
)

//////////////////////
// Netflow10Decoder //
//////////////////////

// Sensor struct contains information about a sensor that has been detected
type Sensor struct {
	SerialNumber  string
	ObservationID uint32
	Address       net.IP
	DeviceID      uint32
}
type decoders map[uint32]*netflow.Decoder
type sensors []Sensor

// Netflow10DecoderConfig contains the Netflow10Decoder configuration
type Netflow10DecoderConfig struct {
	DeviceTypeElementID   uint16
	SerialNumberElementID uint16
	OptionTemplateID      uint16
}

// Netflow10Decoder decode a serial number and IP address from Netflow data
type Netflow10Decoder struct {
	Netflow10DecoderConfig

	sensors  sensors
	decoders decoders
}

// NewNetflow10Decoder creates a new instance of a NetflowDecoder
func NewNetflow10Decoder(config Netflow10DecoderConfig) *Netflow10Decoder {
	return &Netflow10Decoder{
		Netflow10DecoderConfig: config,

		decoders: make(map[uint32]*netflow.Decoder),
	}
}

// Decode tries to decode a netflow packet. The decoder maintains a session for
// ever IP address so devices using different IP address can reuse templates.
// Once a NF10/IPFIX packet is decoded, Decode tries to find a serial number.
// If no serial number has been found the returned value is zero.
func (nd *Netflow10Decoder) Decode(ip uint32, data []byte) (*Sensor, error) {
	decoder, found := nd.decoders[ip]
	if !found {
		decoder = netflow.NewDecoder(session.New())
		nd.decoders[ip] = decoder
	}

	m, err := decoder.Read(bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.New("Error decoding packet: " + err.Error())
	}

	p, ok := m.(*ipfix.Message)
	if !ok {
		return nil, errors.New("Invalid message received: Message is not NF10/IPFIX")
	}

	for _, ots := range p.OptionsTemplateSets {
		for _, record := range ots.Records {
			if record.TemplateID == nd.OptionTemplateID {
				if ok := len(record.ScopeFields) == 1; ok {
					if record.ScopeFields[0].InformationElementID == nd.SerialNumberElementID {
						if ok := len(p.DataSets) == 1 && p.DataSets[0].Header.ID == nd.OptionTemplateID; ok {
							n := bytes.Index(p.DataSets[0].Bytes, []byte{0})
							s := Sensor{
								SerialNumber:  string(p.DataSets[0].Bytes[:n]),
								ObservationID: p.Header.ObservationDomainID,
							}

							if nd.DeviceTypeElementID != 0 {
								s.Address = make(net.IP, 4)
								binary.BigEndian.PutUint32(s.Address, ip)
								nd.sensors = append(nd.sensors, s)
								return nil, nil
							}

							return &s, nil
						}
					}
				}
			}
		}
	}

	if nd.DeviceTypeElementID != 0 {
		for _, ds := range p.DataSets {
			for _, record := range ds.Records {
				for _, field := range record.Fields {
					if field.Translated.InformationElementID == nd.DeviceTypeElementID {
						for _, sensor := range nd.sensors {
							address := make(net.IP, 4)
							binary.BigEndian.PutUint32(address, ip)
							if sensor.Address.Equal(address) &&
								sensor.ObservationID == p.Header.ObservationDomainID {

								sensor.DeviceID = field.Translated.Value.(uint32)
								if sensor.SerialNumber != "" {
									return &sensor, nil
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, nil
}
