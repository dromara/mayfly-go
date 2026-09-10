package ansiterm

import (
	"strings"
)

type ByteStream struct {
	*Stream
	utf8Decoder func(data []byte) (string, error)
}

func (b *ByteStream) Feed(data []byte) {
	var dataStr string
	if b.UseUTF8 {
		dataStr, _ = b.utf8Decoder(data)
	} else {
		dataStr = BytesToString(data)
	}
	b.Stream.Feed(dataStr)
}

func (b *ByteStream) selectOtherCharset(code string) {
	if code == "@" {
		b.UseUTF8 = false
	} else if strings.Contains("G8", code) {
		b.UseUTF8 = true
	}
}

func InitByteStream(screen *Screen, strict bool) *ByteStream {
	stream := initializeStream(screen, strict)
	bs := &ByteStream{
		Stream:      stream,
		utf8Decoder: DecodeUTF8WithReplacement,
	}
	return bs
}
