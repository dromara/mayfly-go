package guac

type Status int

const (
	// Undefined Add to instead null
	Undefined Status = -1

	// Success indicates the operation succeeded.
	Success Status = iota

	// Unsupported indicates the requested operation is unsupported.
	Unsupported

	// ServerError indicates the operation could not be performed due to an internal failure.
	ServerError

	// ServerBusy indicates the operation could not be performed as the server is busy.
	ServerBusy

	// UpstreamTimeout indicates the operation could not be performed because the upstream server is not responding.
	UpstreamTimeout

	// UpstreamError indicates the operation was unsuccessful due to an error or otherwise unexpected
	// condition of the upstream server.
	UpstreamError

	// ResourceNotFound indicates the operation could not be performed as the requested resource does not exist.
	ResourceNotFound

	// ResourceConflict indicates the operation could not be performed as the requested resource is already in use.
	ResourceConflict

	// ResourceClosed indicates the operation could not be performed as the requested resource is now closed.
	ResourceClosed

	// UpstreamNotFound indicates the operation could not be performed because the upstream server does
	// not appear to exist.
	UpstreamNotFound

	// UpstreamUnavailable indicates the operation could not be performed because the upstream server is not
	// available to service the request.
	UpstreamUnavailable

	// SessionConflict indicates the session within the upstream server has ended because it conflicted
	// with another session.
	SessionConflict

	// SessionTimeout indicates the session within the upstream server has ended because it appeared to be inactive.
	SessionTimeout

	// SessionClosed indicates the session within the upstream server has been forcibly terminated.
	SessionClosed

	// ClientBadRequest indicates the operation could not be performed because bad parameters were given.
	ClientBadRequest

	// ClientUnauthorized indicates the user is not authorized.
	ClientUnauthorized

	// ClientForbidden indicates the user is not allowed to do the operation.
	ClientForbidden

	// ClientTimeout indicates the client took too long to respond.
	ClientTimeout

	// ClientOverrun indicates the client sent too much data.
	ClientOverrun

	// ClientBadType indicates the client sent data of an unsupported or unexpected type.
	ClientBadType

	// ClientTooMany indivates the operation failed because the current client is already using too many resources.
	ClientTooMany
)

type statusData struct {
	name string
	// The most applicable HTTP error code.
	httpCode int

	// The most applicable WebSocket error code.
	websocketCode int

	// The Guacamole protocol Status code.
	guacCode int
}

func newStatusData(name string, httpCode, websocketCode, guacCode int) (ret statusData) {
	ret.name = name
	ret.httpCode = httpCode
	ret.websocketCode = websocketCode
	ret.guacCode = guacCode
	return
}

var guacamoleStatusMap = map[Status]statusData{
	Success:             newStatusData("Success", 200, 1000, 0x0000),
	Unsupported:         newStatusData("Unsupported", 501, 1011, 0x0100),
	ServerError:         newStatusData("SERVER_ERROR", 500, 1011, 0x0200),
	ServerBusy:          newStatusData("SERVER_BUSY", 503, 1008, 0x0201),
	UpstreamTimeout:     newStatusData("UPSTREAM_TIMEOUT", 504, 1011, 0x0202),
	UpstreamError:       newStatusData("UPSTREAM_ERROR", 502, 1011, 0x0203),
	ResourceNotFound:    newStatusData("RESOURCE_NOT_FOUND", 404, 1002, 0x0204),
	ResourceConflict:    newStatusData("RESOURCE_CONFLICT", 409, 1008, 0x0205),
	ResourceClosed:      newStatusData("RESOURCE_CLOSED", 404, 1002, 0x0206),
	UpstreamNotFound:    newStatusData("UPSTREAM_NOT_FOUND", 502, 1011, 0x0207),
	UpstreamUnavailable: newStatusData("UPSTREAM_UNAVAILABLE", 502, 1011, 0x0208),
	SessionConflict:     newStatusData("SESSION_CONFLICT", 409, 1008, 0x0209),
	SessionTimeout:      newStatusData("SESSION_TIMEOUT", 408, 1002, 0x020A),
	SessionClosed:       newStatusData("SESSION_CLOSED", 404, 1002, 0x020B),
	ClientBadRequest:    newStatusData("CLIENT_BAD_REQUEST", 400, 1002, 0x0300),
	ClientUnauthorized:  newStatusData("CLIENT_UNAUTHORIZED", 403, 1008, 0x0301),
	ClientForbidden:     newStatusData("CLIENT_FORBIDDEN", 403, 1008, 0x0303),
	ClientTimeout:       newStatusData("CLIENT_TIMEOUT", 408, 1002, 0x0308),
	ClientOverrun:       newStatusData("CLIENT_OVERRUN", 413, 1009, 0x030D),
	ClientBadType:       newStatusData("CLIENT_BAD_TYPE", 415, 1003, 0x030F),
	ClientTooMany:       newStatusData("CLIENT_TOO_MANY", 429, 1008, 0x031D),
}

// String returns the name of the status.
func (s Status) String() string {
	if v, ok := guacamoleStatusMap[s]; ok {
		return v.name
	}
	return ""
}

// GetHTTPStatusCode returns the most applicable HTTP error code.
func (s Status) GetHTTPStatusCode() int {
	if v, ok := guacamoleStatusMap[s]; ok {
		return v.httpCode
	}
	return -1
}

// GetWebSocketCode returns the most applicable HTTP error code.
func (s Status) GetWebSocketCode() int {
	if v, ok := guacamoleStatusMap[s]; ok {
		return v.websocketCode
	}
	return -1
}

// GetGuacamoleStatusCode returns the corresponding Guacamole protocol Status code.
func (s Status) GetGuacamoleStatusCode() int {
	if v, ok := guacamoleStatusMap[s]; ok {
		return v.guacCode
	}
	return -1
}

// FromGuacamoleStatusCode returns the Status corresponding to the given Guacamole protocol Status code.
func FromGuacamoleStatusCode(code int) (ret Status) {
	// Search for a Status having the given Status code
	for k, v := range guacamoleStatusMap {
		if v.guacCode == code {
			ret = k
			return
		}
	}
	// No such Status found
	ret = Undefined
	return

}
