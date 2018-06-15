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
package scripter

import (
	"fmt"
	"github.com/honeytrap/honeytrap/abtester"
	"net"
	"github.com/honeytrap/honeytrap/pushers"
)

// New creates a lua scripter instance that handles the connection to all scripts
// A list where all scripts are stored in is generated
func Dummy(name string, options ...ScripterFunc) (Scripter, error) {
	l := &dummyScripter{
		name: name,
	}

	for _, optionFn := range options {
		optionFn(l)
	}

	return l, nil
}

// The scripter state to which scripter functions are attached
type dummyScripter struct {
	name string

	ab abtester.AbTester

	c pushers.Channel
}

// SetChannel sets the channel over which messages to the log and elasticsearch can be set
func (l *dummyScripter) SetChannel(c pushers.Channel) {
	l.c = c
}

// GetChannel gets the channel over which messages to the log and elasticsearch can be set
func (l *dummyScripter) GetChannel() pushers.Channel {
	return l.c
}

//Set the abTester from which differential responses can be retrieved
func (l *dummyScripter) SetAbTester(ab abtester.AbTester) {
	l.ab = ab
}

//Set the abTester from which differential responses can be retrieved
func (l *dummyScripter) GetAbTester() abtester.AbTester {
	return l.ab
}

// Init initializes the scripts from a specific service
// The service name is given and the method will loop over all files in the scripts folder with the given service name
// All of these scripts are then loaded and stored in the scripts map
func (l *dummyScripter) Init(service string) error {
	return nil
}

//GetConnection returns a connection for the given ip-address, if no connection exists yet, create it.
func (l *dummyScripter) GetConnection(service string, conn net.Conn) ConnectionWrapper {
	return &ConnectionStruct{Service: service, Conn: nil}
}

// CanHandle checks whether scripter can handle incoming connection for the peeked message
// Returns true if there is one script able to handle the connection
func (l *dummyScripter) CanHandle(service string, message string) bool {
	return false
}

// GetScripts return the scripts for this scripter
func (l *dummyScripter) GetScripts() map[string]map[string]string {
	return nil
}

// GetScriptFolder return the folder where the scripts are located for this scripter
func (l *dummyScripter) GetScriptFolder() string {
	return fmt.Sprintf("%s", l.name)
}
