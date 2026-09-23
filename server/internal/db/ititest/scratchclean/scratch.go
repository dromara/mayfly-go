// Package scratchclean 管理测试进程独占的临时数据库。
// 只有本进程成功创建的库才可清理；不枚举或删除共享库中的对象。
// 各 TestMain 通过 Run 无条件收尾，清理失败使测试失败。强制杀进程或数据库离线
// 仍可能留下临时库，错误日志会给出名称；不得通过扫描名称前缀自动删除其他运行的库。
package scratchclean

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
)

type database struct {
	admin dbi.DbInfo
	name  string
}

// Scope 保存本次运行创建的数据库及连接，支持不同测试进程并行运行。
type Scope struct {
	mu    sync.Mutex
	dbs   map[string]database
	conns []*dbi.DbConn
}

var current Scope

// Conn 将约定的测试库名映射到随机临时库；其他连接（如 SQLite 临时文件）保持原样。
func Conn(ctx context.Context, info *dbi.DbInfo) (*dbi.DbConn, error) {
	return current.Conn(ctx, info)
}

func (s *Scope) Conn(ctx context.Context, info *dbi.DbInfo) (*dbi.DbConn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyInfo := *info
	adminDB, suffix, scoped := scratchTarget(info)
	if scoped {
		key := fmt.Sprintf("%s/%s/%d/%s/%s/%s", info.Type, info.Host, info.Port, info.Database, info.Username, info.Params)
		db, ok := s.dbs[key]
		if !ok {
			var token [12]byte
			if _, err := rand.Read(token[:]); err != nil {
				return nil, err
			}
			db = database{admin: *info, name: "mayfly_it_" + hex.EncodeToString(token[:])}
			db.admin.Database = adminDB
			admin, err := dbm.Conn(ctx, &db.admin)
			if err != nil {
				return nil, err
			}
			stmt := "CREATE DATABASE " + admin.GetDialect().Quoter().QuoteIdent(db.name)
			if info.Type == "mysql" {
				stmt += " DEFAULT CHARSET utf8mb4"
			}
			_, createErr := admin.ExecContext(ctx, stmt)
			closeErr := admin.Close()
			if createErr == nil {
				if s.dbs == nil {
					s.dbs = make(map[string]database)
				}
				s.dbs[key] = db
			}
			if err := errors.Join(createErr, closeErr); err != nil {
				return nil, err
			}
		}
		copyInfo.Database = db.name + suffix
	}
	conn, err := dbm.Conn(ctx, &copyInfo)
	if err != nil {
		return nil, err
	}
	s.conns = append(s.conns, conn)
	return conn, nil
}

func scratchTarget(info *dbi.DbInfo) (admin, suffix string, scoped bool) {
	switch {
	case info.Type == "mysql" && info.Database == "mayfly_dbm_it":
		return "information_schema", "", true
	case info.Type == "postgres" && info.Database == "mayfly_pg_it":
		return "postgres", "", true
	case info.Type == "mssql" && info.Database == "mayfly_it/dbo":
		return "master/dbo", "/dbo", true
	default:
		return "", "", false
	}
}

// Cleanup 关闭本次运行的连接并删除独占库；失败资源保留在清单中供显式重试。
func (s *Scope) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for _, conn := range s.conns {
		if err := conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	s.conns = nil
	for key, db := range s.dbs {
		if err := dropDatabase(ctx, db); err != nil {
			errs = append(errs, fmt.Errorf("清理测试库 %s 失败: %w", db.name, err))
		} else {
			delete(s.dbs, key)
		}
	}
	return errors.Join(errs...)
}

func dropDatabase(ctx context.Context, db database) (err error) {
	// 名称由本进程随机生成且仅从创建成功的清单取出，拒绝非托管名称。
	token, decodeErr := hex.DecodeString(strings.TrimPrefix(db.name, "mayfly_it_"))
	if !strings.HasPrefix(db.name, "mayfly_it_") || decodeErr != nil || len(token) != 12 {
		return fmt.Errorf("拒绝清理非托管数据库 %q", db.name)
	}
	conn, err := dbm.Conn(ctx, &db.admin)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	stmt := "DROP DATABASE " + conn.GetDialect().Quoter().QuoteIdent(db.name)
	_, err = conn.ExecContext(ctx, stmt)
	return err
}

// Run 供 TestMain 调用；普通失败同样清理，且不会把清理失败报告为测试成功。
func Run(tests func() int) int {
	return run(tests, current.Cleanup)
}

func run(tests func() int, cleanup func(context.Context) error) (code int) {
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := cleanup(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
	}()
	return tests()
}
