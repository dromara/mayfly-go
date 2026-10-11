package base

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"mayfly-go/pkg/model"
)

// 提交后动作（AfterCommit）的回归用例。
//
// 对应实测复现过的缺陷：动作拿到的是那条**已提交**的事务连接导致任何写入都失败、动作内部再登记
// 的动作被追加进已排空的队列而静默丢失、动作 panic 从事务的 recover 之外穿出去让「已提交」
// 在调用方眼里变成「失败」

// TestAfterCommitCanWrite 提交后动作必须真的能落库。
//
// 若动作拿到的是那条已提交的事务连接，写入会撞
// "sql: transaction has already been committed or rolled back"，整条「提交后再做」的链路随之失效
func TestAfterCommitCanWrite(t *testing.T) {
	db := setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)

	var actionErr error
	err := app.Tx(context.Background(), func(txCtx context.Context) error {
		if err := app.Insert(txCtx, &ReviewEntity{Name: "created"}); err != nil {
			return err
		}
		return AfterCommit(txCtx, func(runCtx context.Context) error {
			actionErr = app.UpdateColsByCond(runCtx, NewPatch().Set("name", "after-commit"),
				model.NewCond().Eq("name", "created"))
			return actionErr
		})
	})
	if err != nil {
		t.Fatalf("tx failed: %v (action err: %v)", err, actionErr)
	}
	if actionErr != nil {
		t.Fatalf("after-commit write failed: %v", actionErr)
	}

	var name string
	if err := db.Table("review_entity").Select("name").Row().Scan(&name); err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if name != "after-commit" {
		t.Errorf("name = %q, want %q：提交后动作的写入必须真的生效", name, "after-commit")
	}
}

// TestAfterCommitCanRead 提交后的动作既能写也能读。
//
// 事务闭包内不收 ctx 的读在单连接池上会等一个永不释放的连接而挂死（由架构守卫
// TestNoCtxlessReadsInsideTxClosure 固定）；这里验的是另一半：动作 ctx 已脱离事务，
// 它去连接池读是安全的——事务此时已结束，不再占着连接
func TestAfterCommitCanRead(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)

	var readBack string
	err := app.Tx(context.Background(), func(txCtx context.Context) error {
		if err := app.Insert(txCtx, &ReviewEntity{Name: "committed"}); err != nil {
			return err
		}
		return AfterCommit(txCtx, func(runCtx context.Context) error {
			list, err := app.ListByCond(model.NewCond().Eq("name", "committed"))
			if err != nil {
				return err
			}
			if len(list) != 1 {
				return fmt.Errorf("rows = %d, want 1: 提交后应能读到本事务已提交的行", len(list))
			}
			readBack = list[0].Name
			return nil
		})
	})
	if err != nil {
		t.Fatalf("tx failed: %v", err)
	}
	if readBack != "committed" {
		t.Errorf("read back %q, want %q", readBack, "committed")
	}
}

// TestAfterCommitPanicBecomesErrAfterCommit 动作 panic 必须收敛成 ErrAfterCommit，不能穿出去。
//
// 事务那层的 recover 在动作执行时已经弹出，panic 会从**已提交**的事务里一路上抛：调用方拿到 500
// 而误判整件事失败并重试，已经生效的副作用因此被再做一遍；排在后面的动作也会被无声跳过
func TestAfterCommitPanicBecomesErrAfterCommit(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)

	errBoom := errors.New("action blew up")
	var secondRan bool

	err := app.Tx(context.Background(), func(txCtx context.Context) error {
		if err := app.Insert(txCtx, &ReviewEntity{Name: "kept"}); err != nil {
			return err
		}
		if err := AfterCommit(txCtx, func(context.Context) error { panic(errBoom) }); err != nil {
			return err
		}
		// panic 的动作之后登记的动作仍须执行：动作之间互不相关，前一个炸了不该剥夺后一个的机会
		return AfterCommit(txCtx, func(context.Context) error {
			secondRan = true
			return nil
		})
	})

	if !errors.Is(err, ErrAfterCommit) {
		t.Errorf("err = %v, want it to match ErrAfterCommit：事务已提交这一事实必须能被调用方判出来", err)
	}
	if !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want it to wrap the panicking action's error：补偿需要知道是哪个动作炸的", err)
	}
	if !secondRan {
		t.Error("前一个动作 panic 后，后续动作被无声跳过了")
	}

	// 事务本身必须已提交：panic 发生在提交之后，不能让它变成回滚
	if !existsRow(t, repo, "kept") {
		t.Error("提交后动作 panic 导致数据不见了：事务应当已经提交")
	}
}

// TestAfterCommitActionCtxIsDetached 动作 ctx 必须脱事务绑定，且仍可再登记新的提交后动作。
//
// 队列已排空，动作内再登记时若仍带着原队列，新动作会追加进一个不再被排空的队列而静默丢失；
// 正确行为是「无事务可等 → 立即执行」
func TestAfterCommitActionCtxIsDetached(t *testing.T) {
	setupTestDb(t)
	repo := newTestRepo()
	app := newTestApp(repo)

	var innerRan bool
	err := app.Tx(context.Background(), func(txCtx context.Context) error {
		return AfterCommit(txCtx, func(runCtx context.Context) error {
			return AfterCommit(runCtx, func(context.Context) error {
				innerRan = true
				return nil
			})
		})
	})
	if err != nil {
		t.Fatalf("tx failed: %v", err)
	}
	if !innerRan {
		t.Error("动作内部登记的提交后动作被丢弃了：脱离事务后它应立即执行")
	}
}

func existsRow(t *testing.T, repo *RepoImpl[*ReviewEntity], name string) bool {
	t.Helper()
	exists, err := repo.Exists(context.Background(), model.NewCond().Eq("name", name))
	if err != nil {
		t.Fatalf("exists check failed: %v", err)
	}
	return exists
}
