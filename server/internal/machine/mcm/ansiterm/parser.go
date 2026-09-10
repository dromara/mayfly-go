package ansiterm

type Parser interface {
	Next() string
	Send(value string) bool
	SetPlain(bool2 bool)
	GetPlain() bool
	Len() int
	Close()
	Running() bool
	Start()
}

type MyParser struct {
	CharChan chan string
	IsPlain  chan bool
	Closed   bool
	State    bool
}

func (m *MyParser) Next() string {
	return <-m.CharChan
}

func (m *MyParser) Len() int {
	return len(m.IsPlain)
}

func (m *MyParser) Send(value string) bool {
	if m.Closed {
		return false
	}
	m.CharChan <- value
	return true
}

func (m *MyParser) GetPlain() bool {
	return <-m.IsPlain
}

func (m *MyParser) SetPlain(bool2 bool) {
	m.IsPlain <- bool2
}

func (m *MyParser) Close() {
	m.Closed = true
	close(m.IsPlain)
	close(m.CharChan)
}

func (m *MyParser) Running() bool {
	return m.State
}

func (m *MyParser) Start() {
	m.State = true
}
