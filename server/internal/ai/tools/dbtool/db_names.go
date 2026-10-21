package dbtool

import (
	"fmt"
	"strings"
)

// nonEmptyDbNames 过滤掉空库名候选。
//
// 存在动机（真机在 sqlite 资产 dbId=2 上复现）：listDbNamesOfAsset 对「既没配置指定库名、又连不上
// 去实时枚举」的资产会兜底返回一条 dbName 为空串的项。直接取用就得到 [""]，于是
// `len(candidates) > 0` 成立、走进「候选清单」分支，输出
// 「该实例可连接的真实库名为 []，dbName 只能取其中之一」——对着一个空清单要求模型选一个，
// 是句谎话；而真正该走的「无法枚举 + 可能是凭证无权限」兜底文案反而永远命中不到。
//
// 兜底项（dbName 为空）不计入候选，让上层正确地走「拿不到清单」那条路
func nonEmptyDbNames(items []dbOptionItem) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.dbName) == "" {
			continue
		}
		names = append(names, item.dbName)
	}
	return names
}

// dbUnknownNameFailure 构造「库名/库权限不对」这类失败给模型的提示（纯函数、不取数）。
//
// 抽出来的理由：这两条文案是模型的行动依据，必须能表驱动断言其**语义**——候选为空时不得要求
// 「从中选一个」、候选非空时不得说「无法枚举」；而不是靠扫源码数文案出现次数（那种静态守卫锚的是
// 措辞，一改文案就红，团队最终连文案一起跟着改，守卫就失去防回归价值）。
//
// 真机实测到的错误形态：`真实库名为 []，dbName 只能取其中之一` —— 候选是空数组却要求模型选一个，
// 而正确的兜底文案永远命中不到
func dbUnknownNameFailure(err error, candidates []string) error {
	if len(candidates) > 0 {
		return fmt.Errorf("%w；该实例可连接的真实库名为 [%s]，dbName 只能取其中之一（资源显示名不是 schema 名）",
			err, strings.Join(candidates, ", "))
	}
	return fmt.Errorf("%w；提示：dbName 必须是实例内的实际库名（schema），不是资源显示名；"+
		"当前无法枚举该实例的库清单，若库名已正确则是该授权凭证对这个库没有权限，请换凭证或找管理员授权", err)
}
