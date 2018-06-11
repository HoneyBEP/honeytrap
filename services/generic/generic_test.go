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
package generic

import (
	"testing"
	"net"
	"bytes"
	"context"
	"github.com/honeytrap/honeytrap/services"
	"github.com/honeytrap/honeytrap/scripter"

	_ "github.com/honeytrap/honeytrap/scripter/lua"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Scripters map[string]toml.Primitive `toml:"scripter"`
}

func TestEcho(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	configString := "[scripter.lua]\r\n" +
		"type=\"lua\"\r\n" +
		"folder=\"../../test-scripts\"\r\n"

	configLua := &Config{}
	if _, err := toml.Decode(configString, configLua); err != nil {
		t.Error(err)
	}

	scripterFunc, ok := scripter.Get("lua")
	if !ok {
		t.Errorf("failed to retrieve scripter func")
	}

	scripter, err := scripterFunc("lua", scripter.WithConfig(configLua.Scripters["lua"]))
	if err != nil {
		t.Error(err)
	}

	g := Generic(services.WithScripter("generic", scripter))
	go g.Handle(context.TODO(), server)

	test := []byte("test")
	if _, err := client.Write(test); err != nil {
		t.Error(err)
	}

	buffer := make([]byte, 255)
	if n, err := client.Read(buffer[:]); err != nil {
		t.Error(err)
	} else {
		buffer = buffer[:n]
	}

	if !bytes.Equal(test, buffer) {
		t.Errorf("Test failed: got %+#v, expected %+#v", buffer, test)
		return
	}
}
