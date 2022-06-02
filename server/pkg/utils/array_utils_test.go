package utils

import (
	"fmt"
	"testing"
)

func TestArrayCompare(t *testing.T) {
	newArr := []interface{}{1, 2, 3, 5}
	oldArr := []interface{}{3, 6}
	add, del, unmodifier := ArrayCompare(newArr, oldArr, func(i1, i2 interface{}) bool {
		return i1.(int) == i2.(int)
	})
	fmt.Println(add...)
	fmt.Println(del...)
	fmt.Println(unmodifier...)
}
