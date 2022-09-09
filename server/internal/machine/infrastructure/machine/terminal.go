package machine

import (
	"bufio"
	"io"

	"golang.org/x/crypto/ssh"
)

type Terminal struct {
	SshSession   *ssh.Session
	StdinPipe    io.WriteCloser
	StdoutReader *bufio.Reader
}

// 新建机器ssh终端
func NewTerminal(cli *Cli) (*Terminal, error) {
	sshSession, err := cli.GetSession()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdoutReader := bufio.NewReader(stdoutPipe)

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}

	terminal := Terminal{
		SshSession:   sshSession,
		StdinPipe:    stdinPipe,
		StdoutReader: stdoutReader,
	}

	return &terminal, nil
}

func (t *Terminal) Write(p []byte) (int, error) {
	return t.StdinPipe.Write(p)
}

func (t *Terminal) ReadRune() (r rune, size int, err error) {
	return t.StdoutReader.ReadRune()
}

func (t *Terminal) Close() error {
	if t.SshSession != nil {
		return t.SshSession.Close()
	}
	return nil
}

func (t *Terminal) WindowChange(h int, w int) error {
	return t.SshSession.WindowChange(h, w)
}

func (t *Terminal) RequestPty(term string, h, w int) error {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	return t.SshSession.RequestPty(term, h, w, modes)
}

func (t *Terminal) Shell() error {
	return t.SshSession.Shell()
}
