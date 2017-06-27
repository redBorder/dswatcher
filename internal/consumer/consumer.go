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

package consumer

// Message can be either an UUID to be blocked or a ResetSignal
type Message interface{}

// BlockOrganization identifies the organization that reached the limit
type BlockOrganization string

// BlockLicense identifies the license that has expired
type BlockLicense string

// ResetSignal notifies that sensors from a given organization should be
// unblocked.
type ResetSignal struct {
	Organization string
}

// FlowData contains the IP address of the Netflow exporter and the flow itself
type FlowData struct {
	IP   uint32
	Data []byte
}

// NetflowConsumer gets an IP address and Netflow data from a resource
type NetflowConsumer interface {
	ConsumeNetflow() (messages chan FlowData, events chan string)
	ConsumeLimits() (messages chan Message, events chan string)
}
