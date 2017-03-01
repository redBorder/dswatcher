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

package decoder

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDecoder(t *testing.T) {
	Convey("Given a Netflow 10 decoder", t, func() {
		decoder := NewNetflow10Decoder(Netflow10DecoderConfig{
			ElementID: 144,
		})

		Convey("For valid template and data sets", func() {
			data := []byte{
				/////////////
				// Headers //
				/////////////
				0x00, 0x0a, // Version: 10
				0x00, 0x24, // Length: 36
				0x58, 0xb0, 0x00, 0x49, // ExportTime: 1487929417
				0x00, 0x00, 0xc6, 0x5b, // FlowSequence: 50779
				0x00, 0x00, 0x00, 0x0a, // Observation Domain Id: 10

				////////////////////////////////////////
				// Set 1 [id=2] (Data Template): 1025 //
				////////////////////////////////////////
				0x00, 0x02, // FlowSet Id: Data Template (V10 [IPFIX]) (2)
				0x00, 0x0c, // FlowSet Length: 12
				// Template (Id = 1025, Count = 1)
				0x04, 0x01, // Template Id: 1025
				0x00, 0x01, // Field Count: 1
				0x00, 0x90, 0x00, 0x04, // Field (1/1): FLOW_EXPORTER

				///////////////////////////////
				// Set 2 [id=1025] (1 flows) //
				///////////////////////////////
				0x04, 0x01, // FlowSet Id: (Data) (1025)
				0x00, 0x08, // FlowSet Length: 8
				// Flow 1
				0x00, 0x00, 0x00, 0x2a, // FlowExporter: 42
			}

			Convey("The id should be decoded", func() {
				id, err := decoder.Decode(3232235777, data)
				So(err, ShouldBeNil)
				So(id, ShouldEqual, 42)
			})
		})

		Convey("For valid template and data sets without FLOW_EXPORTER", func() {
			data := []byte{
				/////////////
				// Headers //
				/////////////
				0x00, 0x0a, // Version: 10
				0x00, 0x24, // Length: 36
				0x58, 0xb0, 0x00, 0x49, // ExportTime: 1487929417
				0x00, 0x00, 0xc6, 0x5b, // FlowSequence: 50779
				0x00, 0x00, 0x00, 0x0a, // Observation Domain Id: 10

				////////////////////////////////////////
				// Set 1 [id=2] (Data Template): 1025 //
				////////////////////////////////////////
				0x00, 0x02, // FlowSet Id: Data Template (V10 [IPFIX]) (2)
				0x00, 0x0c, // FlowSet Length: 12
				// Template (Id = 1025, Count = 1)
				0x04, 0x01, // Template Id: 1025
				0x00, 0x01, // Field Count: 1
				0x00, 0x08, 0x00, 0x04, // Field (1/1): IP_SRC_ADDR

				///////////////////////////////
				// Set 2 [id=1025] (1 flows) //
				///////////////////////////////
				0x04, 0x01, // FlowSet Id: (Data) (1025)
				0x00, 0x08, // FlowSet Length: 8
				// Flow 1
				0xc8, 0xa8, 0xd4, 0x0e, // SrcAddr: 192.168.212.14
			}

			Convey("The retuned ID should be zero", func() {
				id, err := decoder.Decode(3232235777, data)
				So(err, ShouldBeNil)
				So(id, ShouldEqual, 0)
			})
		})

		Convey("For an invalid Netflow 10 packet (netflow 5 packet)", func() {
			data := []byte{
				/////////////////
				// Flow Header //
				/////////////////
				0x00, 0x05, // Version: 5
				0x00, 0x01, // The number of records in PDU
				0x00, 0x00, 0x00, 0x00, // Current time in msecs since router booted
				0x00, 0x00, 0x00, 0x00, // Current seconds since 0000 UTC 1970
				0x00, 0x00, 0x00, 0x00, // Residual nanoseconds since 0000 UTC 1970
				0x00, 0x00, 0x00, 0x01, // Sequence number of total flows seen
				0x00,       // Type of flow switching engine (RP,VIP,etc.)*/
				0x00,       // Slot number of the flow switching engine */
				0x00, 0x00, // Packet capture sample rate */

				/////////////////
				// Flow Record //
				/////////////////
				0x08, 0x08, 0x08, 0x08, /* Source IP Address */
				0x0A, 0x0A, 0x0A, 0x0A, /* Destination IP Address */
				0x00, 0x00, 0x00, 0x00, /* Next hop router's IP Address */
				0x00, 0x00, /* Input interface index */
				0x00, 0x00, /* Output interface index */
				0x01, 0x00, 0x00, 0x00, /* Packets sent in Duration (milliseconds between 1st
				   & last packet in this flow)*/
				0x46, 0x00, 0x00, 0x00, /* Octets sent in Duration (milliseconds between 1st
				   & last packet in  this flow)*/
				0xa8, 0x48, 0x42, 0x05, /* SysUptime at start of flow */
				0xa8, 0x48, 0x42, 0x05, /* and of last packet of the flow */
				0xbb, 0x01, /* ntohs(443)  */ /* TCP/UDP source port number (.e.g, FTP, Telnet, etc.,or equivalent) */
				0x75, 0x27, /* ntohs(10101)*/ /* TCP/UDP destination port number (.e.g, FTP, Telnet, etc.,or equivalent) */
				0x00,       /* pad to word boundary */
				0x00,       /* Cumulative OR of tcp flags */
				0x02,       /* IP protocol, e.g., 6=TCP, 17=UDP, etc... */
				0x00,       /* IP Type-of-Service */
				0x00, 0x00, /* source peer/origin Autonomous System */
				0x00, 0x00, /* dst peer/origin Autonomous System */
				0x00,       /* source route's mask bits */
				0x00,       /* destination route's mask bits */
				0x00, 0x00, /* pad to word boundary */
			}

			Convey("Should error", func() {
				_, err := decoder.Decode(3232235777, data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Invalid message received: Message is not NF10/IPFIX")
			})
		})

		Convey("For an invalid Netflow packet", func() {
			data := []byte{0xca, 0xfe, 0xfa, 0xba, 0xda}

			Convey("Should error", func() {
				_, err := decoder.Decode(3232235777, data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Error decoding packet: netflow: unsupported version 51966")
			})
		})
	})
}
