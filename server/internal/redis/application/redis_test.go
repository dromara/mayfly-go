package application

import (
	"fmt"
	"testing"
)

func TestParseRedisCommand(t *testing.T) {
	res := parseRedisCommand("del 'key l3'")
	fmt.Println(res)
}
