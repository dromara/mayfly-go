package application

import (
	"fmt"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"testing"

	"github.com/may-fly/cast"
)

func TestTagPath(t *testing.T) {
	childOrderTypes := []int{1, 5}
	childOrderTypesMatch := strings.Join(collx.ArrayMap(childOrderTypes, func(tt int) string {
		return cast.ToString(tt) + entity.CodePathResourceSeparator + "%"
	}), entity.CodePathSeparator) + entity.CodePathSeparator
	fmt.Println(childOrderTypesMatch)
}

func TestTagPathMatch(t *testing.T) {
	// accountCodePath := "tag1/tag2/2|xxdd/"
	// resourceCodePath := "tag1/tag2/1|%/11|%/%"

	codePathLike := "default/2|%/22|%/"
	accountCodePath := "default/2|db_local/5|db_local_root/"

	sections := entity.CodePath(accountCodePath).GetPathSections()
	for _, section := range sections {
		if section.Type == entity.TagTypeTag {
			continue
		}
		section.Code = "%"
	}
	accountTagPathPattern := sections.ToCodePath()
	match := strings.HasPrefix(codePathLike, accountTagPathPattern)
	fmt.Println(match)
}
