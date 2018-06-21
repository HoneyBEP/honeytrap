package ssh

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/honeytrap/honeytrap/connectors"
	"github.com/honeytrap/honeytrap/event"
	"github.com/honeytrap/honeytrap/scripter"
	"github.com/honeytrap/honeytrap/services"
	"github.com/honeytrap/honeytrap/services/decoder"
	sshService "github.com/honeytrap/honeytrap/services/ssh"
	"github.com/op/go-logging"
	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"net"
	"strings"
	"sync"
)

var (
	log = logging.MustGetLogger("connectors/ssh-simulator")
	_   = connectors.Register("ssh-simulator", New)
)

func New(fns ...connectors.ConnectorFunc) connectors.Connector {
	g := &sshSimulator{}

	for _, o := range fns {
		err := o(g)
		if err != nil {
			log.Errorf("error while calling connectorFunc: %s", err)
		}
	}

	return g
}

type sshSimulator struct {
	connectors.BaseConnector

	Banner string `toml:"banner"`
	MOTD   string `toml:"motd"`

	MaxAuthTries int `toml:"max-auth-tries"`

	Credentials []string    `toml:"credentials"`
	key         *privateKey `toml:"private-key"`
}

type handshakeData struct {
	id    xid.ID
	sconn *ssh.ServerConn
	chans <-chan ssh.NewChannel
	reqs  <-chan *ssh.Request
	scrConn  scripter.ConnectionWrapper
}

// privateKey holds the ssh.Signer instance to unsign received data.
type privateKey struct {
	ssh.Signer
}

type payloadDecoder struct {
	decoder.Decoder
}

func (pd *payloadDecoder) String() string {
	length := int(pd.Uint32())
	payload := pd.Copy(length)
	return string(payload)
}

func PayloadDecoder(payload []byte) *payloadDecoder {
	return &payloadDecoder{
		decoder.NewDecoder(payload),
	}
}

func (s *sshSimulator) CanHandle(payload []byte) bool {
	return bytes.HasPrefix(payload, []byte("SSH"))
}

func (s *sshSimulator) Handshake(conn net.Conn) (interface{}, error) {
	id := xid.New()

	config := ssh.ServerConfig{
		ServerVersion: s.Banner,
		MaxAuthTries:  s.MaxAuthTries,
		PublicKeyCallback: func(cm ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			s.GetChannel().Send(event.New(
				services.EventOptions,
				event.Category("ssh"),
				event.Type("publickey-authentication"),
				event.SourceAddr(cm.RemoteAddr()),
				event.DestinationAddr(cm.LocalAddr()),
				event.Custom("ssh.sessionid", id.String()),
				event.Custom("ssh.username", cm.User()),
				event.Custom("ssh.publickey-type", key.Type()),
				event.Custom("ssh.publickey", hex.EncodeToString(key.Marshal())),
			))

			return nil, fmt.Errorf("unknown key")
		},
		PasswordCallback: func(cm ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			s.GetChannel().Send(event.New(
				services.EventOptions,
				event.Category("ssh"),
				event.Type("password-authentication"),
				event.SourceAddr(cm.RemoteAddr()),
				event.DestinationAddr(cm.LocalAddr()),
				event.Custom("ssh.sessionid", id.String()),
				event.Custom("ssh.username", cm.User()),
				event.Custom("ssh.password", string(password)),
			))

			for _, credential := range s.Credentials {
				if credential == "*" {
					return nil, nil
				}

				parts := strings.Split(credential, ":")
				if len(parts) != 2 {
					continue
				}

				if cm.User() == parts[0] && string(password) == parts[1] {
					log.Debug("User authenticated successfully. user=%s password=%s", cm.User(), string(password))
					return nil, nil
				}
			}

			return nil, fmt.Errorf("password rejected for %q", cm.User())
		},
	}

	config.AddHostKey(s.key)

	defer conn.Close()

	sconn, chans, reqs, err := ssh.NewServerConn(conn, &config)
	if err == io.EOF {
		// server closed connection
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	defer func() {
		sconn.Close()
	}()

	go ssh.DiscardRequests(reqs)

	scrConn := s.GetScripter().GetConnection("ssh-simulator", conn)

	return handshakeData{
		id: id,
		sconn: sconn,
		chans: chans,
		reqs: reqs,
		scrConn: scrConn,
	}, nil

	}
	//
	//// https://tools.ietf.org/html/rfc4254
	//for newChannel := range chans {
	//	switch newChannel.ChannelType() {
	//	case "session":
	//		// handleSession()
	//	case "forwarded-tcpip":
	//		decoder := PayloadDecoder(newChannel.ExtraData())
	//
	//		s.GetChannel().Send(event.New(
	//			services.EventOptions,
	//			event.Category("ssh"),
	//			event.Type("ssh-channel"),
	//			event.SourceAddr(conn.RemoteAddr()),
	//			event.DestinationAddr(conn.LocalAddr()),
	//			event.Custom("ssh.sessionid", id.String()),
	//			event.Custom("ssh.channel-type", newChannel.ChannelType()),
	//			event.Custom("ssh.forwarded-tcpip.address-that-was-connected", decoder.String()),
	//			event.Custom("ssh.forwarded-tcpip.port-that-was-connected", fmt.Sprintf("%d", decoder.Uint32())),
	//			event.Custom("ssh.forwarded-tcpip.originator-host", decoder.String()),
	//			event.Custom("ssh.forwarded-tcpip.originator-port", fmt.Sprintf("%d", decoder.Uint32())),
	//			event.Payload(newChannel.ExtraData()),
	//		))
	//
	//		newChannel.Reject(ssh.UnknownChannelType, "not allowed")
	//		continue
	//	case "direct-tcpip":
	//		decoder := PayloadDecoder(newChannel.ExtraData())
	//
	//		s.GetChannel().Send(event.New(
	//			services.EventOptions,
	//			event.Category("ssh"),
	//			event.Type("ssh-channel"),
	//			event.SourceAddr(conn.RemoteAddr()),
	//			event.DestinationAddr(conn.LocalAddr()),
	//			event.Custom("ssh.sessionid", id.String()),
	//			event.Custom("ssh.channel-type", newChannel.ChannelType()),
	//			event.Custom("ssh.direct-tcpip.host-to-connect", decoder.String()),
	//			event.Custom("ssh.direct-tcpip.port-to-connect", fmt.Sprintf("%d", decoder.Uint32())),
	//			event.Custom("ssh.direct-tcpip.originator-host", decoder.String()),
	//			event.Custom("ssh.direct-tcpip.originator-port", fmt.Sprintf("%d", decoder.Uint32())),
	//			event.Payload(newChannel.ExtraData()),
	//		))
	//
	//		newChannel.Reject(ssh.UnknownChannelType, "not allowed")
	//		continue
	//	default:
	//		s.GetChannel().Send(event.New(
	//			services.EventOptions,
	//			event.Category("ssh"),
	//			event.Type("ssh-channel"),
	//			event.SourceAddr(conn.RemoteAddr()),
	//			event.DestinationAddr(conn.LocalAddr()),
	//			event.Custom("ssh.sessionid", id.String()),
	//			event.Custom("ssh.channel-type", newChannel.ChannelType()),
	//			event.Payload(newChannel.ExtraData()),
	//		))
	//
	//		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
	//		log.Debugf("Unknown channel type: %s\n", newChannel.ChannelType())
	//		continue
	//	}
	//
	//	channel, requests, err := newChannel.Accept()
	//	if err == io.EOF {
	//		continue
	//	} else if err != nil {
	//		log.Errorf("Could not accept server channel: %s", err.Error())
	//		continue
	//	}
	//
	//	func() {
	//		for req := range requests {
	//			log.Debugf("Request: %s %s %s %s\n", channel, req.Type, req.WantReply, req.Payload)
	//
	//			options := []event.Option{
	//				services.EventOptions,
	//				event.Category("ssh"),
	//				event.Type("ssh-request"),
	//				event.SourceAddr(conn.RemoteAddr()),
	//				event.DestinationAddr(conn.LocalAddr()),
	//				event.Custom("ssh.sessionid", id.String()),
	//				event.Custom("ssh.request-type", req.Type),
	//				event.Custom("ssh.payload", req.Payload),
	//			}
	//
	//			b := false
	//
	//			switch req.Type {
	//			case "shell":
	//				b = true
	//			case "pty-req":
	//				b = true
	//			case "env":
	//				b = true
	//
	//				decoder := PayloadDecoder(req.Payload)
	//
	//				var payloads []string
	//
	//				for {
	//					if decoder.Available() == 0 {
	//						break
	//					}
	//
	//					payload := decoder.String()
	//					payloads = append(payloads, payload)
	//				}
	//
	//				options = append(options, event.Custom("ssh.env", payloads))
	//			case "tcpip-forward":
	//				decoder := PayloadDecoder(req.Payload)
	//
	//				options = append(options, event.Custom("ssh.tcpip-forward.address-to-bind", decoder.String()))
	//				options = append(options, event.Custom("ssh.tcpip-forward.port-to-bind", fmt.Sprintf("%d", decoder.Uint32())))
	//			case "exec":
	//				b = true
	//
	//				decoder := PayloadDecoder(req.Payload)
	//
	//				var payloads []string
	//
	//				for {
	//					if decoder.Available() == 0 {
	//						break
	//					}
	//
	//					payload := decoder.String()
	//					payloads = append(payloads, payload)
	//				}
	//
	//				options = append(options, event.Custom("ssh.exec", payloads))
	//			case "subsystem":
	//				b = true
	//
	//				decoder := PayloadDecoder(req.Payload)
	//				options = append(options, event.Custom("ssh.subsystem", decoder.String()))
	//			default:
	//				log.Errorf("Unsupported request type=%s payload=%s", req.Type, string(req.Payload))
	//			}
	//
	//			if !b {
	//				// no reply
	//			} else if err := req.Reply(b, nil); err != nil {
	//				log.Errorf("wantreply: ", err)
	//			}
	//
	//			if req.Type != "exec" {
	//				s.GetChannel().Send(event.New(
	//					options...,
	//				))
	//			}
	//
	//			func() {
	//				if req.Type == "shell" {
	//					defer channel.Close()
	//
	//					// should only be started in req.Type == shell
	//					twrc := sshService.NewTypeWriterReadCloser(channel)
	//					var wrappedChannel io.ReadWriteCloser = twrc
	//
	//					prompt := "root@host:~$ "
	//
	//					term := terminal.NewTerminal(wrappedChannel, prompt)
	//
	//					term.Write([]byte(s.MOTD))
	//
	//					for {
	//						line, err := term.ReadLine()
	//						if err == io.EOF {
	//							return
	//						} else if err != nil {
	//							log.Errorf("Error reading from connection: %s", err.Error())
	//							return
	//						}
	//
	//						if line == "exit" {
	//							channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
	//							return
	//						}
	//
	//						if line == "" {
	//							continue
	//						}
	//
	//						resp, err := scrConn.Handle(line)
	//						if err != nil {
	//							resp = fmt.Sprintf("%s: command not found\n", line)
	//							log.Errorf("Error running scripter: %s", err.Error())
	//						}
	//
	//						s.GetChannel().Send(event.New(
	//							services.EventOptions,
	//							event.Category("ssh"),
	//							event.Type("ssh-channel"),
	//							event.SourceAddr(conn.RemoteAddr()),
	//							event.DestinationAddr(conn.LocalAddr()),
	//							event.Custom("ssh.sessionid", id.String()),
	//							event.Custom("ssh.command", line),
	//							event.Custom("response", resp),
	//						))
	//
	//						term.Write([]byte(resp))
	//					}
	//				} else if req.Type == "exec" {
	//					defer channel.Close()
	//
	//					decoder := PayloadDecoder(req.Payload)
	//
	//					for {
	//						if decoder.Available() == 0 {
	//							break
	//						}
	//
	//						payload := decoder.String()
	//
	//						resp, err := scrConn.Handle(payload)
	//
	//						if err != nil {
	//							resp = fmt.Sprintf("%s: command not found\n", payload)
	//							log.Errorf("Error running scripter: %s", err.Error())
	//							break
	//						}
	//
	//						options = append(options, event.Custom("response", resp))
	//						s.GetChannel().Send(event.New(
	//							options...,
	//						))
	//
	//						channel.Write([]byte(resp))
	//					}
	//
	//					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
	//					return
	//				} else {
	//				}
	//			}()
	//		}
	//	}()
	//}
	//
	//return nil, nil
}

func (s *sshSimulator) HandleService(conn net.Conn, d interface{}) error {
	cData := d.(handshakeData)

	defer func() {
		cData.sconn.Close()
	}()

	go ssh.DiscardRequests(cData.reqs)

	scrConn := s.GetScripter().GetConnection("ssh-simulator", conn)

	// https://tools.ietf.org/html/rfc4254
	for newChannel := range cData.chans {
		switch newChannel.ChannelType() {
		case "session":
			// handleSession()
		case "forwarded-tcpip":
			decoder := PayloadDecoder(newChannel.ExtraData())

			s.GetChannel().Send(event.New(
				services.EventOptions,
				event.Category("ssh"),
				event.Type("ssh-channel"),
				event.SourceAddr(conn.RemoteAddr()),
				event.DestinationAddr(conn.LocalAddr()),
				event.Custom("ssh.sessionid", cData.id.String()),
				event.Custom("ssh.channel-type", newChannel.ChannelType()),
				event.Custom("ssh.forwarded-tcpip.address-that-was-connected", decoder.String()),
				event.Custom("ssh.forwarded-tcpip.port-that-was-connected", fmt.Sprintf("%d", decoder.Uint32())),
				event.Custom("ssh.forwarded-tcpip.originator-host", decoder.String()),
				event.Custom("ssh.forwarded-tcpip.originator-port", fmt.Sprintf("%d", decoder.Uint32())),
				event.Payload(newChannel.ExtraData()),
			))

			newChannel.Reject(ssh.UnknownChannelType, "not allowed")
			continue
		case "direct-tcpip":
			decoder := PayloadDecoder(newChannel.ExtraData())

			s.GetChannel().Send(event.New(
				services.EventOptions,
				event.Category("ssh"),
				event.Type("ssh-channel"),
				event.SourceAddr(conn.RemoteAddr()),
				event.DestinationAddr(conn.LocalAddr()),
				event.Custom("ssh.sessionid", cData.id.String()),
				event.Custom("ssh.channel-type", newChannel.ChannelType()),
				event.Custom("ssh.direct-tcpip.host-to-connect", decoder.String()),
				event.Custom("ssh.direct-tcpip.port-to-connect", fmt.Sprintf("%d", decoder.Uint32())),
				event.Custom("ssh.direct-tcpip.originator-host", decoder.String()),
				event.Custom("ssh.direct-tcpip.originator-port", fmt.Sprintf("%d", decoder.Uint32())),
				event.Payload(newChannel.ExtraData()),
			))

			newChannel.Reject(ssh.UnknownChannelType, "not allowed")
			continue
		default:
			s.GetChannel().Send(event.New(
				services.EventOptions,
				event.Category("ssh"),
				event.Type("ssh-channel"),
				event.SourceAddr(conn.RemoteAddr()),
				event.DestinationAddr(conn.LocalAddr()),
				event.Custom("ssh.sessionid", cData.id.String()),
				event.Custom("ssh.channel-type", newChannel.ChannelType()),
				event.Payload(newChannel.ExtraData()),
			))

			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			log.Debugf("Unknown channel type: %s\n", newChannel.ChannelType())
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err == io.EOF {
			continue
		} else if err != nil {
			log.Errorf("Could not accept server channel: %s", err.Error())
			continue
		}

		func() {
			for req := range requests {
				log.Debugf("Request: %s %s %s %s\n", channel, req.Type, req.WantReply, req.Payload)

				options := []event.Option{
					services.EventOptions,
					event.Category("ssh"),
					event.Type("ssh-request"),
					event.SourceAddr(conn.RemoteAddr()),
					event.DestinationAddr(conn.LocalAddr()),
					event.Custom("ssh.sessionid", cData.id.String()),
					event.Custom("ssh.request-type", req.Type),
					event.Custom("ssh.payload", req.Payload),
				}

				b := false

				switch req.Type {
				case "shell":
					b = true
				case "pty-req":
					b = true
				case "env":
					b = true

					decoder := PayloadDecoder(req.Payload)

					var payloads []string

					for {
						if decoder.Available() == 0 {
							break
						}

						payload := decoder.String()
						payloads = append(payloads, payload)
					}

					options = append(options, event.Custom("ssh.env", payloads))
				case "tcpip-forward":
					decoder := PayloadDecoder(req.Payload)

					options = append(options, event.Custom("ssh.tcpip-forward.address-to-bind", decoder.String()))
					options = append(options, event.Custom("ssh.tcpip-forward.port-to-bind", fmt.Sprintf("%d", decoder.Uint32())))
				case "exec":
					b = true

					decoder := PayloadDecoder(req.Payload)

					var payloads []string

					for {
						if decoder.Available() == 0 {
							break
						}

						payload := decoder.String()
						payloads = append(payloads, payload)
					}

					options = append(options, event.Custom("ssh.exec", payloads))
				case "subsystem":
					b = true

					decoder := PayloadDecoder(req.Payload)
					options = append(options, event.Custom("ssh.subsystem", decoder.String()))
				default:
					log.Errorf("Unsupported request type=%s payload=%s", req.Type, string(req.Payload))
				}

				if !b {
					// no reply
				} else if err := req.Reply(b, nil); err != nil {
					log.Errorf("wantreply: ", err)
				}

				if req.Type != "exec" {
					s.GetChannel().Send(event.New(
						options...,
					))
				}

				func() {
					if req.Type == "shell" {
						defer channel.Close()

						// should only be started in req.Type == shell
						twrc := sshService.NewTypeWriterReadCloser(channel)
						var wrappedChannel io.ReadWriteCloser = twrc

						prompt := "root@host:~$ "

						term := terminal.NewTerminal(wrappedChannel, prompt)

						term.Write([]byte(s.MOTD))

						for {
							line, err := term.ReadLine()
							if err == io.EOF {
								return
							} else if err != nil {
								log.Errorf("Error reading from connection: %s", err.Error())
								return
							}

							if line == "exit" {
								channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
								return
							}

							if line == "" {
								continue
							}

							resp, err := scrConn.Handle(line)
							if err != nil {
								resp = fmt.Sprintf("%s: command not found\n", line)
								log.Errorf("Error running scripter: %s", err.Error())
							}

							s.GetChannel().Send(event.New(
								services.EventOptions,
								event.Category("ssh"),
								event.Type("ssh-channel"),
								event.SourceAddr(conn.RemoteAddr()),
								event.DestinationAddr(conn.LocalAddr()),
								event.Custom("ssh.sessionid", cData.id.String()),
								event.Custom("ssh.command", line),
								event.Custom("response", resp),
							))

							term.Write([]byte(resp))
						}
					} else if req.Type == "exec" {
						defer channel.Close()

						decoder := PayloadDecoder(req.Payload)

						for {
							if decoder.Available() == 0 {
								break
							}

							payload := decoder.String()

							resp, err := scrConn.Handle(payload)

							if err != nil {
								resp = fmt.Sprintf("%s: command not found\n", payload)
								log.Errorf("Error running scripter: %s", err.Error())
								break
							}

							options = append(options, event.Custom("response", resp))
							s.GetChannel().Send(event.New(
								options...,
							))

							channel.Write([]byte(resp))
						}

						channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
						return
					} else {
					}
				}()
			}
		}()
	}

	return nil
	//return s.GetService().Handle(s.GetContext(), conn)
}

func (s *sshSimulator) HandleScripter(conn net.Conn, sm sync.Map) error {
	return nil
}

func (s *sshSimulator) DialContainer(conn net.Conn, sm sync.Map) error {
	return nil
}
