//go:build it

package transfer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDbgMtLoad(t *testing.T) {
	conn := itSqliteNode(t)
	defer conn.Close()
	spec := &mtSpecs[0]
	dialect := "sqlite"
	mtCreateAndLoad(t, conn, spec, dialect)
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query("SELECT COUNT(*) AS c, MIN(id) AS a, MAX(id) AS b FROM " + quote(spec.name))
	require.NoError(t, err)
	t.Log("count:", rows[0]["c"], "min:", rows[0]["a"], "max:", rows[0]["b"])
}
