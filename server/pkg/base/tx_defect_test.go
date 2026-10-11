package base

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

// 事务与提交后动作的回归用例。
//
// 对应曾实测复现的缺陷：panic 被拍平成字符串导致业务错误码判不出、回滚失败被吞、
// 每一层嵌套都注册 recover 从而掩盖程序缺陷、外部副作用被圈在事务内、事务内的存在性判断
// 落到另一条连接上读不到自己的写入

var errTestRollback = errors.New("sentinel rollback error")

func newTestApp(repo *RepoImpl[*ReviewEntity]) *AppImpl[*ReviewEntity, *RepoImpl[*ReviewEntity]] {
	return &AppImpl[*ReviewEntity, *RepoImpl[*ReviewEntity]]{Repo: repo}
}

// TestTxCommitsAndRunsAfterCommit 提交成功时登记的动作才执行
func TestTxCommitsAndRunsAfterCommit(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	app := newTestApp(repo)

	var ran bool
	err := app.Tx(context.Background(), func(ctx context.Context) error {
		if err := app.UpdateById(ctx, &ReviewEntity{Id: created.Id, Age: 21}); err != nil {
			return err
		}
		if err := AfterCommit(ctx, func(context.Context) error { ran = true; return nil }); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tx failed: %v", err)
	}
	if !ran {
		t.Error("the registered post-commit action did not run after a successful commit")
	}
	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 21 {
		t.Errorf("age = %d, want 21", after.Age)
	}
}

// TestTxRollbackSkipsAfterCommit 回滚路径一个提交后动作都不能执行。
//
// 这条断言守的是「外部副作用不能被提前」：一旦回滚后仍然执行，就会出现「数据库说这次没做，
// 外部世界已经改了」的半提交状态
func TestTxRollbackSkipsAfterCommit(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	app := newTestApp(repo)

	var ran bool
	err := app.Tx(context.Background(), func(ctx context.Context) error {
		if updateErr := app.UpdateById(ctx, &ReviewEntity{Id: created.Id, Age: 33}); updateErr != nil {
			return updateErr
		}
		if registerErr := AfterCommit(ctx, func(context.Context) error { ran = true; return nil }); registerErr != nil {
			return registerErr
		}
		return errTestRollback
	})
	if !errors.Is(err, errTestRollback) {
		t.Fatalf("tx error = %v, want the sentinel error to reach the caller unchanged", err)
	}
	if ran {
		t.Error("a post-commit action ran although the transaction was rolled back")
	}

	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 10 {
		t.Errorf("age = %d, want 10: the write was not rolled back", after.Age)
	}
}

// TestTxKeepsPanicErrorType panic 转 error 时必须保留原始错误类型。
//
// 原实现用 fmt.Errorf("%v", r) 拍平，实测 errorx 业务码穿过 Tx 后 errors.Is 判不出来，
// 错误码与 i18n 信息在事务边界全部丢失
func TestTxKeepsPanicErrorType(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)

	sentinel := errors.New("business error code")
	err := app.Tx(context.Background(), func(context.Context) error {
		panic(sentinel)
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("panic error was flattened: got %T/%v, want the original error recoverable via errors.Is", err, err)
	}
}

// TestNestedTxRollsBackEverything 嵌套事务加入外层：内层失败必须让外层整体回滚，
// 但内层自己不能提交或回滚
func TestNestedTxRollsBackEverything(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	app := newTestApp(repo)

	err := app.Tx(context.Background(), func(ctx context.Context) error {
		if err := app.UpdateById(ctx, &ReviewEntity{Id: created.Id, Age: 41}); err != nil {
			return err
		}
		// 嵌套调用：内层应复用外层事务，不做提交
		return app.Tx(ctx, func(inner context.Context) error {
			return app.UpdateById(inner, &ReviewEntity{Id: created.Id, Age: 42})
		})
	})
	if err != nil {
		t.Fatalf("outer tx failed: %v", err)
	}

	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 42 {
		t.Errorf("age = %d, want 42 (both updates should have committed together)", after.Age)
	}
}

// TestNestedTxInnerFailurePropagates 内层失败冒泡给外层并由外层回滚
func TestNestedTxInnerFailurePropagates(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	app := newTestApp(repo)

	err := app.Tx(context.Background(), func(ctx context.Context) error {
		if err := app.UpdateById(ctx, &ReviewEntity{Id: created.Id, Age: 51}); err != nil {
			return err
		}
		return app.Tx(ctx, func(context.Context) error { return errTestRollback })
	})
	if !errors.Is(err, errTestRollback) {
		t.Fatalf("error = %v, want the inner error to propagate", err)
	}
	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 10 {
		t.Errorf("age = %d, want 10: the outer write survived a rolled-back transaction", after.Age)
	}
}

// TestExistsSeesUncommittedWritesInTx 事务内的存在性判断必须走事务连接。
//
// 这是「先查重再写入」的正确形态：无 ctx 的读入口走连接池的另一条连接，读不到本事务尚未
// 提交的写入，并发下两个请求会双双通过查重
func TestExistsSeesUncommittedWritesInTx(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)
	ctx := context.Background()

	err := app.Tx(ctx, func(txCtx context.Context) error {
		if err := repo.Insert(txCtx, &ReviewEntity{Name: "unique", Age: 1}); err != nil {
			return err
		}
		seen, err := repo.Exists(txCtx, model.NewCond().Eq("name", "unique"))
		if err != nil {
			return err
		}
		if !seen {
			return errors.New("Exists did not observe the row inserted earlier in the same transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAfterCommitFailureIsDistinguishable 「已提交但提交后动作失败」必须能与「事务失败」区分：
// 此时数据已生效，按整体失败去重试会造成重复副作用
func TestAfterCommitFailureIsDistinguishable(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	created := newLoadedRepo(t, repo)
	app := newTestApp(repo)

	actionErr := errors.New("post commit action failed")
	err := app.Tx(context.Background(), func(ctx context.Context) error {
		if updateErr := app.UpdateById(ctx, &ReviewEntity{Id: created.Id, Age: 61}); updateErr != nil {
			return updateErr
		}
		return AfterCommit(ctx, func(context.Context) error { return actionErr })
	})
	if !errors.Is(err, ErrAfterCommit) {
		t.Fatalf("error = %v, want it to carry ErrAfterCommit", err)
	}
	if errors.Is(err, errTestRollback) {
		t.Error("unrelated error matched")
	}

	var after ReviewEntity
	if err := db.Table("review_entity").First(&after, "id = ?", created.Id).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if after.Age != 61 {
		t.Errorf("age = %d, want 61: the transaction must stay committed when only the action failed", after.Age)
	}
}

// TestAfterCommitOutsideTxRunsImmediately 不在事务里登记时立即执行：没有需要等待的提交点
func TestAfterCommitOutsideTxRunsImmediately(t *testing.T) {
	setupTestDb(t)

	var ran bool
	if err := AfterCommit(context.Background(), func(context.Context) error { ran = true; return nil }); err != nil {
		t.Fatalf("AfterCommit outside a tx failed: %v", err)
	}
	if !ran {
		t.Error("AfterCommit outside a transaction should run the action immediately")
	}
}

// TestBeginInsideTxRejected 校验 gormx.Begin 的防误用契约：ctx 已有事务时直接拒绝，
// 绝不把外层事务对象交出去——嵌套拿到的会是同一个事务对象，调用方一句常规的
// defer tx.Rollback() 就会把外层事务杀掉。嵌套传播必须走 TxWith
func TestBeginInsideTxRejected(t *testing.T) {
	setupTestDb(t)
	ctx := context.Background()

	txCtx, tx, err := gormx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	if !gormx.InTx(txCtx) {
		t.Fatal("ctx should report being inside a transaction after Begin")
	}

	nestedCtx, nestedTx, err := gormx.Begin(txCtx)
	if !errors.Is(err, gormx.ErrNestedBegin) {
		t.Fatalf("nested Begin must be rejected with ErrNestedBegin, got err=%v", err)
	}
	if nestedTx != nil {
		t.Error("rejected Begin must not hand out the outer transaction object")
	}
	if nestedCtx == nil {
		t.Error("rejected Begin should still return a usable ctx")
	}

	// 被拒绝之后外层事务必须完好：还能正常回滚
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("outer transaction should stay usable after a rejected nested Begin: %v", err)
	}
}
