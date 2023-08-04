package api

import (
	"fmt"
	"io"
	"io/fs"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/ws"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type MachineFile struct {
	MachineFileApp application.MachineFile
	MsgApp         msgapp.Msg
}

const (
	file          = "-"
	dir           = "d"
	link          = "l"
	max_read_size = 1 * 1024 * 1024
)

func (m *MachineFile) MachineFiles(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.MachineFile{MachineId: GetMachineId(g)}
	rc.ResData = m.MachineFileApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineFileVO))
}

func (m *MachineFile) SaveMachineFiles(rc *req.Ctx) {
	fileForm := new(form.MachineFileForm)
	entity := ginx.BindJsonAndCopyTo[*entity.MachineFile](rc.GinCtx, fileForm, new(entity.MachineFile))
	entity.SetBaseInfo(rc.LoginAccount)

	rc.ReqParam = fileForm
	m.MachineFileApp.Save(entity)
}

func (m *MachineFile) DeleteFile(rc *req.Ctx) {
	m.MachineFileApp.Delete(GetMachineFileId(rc.GinCtx))
}

/***      sftp相关操作      */

func (m *MachineFile) CreateFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	form := ginx.BindJsonAndValid(g, new(form.MachineCreateFileForm))
	path := form.Path

	mi := m.MachineFileApp.GetMachine(fid)
	if form.Type == dir {
		m.MachineFileApp.MkDir(fid, form.Path)
		rc.ReqParam = fmt.Sprintf("%s -> 创建目录: %s", mi.GetLogDesc(), path)
	} else {
		m.MachineFileApp.CreateFile(fid, form.Path)
		rc.ReqParam = fmt.Sprintf("%s -> 创建文件: %s", mi.GetLogDesc(), path)
	}
}

func (m *MachineFile) ReadFileContent(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")
	readType := g.Query("type")

	sftpFile := m.MachineFileApp.ReadFile(fid, readPath)
	defer sftpFile.Close()

	fileInfo, _ := sftpFile.Stat()
	// 如果是读取文件内容，则校验文件大小
	if readType != "1" {
		biz.IsTrue(fileInfo.Size() < max_read_size, "文件超过1m，请使用下载查看")
	}

	mi := m.MachineFileApp.GetMachine(fid)
	// 如果读取类型为下载，则下载文件，否则获取文件内容
	if readType == "1" {
		// 截取文件名，如/usr/local/test.java -》 test.java
		path := strings.Split(readPath, "/")
		rc.Download(sftpFile, path[len(path)-1])
		rc.ReqParam = fmt.Sprintf("%s -> 下载文件: %s", mi.GetLogDesc(), readPath)
	} else {
		datas, err := io.ReadAll(sftpFile)
		biz.ErrIsNilAppendErr(err, "读取文件内容失败: %s")
		rc.ResData = string(datas)
		rc.ReqParam = fmt.Sprintf("%s -> 查看文件: %s", mi.GetLogDesc(), readPath)
	}
}

func (m *MachineFile) GetDirEntry(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")
	rc.ReqParam = fmt.Sprintf("path: %s", readPath)

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
	sort.Sort(vo.MachineFileInfos(fisVO))
	rc.ResData = fisVO
}

func (m *MachineFile) GetDirSize(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	rc.ResData = m.MachineFileApp.GetDirSize(fid, readPath)
}

func (m *MachineFile) GetFileStat(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	rc.ResData = m.MachineFileApp.FileStat(fid, readPath)
}

func (m *MachineFile) WriteFileContent(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	form := new(form.MachineFileUpdateForm)
	ginx.BindJsonAndValid(g, form)
	path := form.Path

	m.MachineFileApp.WriteFileContent(fid, path, []byte(form.Content))

	mi := m.MachineFileApp.GetMachine(fid)
	rc.ReqParam = fmt.Sprintf("%s -> 修改文件内容: %s", mi.GetLogDesc(), path)
}

func (m *MachineFile) UploadFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	path := g.PostForm("path")

	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取文件失败: %s")

	file, _ := fileheader.Open()
	rc.ReqParam = fmt.Sprintf("path: %s", path)

	mi := m.MachineFileApp.GetMachine(fid)
	rc.ReqParam = fmt.Sprintf("%s -> 上传文件: %s/%s", mi.GetLogDesc(), path, fileheader.Filename)

	la := rc.LoginAccount
	go func() {
		defer func() {
			if err := recover(); err != nil {
				switch t := err.(type) {
				case *biz.BizError:
					m.MsgApp.CreateAndSend(la, ws.ErrMsg("文件上传失败", fmt.Sprintf("执行文件上传失败：\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())))
				}
			}
		}()
		defer file.Close()
		m.MachineFileApp.UploadFile(fid, path, fileheader.Filename, file)
		// 保存消息并发送文件上传成功通知
		m.MsgApp.CreateAndSend(la, ws.SuccessMsg("文件上传成功", fmt.Sprintf("[%s]文件已成功上传至 %s[%s:%s]", fileheader.Filename, mi.Name, mi.Ip, path)))
	}()
}

func (m *MachineFile) RemoveFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	path := g.Query("path")

	m.MachineFileApp.RemoveFile(fid, path)

	mi := m.MachineFileApp.GetMachine(fid)
	rc.ReqParam = fmt.Sprintf("%s -> 删除文件: %s", mi.GetLogDesc(), path)
}

func getFileType(fm fs.FileMode) string {
	if fm.IsDir() {
		return dir
	}
	if fm.IsRegular() {
		return file
	}
	return dir
}

func GetMachineFileId(g *gin.Context) uint64 {
	fileId, _ := strconv.Atoi(g.Param("fileId"))
	biz.IsTrue(fileId != 0, "fileId错误")
	return uint64(fileId)
}
