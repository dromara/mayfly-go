package application

import (
	"fmt"
	"mayfly-go/internal/pkg/utils"
	"strings"
	"testing"
)

func TestParseRedisCommand(t *testing.T) {
	utils.SplitStmts(strings.NewReader("del 'key l3'; set key2 key3; set 'key3' 'value3 and value4'; hset field1 key1 'hvalue2 hvalue3'"), func(stmt string) error {
		res := parseRedisCommand(stmt)
		fmt.Println(res)
		return nil
	})

}
