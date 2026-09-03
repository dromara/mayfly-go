package api

import (
	"context"
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
	msgdto "mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/timex"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

type MachineFile struct {
	machineFileApp application.MachineFile `inject:"T"`
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

// progressReader 用于 HTTP 上传时推送进度
type progressReader struct {
	reader     io.Reader
	readSize   int64
	ctx        context.Context
	onProgress func(readSize int64) // 进度回调函数
	lastNotify time.Time            // 上次发送通知的时间
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	// 直接调用底层 Read，不创建 goroutine
	n, err = r.reader.Read(p)

	if n > 0 {
		r.readSize += int64(n)

		// 如果有回调函数，检查是否需要发送进度通知
		// 首次立即通知 + 之后每1秒通知一次（确保小文件也能收到通知）
		if r.onProgress != nil {
			now := time.Now()
			// lastNotify 为零值表示首次读取，立即发送通知
			if r.lastNotify.IsZero() || now.Sub(r.lastNotify) >= time.Second {
				r.lastNotify = now
				r.onProgress(r.readSize)
			}
		}
	}

	return n, err
}

func (m *MachineFile) MachineFiles(rc *req.Ctx) {
	condition := &entity.MachineFile{MachineId: GetMachineId(rc)}
	res, err := m.machineFileApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.MachineFile, *vo.MachineFileVO](res)
}

func (m *MachineFile) SaveMachineFiles(rc *req.Ctx) {
	fileForm, entity := rc.BindJsonAndCopyTo[form.MachineFileForm, entity.MachineFile]()

	rc.ReqParam = fileForm
	biz.ErrIsNil(m.machineFileApp.Save(rc.MetaCtx, entity))
}

func (m *MachineFile) DeleteFile(rc *req.Ctx) {
	biz.ErrIsNil(m.machineFileApp.DeleteById(rc.MetaCtx, GetMachineFileId(rc)))
}

/***      sftp相关操作      */

func (m *MachineFile) CreateFile(rc *req.Ctx) {
	opForm := rc.BindJson[form.CreateFileForm]()
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
	opForm := rc.BindQuery[dto.MachineFileOp]()
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
	opForm := rc.BindQuery[dto.MachineFileOp]()

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
	opForm := rc.BindQuery[dto.MachineFileOp]()
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
	opForm := rc.BindQuery[dto.MachineFileOp]()

	size, err := m.machineFileApp.GetDirSize(rc.MetaCtx, opForm)
	biz.ErrIsNil(err)
	rc.ResData = size
}

func (m *MachineFile) GetFileStat(rc *req.Ctx) {
	opForm := rc.BindQuery[dto.MachineFileOp]()
	res, err := m.machineFileApp.FileStat(rc.MetaCtx, opForm)
	biz.ErrIsNil(err, res)
	rc.ResData = res
}

func (m *MachineFile) WriteFileContent(rc *req.Ctx) {
	opForm := rc.BindJson[form.WriteFileContentForm]()
	path := opForm.Path

	mi, err := m.machineFileApp.WriteFileContent(rc.MetaCtx, opForm.MachineFileOp, []byte(opForm.Content))
	rc.ReqParam = collx.Kvs("machine", mi, "path", path)
	biz.ErrIsNilAppendErr(err, "open file error: %s")
}

func (m *MachineFile) UploadFile(rc *req.Ctx) {
	// 从查询参数读取配置
	opForm := rc.BindQuery[dto.MachineFileOp]()
	path := opForm.Path
	authCertName := opForm.AuthCertName
	uploadId := rc.Query("uploadId")                       // 前端传递的 uploadId
	isFolderUpload := rc.Query("isFolderUpload") == "true" // 是否是文件夹上传的一部分

	body := rc.GetRequest().Body
	defer body.Close()

	// 从查询参数获取文件名
	filename := rc.QueryDefault("filename", "upload_file")
	// 获取内容长度
	contentLength := rc.GetRequest().ContentLength

	// MetaCtx 用于业务逻辑（如获取登录账号、 EventBus 等）
	metaCtx := rc.MetaCtx

	maxUploadFileSize := config.GetMachine().UploadMaxFileSize
	biz.IsTrueI(metaCtx, contentLength <= maxUploadFileSize, imsg.ErrUploadFileOutOfLimit, "size", maxUploadFileSize)

	// 是否需要推送进度通知
	hasProgressNotify := uploadId != ""

	// 进度通知消息模板
	progressMsgTmplChannel := msgdto.MsgTmplMachineFileUploadProgress
	if isFolderUpload {
		progressMsgTmplChannel = msgdto.MsgTmplMachineFolderUploadProgress
	}
	// 推送进度
	var progressMsgEvent *msgdto.MsgTmplSendEvent
	if hasProgressNotify {
		progressMsgEvent = &msgdto.MsgTmplSendEvent{
			TmplChannel: progressMsgTmplChannel,
			Params: collx.M{
				"authCertName": authCertName,
				"path":         path,
				"uploadId":     uploadId,
				"filename":     filename,
				"totalSize":    contentLength,
				"timestamp":    time.Now().UnixMilli(),
			},
			ReceiverIds: []uint64{contextx.GetLoginAccount(metaCtx).Id},
		}
	}

	var mi *mcm.MachineInfo

	var reader io.Reader = body
	if hasProgressNotify {
		reader = &progressReader{
			reader: body,
			ctx:    metaCtx,
			onProgress: func(readSize int64) {
				parmas := collx.CopyM(progressMsgEvent.Params)
				parmas["uploadedSize"] = readSize
				parmas["status"] = "uploading"
				parmas["timestamp"] = time.Now().UnixMilli()
				progressMsgEvent.Params = parmas
				global.EventBus.Publish(metaCtx, event.EventTopicMsgTmplSend, progressMsgEvent)
			},
		}
	}

	uploadedSize, mi, err := m.machineFileApp.UploadFile(metaCtx, opForm, filename, reader)
	rc.ReqParam = collx.Kvs("machine", mi, "path", fmt.Sprintf("%s/%s", path, filename))

	// 检查实际上传的大小（已上传大小 == 总大小）
	if err == nil && uploadedSize < contentLength {
		err = fmt.Errorf("File upload canceld")
	}

	if hasProgressNotify {
		params := collx.CopyM(progressMsgEvent.Params)
		if err != nil {
			logx.WarnfContext(metaCtx, "[UploadFile] File upload incomplete: readSize=%d, total=%d, uploadId: %s", uploadedSize, contentLength, uploadId)
			params["status"] = "error"
		} else {
			params["status"] = "complete"
		}
		progressMsgEvent.Params = params
		global.EventBus.Publish(metaCtx, event.EventTopicMsgTmplSend, progressMsgEvent)
	}

	// 发送文件上传结果消息
	// 文件夹上传时不发送单个文件的成功通知，只在全部完成后由前端发送一次
	if !isFolderUpload {
		msgEvent := &msgdto.MsgTmplSendEvent{
			TmplChannel: msgdto.MsgTmplMachineFileUploadSuccess,
			Params: collx.M{
				"filename": filename,
				"path":     path,
			},
			ReceiverIds: []uint64{contextx.GetLoginAccount(metaCtx).Id},
		}
		if err != nil {
			msgEvent.Params["error"] = err.Error()
			msgEvent.TmplChannel = msgdto.MsgTmplMachineFileUploadFail
		}
		if mi != nil {
			msgEvent.Params["machineName"] = mi.Name
			msgEvent.Params["machineCode"] = mi.Code
		}
		global.EventBus.Publish(metaCtx, event.EventTopicMsgTmplSend, msgEvent)
	}

	biz.ErrIsNilAppendErr(err, "upload file error: %s")
}

func (m *MachineFile) RemoveFile(rc *req.Ctx) {
	opForm := rc.BindJson[form.RemoveFileForm]()

	mi, err := m.machineFileApp.RemoveFile(rc.MetaCtx, opForm.MachineFileOp, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "path", opForm)
	biz.ErrIsNilAppendErr(err, "remove file error: %s")
}

func (m *MachineFile) CopyFile(rc *req.Ctx) {
	opForm := rc.BindJson[form.CopyFileForm]()
	mi, err := m.machineFileApp.Copy(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	biz.ErrIsNilAppendErr(err, "file copy error: %s")
	rc.ReqParam = collx.Kvs("machine", mi, "cp", opForm)
}

func (m *MachineFile) MvFile(rc *req.Ctx) {
	opForm := rc.BindJson[form.CopyFileForm]()
	mi, err := m.machineFileApp.Mv(rc.MetaCtx, opForm.MachineFileOp, opForm.ToPath, opForm.Paths...)
	rc.ReqParam = collx.Kvs("machine", mi, "mv", opForm)
	biz.ErrIsNilAppendErr(err, "file move error: %s")
}

func (m *MachineFile) Rename(rc *req.Ctx) {
	renameForm := rc.BindJson[form.RenameForm]()
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
