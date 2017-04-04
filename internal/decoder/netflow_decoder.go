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
	"strconv"

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
	ProductType   uint32
}
type decoders map[uint32]*netflow.Decoder
type sensors []Sensor

// Netflow10DecoderConfig contains the Netflow10Decoder configuration
type Netflow10DecoderConfig struct {
	ProductTypeElementID  uint16
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

	if len(p.OptionsTemplateSets) < 1 {
		return nil, nil
	}

	if !checkOptionTemplateID(&p.OptionsTemplateSets[0], nd.OptionTemplateID,
		nd.SerialNumberElementID, nd.ProductTypeElementID) {
		return nil, nil
	}

	if len(p.DataSets) != 1 {
		return nil, errors.New("Flow message not supported")
	}

	ds := &p.DataSets[0]

	if ds.Header.ID != nd.OptionTemplateID {
		return nil, errors.New("Data set with ID " +
			strconv.FormatUint(uint64(ds.Header.ID), 10) +
			" does not match the specified option template ID")
	}

	serialNumber := getSerialNumber(ds)
	productType := getProductType(ds)

	s := &Sensor{
		SerialNumber:  serialNumber,
		ProductType:   productType,
		ObservationID: p.Header.ObservationDomainID,
	}

	return s, nil
}

func checkOptionTemplateID(set *ipfix.OptionsTemplateSet, otID, snID, ptID uint16) bool {
	record := set.Records[0]

	if len(record.ScopeFields) != 1 || len(record.Fields) != 1 {
		return false
	}

	if record.TemplateID != otID {
		return false
	}

	productType := record.ScopeFields[0]
	serialNumberField := record.Fields[0]

	if productType.InformationElementID != ptID ||
		serialNumberField.InformationElementID != snID {
		return false
	}

	return true
}

func getProductType(set *ipfix.DataSet) uint32 {
	pt := set.Bytes[:4]
	return binary.BigEndian.Uint32(pt)
}

func getSerialNumber(set *ipfix.DataSet) string {
	n := bytes.Index(set.Bytes[4:len(set.Bytes)], []byte{0})
	return string(set.Bytes[4 : n+4])
}
