package guac

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"net"
	"time"
)

const (
	SocketTimeout  = 15 * time.Second
	MaxGuacMessage = 8192 // TODO is this bytes or runes?
)

// Stream wraps the connection to Guacamole providing timeouts and reading
// a single instruction at a time (since returning partial instructions
// would be an error)
type Stream struct {
	conn net.Conn

	// ConnectionID is the ID Guacamole gives and can be used to reconnect or share sessions
	ConnectionID string
	timeout      time.Duration

	// if more than a single instruction is read, the rest are buffered here
	parseStart int
	buffer     []rune
	reset      []rune
}

// NewStream creates a new stream
func NewStream(conn net.Conn, timeout time.Duration) (ret *Stream) {
	buffer := make([]rune, 0, MaxGuacMessage*3)
	return &Stream{
		conn:    conn,
		timeout: timeout,
		buffer:  buffer,
		reset:   buffer[:cap(buffer)],
	}
}

// Write sends messages to Guacamole with a timeout
func (s *Stream) Write(data []byte) (n int, err error) {
	if err = s.conn.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		logx.Errorf("sends messages to Guacamole error: %v", err)
		return
	}
	return s.conn.Write(data)
}

// Available returns true if there are messages buffered
func (s *Stream) Available() bool {
	return len(s.buffer) > 0
}

// Flush resets the internal buffer
func (s *Stream) Flush() {
	copy(s.reset, s.buffer)
	s.buffer = s.reset[:len(s.buffer)]
}

// ReadSome takes the next instruction (from the network or from the buffer) and returns it.
// io.Reader is not implemented because this seems like the right place to maintain a buffer.
func (s *Stream) ReadSome() (instruction []byte, err error) {
	if err = s.conn.SetReadDeadline(time.Now().Add(s.timeout)); err != nil {
		logx.Errorf("read messages from Guacamole error: %v", err)
		return
	}

	buffer := make([]byte, MaxGuacMessage)
	var n int
	// While we're blocking, or input is available
	for {
		// Length of element
		var elementLength int

		// Resume where we left off
		i := s.parseStart

	parseLoop:
		// Parse instruction in buffer
		for i < len(s.buffer) {
			// ReadSome character
			readChar := s.buffer[i]
			i++

			switch readChar {
			// If digit, update length
			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				elementLength = elementLength*10 + int(readChar-'0')

			// If not digit, check for end-of-length character
			case '.':
				if i+elementLength >= len(s.buffer) {
					// break for i < s.usedLength { ... }
					// Otherwise, read more data
					break parseLoop
				}
				// Check if element present in buffer
				terminator := s.buffer[i+elementLength]
				// Move to character after terminator
				i += elementLength + 1

				// Reset length
				elementLength = 0

				// Continue here if necessary
				s.parseStart = i

				// If terminator is semicolon, we have a full
				// instruction.
				switch terminator {
				case ';':
					instruction = []byte(string(s.buffer[0:i]))
					s.parseStart = 0
					s.buffer = s.buffer[i:]
					return
				case ',':
					// keep going
				default:
					err = ErrServer.NewError("Element terminator of instruction was not ';' nor ','")
					return
				}
			default:
				// Otherwise, parse error
				err = ErrServer.NewError("Non-numeric character in element length:", string(readChar))
				return
			}
		}

		n, err = s.conn.Read(buffer)
		if err != nil && n == 0 {
			switch err.(type) {
			case net.Error:
				ex := err.(net.Error)
				if ex.Timeout() {
					err = ErrUpstreamTimeout.NewError("Connection to guacd timed out.", err.Error())
				} else {
					err = ErrConnectionClosed.NewError("Connection to guacd is closed.", err.Error())
				}
			default:
				err = ErrServer.NewError(err.Error())
			}
			return
		}
		if n == 0 {
			err = ErrServer.NewError("read 0 bytes")
		}
		runes := []rune(string(buffer[:n]))

		if cap(s.buffer)-len(s.buffer) < len(runes) {
			s.Flush()
		}

		n = copy(s.buffer[len(s.buffer):cap(s.buffer)], runes)
		// must reslice so len is changed
		s.buffer = s.buffer[:len(s.buffer)+n]
	}
}

// Close closes the underlying network connection
func (s *Stream) Close() error {
	return s.conn.Close()
}

// Handshake configures the guacd session
func (s *Stream) Handshake(config *Config) error {
	// Get protocol / connection ID
	selectArg := config.ConnectionID
	if len(selectArg) == 0 {
		selectArg = config.Protocol
	}

	// Send requested protocol or connection ID
	_, err := s.Write(NewInstruction("select", selectArg).Byte())
	if err != nil {
		return err
	}

	// Wait for server Args
	args, err := s.AssertOpcode("args")
	if err != nil {
		return err
	}

	// Build Args list off provided names and config
	argNameS := args.Args
	argValueS := make([]string, 0, len(argNameS))
	for _, argName := range argNameS {

		// Retrieve argument name

		// Get defined value for name
		value := config.Parameters[argName]

		// If value defined, set that value
		if len(value) == 0 {
			value = ""
		}
		argValueS = append(argValueS, value)
	}

	// Send size
	_, err = s.Write(NewInstruction("size",
		fmt.Sprintf("%v", config.OptimalScreenWidth),
		fmt.Sprintf("%v", config.OptimalScreenHeight),
		fmt.Sprintf("%v", config.OptimalResolution)).Byte(),
	)

	if err != nil {
		return err
	}

	// Send supported audio formats
	_, err = s.Write(NewInstruction("audio", config.AudioMimetypes...).Byte())
	if err != nil {
		return err
	}

	// Send supported video formats
	_, err = s.Write(NewInstruction("video", config.VideoMimetypes...).Byte())
	if err != nil {
		return err
	}

	// Send supported image formats
	_, err = s.Write(NewInstruction("image", config.ImageMimetypes...).Byte())
	if err != nil {
		return err
	}

	// timezone
	_, err = s.Write(NewInstruction("timezone", "Asia/Shanghai").Byte())
	if err != nil {
		return err
	}

	// Send Args
	_, err = s.Write(NewInstruction("connect", argValueS...).Byte())
	if err != nil {
		return err
	}

	// Wait for ready, store ID
	ready, err := s.AssertOpcode("ready")
	if err != nil {
		return err
	}

	readyArgs := ready.Args
	if len(readyArgs) == 0 {
		err = ErrServer.NewError("No connection ID received")
		return err
	}

	s.Flush()
	s.ConnectionID = readyArgs[0]

	return nil
}

// AssertOpcode checks the next opcode in the stream matches what is expected. Useful during handshake.
func (s *Stream) AssertOpcode(opcode string) (instruction *Instruction, err error) {
	instruction, err = ReadOne(s)
	if err != nil {
		return
	}

	if len(instruction.Opcode) == 0 {
		err = ErrServer.NewError("End of stream while waiting for \"" + opcode + "\".")
		return
	}

	if instruction.Opcode != opcode {
		err = ErrServer.NewError("Expected \"" + opcode + "\" instruction but instead received \"" + instruction.Opcode + "\".")
		return
	}
	return
}
