package connectors

import (
	"context"
	"github.com/BurntSushi/toml"
	"github.com/honeytrap/honeytrap/director"
	"github.com/honeytrap/honeytrap/pushers"
	"github.com/honeytrap/honeytrap/scripter"
	"github.com/honeytrap/honeytrap/services"
	"github.com/op/go-logging"
	"net"
	"fmt"
	"sync"
)

var (
	log        = logging.MustGetLogger("connectors")
	connectors = map[string]func(...ConnectorFunc) Connector{}
)

func Register(key string, fn func(...ConnectorFunc) Connector) func(...ConnectorFunc) Connector {
	connectors[key] = fn
	return fn
}

func Get(key string) (func(...ConnectorFunc) Connector, bool) {
	if fn, ok := connectors[key]; ok {
		return fn, true
	}

	return nil, false
}

func GetAvailableConnectorNames() []string {
	var out []string
	for key := range connectors {
		out = append(out, key)
	}
	return out
}


type ConnectorFunc func(Connector) error

type Connector interface {
	GetName() string
	GetType() string

	SetMode(string)
	GetMode() string

	SetContext(ctx context.Context)
	SetChannel(c pushers.Channel)

	GetContext() context.Context
	GetChannel() pushers.Channel

	SetService(services.Servicer)
	SetDirector(director.Director)
	SetScripter(scripter.Scripter)

	GetService() services.Servicer
	GetDirector() director.Director
	GetScripter() scripter.Scripter
}

type CanHandlerer interface {
	CanHandle([]byte) bool
}

// Handshaker performs handshake on incoming connections if it is implemented
type Handshaker interface {
	Handshake(net.Conn) (sync.Map, error)
}

// ContainerConnector implements connection to the container
type ContainerConnector interface {
	DialContainer(net.Conn, interface{}) error
}

// ServiceConnector handles the connection in the service
type ServiceConnector interface {
	HandleService(net.Conn, interface{}) error
}

// ScripterConnector handles the connection in the scripter
type ScripterConnector interface {
	HandleScripter(net.Conn, interface{}) error
}

func HandleConn(c Connector, conn net.Conn) error {
	var data interface{}

	h, ok := c.(Handshaker)
	if ok {
		data, err := h.Handshake(conn)
		if err != nil {
			return err
		}
	}

	for {
		var err error

		switch c.GetMode() {
		case MODE_SERVICE:
			s, ok := c.(ServiceConnector)
			if !ok {
				return fmt.Errorf("service mode is not implemented")
			}
			err = s.HandleService(conn, data)
			break
		case MODE_DIRECTOR:
			cnt, ok := c.(ContainerConnector)
			if !ok {
				return fmt.Errorf("director mode is not implemented")
			}
			err = cnt.DialContainer(conn, data)
			break
		case MODE_SCRIPTER:
			scr, ok := c.(ScripterConnector)
			if !ok {
				return fmt.Errorf("scripter mode is not implemented")
			}
			err = scr.HandleScripter(conn, data)
			break
		case MODE_TERMINATE:
			// With MODE_TERMINATE we return successfully
			return nil
		default:
			fmt.Errorf("unrecognized mode: '%s'", c.GetMode())
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func WithConfig(p toml.Primitive) ConnectorFunc {
	return func(c Connector) error {
		err := toml.PrimitiveDecode(p, c)
		return err
	}
}

func WithContext(ctx context.Context) ConnectorFunc {
	return func(c Connector) error {
		c.SetContext(ctx)
		return nil
	}
}

func WithChannel(eb pushers.Channel) ConnectorFunc {
	return func(c Connector) error {
		c.SetChannel(eb)
		return nil
	}
}

func WithService(s services.Servicer) ConnectorFunc {
	return func(c Connector) error {
		c.SetService(s)
		return nil
	}
}

func WithDirector(d director.Director) ConnectorFunc {
	return func(c Connector) error {
		c.SetDirector(d)
		return nil
	}
}

func WithScripter(scr scripter.Scripter) ConnectorFunc {
	return func(c Connector) error {
		c.SetScripter(scr)
		return nil
	}
}
