/*
* Honeytrap
* Copyright (C) 2016-2017 DutchSec (https://dutchsec.com/)
*
* This program is free software; you can redistribute it and/or modify it under
* the terms of the GNU Affero General Public License version 3 as published by the
* Free Software Foundation.
*
* This program is distributed in the hope that it will be useful, but WITHOUT
* ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
* FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
* details.
*
* You should have received a copy of the GNU Affero General Public License
* version 3 along with this program in the file "LICENSE".  If not, see
* <http://www.gnu.org/licenses/agpl-3.0.txt>.
*
* See https://honeytrap.io/ for more details. All requests should be sent to
* licensing@honeytrap.io
*
* The interactive user interfaces in modified source and object code versions
* of this program must display Appropriate Legal Notices, as required under
* Section 5 of the GNU Affero General Public License version 3.
*
* In accordance with Section 7(b) of the GNU Affero General Public License version 3,
* these Appropriate Legal Notices must retain the display of the "Powered by
* Honeytrap" logo and retain the original copyright notice. If the display of the
* logo is not reasonably feasible for technical reasons, the Appropriate Legal Notices
* must display the words "Powered by Honeytrap" and retain the original copyright notice.
 */
package arp

import (
	"encoding/binary"
	"fmt"
)

type Frame struct {
	HardwareType uint16
	ProtocolType uint16
	HardwareSize uint8
	ProtocolSize uint8
	Opcode       uint16

	SenderMAC [6]byte
	SenderIP  [4]byte

	TargetMAC [6]byte
	TargetIP  [4]byte
}

func (f *Frame) String() string {
	return fmt.Sprintf("HardwareType: %x, ProtocolType: %x, HardwareSize: %x, ProtocolSize: %x, Opcode: %x, SenderMAC: %#v, SenderIP: %#v, TargetMAC: %#v, TargetIP: %#v",
		f.HardwareType, f.ProtocolType, f.HardwareSize, f.ProtocolSize, f.Opcode, f.SenderMAC, f.SenderIP, f.TargetMAC, f.TargetIP)
}

func Parse(data []byte) (*Frame, error) {
	eh := &Frame{}
	return eh, eh.Unmarshal(data)
}

func (f *Frame) Unmarshal(data []byte) error {
	if len(data) < 28 {
		return fmt.Errorf("Incorrect ARP header size: %d", len(data))
	}

	f.HardwareType = binary.BigEndian.Uint16(data[0:2])
	f.ProtocolType = binary.BigEndian.Uint16(data[2:4])
	f.HardwareSize = data[4]
	f.ProtocolSize = data[5]
	f.Opcode = binary.BigEndian.Uint16(data[6:8])

	copy(f.SenderMAC[:], data[8:14])
	copy(f.SenderIP[:], data[14:18])
	copy(f.TargetMAC[:], data[18:24])
	copy(f.TargetIP[:], data[24:28])

	return nil
}
