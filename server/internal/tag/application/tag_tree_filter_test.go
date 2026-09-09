package application

import (
	"reflect"
	"testing"

	"mayfly-go/internal/tag/domain/entity"
)

// TestBuildTypePathFilter_PureTagAccount 纯标签路径的账号，查询 machine(1)/authcert(5) 类型资源
// 应生成 tag1/1|%/  与 tag1/%/1|%/  两个模糊匹配条件
func TestBuildTypePathFilter_PureTagAccount(t *testing.T) {
	likes, needFilter, appendTypes := buildTypePathFilter(
		[]string{"tag1/"},
		[]entity.TypePath{entity.NewTypePaths(entity.TagTypeMachine, entity.TagTypeAuthCert)},
		false,
	)

	wantLikes := []string{"tag1/1|%/5|%/", "tag1/%/1|%/5|%/"}
	if !reflect.DeepEqual(likes, wantLikes) {
		t.Fatalf("codePathLikes = %v, want %v", likes, wantLikes)
	}
	if len(needFilter) != 0 {
		t.Fatalf("needFilter should be empty, got %v", needFilter)
	}
	// 非GetAllChildren模式，应追加类型路径的最后一段类型(授权凭证)
	if len(appendTypes) != 1 || appendTypes[0] != entity.TagTypeAuthCert {
		t.Fatalf("appendTypes = %v, want [%v]", appendTypes, entity.TagTypeAuthCert)
	}
}

// TestBuildTypePathFilter_ResourceLeafAccount 账号拥有到授权凭证的资源标签路径，
// 查询其父级类型链(machine/authcert)时应直接使用原始路径匹配
func TestBuildTypePathFilter_ResourceLeafAccount(t *testing.T) {
	accountTag := "tag1/1|m1/5|ac1/"
	likes, needFilter, _ := buildTypePathFilter(
		[]string{accountTag},
		[]entity.TypePath{entity.NewTypePaths(entity.TagTypeMachine, entity.TagTypeAuthCert)},
		false,
	)

	// accountMatchPath = tag1/1|%/ 是 codePathLike = tag1/1|%/5|%/ 的前缀，即拥有更深层级，直接使用原始路径
	if !reflect.DeepEqual(likes, []string{accountTag}) {
		t.Fatalf("codePathLikes = %v, want [%v]", likes, accountTag)
	}
	if len(needFilter) != 0 {
		t.Fatalf("needFilter should be empty, got %v", needFilter)
	}
}

// TestBuildTypePathFilter_NeedFilter 账号拥有的路径比查询类型路径更深时
// (账号: tag1/1|m1/，查询: machine 类型)，应使用实际匹配路径查询，并对结果二次过滤防止越权
func TestBuildTypePathFilter_NeedFilter(t *testing.T) {
	accountTag := "tag1/1|m1/5|ac1/"
	likes, needFilter, appendTypes := buildTypePathFilter(
		[]string{accountTag},
		[]entity.TypePath{entity.NewTypePaths(entity.TagTypeMachine)},
		false,
	)

	// codePathLike = tag1/1|%/  为账号路径的祖先，使用实际匹配codePath: tag1/1|m1/
	if !reflect.DeepEqual(likes, []string{"tag1/1|m1/"}) {
		t.Fatalf("codePathLikes = %v, want [tag1/1|m1/]", likes)
	}
	wantFilter := map[string][]string{"tag1/1|m1/": {accountTag}}
	if !reflect.DeepEqual(needFilter, wantFilter) {
		t.Fatalf("needFilter = %v, want %v", needFilter, wantFilter)
	}
	if len(appendTypes) != 1 || appendTypes[0] != entity.TagTypeMachine {
		t.Fatalf("appendTypes = %v, want [%v]", appendTypes, entity.TagTypeMachine)
	}
}

// TestBuildTypePathFilter_GetAllChildren GetAllChildren模式不追加类型过滤
func TestBuildTypePathFilter_GetAllChildren(t *testing.T) {
	_, _, appendTypes := buildTypePathFilter(
		[]string{"tag1/"},
		[]entity.TypePath{entity.NewTypePaths(entity.TagTypeMachine, entity.TagTypeAuthCert)},
		true,
	)
	if len(appendTypes) != 0 {
		t.Fatalf("GetAllChildren时appendTypes应为空, got %v", appendTypes)
	}
}

// TestFilterCodePaths 账号路径与查询条件路径的双向前缀过滤
func TestFilterCodePaths(t *testing.T) {
	accountTagPaths := []string{"a/", "x/y/"}
	queryPaths := []string{"a/b/", "a/", "x/y/z/", "b/"}

	// 有权: a/ x/y/ ; 查询: a/b/ a/ x/y/z/ b/
	// a/b/ 是 a/ 的子集 -> 保留；a/ 与自身匹配 -> 保留；x/y/z/ 是 x/y/ 子集 -> 保留；b/ 无权 -> 剔除
	got := filterCodePaths(accountTagPaths, queryPaths)
	want := []string{"a/b/", "a/", "x/y/z/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCodePaths = %v, want %v", got, want)
	}
}

// TestHasConflictPath 存在祖先/后代路径同时分配时判定为冲突，且与顺序无关
func TestHasConflictPath(t *testing.T) {
	if hasConflictPath([]string{"a/", "a/b/"}) != true {
		t.Fatal("a/ 与 a/b/ 同时存在应判定冲突")
	}
	// 父级路径在后时也应判定冲突(回归用例：原实现仅能检测父级在前的场景)
	if hasConflictPath([]string{"a/b/", "a/"}) != true {
		t.Fatal("a/b/ 与 a/ 同时存在应判定冲突(与顺序无关)")
	}
	if hasConflictPath([]string{"a/b/", "a/c/"}) != false {
		t.Fatal("平级路径不应判定冲突")
	}
	if hasConflictPath([]string{}) != false {
		t.Fatal("空数组不应判定冲突")
	}
}

// TestCodePathCanAccess 标签路径前缀权限判定，需保证不误匹配相似前缀
func TestCodePathCanAccess(t *testing.T) {
	cp := entity.CodePath("tag1/tag2/")
	if !cp.CanAccess("tag1/tag2/test/") {
		t.Fatal("拥有父级路径应可访问子路径")
	}
	if cp.CanAccess("tag1/") {
		t.Fatal("子路径不应反向授权父路径")
	}
	if cp.CanAccess("tag1/tag2x/") {
		t.Fatal("相似前缀不应误匹配")
	}
}
