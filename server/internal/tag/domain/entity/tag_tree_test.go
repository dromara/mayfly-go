package entity

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetPathSection(t *testing.T) {
	fromPath := "tag1/tag2/1|xx/"
	childPath := "tag1/tag2/1|xx/11|yy/"
	toPath := "tag3/"
	parentSection := GetTagPathSections(GetParentPath(fromPath, 0))

	childSection := GetTagPathSections(childPath)
	res := toPath + childSection[len(GetTagPathSections(fromPath)):].ToCodePath()
	res1 := toPath + childSection[len(parentSection):].ToCodePath()

	pPath := GetParentPath(fromPath, 0)
	r := strings.Replace(childPath, pPath, toPath, 1)
	r1 := strings.Replace(fromPath, pPath, toPath, 1)
	fmt.Println(res, res1, r, r1)
}
