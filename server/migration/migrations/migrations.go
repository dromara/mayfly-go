// Package migrations 是数据库迁移链的顺序真源。
//
// 目录与批次始终以 internal/pkg/config/app.go 的 Version 为唯一真源：
// 当前 Version 为 v1.12.2 时，目录就是 v1_12/、批次文件就是 v1_12_2.go、迁移 ID 前缀就是 v1.12.2-；
// Version 未修改前不得新开批次文件或下一大版本目录，升级时必须先改 Version 再按新值建目录。
//
// 目录约定（新增迁移前先看清这里）：
//   - 一个大版本一个目录：initdb/、v1_9/、v1_10/、v1_11/、v1_12/；顶层目录数只随大版本增长；
//   - 同一版本只对应一个批次文件：Version 为 v1.12.2 时批次文件就是 v1_12/v1_12_2.go，
//     Version 未升级前不新增批次文件（如 v1_12_3.go），新迁移追加到当前批次文件末尾；
//   - 一个版本目录只对外暴露一个 Migrations() 入口，批次文件不单独出口；
//   - 跨版本复用的工具与数据放 migrationx/，其他版本不得直接引用某个版本包内的私有符号——
//     已发布批次是冻结的，让新版本依赖旧版本的内部实现，等于把冻结文件改活；
//   - 一个批次内的新增迁移不散落成 feature 命名文件，统一收口在该批次文件，
//     ID 前缀与批次版本一致（如 v1.12.2- 前缀）。
package migrations

import (
	"mayfly-go/migration/migrations/initdb"
	v110 "mayfly-go/migration/migrations/v1_10"
	v111 "mayfly-go/migration/migrations/v1_11"
	v112 "mayfly-go/migration/migrations/v1_12"
	v19 "mayfly-go/migration/migrations/v1_9"

	"github.com/go-gormigrate/gormigrate/v2"
)

// Batch 一个版本的迁移批次构造函数
type Batch = func() []*gormigrate.Migration

// All 返回完整迁移链，切片顺序即执行顺序：按大版本发布先后排列，新版本目录追加在末尾。
//
// 已发布批次的顺序与迁移 ID 一律不得改动：gormigrate 按 ID 记账，改序对存量库
// 要么漏跑要么重跑，而全新库安装看不出问题
func All() []Batch {
	return []Batch{
		initdb.Migrations,
		v19.Migrations,
		v110.Migrations,
		v111.Migrations,
		v112.Migrations,
	}
}
