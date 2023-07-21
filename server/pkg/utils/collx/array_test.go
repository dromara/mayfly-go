package collx

import (
	"fmt"
	"testing"
)

func TestArrayCompare(t *testing.T) {
	newArr := []any{1, 2, 3, 5}
	oldArr := []any{3, 6}
	add, del, unmodifier := ArrayCompare(newArr, oldArr, func(i1, i2 any) bool {
		return i1.(int) == i2.(int)
	})
	fmt.Println(add...)
	fmt.Println(del...)
	fmt.Println(unmodifier...)
}
