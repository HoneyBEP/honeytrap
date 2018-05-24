package scripter

import (
	"bytes"
	"github.com/BurntSushi/toml"
	"github.com/honeytrap/honeytrap/abtester"
	"github.com/honeytrap/honeytrap/pushers"
	"github.com/op/go-logging"
	"net"
	"time"
	"encoding/hex"
	"io/ioutil"
	"crypto/sha1"
	"fmt"
	"github.com/honeytrap/honeytrap/utils/files"
)

var (
	scripters = map[string]func(string, ...ScripterFunc) (Scripter, error){}
)
var log = logging.MustGetLogger("scripter")

//Register the scripter instance
func Register(key string, fn func(string, ...ScripterFunc) (Scripter, error)) func(string, ...ScripterFunc) (Scripter, error) {
	scripters[key] = fn
	return fn
}

type ScripterFunc func(Scripter) error

//Get a scripter instance
func Get(key string) (func(string, ...ScripterFunc) (Scripter, error), bool) {
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

func WithChannel(eb pushers.Channel) ScripterFunc {
	return func(s Scripter) error {
		s.SetChannel(eb)
		return nil
	}
}

//Scripter interface that implements basic scripter methods
type Scripter interface {
	Init(string) error
	GetConnection(service string, conn net.Conn) ConnectionWrapper
	CanHandle(service string, message string) bool
	SetChannel(c pushers.Channel)
	GetScripts() map[string]map[string]*Script
	GetScriptFolder() string
}

//ConnectionWrapper interface that implements the basic method that a connection should have
type ConnectionWrapper interface {
	GetScrConn() ScrConn
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
	GetConnectionBuffer() *bytes.Buffer
}

//Result struct which allows the result to be a string, an empty string and a nil value
//The nil value can be used to indicate that lua has no value to return
type Result struct {
	Content string
}

type Script struct {
	// hash of the file
	Hash string

	// source of the states, initialized per connection: directory/scriptname
	Source string
}

//ScrAbTester exposes methods to interact with the AbTester
type ScrAbTester interface {
	GetAbTester() abtester.Abtester
}

//WithConfig returns a function to attach the config to the scripter
func WithConfig(c toml.Primitive) ScripterFunc {
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

// setScriptInterval sets the interval of checking whether scripts have been changed
func SetScriptInterval(s Scripter) {

	// How often to fire the passed in function
	// in milliseconds
	interval := 10 * time.Second

	// Setup the ticket and the channel to signal
	// the ending of the interval
	ticker := time.NewTicker(interval)
	quit := make(chan struct{})

	// Put the selection in a go routine
	// so that the for loop is none blocking
	go func() {
		for {
			select {
			case <- ticker.C:
				go checkReloadScripts(s)
			case <- quit:
				ticker.Stop()
				return
			}

		}
	}()
}

// checkReloadScripts initializes services again when scripts have been changed within the service
func checkReloadScripts(s Scripter) {
	hasher := sha1.New()
	for service, scripts := range s.GetScripts() {
		isRenewService := false

		// check for edited files
		for _, script := range scripts {
			content, err := ioutil.ReadFile(script.Source)
			if err != nil {
				continue
			}
			hasher.Reset()
			hasher.Write(content)
			hash := hex.EncodeToString(hasher.Sum(nil))
			if script.Hash != hash {
				isRenewService = true
				script.Hash = hash
			}
		}

		// check for new files
		fileNames, _ := ioutil.ReadDir(fmt.Sprintf("%s/%s", s.GetScriptFolder(), service))
		fileCount := 0
		for _, f := range fileNames {
			if !f.IsDir() {
				fileCount++
			}
		}
		if fileCount != len(scripts) {
			isRenewService = true
		}

		// perform reload if needed
		if isRenewService {
			if err := s.Init(service); err != nil {
				log.Errorf("error init service: %s", err)
			} else {
				log.Infof("successfully updated service: %s", service)
			}
		}
	}
}
