package connectors

import (
	"context"
	"github.com/honeytrap/honeytrap/director"
	"github.com/honeytrap/honeytrap/pushers"
	"github.com/honeytrap/honeytrap/scripter"
	"github.com/honeytrap/honeytrap/services"
)

type BaseConnector struct {
	name          string
	connectorType string
	ctx           context.Context
	channel       pushers.Channel

	connectorMode string

	service  services.Servicer
	director director.Director
	scripter scripter.Scripter
}

func (c *BaseConnector) GetName() string {
	return c.name
}

func (c *BaseConnector) GetType() string {
	return c.connectorType
}

func (c *BaseConnector) SetMode(mode string) {
	c.connectorMode = mode
}

func (c *BaseConnector) GetMode() string {
	return c.connectorMode
}

func (c *BaseConnector) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *BaseConnector) SetChannel(channel pushers.Channel) {
	c.channel = channel
}

func (c *BaseConnector) GetContext() context.Context {
	return c.ctx
}

func (c *BaseConnector) GetChannel() pushers.Channel {
	return c.channel
}


func (c *BaseConnector) SetService(s services.Servicer) {
	c.service = s
}

func (c *BaseConnector) SetDirector(d director.Director) {
	c.director = d
}

func (c *BaseConnector) SetScripter(scr scripter.Scripter) {
	c.scripter = scr
}

func (c *BaseConnector) GetService() services.Servicer {
	return c.service
}

func (c *BaseConnector) GetDirector() director.Director {
	return c.director
}

func (c *BaseConnector) GetScripter() scripter.Scripter {
	return c.scripter
}

