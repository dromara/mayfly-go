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

func TestArrayChunk(t *testing.T) {
	arr := []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	res := ArrayChunk[int](arr, 3)
	fmt.Println(res)
}

func TestArraySplit(t *testing.T) {
	// arr := []int{1, 2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	// arr := []int{1, 2, 3}
	res := ArraySplit(arr, 10)
	fmt.Println(res)
}

func TestArrayReduce(t *testing.T) {
	arr := []int{1, 2, 3, 5}
	res := ArrayReduce[int, int](arr, 0, func(i1, i2 int) int { return i1 + i2 })
	fmt.Println(res)
}
