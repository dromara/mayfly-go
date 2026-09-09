package entity

import (
	"fmt"
	"mayfly-go/internal/pkg/consts"
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

// TestTagTypeAliasesMatchResourceTypes TagType 别名与 consts 资源类型常量一致性契约测试：
// 防止别名漂移导致标签树类型判断与资源权限判断串类型
func TestTagTypeAliasesMatchResourceTypes(t *testing.T) {
	mappings := map[string]struct {
		tagType TagType
		want    int8
	}{
		"TagTypeMachine":    {TagTypeMachine, consts.ResourceTypeMachine},
		"TagTypeDbInstance": {TagTypeDbInstance, consts.ResourceTypeDbInstance},
		"TagTypeRedis":      {TagTypeRedis, consts.ResourceTypeRedis},
		"TagTypeMongo":      {TagTypeMongo, consts.ResourceTypeMongo},
		"TagTypeAuthCert":   {TagTypeAuthCert, consts.ResourceTypeAuthCert},
		"TagTypeEsInstance": {TagTypeEsInstance, consts.ResourceTypeEsInstance},
		"TagTypeContainer":  {TagTypeContainer, consts.ResourceTypeContainer},
		"TagTypeMqKafka":    {TagTypeMqKafka, consts.ResourceTypeMqKafka},
		"TagTypeMilvus":     {TagTypeMilvus, consts.ResourceTypeMilvus},
		"TagTypeDb":         {TagTypeDb, consts.ResourceTypeDbName},
	}

	for name, m := range mappings {
		if int8(m.tagType) != m.want {
			t.Errorf("%s = %d, 与 consts.ResourceType 常量值 %d 不一致", name, int8(m.tagType), m.want)
		}
	}
}

func TestAppendResource(t *testing.T) {
	// 纯标签路径追加资源段
	if got := CodePath("tag1/tag2/").AppendResource(TagTypeMachine, "m1"); got != CodePath("tag1/tag2/1|m1/") {
		t.Errorf("unexpected codePath: %s", got)
	}
	// 资源路径继续追加子资源段
	if got := CodePath("tag1/1|m1/").AppendResource(TagTypeAuthCert, "ac1"); got != CodePath("tag1/1|m1/5|ac1/") {
		t.Errorf("unexpected codePath: %s", got)
	}
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
