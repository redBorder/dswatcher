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
	"errors"

	"github.com/tehmaze/netflow"
	"github.com/tehmaze/netflow/ipfix"
	"github.com/tehmaze/netflow/session"
)

//////////////////////
// Netflow10Decoder //
//////////////////////

type decoders map[uint32]*netflow.Decoder

// Netflow10DecoderConfig contains the Netflow10Decoder configuration
type Netflow10DecoderConfig struct {
	ElementID        uint16
	OptionTemplateID uint16
}

// Netflow10Decoder decode a serial number and IP address from Netflow data
type Netflow10Decoder struct {
	Netflow10DecoderConfig

	d decoders
}

// NewNetflow10Decoder creates a new instance of a NetflowDecoder
func NewNetflow10Decoder(config Netflow10DecoderConfig) *Netflow10Decoder {
	return &Netflow10Decoder{
		Netflow10DecoderConfig: config,

		d: make(map[uint32]*netflow.Decoder),
	}
}

// Decode tries to decode a netflow packet. The decoder maintains a session for
// ever IP address so devices using different IP address can reuse templates.
// Once a NF10/IPFIX packet is decoded, Decode tries to find a device ID.
// If no device ID has been found the returned value is zero.
func (nd Netflow10Decoder) Decode(ip uint32, data []byte) (string, uint32, error) {
	decoder, found := nd.d[ip]
	if !found {
		decoder = netflow.NewDecoder(session.New())
		nd.d[ip] = decoder
	}

	m, err := decoder.Read(bytes.NewBuffer(data))
	if err != nil {
		return "", 0, errors.New("Error decoding packet: " + err.Error())
	}

	p, ok := m.(*ipfix.Message)
	if !ok {
		return "", 0, errors.New("Invalid message received: Message is not NF10/IPFIX")
	}

	for _, ots := range p.OptionsTemplateSets {
		for _, record := range ots.Records {
			if record.TemplateID == nd.OptionTemplateID {
				if ok := len(record.ScopeFields) == 1; ok {
					if record.ScopeFields[0].InformationElementID == nd.ElementID {
						if ok := len(p.DataSets) == 1 && p.DataSets[0].Header.ID == nd.OptionTemplateID; ok {
							n := bytes.Index(p.DataSets[0].Bytes, []byte{0})
							serialNumer := string(p.DataSets[0].Bytes[:n])

							return serialNumer, p.Header.ObservationDomainID, nil
						}
					}
				}
			}
		}
	}

	return "", p.Header.ObservationDomainID, nil
}
