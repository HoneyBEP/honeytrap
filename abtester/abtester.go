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
package abtester

import (
	"github.com/honeytrap/honeytrap/storage"
	"fmt"
	"strings"
	"math/rand"
	"encoding/json"
	"io/ioutil"
)

type Abtester interface {
	Get(key string, item int) (string, error)
	GetForGroup(group string, key string, item int) (string, error)
	Set(key string, value string) error
	SetForGroup(group string, key string, value string) error
	LoadFromFile(fileName string) error
}

func Namespace(namespace string) (*abTester, error) {
	st, err := storage.Namespace(fmt.Sprintf("abtester_%s", namespace))
	return &abTester{
		st: st,
	}, err
}

type abTester struct {
	st storage.Storage
}

func (s *abTester) Get(key string, item int) (string, error) {
	data, err := s.st.Get(key)
	if err != nil {
		return "", err
	}
	options := byteToString(data)

	return getItem(options, item)
}

func (s *abTester) GetForGroup(group string, key string, item int) (string, error) {
	return s.Get(fmt.Sprintf("%s_%s", group, key), item)
}

func (s *abTester) Set(key string, value string) error {
	data, err := s.st.Get(key)
	if err != nil && len(data) > 0 {
		return err
	}

	options := append(byteToString(data), value)

	s.st.Set(key, stringToByte(options))

	return nil
}

func (s *abTester) SetForGroup(group string, key string, value string) error {
	return s.Set(fmt.Sprintf("%s_%s", group, key), value)
}

func (s *abTester) LoadFromFile(fileName string) error {
	file, _ := ioutil.ReadFile(fileName)
	var objmap map[string]map[string][]string
	err := json.Unmarshal(file, &objmap)

	for groupName, group := range objmap {
		for key, values := range group {
			for _, value := range values {
				s.SetForGroup(groupName, key, value)
			}
		}
	}

	return err
}

func byteToString(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	return strings.Split(string(data), ";;;")
}

func stringToByte(data []string) []byte {
	return []byte(strings.Join(data, ";;;"))
}

func getItem(options []string, item int) ( string, error ) {
	var result string
	if len(options) == 0 {
		return "", fmt.Errorf("no abTest found")
	}
	if item > 0 && item < len(options) {
		result = options[item]
	} else {
		result = options[rand.Intn(len(options)-1)]
	}
	return result, nil
}