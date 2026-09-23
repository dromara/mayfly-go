package dbi

import (
	"context"
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

// fakeTunnelOpener 记录每次建立请求的 spec 并返回固定的本地可拨地址
type fakeTunnelOpener struct {
	got []TunnelSpec
}

func (f *fakeTunnelOpener) Open(_ context.Context, spec TunnelSpec) (*Tunnel, error) {
	f.got = append(f.got, spec)
	return NewTunnel("127.0.0.1", 33306, func() {}), nil
}

// dialTunnel 隧道建立的地址改写与原始地址保留（防二次改写连错目标）
func TestDbInfo_dialTunnel(t *testing.T) {
	opener := &fakeTunnelOpener{}
	RegisterTunnelOpener(opener)
	defer func() { tunnelOpener = nil }() // 复位全局，避免污染其他用例

	// 未配置中转：返回 nil 句柄，不改写地址，不触碰 opener
	plain := &DbInfo{Host: "10.0.0.1", Port: 3306}
	tun, err := plain.dialTunnel(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, tun)
	assert.Equal(t, "10.0.0.1", plain.Host)
	assert.Empty(t, opener.got)

	// 配置中转：目标地址改写为通道本地地址，原始地址被记录进 RemoteAddr
	di := &DbInfo{Host: "10.0.0.1", Port: 3306, SshTunnelMachineId: 7}
	tun, err = di.dialTunnel(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, tun)
	assert.Equal(t, "127.0.0.1", di.Host)
	assert.Equal(t, 33306, di.Port)
	assert.Equal(t, "10.0.0.1:3306", di.RemoteAddr)
	// opener 收到的是「中转机id + 原始地址」，而非改写后的本地地址
	assert.Equal(t, TunnelSpec{MachineId: 7, RemoteAddr: "10.0.0.1:3306"}, opener.got[0])

	// 二次进入不得把已被改写的本地地址当作原始地址覆盖（否则隧道重建会连错目标）
	_, _ = di.dialTunnel(context.Background())
	assert.Equal(t, "10.0.0.1:3306", di.RemoteAddr)
}

// Tunnel.Close 对 nil 接收者安全，且释放钩子只触发一次语义由实现方保证，此处仅验证 nil 安全
func TestTunnel_CloseNilSafe(t *testing.T) {
	var nilTunnel *Tunnel
	assert.NotPanics(t, func() { nilTunnel.Close() })

	released := 0
	tun := NewTunnel("127.0.0.1", 1, func() { released++ })
	tun.Close()
	assert.Equal(t, 1, released)
}

func TestDbType_Equal(t *testing.T) {
	assert.True(t, DbType("mysql").Equal("mysql"))
	assert.False(t, DbType("mysql").Equal("MYSQL"))
	assert.False(t, DbType("mysql").Equal("postgres"))
}
