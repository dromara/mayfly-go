package api

import (
	"fmt"
	"io"
	"io/fs"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/mcm"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/timex"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/may-fly/cast"
	"github.com/pkg/sftp"
)

type MachineFile struct {
	MachineApp     application.Machine     `inject:""`
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
	condition := &entity.MachineFile{MachineId: GetMachineId(rc)}
	res, err := m.MachineFileApp.GetPageList(condition, rc.GetPageParam(), new([]vo.MachineFileVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MachineFile) SaveMachineFiles(rc *req.Ctx) {
	fileForm := new(form.MachineFileForm)
	entity := req.BindJsonAndCopyTo[*entity.MachineFile](rc, fileForm, new(entity.MachineFile))

	rc.ReqParam = fileForm
	biz.ErrIsNil(m.MachineFileApp.Save(rc.MetaCtx, entity))
}

func (m *MachineFile) DeleteFile(rc *req.Ctx) {
	biz.ErrIsNil(m.MachineFileApp.DeleteById(rc.MetaCtx, GetMachineFileId(rc)))
}

/***      sftp相关操作      */

func (m *MachineFile) CreateFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid(rc, new(form.CreateFileForm))
	path := opForm.Path

	attrs := collx.Kvs("path", path)
	var mi *mcm.MachineInfo
	var err error
	if opForm.Type == dir {
		attrs["type"] = "目录"
		mi, err = m.MachineFileApp.MkDir(rc.MetaCtx, opForm.MachineFileOp)
	} else {
		attrs["type"] = "文件"
		mi, err = m.MachineFileApp.CreateFile(rc.MetaCtx, opForm.MachineFileOp)
	}
	attrs["machine"] = mi
	rc.ReqParam = attrs
	biz.ErrIsNilAppendErr(err, "创建目录失败: %s")
}

func (m *MachineFile) ReadFileContent(rc *req.Ctx) {
	opForm := req.BindQuery(rc, new(dto.MachineFileOp))
	readPath := opForm.Path
	// 特殊处理rdp文件
	if opForm.Protocol == entity.MachineProtocolRdp {
		path := m.MachineFileApp.GetRdpFilePath(rc.GetLoginAccount(), opForm.Path)
		fi, err := os.Stat(path)
		biz.ErrIsNilAppendErr(err, "读取文件内容失败: %s")
		biz.IsTrue(fi.Size() < max_read_size, "文件超过1m，请使用下载查看")
		datas, err := os.ReadFile(path)
		biz.ErrIsNilAppendErr(err, "读取文件内容失败: %s")
		rc.ResData = string(datas)
		return
	}

	sftpFile, mi, err := m.MachineFileApp.ReadFile(rc.MetaCtx, opForm)
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
	opForm := req.BindQuery(rc, new(dto.MachineFileOp))

	readPath := opForm.Path

	// 截取文件名，如/usr/local/test.java -》 test.java
	path := strings.Split(readPath, "/")
	fileName := path[len(path)-1]

	if opForm.Protocol == entity.MachineProtocolRdp {
		path := m.MachineFileApp.GetRdpFilePath(rc.GetLoginAccount(), opForm.Path)
		file, err := os.Open(path)
		if err != nil {
			return
		}
		defer file.Close()
		rc.Download(file, fileName)
		return
	}

	sftpFile, mi, err := m.MachineFileApp.ReadFile(rc.MetaCtx, opForm)
	rc.ReqParam = collx.Kvs("machine", mi, "path", readPath)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	defer sftpFile.Close()

	rc.Download(sftpFile, fileName)
}

func (m *MachineFile) GetDirEntry(rc *req.Ctx) {
	opForm := req.BindQuery(rc, new(dto.MachineFileOp))
	readPath := opForm.Path
	rc.ReqParam = fmt.Sprintf("path: %s", readPath)

	fis, err := m.MachineFileApp.ReadDir(rc.MetaCtx, opForm)
	biz.ErrIsNilAppendErr(err, "读取目录失败: %s")

	fisVO := make([]vo.MachineFileInfo, 0)
	for _, fi := range fis {
		name := fi.Name()
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		path := name
		if readPath != "/" && readPath != "" {
			path = readPath + name
		}

		mfi := vo.MachineFileInfo{
			Name:    fi.Name(),
			Size:    fi.Size(),
			Path:    path,
			Type:    getFileType(fi.Mode()),
			Mode:    fi.Mode().String(),
			ModTime: timex.DefaultFormat(fi.ModTime()),
		}

		if sftpFs, ok := fi.Sys().(*sftp.FileStat); ok {
			mfi.UID = sftpFs.UID
			mfi.GID = sftpFs.GID
		}

		fisVO = append(fisVO, mfi)
	}
	sort.Sort(vo.MachineFileInfos(fisVO))
	rc.ResData = fisVO
}

func (m *MachineFile) GetDirSize(rc *req.Ctx) {
	opForm := req.BindQuery(rc, new(dto.MachineFileOp))

	size, err := m.MachineFileApp.GetDirSize(rc.MetaCtx, opForm)
	biz.ErrIsNil(err)
	rc.ResData = size
}

func (m *MachineFile) GetFileStat(rc *req.Ctx) {
	opForm := req.BindQuery(rc, new(dto.MachineFileOp))
	res, err := m.MachineFileApp.FileStat(rc.MetaCtx, opForm)
	biz.ErrIsNil(err, res)
	rc.ResData = res
}

func (m *MachineFile) WriteFileContent(rc *req.Ctx) {
	opForm := req.BindJsonAndValid(rc, new(form.WriteFileContentForm))
	path := opForm.Path

	mi, err := m.MachineFileApp.WriteFileContent(rc.MetaCtx, opForm.MachineFileOp, []byte(opForm.Content))
	rc.ReqParam = collx.Kvs("machine", mi, "path", path)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
}

func (m *MachineFile) UploadFile(rc *req.Ctx) {
	path := rc.PostForm("path")
	protocol := cast.ToInt(rc.PostForm("protocol"))
	machineId := cast.ToUint64(rc.PostForm("machineId"))
	authCertName := rc.PostForm("authCertName")

	fileheader, err := rc.FormFile("file")
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

	opForm := &dto.MachineFileOp{
		MachineId:    machineId,
		AuthCertName: authCertName,
		Protocol:     protocol,
		Path:         path,
	}

	mi, err := m.MachineFileApp.UploadFile(rc.MetaCtx, opForm, fileheader.Filename, file)
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
	mf, err := rc.MultipartForm()
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
	authCertName := mf.Value["authCertName"][0]
	machineId := cast.ToUint64(mf.Value["machineId"][0])
	// protocol
	protocol := cast.ToInt(mf.Value["protocol"][0])

	opForm := &dto.MachineFileOp{
		MachineId:    machineId,
		Protocol:     protocol,
		AuthCertName: authCertName,
	}

	if protocol == entity.MachineProtocolRdp {
		m.MachineFileApp.UploadFiles(rc.MetaCtx, opForm, basePath, fileheaders, paths)
		return
	}

	folderName := filepath.Dir(paths[0])
	mcli, err := m.MachineFileApp.GetMachineCli(authCertName)
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
	opForm := req.BindJsonAndValid(rc, new(form.RemoveFileForm))

	mi, err := m.MachineFileApp.RemoveFile(rc.MetaCtx, opForm.MachineFileOp, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "path", opForm)
	biz.ErrIsNilAppendErr(err, "删除文件失败: %s")
}

func (m *MachineFile) CopyFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid(rc, new(form.CopyFileForm))
	mi, err := m.MachineFileApp.Copy(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	biz.ErrIsNilAppendErr(err, "文件拷贝失败: %s")
	rc.ReqParam = collx.Kvs("machine", mi, "cp", opForm)
}

func (m *MachineFile) MvFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid(rc, new(form.CopyFileForm))
	mi, err := m.MachineFileApp.Mv(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "mv", opForm)
	biz.ErrIsNilAppendErr(err, "文件移动失败: %s")
}

func (m *MachineFile) Rename(rc *req.Ctx) {
	renameForm := req.BindJsonAndValid(rc, new(form.RenameForm))
	mi, err := m.MachineFileApp.Rename(rc.MetaCtx, renameForm.MachineFileOp, renameForm.Newname)
	rc.ReqParam = collx.Kvs("machine", mi, "rename", renameForm)
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

func GetMachineFileId(rc *req.Ctx) uint64 {
	fileId := rc.PathParamInt("fileId")
	biz.IsTrue(fileId != 0, "fileId错误")
	return uint64(fileId)
}
