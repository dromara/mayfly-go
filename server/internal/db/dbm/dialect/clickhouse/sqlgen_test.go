package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClickHouseGenTruncate(t *testing.T) {
	gen := newTestSQLGenerator()
	sqls := gen.GenTruncate("t_user")
	assert.Nil(t, sqls, "ClickHouse does not support TRUNCATE TABLE, should return nil")
}

func TestClickHouseGenBatchDelete_SinglePK(t *testing.T) {
	gen := newTestSQLGenerator()
	sqls := gen.GenBatchDelete("t_user", []string{"id"}, [][]any{{1}, {2}, {3}}, nil)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "ALTER TABLE")
	assert.Contains(t, sqls[0], "DELETE WHERE")
	assert.Contains(t, sqls[0], "`id` NOT IN ('1', '2', '3')")
}

func TestClickHouseGenBatchDelete_CompositePK(t *testing.T) {
	gen := newTestSQLGenerator()
	sqls := gen.GenBatchDelete("t_user", []string{"k1", "k2"}, [][]any{{1, "a"}, {2, "b"}}, nil)
	assert.Len(t, sqls, 1)
	assert.Contains(t, sqls[0], "(`k1`, `k2`) NOT IN (('1', 'a'), ('2', 'b'))")
}

func TestClickHouseGenBatchDelete_EmptyInput(t *testing.T) {
	gen := newTestSQLGenerator()
	assert.Nil(t, gen.GenBatchDelete("t", nil, nil, nil))
	assert.Nil(t, gen.GenBatchDelete("t", []string{"id"}, [][]any{}, nil))
}
