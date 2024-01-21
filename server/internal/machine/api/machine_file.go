package api

import (
	"fmt"
	"io"
	"io/fs"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/mcm"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/timex"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type MachineFile struct {
	MachineFileApp application.MachineFile `inject:""`
	MsgApp         msgapp.Msg              `inject:""`
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
	res, err := m.MachineFileApp.GetPageList(condition, ginx.GetPageParam(g), new([]vo.MachineFileVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MachineFile) SaveMachineFiles(rc *req.Ctx) {
	fileForm := new(form.MachineFileForm)
	entity := ginx.BindJsonAndCopyTo[*entity.MachineFile](rc.GinCtx, fileForm, new(entity.MachineFile))

	rc.ReqParam = fileForm
	biz.ErrIsNil(m.MachineFileApp.Save(rc.MetaCtx, entity))
}

func (m *MachineFile) DeleteFile(rc *req.Ctx) {
	biz.ErrIsNil(m.MachineFileApp.DeleteById(rc.MetaCtx, GetMachineFileId(rc.GinCtx)))
}

/***      sftp相关操作      */

func (m *MachineFile) CreateFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	form := ginx.BindJsonAndValid(g, new(form.MachineCreateFileForm))
	path := form.Path

	attrs := collx.Kvs("path", path)
	var mi *mcm.MachineInfo
	var err error
	if form.Type == dir {
		attrs["type"] = "目录"
		mi, err = m.MachineFileApp.MkDir(fid, form.Path)
	} else {
		attrs["type"] = "文件"
		mi, err = m.MachineFileApp.CreateFile(fid, form.Path)
	}
	attrs["machine"] = mi
	rc.ReqParam = attrs
	biz.ErrIsNilAppendErr(err, "创建目录失败: %s")
}

func (m *MachineFile) ReadFileContent(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	sftpFile, mi, err := m.MachineFileApp.ReadFile(fid, readPath)
	rc.ReqParam = collx.Kvs("machine", mi, "path", readPath)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	defer sftpFile.Close()

	fileInfo, _ := sftpFile.Stat()
	filesize := fileInfo.Size()

	biz.IsTrue(filesize < max_read_size, "文件超过1m，请使用下载查看")
	datas, err := io.ReadAll(sftpFile)
	biz.ErrIsNilAppendErr(err, "读取文件内容失败: %s")

	rc.ResData = string(datas)
}

func (m *MachineFile) DownloadFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	sftpFile, mi, err := m.MachineFileApp.ReadFile(fid, readPath)
	rc.ReqParam = collx.Kvs("machine", mi, "path", readPath)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	defer sftpFile.Close()

	// 截取文件名，如/usr/local/test.java -》 test.java
	path := strings.Split(readPath, "/")
	rc.Download(sftpFile, path[len(path)-1])
}

func (m *MachineFile) GetDirEntry(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")
	rc.ReqParam = fmt.Sprintf("path: %s", readPath)

	if !strings.HasSuffix(readPath, "/") {
		readPath = readPath + "/"
	}
	fis, err := m.MachineFileApp.ReadDir(fid, readPath)
	biz.ErrIsNilAppendErr(err, "读取目录失败: %s")

	fisVO := make([]vo.MachineFileInfo, 0)
	for _, fi := range fis {
		fisVO = append(fisVO, vo.MachineFileInfo{
			Name:    fi.Name(),
			Size:    fi.Size(),
			Path:    readPath + fi.Name(),
			Type:    getFileType(fi.Mode()),
			Mode:    fi.Mode().String(),
			ModTime: timex.DefaultFormat(fi.ModTime()),
		})

	}
	sort.Sort(vo.MachineFileInfos(fisVO))
	rc.ResData = fisVO
}

func (m *MachineFile) GetDirSize(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	size, err := m.MachineFileApp.GetDirSize(fid, readPath)
	biz.ErrIsNil(err)
	rc.ResData = size
}

func (m *MachineFile) GetFileStat(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	readPath := g.Query("path")

	res, err := m.MachineFileApp.FileStat(fid, readPath)
	biz.ErrIsNil(err, res)
	rc.ResData = res
}

func (m *MachineFile) WriteFileContent(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	form := new(form.MachineFileUpdateForm)
	ginx.BindJsonAndValid(g, form)
	path := form.Path

	mi, err := m.MachineFileApp.WriteFileContent(fid, path, []byte(form.Content))
	rc.ReqParam = collx.Kvs("machine", mi, "path", path)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
}

func (m *MachineFile) UploadFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)
	path := g.PostForm("path")

	fileheader, err := g.FormFile("file")
	biz.ErrIsNilAppendErr(err, "读取文件失败: %s")

	maxUploadFileSize := config.GetMachine().UploadMaxFileSize
	biz.IsTrue(fileheader.Size <= maxUploadFileSize, "文件大小不能超过%d字节", maxUploadFileSize)

	file, _ := fileheader.Open()
	defer file.Close()

	la := rc.GetLoginAccount()
	defer func() {
		if anyx.ToString(recover()) != "" {
			logx.Errorf("文件上传失败: %s", err)
			m.MsgApp.CreateAndSend(la, msgdto.ErrSysMsg("文件上传失败", fmt.Sprintf("执行文件上传失败：\n<-e : %s", err)))
		}
	}()

	mi, err := m.MachineFileApp.UploadFile(fid, path, fileheader.Filename, file)
	rc.ReqParam = collx.Kvs("machine", mi, "path", fmt.Sprintf("%s/%s", path, fileheader.Filename))
	biz.ErrIsNilAppendErr(err, "创建文件失败: %s")
	// 保存消息并发送文件上传成功通知
	m.MsgApp.CreateAndSend(la, msgdto.SuccessSysMsg("文件上传成功", fmt.Sprintf("[%s]文件已成功上传至 %s[%s:%s]", fileheader.Filename, mi.Name, mi.Ip, path)))
}

type FolderFile struct {
	Dir        string
	Fileheader *multipart.FileHeader
}

func (m *MachineFile) UploadFolder(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	mf, err := g.MultipartForm()
	biz.ErrIsNilAppendErr(err, "获取表单信息失败: %s")
	basePath := mf.Value["basePath"][0]
	biz.NotEmpty(basePath, "基础路径不能为空")

	fileheaders := mf.File["files"]
	biz.IsTrue(len(fileheaders) > 0, "文件不能为空")
	allFileSize := collx.ArrayReduce(fileheaders, 0, func(i int64, fh *multipart.FileHeader) int64 {
		return i + fh.Size
	})

	maxUploadFileSize := config.GetMachine().UploadMaxFileSize
	biz.IsTrue(allFileSize <= maxUploadFileSize, "文件夹总大小不能超过%d字节", maxUploadFileSize)

	paths := mf.Value["paths"]

	folderName := filepath.Dir(paths[0])
	mcli, err := m.MachineFileApp.GetMachineCli(fid, basePath+"/"+folderName)
	biz.ErrIsNil(err)
	mi := mcli.Info

	sftpCli, err := mcli.GetSftpCli()
	biz.ErrIsNil(err)
	rc.ReqParam = collx.Kvs("machine", mi, "path", fmt.Sprintf("%s/%s", basePath, folderName))

	folderFiles := make([]FolderFile, len(paths))
	// 先创建目录，并将其包装为folderFile结构
	mkdirs := make(map[string]bool, 0)
	for i, path := range paths {
		dir := filepath.Dir(path)
		// 目录已建，则无需重复建
		if !mkdirs[dir] {
			biz.ErrIsNilAppendErr(sftpCli.MkdirAll(basePath+"/"+dir), "创建目录失败: %s")
			mkdirs[dir] = true
		}
		folderFiles[i] = FolderFile{
			Dir:        dir,
			Fileheader: fileheaders[i],
		}
	}

	// 分组处理
	groupNum := 30
	chunks := collx.ArraySplit(folderFiles, groupNum)

	var wg sync.WaitGroup
	// 设置要等待的协程数量
	wg.Add(len(chunks))

	isSuccess := true
	la := rc.GetLoginAccount()
	for _, chunk := range chunks {
		go func(files []FolderFile, wg *sync.WaitGroup) {
			defer func() {
				// 协程执行完成后调用Done方法
				wg.Done()
				if err := recover(); err != nil {
					isSuccess = false
					logx.Errorf("文件上传失败: %s", err)
					switch t := err.(type) {
					case errorx.BizError:
						m.MsgApp.CreateAndSend(la, msgdto.ErrSysMsg("文件上传失败", fmt.Sprintf("执行文件上传失败：\n<-e errCode: %d, errMsg: %s", t.Code(), t.Error())))
					}
				}
			}()

			for _, file := range files {
				fileHeader := file.Fileheader
				dir := file.Dir
				file, _ := fileHeader.Open()
				defer file.Close()

				logx.Debugf("上传文件夹: dir=%s -> filename=%s", dir, fileHeader.Filename)

				createfile, err := sftpCli.Create(fmt.Sprintf("%s/%s/%s", basePath, dir, fileHeader.Filename))
				biz.ErrIsNilAppendErr(err, "创建文件失败: %s")
				defer createfile.Close()
				io.Copy(createfile, file)
			}
		}(chunk, &wg)
	}

	// 等待所有协程执行完成
	wg.Wait()
	if isSuccess {
		// 保存消息并发送文件上传成功通知
		m.MsgApp.CreateAndSend(la, msgdto.SuccessSysMsg("文件上传成功", fmt.Sprintf("[%s]文件夹已成功上传至 %s[%s:%s]", folderName, mi.Name, mi.Ip, basePath)))
	}
}

func (m *MachineFile) RemoveFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	rmForm := new(form.MachineFileOpForm)
	ginx.BindJsonAndValid(g, rmForm)

	mi, err := m.MachineFileApp.RemoveFile(fid, rmForm.Path...)
	rc.ReqParam = collx.Kvs("machine", mi, "path", rmForm.Path)
	biz.ErrIsNilAppendErr(err, "删除文件失败: %s")
}

func (m *MachineFile) CopyFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	cpForm := new(form.MachineFileOpForm)
	ginx.BindJsonAndValid(g, cpForm)
	mi, err := m.MachineFileApp.Copy(fid, cpForm.ToPath, cpForm.Path...)
	biz.ErrIsNilAppendErr(err, "文件拷贝失败: %s")
	rc.ReqParam = collx.Kvs("machine", mi, "cp", cpForm)
}

func (m *MachineFile) MvFile(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	cpForm := new(form.MachineFileOpForm)
	ginx.BindJsonAndValid(g, cpForm)
	mi, err := m.MachineFileApp.Mv(fid, cpForm.ToPath, cpForm.Path...)
	rc.ReqParam = collx.Kvs("machine", mi, "mv", cpForm)
	biz.ErrIsNilAppendErr(err, "文件移动失败: %s")
}

func (m *MachineFile) Rename(rc *req.Ctx) {
	g := rc.GinCtx
	fid := GetMachineFileId(g)

	rename := new(form.MachineFileRename)
	ginx.BindJsonAndValid(g, rename)
	mi, err := m.MachineFileApp.Rename(fid, rename.Oldname, rename.Newname)
	rc.ReqParam = collx.Kvs("machine", mi, "rename", rename)
	biz.ErrIsNilAppendErr(err, "文件重命名失败: %s")
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
