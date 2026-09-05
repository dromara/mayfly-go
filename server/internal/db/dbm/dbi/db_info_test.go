package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// CurrentSchema 解析 database/schema 格式，schema部分为空或无斜杠时返回空
func TestDbInfo_CurrentSchema(t *testing.T) {
	cases := []struct {
		database string
		expected string
	}{
		{"", ""},
		{"mydb", ""},
		{"mydb/public", "public"},
		{"mydb/dbo", "dbo"},
		// 超过2段视为非法格式，不取值
		{"a/b/c", ""},
	}
	for _, c := range cases {
		di := &DbInfo{Database: c.database}
		assert.Equal(t, c.expected, di.CurrentSchema(), "database: %s", c.database)
	}
}

// GetDatabase 返回 database/schema 格式中的 database 部分
func TestDbInfo_GetDatabase(t *testing.T) {
	cases := []struct {
		database string
		expected string
	}{
		{"", ""},
		{"mydb", "mydb"},
		{"mydb/public", "mydb"},
		{"a/b/c", "a"},
	}
	for _, c := range cases {
		di := &DbInfo{Database: c.database}
		assert.Equal(t, c.expected, di.GetDatabase(), "database: %s", c.database)
	}
}

// GetDbConnId：dbId为0（未持久化的临时连接）必须返回空id，避免与其他池key冲突
func TestGetDbConnId(t *testing.T) {
	assert.Equal(t, "", GetDbConnId(0, "mydb"))
	assert.Equal(t, "db-5:mydb", GetDbConnId(5, "mydb"))
	assert.Equal(t, "db-5:mydb/public", GetDbConnId(5, "mydb/public"))
}

// GetRemoteAddr：ssh隧道映射后地址优先于原始host:port
func TestDbInfo_GetRemoteAddr(t *testing.T) {
	di := &DbInfo{Host: "10.0.0.1", Port: 3306}
	assert.Equal(t, "10.0.0.1:3306", di.GetRemoteAddr())

	di.RemoteAddr = "127.0.0.1:33306"
	assert.Equal(t, "127.0.0.1:33306", di.GetRemoteAddr())
}

func TestDbType_Equal(t *testing.T) {
	assert.True(t, DbType("mysql").Equal("mysql"))
	assert.False(t, DbType("mysql").Equal("MYSQL"))
	assert.False(t, DbType("mysql").Equal("postgres"))
}
