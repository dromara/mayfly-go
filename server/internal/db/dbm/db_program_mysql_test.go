package dbm

import (
	"github.com/stretchr/testify/require"
	"mayfly-go/internal/db/domain/entity"
	"strings"
	"testing"
)

func Test_readBinlogInfoFromBackup(t *testing.T) {
	text := `
--
-- Position to start replication or point-in-time recovery from
--

-- CHANGE MASTER TO MASTER_LOG_FILE='binlog.000003', MASTER_LOG_POS=379;
`
	got, err := readBinlogInfoFromBackup(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, &entity.BinlogInfo{
		FileName: "binlog.000003",
		Sequence: 3,
		Position: 379,
	}, got)
}
