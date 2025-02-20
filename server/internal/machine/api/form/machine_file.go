package form

import (
	"mayfly-go/internal/machine/application/dto"
)

type MachineFileForm struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name" binding:"required"`
	MachineId uint64 `json:"machineId" binding:"required"`
	Type      int8   `json:"type" binding:"required"`
	Path      string `json:"path" binding:"required"`
}

type MachineFileUpdateForm struct {
	Content string `json:"content" binding:"required"`
	Id      uint64 `json:"id" binding:"required"`
	Path    string `json:"path" binding:"required"`
}

type CreateFileForm struct {
	*dto.MachineFileOp

	Type string `json:"type"`
}

type WriteFileContentForm struct {
	*dto.MachineFileOp

	Content string `json:"content" binding:"required"`
}

type RemoveFileForm struct {
	*dto.MachineFileOp

	Paths []string `json:"paths" binding:"required"`
}

type CopyFileForm struct {
	*dto.MachineFileOp

	Paths  []string `json:"paths" binding:"required"`
	ToPath string   `json:"toPath" binding:"required"`
}

type RenameForm struct {
	*dto.MachineFileOp

	Newname string `json:"newname" binding:"required"`
}
