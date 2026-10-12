package application

// 文件访问模式判定的回归测试。
//
// 缺陷现场：通用文件内容路由（GET /sys/files/:key）不校验 token——头像与 AI 附件由
// <img src>/<a href> 直接引用，浏览器无法附加 Authorization 头。若所有文件都吃这条匿名
// 路由，数据库备份/导出等整库明文产物只要 fileKey 泄露就能被无凭证拉走，且界面上声明的
// 下载权限码形同虚设（后端从未校验）。收口后按文件访问模式判定：内容获取仅 public 可匿名读，
// private 必须走所属模块带权限码的下载入口。
//
// 元数据（文件名/大小）查询不设模式门槛：界面统一经登录态的 detail 端点回显，私有产物同样
// 回显（迁移导出文件列表依赖此行为），访问模式只在获取内容时生效。
//
// 运行：cd server && go test -count=1 -run TestFileAccess ./internal/file/application/

import (
	"context"
	"testing"

	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/file/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stubFileRepo 只为满足 repository.File（即 base.Repo）类型要求，无额外方法
type stubFileRepo struct {
	*base.RepoImpl[*entity.File]
}

func newFileApp() *fileAppImpl {
	// 内嵌的 AppImpl 以接口参数实例化（与 IOC 注入后的运行时形状一致），故 Repo 字段装的是 stub
	return &fileAppImpl{AppImpl: base.AppImpl[*entity.File, repository.File]{Repo: &stubFileRepo{RepoImpl: &base.RepoImpl[*entity.File]{}}}}
}

// setupFileDb 内存 sqlite 建表并接管全局 db（仓储默认取 global.Db）
func setupFileDb(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.File{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })
	return db
}

// insertFileMode 落库一条指定访问模式的文件记录。
func insertFileMode(t *testing.T, db *gorm.DB, fileKey string, accessMode entity.AccessMode) {
	t.Helper()
	file := &entity.File{FileKey: fileKey, Filename: fileKey + ".sql", Path: "2026/1/1/" + fileKey + ".sql", StorageType: "local"}
	file.AccessMode = accessMode
	file.FillBaseInfo(nil)
	require.NoError(t, db.Create(file).Error)
}

// insertUnknownModeFile 落库一条 access_mode 为未知值 0 的文件记录。
//
// 不能靠 Create 传 0：实体字段带了 `default:1` 标签，GORM 会在插入时略过零值列、由数据库填默认值，
// 这样写出的行实际是 1，根本测不到「遗留零值行」这条分支（本轮首跑即被此假象掩盖）。
// 因此先正常入库，再用原生 SQL 强制把列改成 0。
func insertUnknownModeFile(t *testing.T, db *gorm.DB, fileKey string) {
	t.Helper()
	insertFileMode(t, db, fileKey, entity.FileAccessModePublic)
	require.NoError(t, db.Exec("UPDATE t_sys_file SET access_mode = 0 WHERE file_key = ?", fileKey).Error)
}

func TestFileAccessCanAnonymousRead(t *testing.T) {
	db := setupFileDb(t)
	app := newFileApp()
	ctx := context.Background()

	insertFileMode(t, db, "public_file", entity.FileAccessModePublic)
	insertFileMode(t, db, "private_file", entity.FileAccessModePrivate)
	insertUnknownModeFile(t, db, "legacy_zero_file")

	t.Run("公开文件允许匿名读取", func(t *testing.T) {
		assert.NoError(t, app.CanAnonymousRead(ctx, "public_file"))
	})

	t.Run("私有文件拒绝匿名读取", func(t *testing.T) {
		assert.Error(t, app.CanAnonymousRead(ctx, "private_file"))
	})

	// 未知取值必须按私有处理（fail-closed）：若按「非 private 即公开」解释，
	// 零值行就成了匿名可下的数据文件，而这类行恰恰来自未显式声明模式的历史写入
	t.Run("访问模式未知的遗留行按私有处理", func(t *testing.T) {
		assert.Error(t, app.CanAnonymousRead(ctx, "legacy_zero_file"))
	})

	t.Run("文件不存在返回错误而非放行", func(t *testing.T) {
		assert.Error(t, app.CanAnonymousRead(ctx, "no_such_file"))
	})
}

func TestFileAccessListByKeys(t *testing.T) {
	db := setupFileDb(t)
	app := newFileApp()

	insertFileMode(t, db, "p1", entity.FileAccessModePublic)
	insertFileMode(t, db, "s1", entity.FileAccessModePrivate)
	insertUnknownModeFile(t, db, "z1")

	type fileRow struct {
		FileKey  string `json:"fileKey"`
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
	}

	// 元数据查询不设访问模式门槛：界面由统一文件组件按 key 回显文件名与大小，
	// 私有产物同样要回显（否则迁移导出文件列表里已生成文件的行会显示成空白）
	var rows []fileRow
	require.NoError(t, app.ListByKeys([]string{"p1", "s1", "z1"}, &rows))
	require.Len(t, rows, 3)

	byKey := make(map[string]string, len(rows))
	for _, r := range rows {
		byKey[r.FileKey] = r.Filename
	}
	assert.Equal(t, "s1.sql", byKey["s1"], "私有文件的文件名也必须能查到")
	assert.Equal(t, "z1.sql", byKey["z1"], "遗留零值行同样回显")

	// 空 key 集合不查库也不报错
	var none []fileRow
	require.NoError(t, app.ListByKeys(nil, &none))
	assert.Empty(t, none)
}
