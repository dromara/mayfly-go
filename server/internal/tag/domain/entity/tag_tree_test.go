package entity

import (
	"fmt"
	"testing"
)

func TestGetTag(t *testing.T) {
	cp := CodePath("tag1/tag2/1|xx/11|yy/111|zz/")
	v := cp.GetTag()
	pv := cp.GetParent(0)
	av := cp.GetAllPath()
	ps := cp.GetPathSections()
	code := cp.GetCode(TagType(11))
	fmt.Println(v, pv, av, ps, code)
}

// func TestGetPathSection(t *testing.T) {
// 	fromPath := "tag1/tag2/1|xx/"
// 	childPath := "tag1/tag2/1|xx/11|yy/"
// 	toPath := "tag3/"
// 	parentSection := GetTagPathSections(GetParentPath(fromPath, 0))

// 	childSection := GetTagPathSections(childPath)
// 	res := toPath + childSection[len(GetTagPathSections(fromPath)):].ToCodePath()
// 	res1 := toPath + childSection[len(parentSection):].ToCodePath()

// 	pPath := GetParentPath(fromPath, 0)
// 	r := strings.Replace(childPath, pPath, toPath, 1)
// 	r1 := strings.Replace(fromPath, pPath, toPath, 1)
// 	fmt.Println(res, res1, r, r1)
// }

// func TestGetPathSection2(t *testing.T) {
// 	tagpath := "tag1/tag2/1|xx/11|yy/"
// 	sections := GetTagPathSections(GetParentPath(tagpath, 0))
// 	fmt.Println(sections)
// }
