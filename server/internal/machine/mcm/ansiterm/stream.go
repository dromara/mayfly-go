package ansiterm

import (
	"fmt"
	"math"
	. "mayfly-go/internal/machine/mcm/ansiterm/consts"
	"mayfly-go/pkg/gox"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	Bell             = "bell"
	Backspace        = "backspace"
	Tab              = "tab"
	Linefeed         = "linefeed"
	CarriageReturn   = "carriage_return"
	ShiftOut         = "shift_out"
	ShiftIn          = "shift_in"
	Reset            = "reset"
	Index            = "index"
	ReverseIndex     = "reverse_index"
	SetTabStop       = "set_tab_stop"
	SaveCursor       = "save_cursor"
	RestoreCursor    = "restore_cursor"
	AlignmentDisplay = "alignment_display"
)

type Stream struct {
	Listener        *Screen
	Strict          bool
	UseUTF8         bool
	TakingPlainText bool
	Basic           map[string]struct{}
	Escape          map[string]struct{}
	Sharp           map[string]struct{}
	Csi             map[string]struct{}
	//Events          map[string]struct{} // or []string
	TextPattern *regexp.Regexp
	parser      Parser
}

func generateTextPattern() (*regexp.Regexp, error) {
	special := map[string]struct{}{
		RegBEL:   {},
		RegBS:    {},
		RegHT:    {},
		RegLF:    {},
		RegVT:    {},
		RegFF:    {},
		RegCR:    {},
		RegSO:    {},
		RegSI:    {},
		RegESC:   {},
		RegCSIC1: {},
		RegNUL:   {},
		RegDEL:   {},
		RegOSCC1: {},
	}
	//var specialChars []string
	//for k := range special {
	//	specialChars = append(specialChars, k)
	//	//specialChars = append(specialChars, regexp.QuoteMeta(k))
	//}

	var escaped string
	for s := range special {
		//escaped += regexp.QuoteMeta(s)
		escaped += s
	}

	// 创建正则表达式
	pattern := "[^" + escaped + "]+"

	//pattern := "[^" + strings.Join(specialChars, "") + "]+"
	return regexp.Compile(pattern)
}

func initializeStream(screen *Screen, strict bool) *Stream {
	textPattern, err := generateTextPattern()
	if err != nil {
		fmt.Println(err)
	}

	s := &Stream{
		Listener:        nil,
		Strict:          strict,
		UseUTF8:         true,
		TakingPlainText: false,
		TextPattern:     textPattern,
		Basic:           Basic,
		Escape:          Escape,
		Sharp:           Sharp,
		Csi:             Csi,
	}

	//if screen != nil {
	//	s.Attach(screen)
	//}

	return s
}

func (s *Stream) Attach(screen *Screen) {
	s.Listener = screen
	s.InitializeParser()
}

func (s *Stream) InitializeParser() {
	s.parser = &MyParser{
		CharChan: make(chan string, 2048),
		IsPlain:  make(chan bool),
	}
	gox.Go(func() { s.parseFsm() })
	s.TakingPlainText = true
	s.parser.Running()
}

func (s *Stream) Feed(data string) {
	//matchText := s.TextPattern.MatchString
	matchText := s.TextPattern.FindStringSubmatchIndex
	takingPlainText := s.TakingPlainText
	if s.Listener == nil {
		panic("Listener is nil")
	}

	length := len(data)
	offset := 0
	if !s.parser.Running() {
		s.parser.Start()
		s.parser.GetPlain()
	}
	for offset < length {
		if takingPlainText {
			matches := matchText(data[offset:])

			if matches != nil && matches[0] == 0 {
				start, end := matches[0]+offset, matches[1]+offset
				s.Listener.Draw(data[start:end])
				offset = end
			} else {
				takingPlainText = false
			}
		} else {
			if s.parser.Send(data[offset : offset+1]) {
				takingPlainText = s.parser.GetPlain()
			} else {
				s.InitializeParser()
			}
			offset++
		}
	}
	s.TakingPlainText = takingPlainText
}

func (s *Stream) parseFsm() {
	if s.Listener == nil {
		panic("listener is nil")
	}

	SpOrGt := SP + ">"
	NulOrDel := NUL + DEL
	CanOrSub := CAN + SUB
	AllowedInCsi := BEL + BS + HT + LF + VT + FF + CR
	OscTermINATORS := map[string]struct{}{
		STC0: {},
		STC1: {},
		BEL:  {},
	}

	var char string
	defer func() {
		s.parser.Close()
	}()
	for {
		s.parser.SetPlain(true)
		char = s.parser.Next()
		if char == ESC {
			s.parser.SetPlain(false)

			char = s.parser.Next()
			switch char {
			case "[":
				char = CSIC1
			case "]":
				char = OSCC1
			default:
				if char == "#" {
					s.parser.SetPlain(false)
					s.HandleSharp(s.parser.Next())
				} else if char == "%" {
					s.parser.SetPlain(false)
					s.selectOtherCharset(s.parser.Next())
				} else if char == "(" || char == ")" {
					s.parser.SetPlain(false)
					code := s.parser.Next()
					if s.UseUTF8 {
						continue
					}
					s.Listener.defineCharset(code, char)
				} else {
					s.HandleEscape(char)
				}
				continue
			}
		}
		if _, ok := s.Basic[char]; ok {
			if (char == SI || char == SO) && s.UseUTF8 {
				continue
			}
			s.HandleBasic(char)
		} else if char == CSIC1 {
			var params []int
			current := ""
			private := false
			for {
				s.parser.SetPlain(false)
				char = s.parser.Next()
				if char == "?" {
					private = true
				} else if strings.Contains(AllowedInCsi, char) {
					s.HandleBasic(char)
				} else if strings.Contains(SpOrGt, char) {
				} else if strings.Contains(CanOrSub, char) {
					s.Listener.Draw(char)
					break
				} else if unicode.IsDigit(rune(char[0])) {
					current += char
				} else if char == "$" {
					s.parser.SetPlain(false)
					char = s.parser.Next()
					break
				} else {
					num, _ := strconv.Atoi(current)
					params = append(params, int(math.Min(float64(num), 9999)))
					if char == ";" {
						current = ""
					} else {
						if private {
							s.HandleCSI(char, params, map[string]any{"private": true})
						} else {
							s.HandleCSI(char, params, nil)
						}
						break
					}
				}
			}
		} else if char == OSCC1 {
			s.parser.SetPlain(false)
			code := s.parser.Next()
			switch code {
			case "R", "P":
				continue
			}
			param := ""
			for {
				s.parser.SetPlain(false)
				char = s.parser.Next()
				if char == ESC {
					s.parser.SetPlain(false)
					char += s.parser.Next()
				}
				if _, ok := OscTermINATORS[char]; ok {
					break
				} else {
					param += char
				}
			}
			if len(param) > 0 {
				if strings.Contains("01", code) {
					s.Listener.setIconName(param)
				}
				if strings.Contains("02", code) {
					s.Listener.setTitle(param)
				}
			}
		} else if strings.Contains(NulOrDel, char) {
			s.Listener.Draw(char)
		}
	}

}

func (s *Stream) selectOtherCharset(code string) {

}
