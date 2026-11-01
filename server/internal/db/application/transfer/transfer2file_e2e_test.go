package transfer

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer/backupenc"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	fileapp "mayfly-go/internal/file/application"
	fileentity "mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/pkg/config"

	"github.com/stretchr/testify/require"
)

// closeTrackingWriter 模拟真实文件的「关闭后不可再写」语义。
//
// 这一点决定了用例的鉴别力：若 closeFunc 只是空操作，收尾顺序反了（先关文件再写加密终止块）
// 也不会报错，测试就成了恒绿的空壳。真实实现里 closeFunc 会关闭并保存文件。
type closeTrackingWriter struct {
	buf    *bytes.Buffer
	closed bool
}

func (w *closeTrackingWriter) Write(p []byte) (int, error) {
	if w.closed {
		return 0, errors.New("write after close")
	}
	return w.buf.Write(p)
}

// fakeFileApp 只实现 NewWriter：把产物写到内存 buffer，closeFunc 关闭并锁死写入
type fakeFileApp struct {
	fileapp.File
	w *closeTrackingWriter
}

func (f *fakeFileApp) NewWriter(ctx context.Context, fileKey, filename string, accessMode fileentity.AccessMode) (string, io.Writer, func(*error) error, error) {
	return "fake-key", f.w, func(*error) error {
		f.w.closed = true
		return nil
	}, nil
}

// fakeDbApp 的 DumpDb 直接写出可识别的 SQL（等价于 dump 层产出）
type fakeDbApp struct {
	sql string
}

func (f *fakeDbApp) DumpDb(ctx context.Context, req *dto.DumpDb) error {
	_, err := io.WriteString(req.Writer, f.sql)
	return err
}

func (f *fakeDbApp) GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	return nil, nil
}

// fakeTransferFileApp 只记录文件状态变更
type fakeTransferFileApp struct {
	DbTransferFile
	last *entity.DbTransferFile
}

func (f *fakeTransferFileApp) Save(ctx context.Context, file *entity.DbTransferFile) error {
	f.last = file
	return nil
}

func (f *fakeTransferFileApp) UpdateById(ctx context.Context, file *entity.DbTransferFile) error {
	f.last = file
	return nil
}

// TestTransfer2FileProducesRecoverableEncryptedBackup 库→文件迁移的端到端不变量：
// 产物必须是「可解密的密文」，且解密后能看到完整的建表 + 插入语句。
//
// 历史缺陷有两层：① 加密器在外层作用域 defer Close（数据写入在协程里）→ 文件只剩终止块；
// ② 即便搬进协程，defer 注册顺序仍反了 → closeFunc 先关闭保存文件，加密终止块写不进去。
// 两者都表现为「备份看起来成功、恢复出来是空的/不可解密」。
func TestTransfer2FileProducesRecoverableEncryptedBackup(t *testing.T) {
	prevConf := config.Conf
	config.Conf = &config.Config{Aes: config.Aes{Key: "0123456789abcdef0123456789abcdef"}}
	t.Cleanup(func() { config.Conf = prevConf })

	db := setupTransferLogDb(t)
	app := newLogTestApp()

	var buf bytes.Buffer
	app.fileApp = &fakeFileApp{w: &closeTrackingWriter{buf: &buf}}
	app.dbApp = &fakeDbApp{sql: "CREATE TABLE t (id int);\nINSERT INTO t (id) VALUES (1);\n"}
	app.transferFileApp = &fakeTransferFileApp{}

	now := time.Now()
	task := &entity.DbTransferTask{
		TaskName:         "e2e-file",
		RunningState:     entity.DbTransferTaskRunStateRunning,
		SrcDbId:          1,
		SrcDbName:        "src",
		TargetFileDbType: "mysql",
	}
	task.Id = 7001
	task.CreateTime = &now
	logId := insertRunningLog(t, db, task.Id)
	app.setTransferLog(logId, getLog(t, db, logId))

	app.transfer2File(context.Background(), logId, task, []dbi.Table{{TableName: "t"}})

	// 异步执行：等日志离开 Running 状态
	require.Eventually(t, func() bool {
		l := app.getTransferLog(logId)
		return l != nil && l.Status != entity.DbTransferLogStatusRunning
	}, 5*time.Second, 20*time.Millisecond, "transfer2File 未在超时内完成")

	require.NotEmpty(t, buf.Bytes(), "必须产出文件内容（空文件意味着数据块根本没写进去）")

	reader, err := backupenc.WrapReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err, "产物必须是可识别的加密格式")
	plain, err := io.ReadAll(reader)
	require.NoError(t, err, "解密必须成功（终止块缺失会在这里报错）")
	require.Contains(t, string(plain), "CREATE TABLE t", "解密后必须看到建表语句")
	require.Contains(t, string(plain), "INSERT INTO t", "解密后必须看到插入语句（数据块未被终止块截断）")
}

// TestTransfer2FileZipProducesRecoverableArchive zip 分支的端到端验证：解密后必须是合法 zip，
// 且压缩包内的 .sql 条目含完整 SQL。
//
// 为什么单列一条：sql 分支的 e2e 走不到 zip 收尾（zipWriter.Close 先于加密终止块），
// 而 "zip.Close → encWriter.Close → closeFunc" 这条顺序若只靠源码下标断言，改坏 zip 收尾实现
// 并不会让测试变红（下标断言只能证明书写顺序）。
func TestTransfer2FileZipProducesRecoverableArchive(t *testing.T) {
	prevConf := config.Conf
	config.Conf = &config.Config{Aes: config.Aes{Key: "0123456789abcdef0123456789abcdef"}}
	t.Cleanup(func() { config.Conf = prevConf })

	db := setupTransferLogDb(t)
	app := newLogTestApp()

	var buf bytes.Buffer
	app.fileApp = &fakeFileApp{w: &closeTrackingWriter{buf: &buf}}
	app.dbApp = &fakeDbApp{sql: "CREATE TABLE t (id int);\nINSERT INTO t (id) VALUES (1);\n"}
	app.transferFileApp = &fakeTransferFileApp{}

	now := time.Now()
	task := &entity.DbTransferTask{
		TaskName:         "e2e-zip",
		RunningState:     entity.DbTransferTaskRunStateRunning,
		SrcDbId:          1,
		SrcDbName:        "src",
		TargetFileDbType: "mysql",
	}
	task.Id = 7002
	task.CreateTime = &now
	task.Extra = map[string]any{}
	task.SetExtraValue("fileType", "zip")
	logId := insertRunningLog(t, db, task.Id)
	app.setTransferLog(logId, getLog(t, db, logId))

	app.transfer2File(context.Background(), logId, task, []dbi.Table{{TableName: "t"}})

	require.Eventually(t, func() bool {
		l := app.getTransferLog(logId)
		return l != nil && l.Status != entity.DbTransferLogStatusRunning
	}, 5*time.Second, 20*time.Millisecond, "transfer2File(zip) 未在超时内完成")

	reader, err := backupenc.WrapReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err, "产物必须是可识别的加密格式")
	plain, err := io.ReadAll(reader)
	require.NoError(t, err, "解密必须成功")

	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	require.NoError(t, err, "解密后必须是合法 zip（中央目录先于加密终止块写出）")
	require.NotEmpty(t, zr.File, "压缩包内应有 sql 条目")

	rc, err := zr.File[0].Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()
	content, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Contains(t, string(content), "CREATE TABLE t", "压缩包内 sql 必须完整")
	require.Contains(t, string(content), "INSERT INTO t", "压缩包内 sql 必须完整")
}
