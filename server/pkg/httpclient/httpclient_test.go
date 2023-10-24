package httpclient

import (
	"fmt"
	"mayfly-go/pkg/utils/collx"
	"testing"
)

type TestStruct struct {
	Id       uint64
	Username string
}

func TestGet(t *testing.T) {
	res, err := NewRequest("www.baidu.com").Get().BodyToString()
	fmt.Println(err)
	fmt.Println(res)
}

func TestGetBodyToMap(t *testing.T) {
	res, err := NewRequest("http://go.mayfly.run/api/syslogs?pageNum=1&pageSize=10").Get().BodyToMap()
	fmt.Println(err)
	fmt.Println(res["msg"])
	fmt.Println(res["code"])
}

func TestGetQueryBodyToMap(t *testing.T) {
	res, err := NewRequest("http://go.mayfly.run/api/syslogs").
		Header("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2OTUzOTQ5NTIsImlkIjoxLCJ1c2VybmFtZSI6ImFkbWluIn0.pGrczVZqk5nlId-FZPkjW_O5Sw3-2yjgzACp_j4JEXY").
		GetByQuery(collx.M{"pageNum": 1, "pageSize": 10}).
		BodyToMap()
	fmt.Println(err)
	fmt.Println(res["msg"])
	fmt.Println(res["code"])
}
