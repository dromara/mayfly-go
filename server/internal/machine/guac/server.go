package guac

import (
	"fmt"
	"io"
	"mayfly-go/pkg/logx"
	"net/http"
	"strings"
)

const (
	readPrefix        string = "read:"
	writePrefix       string = "write:"
	readPrefixLength         = len(readPrefix)
	writePrefixLength        = len(writePrefix)
	uuidLength               = 36
)

// Server uses HTTP requests to talk to guacd (as opposed to WebSockets in ws_server.go)
type Server struct {
	tunnels *TunnelMap
	connect func(*http.Request) (Tunnel, error)
}

// NewServer constructor
func NewServer(connect func(r *http.Request) (Tunnel, error)) *Server {
	return &Server{
		tunnels: NewTunnelMap(),
		connect: connect,
	}
}

// Registers the given tunnel such that future read/write requests to that tunnel will be properly directed.
func (s *Server) registerTunnel(tunnel Tunnel) {
	s.tunnels.Put(tunnel.GetUUID(), tunnel)
	logx.Debugf("Registered tunnel %v.", tunnel.GetUUID())
}

// Deregisters the given tunnel such that future read/write requests to that tunnel will be rejected.
func (s *Server) deregisterTunnel(tunnel Tunnel) {
	s.tunnels.Remove(tunnel.GetUUID())
	logx.Debugf("Deregistered tunnel %v.", tunnel.GetUUID())
}

// Returns the tunnel with the given UUID.
func (s *Server) getTunnel(tunnelUUID string) (ret Tunnel, err error) {
	var ok bool
	ret, ok = s.tunnels.Get(tunnelUUID)

	if !ok {
		err = ErrResourceNotFound.NewError("No such tunnel.")
	}
	return
}

func (s *Server) sendError(response http.ResponseWriter, guacStatus Status, message string) {
	response.Header().Set("Guacamole-Status-Code", fmt.Sprintf("%v", guacStatus.GetGuacamoleStatusCode()))
	response.Header().Set("Guacamole-Error-Message", message)
	response.WriteHeader(guacStatus.GetHTTPStatusCode())
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := s.handleTunnelRequestCore(w, r)
	if err == nil {
		return
	}
	guacErr := err.(*ErrGuac)
	switch guacErr.Kind {
	case ErrClient:
		logx.Warnf("HTTP tunnel request rejected: %s", err.Error())
		s.sendError(w, guacErr.Status, err.Error())
	default:
		logx.Errorf("HTTP tunnel request failed: %s", err.Error())
		s.sendError(w, guacErr.Status, "Internal server error.")
	}
	return
}

func (s *Server) handleTunnelRequestCore(response http.ResponseWriter, request *http.Request) (err error) {
	query := request.URL.RawQuery
	if len(query) == 0 {
		return ErrClient.NewError("No query string provided.")
	}

	// Call the supplied connect callback upon HTTP connect request
	if query == "connect" {
		tunnel, e := s.connect(request)
		if e != nil {
			err = ErrResourceNotFound.NewError("No tunnel created.", e.Error())
			return
		}

		s.registerTunnel(tunnel)

		// Ensure buggy browsers do not cache response
		response.Header().Set("Cache-Control", "no-cache")

		_, e = response.Write([]byte(tunnel.GetUUID()))

		if e != nil {
			err = ErrServer.NewError(e.Error())
			return
		}
		return
	}

	// Connect has already been called so we use the UUID to do read and writes to the existing session
	if strings.HasPrefix(query, readPrefix) && len(query) >= readPrefixLength+uuidLength {
		err = s.doRead(response, request, query[readPrefixLength:readPrefixLength+uuidLength])
	} else if strings.HasPrefix(query, writePrefix) && len(query) >= writePrefixLength+uuidLength {
		err = s.doWrite(response, request, query[writePrefixLength:writePrefixLength+uuidLength])
	} else {
		err = ErrClient.NewError("Invalid tunnel operation: " + query)
	}

	return
}

// doRead takes guacd messages and sends them in the response
func (s *Server) doRead(response http.ResponseWriter, request *http.Request, tunnelUUID string) error {
	tunnel, err := s.getTunnel(tunnelUUID)
	if err != nil {
		return err
	}

	reader := tunnel.AcquireReader()
	defer tunnel.ReleaseReader()

	// Note that although we are sending text, Webkit browsers will
	// buffer 1024 bytes before starting a normal stream if we use
	// anything but application/octet-stream.
	response.Header().Set("Content-Type", "application/octet-stream")
	response.Header().Set("Cache-Control", "no-cache")

	if v, ok := response.(http.Flusher); ok {
		v.Flush()
	}

	err = s.writeSome(response, reader, tunnel)

	if err == nil {
		// success
		return err
	}

	switch err.(*ErrGuac).Kind {
	// Send end-of-stream marker and close tunnel if connection is closed
	case ErrConnectionClosed:
		s.deregisterTunnel(tunnel)
		tunnel.Close()

		// End-of-instructions marker
		_, _ = response.Write([]byte("0.;"))
		if v, ok := response.(http.Flusher); ok {
			v.Flush()
		}
	default:
		logx.Debugf("Error writing to output, %v", err)
		s.deregisterTunnel(tunnel)
		tunnel.Close()
	}

	return err
}

// writeSome drains the guacd buffer holding instructions into the response
func (s *Server) writeSome(response http.ResponseWriter, guacd InstructionReader, tunnel Tunnel) (err error) {
	var message []byte

	for {
		message, err = guacd.ReadSome()
		if err != nil {
			s.deregisterTunnel(tunnel)
			tunnel.Close()
			return
		}

		if len(message) == 0 {
			return
		}

		_, e := response.Write(message)
		if e != nil {
			err = ErrOther.NewError(e.Error())
			return
		}

		if !guacd.Available() {
			if v, ok := response.(http.Flusher); ok {
				v.Flush()
			}
		}

		// No more messages another guacd can take over
		if tunnel.HasQueuedReaderThreads() {
			break
		}
	}

	// End-of-instructions marker
	if _, err = response.Write([]byte("0.;")); err != nil {
		return err
	}
	if v, ok := response.(http.Flusher); ok {
		v.Flush()
	}
	return nil
}

// doWrite takes data from the request and sends it to guacd
func (s *Server) doWrite(response http.ResponseWriter, request *http.Request, tunnelUUID string) error {
	tunnel, err := s.getTunnel(tunnelUUID)
	if err != nil {
		return err
	}

	// We still need to set the content type to avoid the default of
	// text/html, as such a content type would cause some browsers to
	// attempt to parse the result, even though the JavaScript client
	// does not explicitly request such parsing.
	response.Header().Set("Content-Type", "application/octet-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Content-Length", "0")

	writer := tunnel.AcquireWriter()
	defer tunnel.ReleaseWriter()

	_, err = io.Copy(writer, request.Body)

	if err != nil {
		s.deregisterTunnel(tunnel)
		if err = tunnel.Close(); err != nil {
			logx.Debugf("Error closing tunnel: %v", err)
		}
	}

	return err
}
