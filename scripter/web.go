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
	"encoding/json"
	"github.com/honeytrap/honeytrap/utils/files"
	"io/ioutil"
	"strings"
	"os"
	"encoding/base64"
)

// fileInfo covers the file info for responses
type fileInfo struct {
	Path string `json:"path"`
	Content string `json:"content"`
}

// response struct is used for JSON responses
type response struct {
	Type string `json:"type"`
	Data interface{} `json:"data"`
}

// HandleRequests handles the request coming from other environments
func HandleRequests(scripters map[string]Scripter, message []byte) ([]byte, error) {
	var js map[string]interface{}
	json.Unmarshal(message, &js)

	basepath := "scripts/"

	switch val, _ := js["action"]; val {
	// reload scripts
	case "script_reload":
		ReloadAllScripters(scripters)
		// put script
	case "script_put":
		if path, ok := js["path"].(string); ok {
			if content, ok := js["file"].(string); ok {
				if err := files.Put(basepath + path, content); err == nil {
					ReloadAllScripters(scripters)
				}
			}
		}
		// delete script
	case "script_delete":
		if path, ok := js["path"].(string); ok {
			if err := files.Delete(basepath + path); err == nil {
				ReloadAllScripters(scripters)
			}
		}
		// read scripts
	case "script_read":
		dir, ok := js["dir"].(string)
		if !ok {
			dir = ""
		}

		arrFileInfo := readFiles(dir)
		return generateResponse("scripts", arrFileInfo)
	}

	return nil, nil
}

// readFiles reads the files in the scripts directory
func readFiles(dir string) []fileInfo {
	var fileInfos []fileInfo
	basepath := "scripts/"

	dirFiles, err := files.Walker(basepath + dir)
	if err != nil {
		return nil
	}

	for _, file := range dirFiles {
		content, err := ioutil.ReadFile(basepath + dir + file)
		if err != nil {
			return nil
		}

		fileInfos = append(fileInfos, fileInfo{Path: strings.Replace(basepath + dir + file, string(os.PathSeparator), "/", -1), Content: base64.StdEncoding.EncodeToString(content)})
	}

	return fileInfos
}

// generateResponse generates a JSON response for web requests
func generateResponse(responseType string, data interface{}) ([]byte, error) {
	response := response{ Type: responseType, Data: data }
	if response, err := json.Marshal(response); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}
