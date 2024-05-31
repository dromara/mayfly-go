package guac

// Config is the data sent to guacd to configure the session during the handshake.
type Config struct {
	// ConnectionID is used to reconnect to an existing session, otherwise leave blank for a new session.
	ConnectionID string
	// Protocol is the protocol of the connection from guacd to the remote (rdp, ssh, etc).
	Protocol string
	// Parameters are used to configure protocol specific options like sla for rdp or terminal color schemes.
	Parameters map[string]string
	// OptimalScreenWidth is the desired width of the screen
	OptimalScreenWidth int
	// OptimalScreenHeight is the desired height of the screen
	OptimalScreenHeight int
	// OptimalResolution is the desired resolution of the screen
	OptimalResolution int
	// AudioMimetypes is an array of the supported audio types
	AudioMimetypes []string
	// VideoMimetypes is an array of the supported video types
	VideoMimetypes []string
	// ImageMimetypes is an array of the supported image types
	ImageMimetypes []string
}

// NewGuacamoleConfiguration returns a Config with sane defaults
func NewGuacamoleConfiguration() *Config {
	return &Config{
		Parameters:          map[string]string{},
		OptimalScreenWidth:  1024,
		OptimalScreenHeight: 768,
		OptimalResolution:   96,
		AudioMimetypes:      make([]string, 0, 1),
		VideoMimetypes:      make([]string, 0, 1),
		ImageMimetypes:      make([]string, 0, 1),
	}
}
