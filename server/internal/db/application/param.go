package application

import "io"

type DumpDbReq struct {
	DbId     uint64
	DbName   string
	Tables   []string
	DumpDDL  bool // 是否dump ddl
	DumpData bool // 是否dump data

	Writer io.Writer
}
