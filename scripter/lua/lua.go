package lua

import (
	"fmt"
	"github.com/honeytrap/honeytrap/abtester"
	"github.com/honeytrap/honeytrap/scripter"
	"github.com/op/go-logging"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"net"
	"strings"
	"github.com/honeytrap/honeytrap/pushers"
	"time"
	"crypto/sha1"
	"encoding/hex"
)

var log = logging.MustGetLogger("scripter/lua")

var (
	_ = scripter.Register("lua", New)
)

// New creates a lua scripter instance that handles the connection to all lua-scripts
// A list where all scripts are stored in is generated
func New(name string, options ...scripter.ScripterFunc) (scripter.Scripter, error) {
	l := &luaScripter{
		name: name,
	}

	for _, optionFn := range options {
		optionFn(l)
	}

	log.Infof("Using folder: %s", l.Folder)
	l.scripts = map[string]map[string]*luaScript{}
	l.connections = map[string]*luaConn{}
	l.canHandleStates = map[string]map[string]*lua.LState{}
	l.abTester, _ = abtester.Namespace("lua")

	if err := l.abTester.LoadFromFile("scripter/abtests.json"); err != nil {
		return nil, err
	}

	l.setScriptInterval()

	return l, nil
}

// The scripter state to which scripter functions are attached
type luaScripter struct {
	name string

	Folder string `toml:"folder"`

	//Source of the states, initialized per connection: directory/scriptname
	scripts map[string]map[string]*luaScript
	//List of connections keyed by 'ip'
	connections map[string]*luaConn
	//Lua states to check whether the connection can be handled with the script
	canHandleStates map[string]map[string]*lua.LState

	abTester abtester.Abtester

	c pushers.Channel
}

//Set the channel over which messages to the log and elasticsearch can be set
func (l *luaScripter) SetChannel(c pushers.Channel) {
	l.c = c
}

type luaScript struct {
	// hash of the file
	hash string

	// source of the states, initialized per connection: directory/scriptname
	source string
}

// Init initializes the scripts from a specific service
// The service name is given and the method will loop over all files in the lua-scripts folder with the given service name
// All of these scripts are then loaded and stored in the scripts map
func (l *luaScripter) Init(service string) error {
	fileNames, err := ioutil.ReadDir(fmt.Sprintf("%s/%s/%s", l.Folder, l.name, service))
	if err != nil {
		return err
	}

	// TODO: Load basic lua functions from shared context
	hasher := sha1.New()

	l.connections = map[string]*luaConn{}
	l.scripts[service] = map[string]*luaScript{}
	l.canHandleStates[service] = map[string]*lua.LState{}

	for _, f := range fileNames {
		sf := fmt.Sprintf("%s/%s/%s/%s", l.Folder, l.name, service, f.Name())

		hash := ""
		content, err := ioutil.ReadFile(sf)
		if err == nil {
			hasher.Reset()
			hasher.Write(content)
			hash = hex.EncodeToString(hasher.Sum(nil))
		}

		l.scripts[service][f.Name()] = &luaScript{hash, sf}

		ls := lua.NewState()
		if err := ls.DoFile(sf); err != nil {
			return err
		}
		l.canHandleStates[service][f.Name()] = ls
	}

	return nil
}

//GetConnection returns a connection for the given ip-address, if no connection exists yet, create it.
func (l *luaScripter) GetConnection(service string, conn net.Conn) scripter.ConnectionWrapper {
	ip := getConnIP(conn)

	sConn, ok := l.connections[ip]
	if !ok {
		sConn = &luaConn{
			conn: conn,
			scripts: map[string]map[string]*lua.LState{},
			abTester: l.abTester,
		}
		l.connections[ip] = sConn
	} else {
		sConn.conn = conn
	}

	if !sConn.HasScripts(service) {
		scripts := make(map[string]string)
		for k, v := range l.scripts[service] {
			scripts[k] = v.source
		}
		sConn.AddScripts(service, scripts)
	}

	return &scripter.ConnectionStruct{Service: service, Conn: sConn}
}

// CanHandle checks whether scripter can handle incoming connection for the peeked message
// Returns true if there is one script able to handle the connection
func (l *luaScripter) CanHandle(service string, message string) bool {
	for _, ls := range l.canHandleStates[service] {
		canHandle, err := callCanHandle(ls, message)
		if err != nil {
			log.Errorf("%s", err)
		} else if canHandle {
			return true
		}
	}

	return false
}

// getConnIP retrieves the IP fron a connection's remote address
func getConnIP(conn net.Conn) string {
	s := strings.Split(conn.RemoteAddr().String(), ":")
	s = s[:len(s)-1]
	return strings.Join(s, ":")
}

// setScriptInterval sets the interval of checking whether scripts have been changed
func (l *luaScripter) setScriptInterval() {

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
				go l.checkReloadScripts()
			case <- quit:
				ticker.Stop()
				return
			}

		}
	}()
}

// checkReloadScripts initializes services again when scripts have been changed within the service
func (l *luaScripter) checkReloadScripts() {
	hasher := sha1.New()
	for service, scripts := range l.scripts {
		isRenewService := false
		for _, script := range scripts {
			content, err := ioutil.ReadFile(script.source);
			if err != nil {
				continue
			}
			hasher.Reset()
			hasher.Write(content)
			hash := hex.EncodeToString(hasher.Sum(nil))
			if script.hash != hash {
				isRenewService = true
				script.hash = hash
			}
		}

		if isRenewService {
			l.Init(service)
		}
	}
}