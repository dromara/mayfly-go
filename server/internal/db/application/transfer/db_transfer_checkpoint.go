package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	dbentity "mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/model"
)

// transferCheckpoint 迁移断点续传检查点（内存形态）
type transferCheckpoint struct {
	// PlannedTables 计划迁移表清单（任务启动时快照，保证续传确定性）
	PlannedTables []string `json:"plannedTables"`
	// DoneTables 已完成迁移的表清单
	DoneTables []string `json:"doneTables"`
	// StartedAt 本次迁移链启动时间
	StartedAt time.Time `json:"startedAt"`
}

// checkpointStore 检查点存取抽象（生产为gorm实现，测试可注入内存实现）
type checkpointStore interface {
	// Load 加载任务检查点，不存在返回(nil, nil)
	Load(taskId uint64) (*transferCheckpoint, error)

	// Save 全量保存检查点（用于初始化）
	Save(taskId uint64, cp *transferCheckpoint) error

	// AppendDone 原子追加已完成表并持久化（并发安全，避免多表同时完成时丢失更新）
	AppendDone(taskId uint64, table string) error

	// Clear 清除任务检查点（迁移全部完成后调用）
	Clear(taskId uint64) error
}

// gormCheckpointStore 基于t_db_transfer_checkpoint表的检查点存储
type gormCheckpointStore struct {
	repo repository.DbTransferCheckpoint

	mu sync.Mutex // 保护Load-Modify-Save序列的并发安全
}

func newGormCheckpointStore(repo repository.DbTransferCheckpoint) *gormCheckpointStore {
	return &gormCheckpointStore{repo: repo}
}

func (s *gormCheckpointStore) Load(taskId uint64) (*transferCheckpoint, error) {
	entityCp, err := s.repo.GetByTaskId(taskId)
	if err != nil || entityCp == nil {
		return nil, err
	}
	planned, err := unmarshalStringSlice(entityCp.PlannedTables)
	if err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint planned tables: %w", err)
	}
	done, err := unmarshalStringSlice(entityCp.DoneTables)
	if err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint done tables: %w", err)
	}
	startedAt := time.Time{}
	if entityCp.CreateTime != nil {
		startedAt = *entityCp.CreateTime
	}
	return &transferCheckpoint{
		PlannedTables: planned,
		DoneTables:    done,
		StartedAt:     startedAt,
	}, nil
}

func (s *gormCheckpointStore) Save(taskId uint64, cp *transferCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(taskId, cp)
}

func (s *gormCheckpointStore) AppendDone(taskId uint64, table string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp, err := s.Load(taskId)
	if err != nil {
		return err
	}
	if cp == nil {
		return fmt.Errorf("checkpoint of transfer task [%d] not found", taskId)
	}
	for _, t := range cp.DoneTables {
		if t == table {
			return nil // 幂等
		}
	}
	cp.DoneTables = append(cp.DoneTables, table)
	return s.saveLocked(taskId, cp)
}

func (s *gormCheckpointStore) Clear(taskId uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repo.DeleteByCond(context.Background(), model.NewCond().Eq("task_id", taskId))
}

// saveLocked 保存检查点（调用方需持有mu；不存在则新增，否则更新）
func (s *gormCheckpointStore) saveLocked(taskId uint64, cp *transferCheckpoint) error {
	planned, err := marshalStringSlice(cp.PlannedTables)
	if err != nil {
		return err
	}
	done, err := marshalStringSlice(cp.DoneTables)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetByTaskId(taskId)
	if err != nil {
		return err
	}
	if existing == nil {
		entityCp := &dbentity.DbTransferCheckpoint{
			TaskId:        taskId,
			PlannedTables: planned,
			DoneTables:    done,
		}
		return s.repo.Insert(context.Background(), entityCp)
	}
	existing.PlannedTables = planned
	existing.DoneTables = done
	now := time.Now()
	existing.UpdateTime = &now
	return s.repo.UpdateById(context.Background(), existing)
}

// marshalStringSlice 序列化字符串数组；nil统一序列化为"[]"，保证往返一致
func marshalStringSlice(ss []string) (string, error) {
	if ss == nil {
		ss = []string{}
	}
	bytes, err := json.Marshal(ss)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// unmarshalStringSlice 反序列化字符串数组；空串/null视为空数组
func unmarshalStringSlice(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return []string{}, nil
	}
	var res []string
	if err := json.Unmarshal([]byte(s), &res); err != nil {
		return nil, err
	}
	if res == nil {
		res = []string{}
	}
	return res, nil
}

// initCheckpoint 初始化或加载断点检查点。
// 返回有效检查点与是否为断点续传：
//   - 无存量检查点：新建（快照本次计划表清单），非续传
//   - 有存量检查点且计划表清单与本次一致：续传（跳过已完成表）
//   - 有存量检查点但计划表清单不一致（任务配置变更）：作废重建，非续传
func initCheckpoint(store checkpointStore, taskId uint64, plannedTables []string) (*transferCheckpoint, bool, error) {
	existing, err := store.Load(taskId)
	if err != nil {
		return nil, false, err
	}
	if existing != nil && equalStringSlice(existing.PlannedTables, plannedTables) {
		return existing, true, nil
	}
	cp := &transferCheckpoint{
		PlannedTables: plannedTables,
		DoneTables:    []string{},
		StartedAt:     time.Now(),
	}
	if err := store.Save(taskId, cp); err != nil {
		return nil, false, err
	}
	return cp, false, nil
}

// remainingTables 返回planned中未被done包含的表（保持planned顺序）
func remainingTables(planned, done []string) []string {
	doneSet := make(map[string]struct{}, len(done))
	for _, t := range done {
		doneSet[t] = struct{}{}
	}
	res := make([]string, 0, len(planned))
	for _, t := range planned {
		if _, ok := doneSet[t]; !ok {
			res = append(res, t)
		}
	}
	return res
}

// equalStringSlice 判断两个字符串数组内容是否一致（顺序敏感，表清单在存取前均已排序）
func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
