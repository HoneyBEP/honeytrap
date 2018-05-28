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
package services

import (
	"context"
	"github.com/honeytrap/honeytrap/pushers"
	"github.com/honeytrap/honeytrap/scripter"
	"net"
	"github.com/honeytrap/honeytrap/utils"
	"net/http"
	"time"
	"encoding/json"
	"io/ioutil"
	"bytes"
	"io"
	"bufio"
	"strconv"
)

var (
	_ = Register("generic", generic)
)

func generic(options ...ServicerFunc) Servicer {
	s := &genericService{}

	for _, o := range options {
		o(s)
	}

	return s
}

type genericService struct {
	scr scripter.Scripter
	c   pushers.Channel
}

func (s *genericService) CanHandle(payload []byte) bool {
	return s.scr.CanHandle("generic", string(payload))
}

func (s *genericService) SetScripter(scr scripter.Scripter) {
	s.scr = scr
}

func (s *genericService) SetChannel(c pushers.Channel) {
	s.c = c
}

func (s *genericService) Handle(ctx context.Context, conn net.Conn) error {
	buffer := make([]byte, 4096)
	pConn := utils.PeekConnection(conn)
	n, _ := pConn.Peek(buffer)

	connW := s.scr.GetConnection("generic", pConn)

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

func (s *genericService) setMethods(connW scripter.ConnectionWrapper) error {
	connW.SetStringFunction("getRequest", func() string {
		params, _ := connW.GetParameters([]string{"withBody"})

		buf := connW.GetScrConn().GetConnectionBuffer()
		buf.Reset()
		tee := io.TeeReader(connW.GetScrConn().GetConn(), buf)

		br := bufio.NewReader(tee)

		req, err := http.ReadRequest(br)
		if err == io.EOF {
			log.Infof("Payload is empty.", err)
			return ""
		} else if err != nil {
			log.Errorf("Failed to parse payload to HTTP Request, Error: %s", err)
			return ""
		}

		m := map[string]interface{}{}
		m["method"] = req.Method
		m["header"] = req.Header
		m["host"] = req.Host
		m["form"] = req.Form
		body := make([]byte, 1024)
		if params["withBody"] == "1" {

			defer req.Body.Close()
			n, _ := req.Body.Read(body)

			body = body[:n]
			var js2 map[string]interface{}
			if json.Unmarshal([]byte(body), &js2) == nil {
				m["body"] = js2
			} else {
				m["body"] = string(body)
			}
		}

		result, err := json.Marshal(m)
		if err != nil {
			log.Errorf("Failed to parse request struct to json, Error: %s", err)
			return "{}"
		}

		return string(result)
	})

	connW.SetVoidFunction("restWrite", func() {
		params, _ := connW.GetParameters([]string{"status", "response", "headers"})

		status, _ := strconv.Atoi(params["status"])
		buf := connW.GetScrConn().GetConnectionBuffer()
		br := bufio.NewReader(buf)

		req, err := http.ReadRequest(br)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Errorf("Error while reading buffered request connection, %s", err)
			return
		}

		defer req.Body.Close()

		header := http.Header{}

		header.Set("date", (time.Now()).String())
		header.Set("connection", "Keep-Alive")
		header.Set("content-type", "application/json")

		var headers map[string]string
		json.Unmarshal([]byte(params["data"]), &headers)
		for name, value := range headers {
			header.Set(name, value)
		}

		resp := http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Proto:      req.Proto,
			ProtoMajor: req.ProtoMajor,
			ProtoMinor: req.ProtoMinor,
			Request:    req,
			Header: header,
			Body:          ioutil.NopCloser(bytes.NewBufferString(params["response"])),
			ContentLength: int64(len(params["response"])),
		}

		if err := resp.Write(connW.GetScrConn().GetConn()); err != nil {
			log.Errorf("Writing of scripter - REST message was not successful, %s", err)
		}
	})
}
