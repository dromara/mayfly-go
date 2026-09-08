package dbi

import (
	"context"
	"errors"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// newMockConn 构造基于sqlmock的DbConn（同包可直接注入私有db字段）
func newMockConn(t *testing.T) (*DbConn, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return &DbConn{Id: "test-conn", Info: &DbInfo{Type: DbType("test-mock-db")}, db: db}, mock
}

func newMockRows() *sqlmock.Rows {
	return sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
		sqlmock.NewColumn("name").OfType("VARCHAR", ""),
	)
}

// WalkQueryRows参数绑定与行数据映射链路（GetByCond/DataSync等均依赖此链路）
func TestWalkQueryRows_ArgsBindingAndRowMap(t *testing.T) {
	conn, mock := newMockConn(t)

	rows := newMockRows().AddRow(int64(1), "a").AddRow(int64(2), "b")
	mock.ExpectQuery("SELECT id, name FROM t1 WHERE id > \\$1").WithArgs(int64(0)).WillReturnRows(rows)

	var gotRows []map[string]any
	cols, err := conn.WalkQueryRows(context.Background(),
		"SELECT id, name FROM t1 WHERE id > $1",
		func(row map[string]any, columns []*QueryColumn) error {
			gotRows = append(gotRows, row)
			return nil
		}, int64(0))

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Len(t, cols, 2)
	assert.Equal(t, "id", cols[0].Name)
	assert.Equal(t, "name", cols[1].Name)
	assert.Len(t, gotRows, 2)
	assert.Equal(t, "1", gotRows[0]["id"]) // BIGINT列未注册具体类型，兜底string valuer
	assert.Equal(t, "a", gotRows[0]["name"])
	assert.Equal(t, "b", gotRows[1]["name"])
}

// walkFn返回StopWalkQueryError（即使被包装）必须视为正常停止，返回nil错误与已遍历数据。
// 原实现用类型断言，包装错误（fmt.Errorf %w）会误判为异常并向上传播
func TestWalkQueryRows_StopErrorWrapped(t *testing.T) {
	conn, mock := newMockConn(t)

	rows := newMockRows().AddRow(int64(1), "a").AddRow(int64(2), "b").AddRow(int64(3), "c")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	// 第二行返回包装后的StopWalkQueryError
	wrapped := fmt.Errorf("wrap: %w", NewStopWalkQueryError("reached limit"))
	var count int
	cols, err := conn.WalkQueryRows(context.Background(), "SELECT", func(row map[string]any, columns []*QueryColumn) error {
		count++
		if count == 2 {
			return wrapped
		}
		return nil
	})

	// 停止不是异常：错误为nil，已遍历的2行有效
	assert.NoError(t, err)
	assert.Len(t, cols, 2)
	assert.Equal(t, 2, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// 游标中途IO/网络错误必须显式失败，不得静默按正常结束处理。
// 回归背景：原实现在for rows.Next()后不检查rows.Err()，断连时dump会产出截断的备份文件且无任何报错
func TestWalkQueryRows_MidIterationError(t *testing.T) {
	conn, mock := newMockConn(t)

	rows := newMockRows().AddRow(int64(1), "a").AddRow(int64(2), "b").AddRow(int64(3), "c")
	// 第三行迭代时注入IO错误（前两行已正常walkFn），模拟游标中途断连：
	// 必须走到for循环结束后的rows.Err()检查路径（而非Scan错误路径）才能真正回归保护
	midErr := errors.New("driver: bad connection")
	rows.RowError(2, midErr)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	var count int
	_, err := conn.WalkQueryRows(context.Background(), "SELECT", func(row map[string]any, columns []*QueryColumn) error {
		count++
		return nil
	})

	// 错误必须向上传播（dump依赖此失败中断导出，而非产出截断文件）
	assert.ErrorIs(t, err, midErr)
	assert.Equal(t, 2, count) // 已成功读取的2行仍被walkFn处理
}

// walkFn返回普通错误必须向上传播（而非被误判为停止）
func TestWalkQueryRows_NormalErrorPropagated(t *testing.T) {
	conn, mock := newMockConn(t)

	rows := newMockRows().AddRow(int64(1), "a")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	bizErr := assert.AnError
	_, err := conn.WalkQueryRows(context.Background(), "SELECT", func(row map[string]any, columns []*QueryColumn) error {
		return bizErr
	})

	assert.ErrorIs(t, err, bizErr)
	// 普通错误不是StopWalkQueryError，不应被errors.As误判为停止
	var stopErr *StopWalkQueryError
	assert.False(t, errors.As(err, &stopErr))
}
