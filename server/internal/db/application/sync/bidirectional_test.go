package sync

import (
	"mayfly-go/internal/db/domain/entity"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== ConflictDetector Tests ==========

func TestConflictDetector_NoTimestampField(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategySourceWins, "")
	srcRow := map[string]any{"id": 1, "name": "alice"}
	targetRow := map[string]any{"id": 1, "name": "bob"}

	assert.False(t, detector.DetectConflict(srcRow, targetRow), "no timestamp field → no conflict")
}

func TestConflictDetector_SameTimestamp(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategySourceWins, "updated_at")
	srcRow := map[string]any{"id": 1, "updated_at": "2024-01-01 12:00:00"}
	targetRow := map[string]any{"id": 1, "updated_at": "2024-01-01 12:00:00"}

	assert.False(t, detector.DetectConflict(srcRow, targetRow), "same timestamp → no conflict")
}

func TestConflictDetector_SourceWins(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategySourceWins, "updated_at")
	srcRow := map[string]any{"id": 1, "updated_at": "2024-01-02 12:00:00"}
	targetRow := map[string]any{"id": 1, "updated_at": "2024-01-01 12:00:00"}

	assert.False(t, detector.DetectConflict(srcRow, targetRow), "source wins → no skip")
}

func TestConflictDetector_TargetWins(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategyTargetWins, "updated_at")
	srcRow := map[string]any{"id": 1, "updated_at": "2024-01-02 12:00:00"}
	targetRow := map[string]any{"id": 1, "updated_at": "2024-01-01 12:00:00"}

	assert.True(t, detector.DetectConflict(srcRow, targetRow), "target wins → skip sync")
}

func TestConflictDetector_SkipOnConflict(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategySkip, "updated_at")
	srcRow := map[string]any{"id": 1, "updated_at": "2024-01-02 12:00:00"}
	targetRow := map[string]any{"id": 1, "updated_at": "2024-01-01 12:00:00"}

	assert.True(t, detector.DetectConflict(srcRow, targetRow), "skip strategy → skip sync")
}

func TestConflictDetector_MissingTimestamp(t *testing.T) {
	detector := NewConflictDetector(entity.ConflictStrategySourceWins, "updated_at")
	srcRow := map[string]any{"id": 1}
	targetRow := map[string]any{"id": 1, "updated_at": "2024-01-01"}

	assert.False(t, detector.DetectConflict(srcRow, targetRow), "missing timestamp → no conflict")
}

// ========== reverseFieldMapJSON Tests ==========

func TestReverseFieldMapJSON(t *testing.T) {
	input := `[{"src":"id","target":"user_id"},{"src":"name","target":"user_name"}]`
	reversed := reverseFieldMapJSON(input)

	// Parse and verify
	assert.Contains(t, reversed, `"src":"user_id"`)
	assert.Contains(t, reversed, `"target":"id"`)
	assert.Contains(t, reversed, `"src":"user_name"`)
	assert.Contains(t, reversed, `"target":"name"`)
}

func TestReverseFieldMapJSON_InvalidJSON(t *testing.T) {
	input := "not json"
	result := reverseFieldMapJSON(input)
	assert.Equal(t, input, result, "invalid JSON should be returned as-is")
}

// ========== SyncMetrics Tests ==========

func TestSyncMetrics_RecordBatch(t *testing.T) {
	m := NewSyncMetrics()
	m.RecordBatch(100)
	m.RecordBatch(200)

	assert.Equal(t, 300, m.TotalRows)
	assert.Equal(t, 2, m.BatchCount)
}

func TestSyncMetrics_ToSyncLog(t *testing.T) {
	m := NewSyncMetrics()
	m.StartTime = 1000
	m.EndTime = 2000
	m.TotalRows = 500
	m.BatchCount = 5
	m.InsertCount = 400
	m.UpdateCount = 80
	m.DeleteCount = 20
	m.SkipCount = 10

	log := &entity.DataSyncLog{}
	m.ToSyncLog(log)

	assert.Equal(t, 500, log.ResNum)
	assert.Equal(t, 5, log.BatchCount)
	assert.Equal(t, 400, log.InsertCount)
	assert.Equal(t, 80, log.UpdateCount)
	assert.Equal(t, 20, log.DeleteCount)
	assert.Equal(t, 10, log.SkipCount)
	assert.Equal(t, int64(1000), log.DurationMs)
	assert.Equal(t, 500, log.Throughput, "500 rows / 1 second = 500 rows/s")
}

func TestSyncMetrics_ToSyncLog_ZeroDuration(t *testing.T) {
	m := NewSyncMetrics()
	m.StartTime = 1000
	m.EndTime = 1000 // same as start
	m.TotalRows = 100

	log := &entity.DataSyncLog{}
	m.ToSyncLog(log)

	assert.Equal(t, int64(0), log.DurationMs)
	assert.Equal(t, 0, log.Throughput, "zero duration → zero throughput")
}

// ========== ResolveConflict Tests ==========

func TestConflictDetector_ResolveConflict(t *testing.T) {
	tests := []struct {
		strategy entity.ConflictStrategy
		contains string
	}{
		{entity.ConflictStrategySourceWins, "source wins"},
		{entity.ConflictStrategyTargetWins, "target wins"},
		{entity.ConflictStrategySkip, "skipping"},
	}

	for _, tt := range tests {
		detector := NewConflictDetector(tt.strategy, "updated_at")
		msg := detector.ResolveConflict("test-task", "id=1")
		assert.Contains(t, msg, tt.contains)
	}
}
