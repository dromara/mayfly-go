package guac

import (
	"fmt"
	"strings"
)

type ErrGuac struct {
	error
	Status Status
	Kind   ErrKind
}

type ErrKind int

const (
	ErrClientBadType ErrKind = iota
	ErrClient
	ErrClientOverrun
	ErrClientTimeout
	ErrClientTooMany
	ErrConnectionClosed
	ErrOther
	ErrResourceClosed
	ErrResourceConflict
	ErrResourceNotFound
	ErrSecurity
	ErrServerBusy
	ErrServer
	ErrSessionClosed
	ErrSessionConflict
	ErrSessionTimeout
	ErrUnauthorized
	ErrUnsupported
	ErrUpstream
	ErrUpstreamNotFound
	ErrUpstreamTimeout
	ErrUpstreamUnavailable
)

// Status convert ErrKind to Status
func (e ErrKind) Status() (state Status) {
	switch e {
	case ErrClientBadType:
		return ClientBadType
	case ErrClient:
		return ClientBadRequest
	case ErrClientOverrun:
		return ClientOverrun
	case ErrClientTimeout:
		return ClientTimeout
	case ErrClientTooMany:
		return ClientTooMany
	case ErrConnectionClosed:
		return ServerError
	case ErrOther:
		return ServerError
	case ErrResourceClosed:
		return ResourceClosed
	case ErrResourceConflict:
		return ResourceConflict
	case ErrResourceNotFound:
		return ResourceNotFound
	case ErrSecurity:
		return ClientForbidden
	case ErrServerBusy:
		return ServerBusy
	case ErrServer:
		return ServerError
	case ErrSessionClosed:
		return SessionClosed
	case ErrSessionConflict:
		return SessionConflict
	case ErrSessionTimeout:
		return SessionTimeout
	case ErrUnauthorized:
		return ClientUnauthorized
	case ErrUnsupported:
		return Unsupported
	case ErrUpstream:
		return UpstreamError
	case ErrUpstreamNotFound:
		return UpstreamNotFound
	case ErrUpstreamTimeout:
		return UpstreamTimeout
	case ErrUpstreamUnavailable:
		return UpstreamUnavailable
	}
	return
}

// NewError creates a new error struct instance with Kind and included message
func (e ErrKind) NewError(args ...string) error {
	return &ErrGuac{
		error:  fmt.Errorf("%v", strings.Join(args, ", ")),
		Status: e.Status(),
		Kind:   e,
	}
}
