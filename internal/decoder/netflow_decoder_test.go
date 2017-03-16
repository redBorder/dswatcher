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
			ElementID:        300,
			OptionTemplateID: 258,
		})

		Convey("For valid template and data sets", func() {
			data := []byte{
				/////////////
				// Headers //
				/////////////
				0x00, 0x0a, // Version: 10
				0x00, 0x64, // Length: 100
				0x58, 0xb0, 0x00, 0x49, // ExportTime: 1487929417
				0x00, 0x00, 0xc6, 0x5b, // FlowSequence: 50779
				0x00, 0x00, 0x00, 0x0a, // Observation Domain Id: 10

				//////////////////////////////////////////
				// Set 1 [id=3] (Options Template): 258 //
				//////////////////////////////////////////
				0x00, 0x03, // FlowSet Id: Options Template (V10 [IPFIX]) (3)
				0x00, 0x0e, // FlowSet Length: 14
				// Options Template (Id = 258) (Scope Count = 1; Data Count = 0)
				0x01, 0x02, // Template Id: 258
				0x00, 0x01, // Total Field Count: 1
				0x00, 0x01, // Scope Field Count: 1
				0x01, 0x2c, 0x00, 0x40, // Field (1/1) [Scope]: observationDomainName

				//////////////////////////////
				// Set 2 [id=258] (1 flows) //
				//////////////////////////////
				0x01, 0x02, // FlowSet Id: (Data) (258)
				0x00, 0x46, // FlowSet Length: 70
				// Flow 1: Serial number "tim/88888888"
				0x74, 0x69, 0x6d, 0x2f, 0x38, 0x38, 0x38, 0x38,
				0x38, 0x38, 0x38, 0x38, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				// Padding
				0x00, 0x00,
			}

			Convey("The serial number and observation domain ID should be decoded", func() {
				sn, obsID, err := decoder.Decode(3232235777, data) // 192.168.1.1
				So(err, ShouldBeNil)
				So(sn, ShouldEqual, "tim/88888888")
				So(obsID, ShouldEqual, 10)
			})
		})

		Convey("For valid template and data sets with different option template id", func() {
			data := []byte{
				/////////////
				// Headers //
				/////////////
				0x00, 0x0a, // Version: 10
				0x00, 0x64, // Length: 100
				0x58, 0xb0, 0x00, 0x49, // ExportTime: 1487929417
				0x00, 0x00, 0xc6, 0x5b, // FlowSequence: 50779
				0x00, 0x00, 0x00, 0x0a, // Observation Domain Id: 10

				//////////////////////////////////////////
				// Set 1 [id=3] (Options Template): 258 //
				//////////////////////////////////////////
				0x00, 0x03, // FlowSet Id: Options Template (V10 [IPFIX]) (3)
				0x00, 0x0e, // FlowSet Length: 14
				// Options Template (Id = 258) (Scope Count = 1; Data Count = 0)
				0x01, 0x02, // Template Id: 258
				0x00, 0x01, // Total Field Count: 1
				0x00, 0x01, // Scope Field Count: 1
				0x01, 0x2c, 0x00, 0x40, // Field (1/1) [Scope]: observationDomainName

				//////////////////////////////
				// Set 2 [id=257] (1 flows) //
				//////////////////////////////
				0x01, 0x00, // FlowSet Id: (Data) (257)
				0x00, 0x46, // FlowSet Length: 70
				// Flow 1: Serial number "tim/88888888"
				0x74, 0x69, 0x6d, 0x2f, 0x38, 0x38, 0x38, 0x38,
				0x38, 0x38, 0x38, 0x38, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				// Padding
				0x00, 0x00,
			}

			Convey("The serial number and observation domain ID should NOT be decoded", func() {
				sn, obsID, err := decoder.Decode(3232235777, data) // 192.168.1.1
				So(err, ShouldBeNil)
				So(sn, ShouldEqual, "")
				So(obsID, ShouldEqual, 10)
			})
		})

		Convey("For valid template and data sets without serial number", func() {
			data := []byte{
				/////////////
				// Headers //
				/////////////
				0x00, 0x0a, // Version: 10
				0x00, 0x1e, // Length: 100
				0x58, 0xb0, 0x00, 0x49, // ExportTime: 1487929417
				0x00, 0x00, 0xc6, 0x5b, // FlowSequence: 50779
				0x00, 0x00, 0x00, 0x0a, // Observation Domain Id: 10

				//////////////////////////////////////////
				// Set 1 [id=3] (Options Template): 258 //
				//////////////////////////////////////////
				0x00, 0x03, // FlowSet Id: Options Template (V10 [IPFIX]) (3)
				0x00, 0x0e, // FlowSet Length: 14
				// Options Template (Id = 258) (Scope Count = 1; Data Count = 0)
				0x01, 0x02, // Template Id: 258
				0x00, 0x01, // Total Field Count: 1
				0x00, 0x01, // Scope Field Count: 1
				0x01, 0x2c, 0x00, 0x40, // Field (1/1) [Scope]: observationDomainName
			}

			Convey("The retuned serial number should be empty", func() {
				id, obsID, err := decoder.Decode(3232235777, data)
				So(err, ShouldBeNil)
				So(id, ShouldEqual, "")
				So(obsID, ShouldEqual, 10)
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
				_, _, err := decoder.Decode(3232235777, data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Invalid message received: Message is not NF10/IPFIX")
			})
		})

		Convey("For an invalid Netflow packet", func() {
			data := []byte{0xca, 0xfe, 0xfa, 0xba, 0xda}

			Convey("Should error", func() {
				_, _, err := decoder.Decode(3232235777, data)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "Error decoding packet: netflow: unsupported version 51966")
			})
		})
	})
}
