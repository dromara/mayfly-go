package apis

import (
	"fmt"
	"io"
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
	"os"
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

	biz.NotNil(m.MachineApp.GetById(entity.MachineId, "Id"), "机器不存在")
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
	machineId := GetMachineId(g)
	readPath := g.Query("path")
	readType := g.Query("type")

	readPath = m.checkAndReturnPath(machineId, fid, readPath)

	sftpCli := m.MachineApp.GetCli(machineId).GetSftpCli()
	// 读取文件内容
	fc, err := sftpCli.Open(readPath)
	biz.ErrIsNilAppendErr(err, "打开文件失败：%s")
	defer fc.Close()

	fileInfo, _ := fc.Stat()
	biz.IsTrue(!fileInfo.IsDir(), "该文件为目录")
	// 如果是读取文件内容，则校验文件大小
	if readType != "1" {
		biz.IsTrue(fileInfo.Size() < max_read_size, "文件超过5m，请使用下载查看")
	}
	dataByte, err := ioutil.ReadAll(fc)
	if err != nil && err != io.EOF {
		panic(biz.NewBizErr("读取文件内容失败"))
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
	machineId := GetMachineId(g)
	readPath := g.Query("path")

	readPath = m.checkAndReturnPath(machineId, fid, readPath)
	if !strings.HasSuffix(readPath, "/") {
		readPath = readPath + "/"
	}

	sftpCli := m.MachineApp.GetCli(machineId).GetSftpCli()
	fis, err := sftpCli.ReadDir(readPath)
	biz.ErrIsNilAppendErr(err, "读取目录失败: %s")
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
	machineId := GetMachineId(g)

	form := new(form.MachineFileUpdateForm)
	ginx.BindJsonAndValid(g, form)
	path := form.Path
	path = m.checkAndReturnPath(machineId, fid, path)

	sftpCli := m.MachineApp.GetCli(machineId).GetSftpCli()
	f, err := sftpCli.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_RDWR)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	defer f.Close()

	fi, _ := f.Stat()
	biz.IsTrue(!fi.IsDir(), "该路径不是文件")
	f.Write([]byte(form.Content))

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

func (m *MachineFile) UploadFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	mid := GetMachineId(g)
	path := g.PostForm("path")
	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取文件失败: %s")

	path = m.checkAndReturnPath(mid, fid, path)
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	sftpCli := m.MachineApp.GetCli(mid).GetSftpCli()
	createfile, err := sftpCli.Create(path + fileheader.Filename)
	biz.ErrIsNilAppendErr(err, "创建文件失败: %s")
	defer createfile.Close()

	file, _ := fileheader.Open()
	bytes, err := ioutil.ReadAll(file)
	createfile.Write(bytes)

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

func (m *MachineFile) RemoveFile(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	mid := GetMachineId(g)
	path := g.Query("path")

	path = m.checkAndReturnPath(mid, fid, path)

	sftpCli := m.MachineApp.GetCli(mid).GetSftpCli()
	file, err := sftpCli.Open(path)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	fi, err := file.Stat()
	if fi.IsDir() {
		err = sftpCli.RemoveDirectory(path)
	} else {
		err = sftpCli.Remove(path)
	}
	biz.ErrIsNilAppendErr(err, "删除文件失败: %s")

	rc.ReqParam = fmt.Sprintf("path: %s", path)
}

// 校验并返回实际可访问的文件path
func (m *MachineFile) checkAndReturnPath(mid, fid uint64, inputPath string) string {
	biz.IsTrue(fid != 0, "文件id不能为空")
	mf := m.MachineFileApp.GetById(uint64(fid))
	biz.NotNil(mf, "文件不存在")
	biz.IsEquals(mid, mf.MachineId, "机器id与文件id不匹配")
	if inputPath != "" {
		biz.IsTrue(strings.HasPrefix(inputPath, mf.Path), "无权访问该目录或文件")
		return inputPath
	} else {
		return mf.Path
	}
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
