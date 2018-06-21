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
	"github.com/honeytrap/honeytrap/utils"
	"github.com/op/go-logging"
	"net"
	"github.com/honeytrap/honeytrap/connectors"
)

var (
	_   = connectors.Register("generic", Generic)
	log = logging.MustGetLogger("connectors/generic")
)

func Generic(options ...connectors.ConnectorFunc) connectors.Connector {
	s := &genericService{}

	for _, o := range options {
		o(s)
	}

	return s
}

type genericService struct {
	connectors.BaseConnector
}

func (s *genericService) CanHandle(payload []byte) bool {
	return s.GetScripter().CanHandle("generic", string(payload))
}

func (s *genericService) HandleScripter(conn net.Conn, cData interface{}) error {
	buffer := make([]byte, 4096)
	pConn := utils.PeekConnection(conn)
	n, _ := pConn.Peek(buffer)

	// Add the go methods that have to be exposed to the scripts
	connW := s.GetScripter().GetConnection("generic", pConn)

	s.setMethods(connW)

	for {
		//Handle incoming message with the scripter
		response, err := connW.Handle(string(buffer[:n]))
		if err != nil {
			return err
		} else if response == "_return" {
			// Return called from script
			return nil
		}

		//Write message to the connection
		if _, err := conn.Write([]byte(response)); err != nil {
			return err
		}
	}
	return nil
}
