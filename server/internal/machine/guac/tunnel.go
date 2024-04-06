package guac

import (
	"fmt"
	"github.com/google/uuid"
	"io"
)

// The Guacamole protocol instruction Opcode reserved for arbitrary
// internal use by tunnel implementations. The value of this Opcode is
// guaranteed to be the empty string (""). Tunnel implementations may use
// this Opcode for any purpose. It is currently used by the HTTP tunnel to
// mark the end of the HTTP response, and by the WebSocket tunnel to
// transmit the tunnel UUID.
const InternalDataOpcode = ""

var internalOpcodeIns = []byte(fmt.Sprint(len(InternalDataOpcode), ".", InternalDataOpcode))

// InstructionReader provides reading functionality to a Stream
type InstructionReader interface {
	// ReadSome returns the next complete guacd message from the stream
	ReadSome() ([]byte, error)
	// Available returns true if there are bytes buffered in the stream
	Available() bool
	// Flush resets the internal buffer for reuse
	Flush()
}

// Tunnel provides a unique identifier and synchronized access to the InstructionReader and Writer
// associated with a Stream.
type Tunnel interface {
	// AcquireReader returns a reader to the tunnel if it isn't locked
	AcquireReader() InstructionReader
	// ReleaseReader releases the lock on the reader
	ReleaseReader()
	// HasQueuedReaderThreads returns true if there is a reader locked
	HasQueuedReaderThreads() bool
	// AcquireWriter returns a writer to the tunnel if it isn't locked
	AcquireWriter() io.Writer
	// ReleaseWriter releases the lock on the writer
	ReleaseWriter()
	// HasQueuedWriterThreads returns true if there is a writer locked
	HasQueuedWriterThreads() bool
	// GetUUID returns the uuid of the tunnel
	GetUUID() string
	// ConnectionId returns the guacd Connection ID of the tunnel
	ConnectionID() string
	// Close closes the tunnel
	Close() error
}

// Base Tunnel implementation which synchronizes access to the underlying reader and writer with locks
type SimpleTunnel struct {
	stream *Stream
	/**
	 * The UUID associated with this tunnel. Every tunnel must have a
	 * corresponding UUID such that tunnel read/write requests can be
	 * directed to the proper tunnel.
	 */
	uuid       uuid.UUID
	readerLock CountedLock
	writerLock CountedLock
}

// NewSimpleTunnel creates a new tunnel
func NewSimpleTunnel(stream *Stream) *SimpleTunnel {
	return &SimpleTunnel{
		stream: stream,
		uuid:   uuid.New(),
	}
}

// AcquireReader acquires the reader lock
func (t *SimpleTunnel) AcquireReader() InstructionReader {
	t.readerLock.Lock()
	return t.stream
}

// ReleaseReader releases the reader
func (t *SimpleTunnel) ReleaseReader() {
	t.readerLock.Unlock()
}

// HasQueuedReaderThreads returns true if more than one goroutine is trying to read
func (t *SimpleTunnel) HasQueuedReaderThreads() bool {
	return t.readerLock.HasQueued()
}

// AcquireWriter locks the writer lock
func (t *SimpleTunnel) AcquireWriter() io.Writer {
	t.writerLock.Lock()
	return t.stream
}

// ReleaseWriter releases the writer lock
func (t *SimpleTunnel) ReleaseWriter() {
	t.writerLock.Unlock()
}

// ConnectionID returns the underlying Guacamole connection ID
func (t *SimpleTunnel) ConnectionID() string {
	return t.stream.ConnectionID
}

// HasQueuedWriterThreads returns true if more than one goroutine is trying to write
func (t *SimpleTunnel) HasQueuedWriterThreads() bool {
	return t.writerLock.HasQueued()
}

// Close closes the underlying stream
func (t *SimpleTunnel) Close() (err error) {
	return t.stream.Close()
}

// GetUUID returns the tunnel's UUID
func (t *SimpleTunnel) GetUUID() string {
	return t.uuid.String()
}
