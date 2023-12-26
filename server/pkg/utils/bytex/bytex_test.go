package bytex

import (
	"fmt"
	"testing"
)

func TestParseSize(t *testing.T) {
	res, _ := ParseSize("1MB")
	fmt.Println(res)
}
