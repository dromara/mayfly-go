package form

import "mayfly-go/internal/machine/application"

type MachineFileForm struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name" binding:"required"`
	MachineId uint64 `json:"machineId" binding:"required"`
	Type      int    `json:"type" binding:"required"`
	Path      string `json:"path" binding:"required"`
}

type MachineFileUpdateForm struct {
	Content string `json:"content" binding:"required"`
	Id      uint64 `json:"id" binding:"required"`
	Path    string `json:"path" binding:"required"`
}

type CreateFileForm struct {
	*application.MachineFileOpParam

	Type string `json:"type"`
}

type WriteFileContentForm struct {
	*application.MachineFileOpParam

	Content string `json:"content" binding:"required"`
}

type RemoveFileForm struct {
	*application.MachineFileOpParam

	Paths []string `json:"paths" binding:"required"`
}

type CopyFileForm struct {
	*application.MachineFileOpParam

	Paths  []string `json:"paths" binding:"required"`
	ToPath string   `json:"toPath" binding:"required"`
}

type RenameForm struct {
	*application.MachineFileOpParam

	Newname string `json:"newname" binding:"required"`
}
