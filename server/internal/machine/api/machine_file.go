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
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
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
	machineFileApp application.MachineFile `inject:"T"`
	msgApp         msgapp.Msg              `inject:"T"`
}

func (mf *MachineFile) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取指定机器文件列表
		req.NewGet(":machineId/files", mf.MachineFiles),

		req.NewPost(":machineId/files", mf.SaveMachineFiles).Log(req.NewLogSaveI(imsg.LogMachineFileConfSave)).RequiredPermissionCode("machine:file:add"),

		req.NewDelete(":machineId/files/:fileId", mf.DeleteFile).Log(req.NewLogSaveI(imsg.LogMachineFileConfDelete)).RequiredPermissionCode("machine:file:del"),

		req.NewGet(":machineId/files/:fileId/read", mf.ReadFileContent).Log(req.NewLogSaveI(imsg.LogMachineFileRead)),

		req.NewGet(":machineId/files/:fileId/download", mf.DownloadFile).NoRes().Log(req.NewLogSaveI(imsg.LogMachineFileDownload)),

		req.NewGet(":machineId/files/:fileId/read-dir", mf.GetDirEntry),

		req.NewGet(":machineId/files/:fileId/dir-size", mf.GetDirSize),

		req.NewGet(":machineId/files/:fileId/file-stat", mf.GetFileStat),

		req.NewPost(":machineId/files/:fileId/write", mf.WriteFileContent).Log(req.NewLogSaveI(imsg.LogMachineFileModify)).RequiredPermissionCode("machine:file:write"),

		req.NewPost(":machineId/files/:fileId/create-file", mf.CreateFile).Log(req.NewLogSaveI(imsg.LogMachineFileCreate)),

		req.NewPost(":machineId/files/:fileId/upload", mf.UploadFile).Log(req.NewLogSaveI(imsg.LogMachineFileUpload)).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/upload-folder", mf.UploadFolder).Log(req.NewLogSaveI(imsg.LogMachineFileUploadFolder)).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/remove", mf.RemoveFile).Log(req.NewLogSaveI(imsg.LogMachineFileDelete)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/cp", mf.CopyFile).Log(req.NewLogSaveI(imsg.LogMachineFileCopy)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/mv", mf.MvFile).Log(req.NewLogSaveI(imsg.LogMachineFileMove)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/rename", mf.Rename).Log(req.NewLogSaveI(imsg.LogMachineFileRename)).RequiredPermissionCode("machine:file:write"),
	}

	return req.NewConfs("machines", reqs[:]...)
}

const (
	file          = "-"
	dir           = "d"
	link          = "l"
	max_read_size = 1 * 1024 * 1024
)

func (m *MachineFile) MachineFiles(rc *req.Ctx) {
	condition := &entity.MachineFile{MachineId: GetMachineId(rc)}
	res, err := m.machineFileApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.MachineFile, *vo.MachineFileVO](res)
}

func (m *MachineFile) SaveMachineFiles(rc *req.Ctx) {
	fileForm, entity := req.BindJsonAndCopyTo[*form.MachineFileForm, *entity.MachineFile](rc)

	rc.ReqParam = fileForm
	biz.ErrIsNil(m.machineFileApp.Save(rc.MetaCtx, entity))
}

func (m *MachineFile) DeleteFile(rc *req.Ctx) {
	biz.ErrIsNil(m.machineFileApp.DeleteById(rc.MetaCtx, GetMachineFileId(rc)))
}

/***      sftp相关操作      */

func (m *MachineFile) CreateFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid[*form.CreateFileForm](rc)
	path := opForm.Path

	attrs := collx.Kvs("path", path)
	var mi *mcm.MachineInfo
	var err error
	if opForm.Type == dir {
		attrs["type"] = "Folder"
		mi, err = m.machineFileApp.MkDir(rc.MetaCtx, opForm.MachineFileOp)
	} else {
		attrs["type"] = "File"
		mi, err = m.machineFileApp.CreateFile(rc.MetaCtx, opForm.MachineFileOp)
	}
	attrs["machine"] = mi
	rc.ReqParam = attrs
	biz.ErrIsNil(err)
}

func (m *MachineFile) ReadFileContent(rc *req.Ctx) {
	opForm := req.BindQuery[*dto.MachineFileOp](rc)
	readPath := opForm.Path
	ctx := rc.MetaCtx

	// 特殊处理rdp文件
	if opForm.Protocol == entity.MachineProtocolRdp {
		path := m.machineFileApp.GetRdpFilePath(rc.GetLoginAccount(), opForm.Path)
		fi, err := os.Stat(path)
		biz.ErrIsNil(err)
		biz.IsTrueI(ctx, fi.Size() < max_read_size, imsg.ErrFileTooLargeUseDownload)
		datas, err := os.ReadFile(path)
		biz.ErrIsNil(err)
		rc.ResData = string(datas)
		return
	}

	sftpFile, mi, err := m.machineFileApp.ReadFile(rc.MetaCtx, opForm)
	rc.ReqParam = collx.Kvs("machine", mi, "path", readPath)
	biz.ErrIsNil(err)
	defer sftpFile.Close()

	fileInfo, _ := sftpFile.Stat()
	filesize := fileInfo.Size()
	biz.IsTrueI(ctx, filesize < max_read_size, imsg.ErrFileTooLargeUseDownload)

	datas, err := io.ReadAll(sftpFile)
	biz.ErrIsNil(err)

	rc.ResData = string(datas)
}

func (m *MachineFile) DownloadFile(rc *req.Ctx) {
	opForm := req.BindQuery[*dto.MachineFileOp](rc)

	readPath := opForm.Path

	// 截取文件名，如/usr/local/test.java -》 test.java
	path := strings.Split(readPath, "/")
	fileName := path[len(path)-1]

	if opForm.Protocol == entity.MachineProtocolRdp {
		path := m.machineFileApp.GetRdpFilePath(rc.GetLoginAccount(), opForm.Path)
		file, err := os.Open(path)
		if err != nil {
			return
		}
		defer file.Close()
		rc.Download(file, fileName)
		return
	}

	sftpFile, mi, err := m.machineFileApp.ReadFile(rc.MetaCtx, opForm)
	rc.ReqParam = collx.Kvs("machine", mi, "path", readPath)
	biz.ErrIsNilAppendErr(err, "open file error: %s")
	defer sftpFile.Close()

	rc.Download(sftpFile, fileName)
}

func (m *MachineFile) GetDirEntry(rc *req.Ctx) {
	opForm := req.BindQuery[*dto.MachineFileOp](rc)
	readPath := opForm.Path
	rc.ReqParam = fmt.Sprintf("path: %s", readPath)

	fis, err := m.machineFileApp.ReadDir(rc.MetaCtx, opForm)
	biz.ErrIsNilAppendErr(err, "read dir error: %s")

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
	opForm := req.BindQuery[*dto.MachineFileOp](rc)

	size, err := m.machineFileApp.GetDirSize(rc.MetaCtx, opForm)
	biz.ErrIsNil(err)
	rc.ResData = size
}

func (m *MachineFile) GetFileStat(rc *req.Ctx) {
	opForm := req.BindQuery[*dto.MachineFileOp](rc)
	res, err := m.machineFileApp.FileStat(rc.MetaCtx, opForm)
	biz.ErrIsNil(err, res)
	rc.ResData = res
}

func (m *MachineFile) WriteFileContent(rc *req.Ctx) {
	opForm := req.BindJsonAndValid[*form.WriteFileContentForm](rc)
	path := opForm.Path

	mi, err := m.machineFileApp.WriteFileContent(rc.MetaCtx, opForm.MachineFileOp, []byte(opForm.Content))
	rc.ReqParam = collx.Kvs("machine", mi, "path", path)
	biz.ErrIsNilAppendErr(err, "open file error: %s")
}

func (m *MachineFile) UploadFile(rc *req.Ctx) {
	path := rc.PostForm("path")
	protocol := cast.ToInt(rc.PostForm("protocol"))
	machineId := cast.ToUint64(rc.PostForm("machineId"))
	authCertName := rc.PostForm("authCertName")

	fileheader, err := rc.FormFile("file")
	biz.ErrIsNilAppendErr(err, "read form file error: %s")

	ctx := rc.MetaCtx

	maxUploadFileSize := config.GetMachine().UploadMaxFileSize
	biz.IsTrueI(ctx, fileheader.Size <= maxUploadFileSize, imsg.ErrUploadFileOutOfLimit, "size", maxUploadFileSize)

	file, _ := fileheader.Open()
	defer file.Close()

	la := rc.GetLoginAccount()
	defer func() {
		if anyx.ToString(recover()) != "" {
			logx.Errorf("upload file error: %s", err)
			m.msgApp.CreateAndSend(la, msgdto.ErrSysMsg(i18n.TC(ctx, imsg.ErrFileUploadFail), fmt.Sprintf("%s: \n<-e : %s", i18n.TC(ctx, imsg.ErrFileUploadFail), err)))
		}
	}()

	opForm := &dto.MachineFileOp{
		MachineId:    machineId,
		AuthCertName: authCertName,
		Protocol:     protocol,
		Path:         path,
	}

	mi, err := m.machineFileApp.UploadFile(ctx, opForm, fileheader.Filename, file)
	rc.ReqParam = collx.Kvs("machine", mi, "path", fmt.Sprintf("%s/%s", path, fileheader.Filename))
	biz.ErrIsNilAppendErr(err, "upload file error: %s")
	// 保存消息并发送文件上传成功通知
	m.msgApp.CreateAndSend(la, msgdto.SuccessSysMsg(i18n.TC(ctx, imsg.MsgUploadFileSuccess), fmt.Sprintf("[%s] -> %s[%s:%s]", fileheader.Filename, mi.Name, mi.Ip, path)))
}

type FolderFile struct {
	Dir        string
	Fileheader *multipart.FileHeader
}

func (m *MachineFile) UploadFolder(rc *req.Ctx) {
	mf, err := rc.MultipartForm()
	biz.ErrIsNilAppendErr(err, "get multipart form error: %s")
	basePath := mf.Value["basePath"][0]
	biz.NotEmpty(basePath, "basePath cannot be empty")

	fileheaders := mf.File["files"]
	biz.IsTrue(len(fileheaders) > 0, "files cannot be empty")
	allFileSize := collx.ArrayReduce(fileheaders, 0, func(i int64, fh *multipart.FileHeader) int64 {
		return i + fh.Size
	})

	ctx := rc.MetaCtx
	maxUploadFileSize := config.GetMachine().UploadMaxFileSize
	biz.IsTrueI(ctx, allFileSize <= maxUploadFileSize, imsg.ErrUploadFileOutOfLimit, "size", maxUploadFileSize)

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
		m.machineFileApp.UploadFiles(ctx, opForm, basePath, fileheaders, paths)
		return
	}

	folderName := filepath.Dir(paths[0])
	mcli, err := m.machineFileApp.GetMachineCli(rc.MetaCtx, authCertName)
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
			biz.ErrIsNilAppendErr(sftpCli.MkdirAll(basePath+"/"+dir), "create dir error: %s")
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
					logx.Errorf("upload file error: %s", err)
					switch t := err.(type) {
					case *errorx.BizError:
						m.msgApp.CreateAndSend(la, msgdto.ErrSysMsg(i18n.TC(ctx, imsg.ErrFileUploadFail), fmt.Sprintf("%s: \n<-e errCode: %d, errMsg: %s", i18n.TC(ctx, imsg.ErrFileUploadFail), t.Code(), t.Error())))
					}
				}
			}()

			for _, file := range files {
				fileHeader := file.Fileheader
				dir := file.Dir
				file, _ := fileHeader.Open()
				defer file.Close()

				logx.Debugf("upload folder: dir=%s -> filename=%s", dir, fileHeader.Filename)

				createfile, err := sftpCli.Create(fmt.Sprintf("%s/%s/%s", basePath, dir, fileHeader.Filename))
				biz.ErrIsNilAppendErr(err, "create file error: %s")
				defer createfile.Close()
				io.Copy(createfile, file)
			}
		}(chunk, &wg)
	}

	// 等待所有协程执行完成
	wg.Wait()
	if isSuccess {
		// 保存消息并发送文件上传成功通知
		m.msgApp.CreateAndSend(la, msgdto.SuccessSysMsg(i18n.TC(ctx, imsg.MsgUploadFileSuccess), fmt.Sprintf("[%s] -> %s[%s:%s]", folderName, mi.Name, mi.Ip, basePath)))
	}
}

func (m *MachineFile) RemoveFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid[*form.RemoveFileForm](rc)

	mi, err := m.machineFileApp.RemoveFile(rc.MetaCtx, opForm.MachineFileOp, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "path", opForm)
	biz.ErrIsNilAppendErr(err, "remove file error: %s")
}

func (m *MachineFile) CopyFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid[*form.CopyFileForm](rc)
	mi, err := m.machineFileApp.Copy(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	biz.ErrIsNilAppendErr(err, "file copy error: %s")
	rc.ReqParam = collx.Kvs("machine", mi, "cp", opForm)
}

func (m *MachineFile) MvFile(rc *req.Ctx) {
	opForm := req.BindJsonAndValid[*form.CopyFileForm](rc)
	mi, err := m.machineFileApp.Mv(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "mv", opForm)
	biz.ErrIsNilAppendErr(err, "file move error: %s")
}

func (m *MachineFile) Rename(rc *req.Ctx) {
	renameForm := req.BindJsonAndValid[*form.RenameForm](rc)
	mi, err := m.machineFileApp.Rename(rc.MetaCtx, renameForm.MachineFileOp, renameForm.Newname)
	rc.ReqParam = collx.Kvs("machine", mi, "rename", renameForm)
	biz.ErrIsNilAppendErr(err, "file rename error: %s")
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
	biz.IsTrue(fileId != 0, "fileId error")
	return uint64(fileId)
}
