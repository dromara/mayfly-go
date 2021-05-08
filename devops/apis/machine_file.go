package apis

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"mayfly-go/base/biz"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/devops/apis/form"
	"mayfly-go/devops/apis/vo"
	"mayfly-go/devops/application"
	"mayfly-go/devops/domain/entity"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type MachineFile struct {
	MachineFileApp application.IMachineFile
	MachineApp     application.IMachine
}

const (
	file          = "-"
	dir           = "d"
	link          = "l"
	max_read_size = 5 * 1024 * 1024
)

func (m *MachineFile) MachineFiles(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	condition := &entity.MachineFile{MachineId: GetMachineId(g)}
	rc.ResData = m.MachineFileApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineFileVO))
}

func (m *MachineFile) SaveMachineFiles(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fileForm := new(form.MachineFileForm)
	ginx.BindJsonAndValid(g, fileForm)

	entity := new(entity.MachineFile)
	utils.Copy(entity, fileForm)

	entity.SetBaseInfo(rc.LoginAccount)

	m.MachineFileApp.Save(entity)
}

func (m *MachineFile) DeleteFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	m.MachineFileApp.Delete(fid)
}

/***      sftp相关操作      */

func (m *MachineFile) ReadFileContent(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")
	readType := g.Query("type")

	dataByte, fileInfo := m.MachineFileApp.ReadFile(fid, readPath)
	// 如果是读取文件内容，则校验文件大小
	if readType != "1" {
		biz.IsTrue(fileInfo.Size() < max_read_size, "文件超过5m，请使用下载查看")
	}

	rc.ReqParam = fmt.Sprintf("path: %s", readPath)
	// 如果读取类型为下载，则下载文件，否则获取文件内容
	if readType == "1" {
		// 截取文件名，如/usr/local/test.java -》 test.java
		path := strings.Split(readPath, "/")
		rc.Download(dataByte, path[len(path)-1])
	} else {
		rc.ResData = string(dataByte)
	}
}

func (m *MachineFile) GetDirEntry(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	if !strings.HasSuffix(readPath, "/") {
		readPath = readPath + "/"
	}
	fis := m.MachineFileApp.ReadDir(fid, readPath)
	fisVO := make([]vo.MachineFileInfo, 0)
	for _, fi := range fis {
		fisVO = append(fisVO, vo.MachineFileInfo{
			Name: fi.Name(),
			Size: fi.Size(),
			Path: readPath + fi.Name(),
			Type: getFileType(fi.Mode()),
		})
	}
	rc.ResData = fisVO
	rc.ReqParam = fmt.Sprintf("path: %s", readPath)
}

func (m *MachineFile) WriteFileContent(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	form := new(form.MachineFileUpdateForm)
	ginx.BindJsonAndValid(g, form)
	path := form.Path

	m.MachineFileApp.WriteFileContent(fid, path, []byte(form.Content))

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

func (m *MachineFile) UploadFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	path := g.PostForm("path")
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取文件失败: %s")

	file, _ := fileheader.Open()
	bytes, err := ioutil.ReadAll(file)
	m.MachineFileApp.UploadFile(fid, path, fileheader.Filename, bytes)

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

func (m *MachineFile) RemoveFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	// mid := GetMachineId(g)
	path := g.Query("path")

	m.MachineFileApp.RemoveFile(fid, path)

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

func getFileType(fm fs.FileMode) string {
	if fm.IsDir() {
		return dir
	}
	return file
}

func GetMachineFileId(g *gin.Context) uint64 {
	fileId, _ := strconv.Atoi(g.Param("fileId"))
	biz.IsTrue(fileId != 0, "fileId错误")
	return uint64(fileId)
}
