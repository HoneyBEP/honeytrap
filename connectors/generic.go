package connectors

import "net"

type generic struct {
	BaseConnector
}

func (g *generic) CanHandle(message []byte) bool {
	return false
}

func (g *generic) HandleService(conn net.Conn) error {
	return g.service.Handle(g.ctx, conn)
}

func (g *generic) HandleScripter(conn net.Conn) error {
	return nil
}

func (g *generic) DialContainer(conn net.Conn) error {
	return nil
}
