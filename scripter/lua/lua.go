package lua

import (
	"fmt"
	"github.com/honeytrap/honeytrap/scripter"
	"github.com/op/go-logging"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
	"errors"
	"net"
	"context"
)

var log = logging.MustGetLogger("scripter/lua")

var (
	_ = scripter.Register("lua", New)
)

// Create a lua scripter instance that handles the connection to all lua-scripts
// A list where all scripts are stored in is generated
func New(name string, options ...func(scripter.Scripter) error) (scripter.Scripter, error) {
	s := &luaScripter{
		name: name,
	}

	for _, optionFn := range options {
		optionFn(s)
	}

	log.Infof("Using folder: %s", s.Folder)
	s.scripts = map[string]map[string]*lua.LState{}

	return s, nil
}

// The scripter state to which scripter functions are attached
type luaScripter struct {
	name string
	service string

	Folder string `toml:"folder"`

	//Source of the states, initialized per connection: directory/scriptname
	scripts map[string]map[string]*lua.LState
	//List of connections keyed by 'ip'
	connections map[string]scripterConn
}

// Initialize the scripts from a specific service
// The service name is given and the method will loop over all files in the lua-scripts folder with the given service name
// All of these scripts are then loaded and stored in the scripts map
func (l *luaScripter) InitScripts(service string) error {
	files, err := ioutil.ReadDir(fmt.Sprintf("%s/%s/%s", l.Folder, service, service))
	if err != nil {
		return err
	}

	// TODO: Load basic lua functions from shared context
	l.scripts[service] = map[string]*lua.LState {}

	for _, f := range files {
		ls := lua.NewState()
		ls.DoFile(fmt.Sprintf("%s/%s/%s/%s", l.Folder, service, service, f.Name()))
		if err != nil {
			return err
		}

		l.scripts[service][f.Name()] = ls
	}

	return nil
}

//func (l *luaScripter) SetGlobalFn(name string, fn func() string) error {
//	//for _, script := range l.scripts {
//	//	return l.SetStringFunction(name, fn)
//	//}
//}

// Run the given script on a given message
// Return the value that come out of function(message)
func handleScript(ls *lua.LState, message string) (string, error) {
	// Call method to handle the message
	if err := ls.CallByParam(lua.P{
		Fn:      ls.GetGlobal("handle"),
		NRet:    1,
		Protect: true,
	}, lua.LString(message)); err != nil {
		return "", errors.New(fmt.Sprintf("error calling handle method:%s", err))
	}

	// Get result of the function
	result := ls.Get(-1).String()
	ls.Pop(1)

	return result, nil
}

// Closes the scripter state
func (l *luaScripter) Close() {
	l.Close()
}

func (l *luaScripter) GetConnection(service string, conn net.Conn) ConnectionWrapper {
	ip := conn.RemoteAddr().String()
	var sConn scripterConn
	var ok bool
	if sConn, ok = l.connections[ip]; !ok {
		sConn = scripterConn{}
		sConn.conn = conn
		sConn.scripts = map[string]map[string]*lua.LState{}
		sConn.cancelFuncs = map[string]map[string]context.CancelFunc{}
	}

	if !sConn.hasScripts(service) {
		sConn.addScripts(service, l.scripts[service])
	}

	return ConnectionWrapper{service, sConn}
}

//////////  Connection Wrapper struct \\\\\\\\\\\
type ConnectionWrapper struct {
	service string
	conn scripterConn
}

// Handle incoming message string
// Get all scripts for a given service and pass the string to each script
func (w *ConnectionWrapper) Handle(message string) (string, error) {
	result := message
	var err error

	// TODO: Figure out the correct way to call all handle methods
	for _, script := range w.getScripts() {
		result, err = handleScript(script, result)
		if err != nil {
			return "", err
		}
	}

	return result, nil
}

func (w *ConnectionWrapper) getScripts() (map[string]*lua.LState) {
	return w.conn.scripts[w.service]
}

// Set a function that is available in all scripts for a service
func (w *ConnectionWrapper) SetStringFunction(name string, getString func() string) error {
	for _, script := range w.getScripts() {
		script.Register(name, func(state *lua.LState) int {
			state.Push(lua.LString(getString()))
			return 1
		})
	}

	return nil
}


//////////  Scripter Connection struct \\\\\\\\\\\
type scripterConn struct {
	conn net.Conn

	//List of lua scripts running for this connection: directory/scriptname
	scripts map[string]map[string]*lua.LState
	cancelFuncs map[string]map[string]context.CancelFunc
}

func (c *scripterConn) hasScripts(service string) bool {
	_, ok := c.scripts[service]
	return ok
}

func (c *scripterConn) addScripts(service string, scripts map[string]*lua.LState) {
	for name, script := range scripts {
		c.scripts[service][name], c.cancelFuncs[service][name] = script.NewThread()
	}
}