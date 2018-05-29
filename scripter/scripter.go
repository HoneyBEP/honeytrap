package scripter

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/honeytrap/honeytrap/abtester"
	"github.com/honeytrap/honeytrap/utils/files"
	"github.com/op/go-logging"
	"net"
	"time"
	"io/ioutil"
	"strings"
	"encoding/json"
	"os"
	"encoding/base64"
)

var (
	scripters = map[string]func(string, ...func(Scripter) error) (Scripter, error){}
)
var log = logging.MustGetLogger("scripter")

//Register the scripter instance
func Register(key string, fn func(string, ...func(Scripter) error) (Scripter, error)) func(string, ...func(Scripter) error) (Scripter, error) {
	scripters[key] = fn
	return fn
}

//Get a scripter instance
func Get(key string) (func(string, ...func(Scripter) error) (Scripter, error), bool) {
	if fn, ok := scripters[key]; ok {
		return fn, true
	}

	return nil, false
}

//GetAvailableScripterNames gets all scripters that are registered
func GetAvailableScripterNames() []string {
	var out []string
	for key := range scripters {
		out = append(out, key)
	}
	return out
}

//Scripter interface that implements basic scripter methods
type Scripter interface {
	Init(string) error
	//SetGlobalFn(name string, fn func() string) error
	GetConnection(service string, conn net.Conn) ConnectionWrapper
	CanHandle(service string, message string) bool
	GetScripts() map[string]map[string]string
	GetScriptFolder() string
}

//ConnectionWrapper interface that implements the basic method that a connection should have
type ConnectionWrapper interface {
	Handle(message string) (string, error)
	SetStringFunction(name string, getString func() string) error
	SetFloatFunction(name string, getFloat func() float64) error
	SetVoidFunction(name string, doVoid func()) error
	GetParameters(params []string) (map[string]string, error)
}

//ScrConn wraps a connection and exposes methods to interact with the connection and scripter
type ScrConn interface {
	GetConn() net.Conn
	SetStringFunction(name string, getString func() string, service string) error
	SetFloatFunction(name string, getFloat func() float64, service string) error
	SetVoidFunction(name string, doVoid func(), service string) error
	GetParameters(params []string, service string) (map[string]string, error)
	HasScripts(service string) bool
	AddScripts(service string, scripts map[string]string)
	Handle(service string, message string) (*Result, error)
}

//Result struct which allows the result to be a string, an empty string and a nil value
//The nil value can be used to indicate that lua has no value to return
type Result struct {
	Content string
}

//ScrAbTester exposes methods to interact with the AbTester
type ScrAbTester interface {
	GetAbTester() abtester.Abtester
}

//WithConfig returns a function to attach the config to the scripter
func WithConfig(c toml.Primitive) func(Scripter) error {
	return func(scr Scripter) error {
		return toml.PrimitiveDecode(c, scr)
	}
}

//SetBasicMethods sets methods that can be called by each script, returning basic functionality
func SetBasicMethods(c ScrConn, service string) {
	c.SetStringFunction("getRemoteAddr", func() string { return c.GetConn().RemoteAddr().String() }, service)
	c.SetStringFunction("getLocalAddr", func() string { return c.GetConn().LocalAddr().String() }, service)

	c.SetStringFunction("getDatetime", func() string {
		t := time.Now()
		return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d-00:00\n",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second())
	}, service)

	c.SetStringFunction("getFileDownload", func() string {
		params, _ := c.GetParameters([]string{"url", "path"}, service)

		if err := files.Download(params["url"], params["path"]); err != nil {
			log.Errorf("error downloading file: %s", err)
			return "no"
		}
		return "yes"
	}, service)

	if ab, ok := c.(ScrAbTester); ok {
		//In the script the function 'getAbTest(key)' can be called, returning a random result for the given key
		c.SetStringFunction("getAbTest", func() string {
			params, _ := c.GetParameters([]string{"key"}, service)

			val, err := ab.GetAbTester().GetForGroup(service, params["key"], -1)
			if err != nil {
				return "_" //No response, _ so lua knows it has no ab-test
			}

			return val
		}, service)
	}

	//In the script the function 'doLog(type, message)' can be called, with type = logging type and message the message
	c.SetVoidFunction("doLog", func() {
		params, _ := c.GetParameters([]string{"logType", "message"}, service)
		logType := params["logType"]
		message := params["message"]

		if logType == "critical" {
			log.Critical(message)
		}
		if logType == "debug" {
			log.Debug(message)
		}
		if logType == "error" {
			log.Error(message)
		}
		if logType == "fatal" {
			log.Fatal(message)
		}
		if logType == "info" {
			log.Info(message)
		}
		if logType == "notice" {
			log.Notice(message)
		}
		if logType == "panic" {
			log.Panic(message)
		}
		if logType == "warning" {
			log.Warning(message)
		}
	}, service)
}

// ReloadScripts reloads the scripts from the scripter
func ReloadScripts(s Scripter) {
	for service := range s.GetScripts() {
		if err := s.Init(service); err != nil {
			log.Errorf("error init service: %s", err)
		} else {
			log.Infof("successfully updated service: %s", service)
		}
	}
}

// HandleRequests handles the request coming from other environments
func HandleRequests(scripters map[string]Scripter, message []byte) ([]byte, error) {
	var js map[string]interface{}
	json.Unmarshal(message, &js)

	type fileInfo struct {
		Path string `json:"path"`
		Content string `json:"content"`
	}

	type response struct {
		Type string `json:"type"`
		Data interface{} `json:"data"`
	}

	if val, ok := js["action"]; ok && val == "file_reload" {
		for _, script := range scripters {
			ReloadScripts(script)
		}
	} else if ok && val == "file_put" {
		if path, ok := js["path"].(string); ok {
			if content, ok := js["file"].(string); ok {
				if err := files.Put(path, content); err == nil {
					for _, script := range scripters {
						ReloadScripts(script)
					}
				}
			}
		}
	} else if ok && val == "file_delete" {
		if path, ok := js["path"].(string); ok {
			if err := files.Delete(path); err == nil {
				for _, script := range scripters {
					ReloadScripts(script)
				}
			}
		}
	} else if ok && val == "file_read" {
		dir, ok := js["dir"].(string)
		if !ok {
			dir = ""
		}

		dirFiles, err := files.Walker("scripts/" + dir)
		if err != nil {
			return nil, err
		}

		var fileInfos []fileInfo
		for _, file := range dirFiles {
			content, err := ioutil.ReadFile("scripts/" + dir + file)
			if err != nil {
				return nil, err
			}

			fileInfos = append(fileInfos, fileInfo{Path: strings.Replace("scripts/" + dir + file, string(os.PathSeparator), "/", -1), Content: base64.StdEncoding.EncodeToString(content)})
		}

		fileJSON := response{ Type: "files", Data: fileInfos }
		if fileJSON, err := json.Marshal(fileJSON); err != nil {
			return nil, err
		} else {
			return fileJSON, nil
		}

		for _, script := range scripters {
			ReloadScripts(script)
		}
	}

	return nil, nil
}
