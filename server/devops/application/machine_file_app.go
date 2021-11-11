package application

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/base/ws"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/persistence"
	sysApplication "mayfly-go/server/sys/application"
	"os"
	"strings"

	"github.com/pkg/sftp"
)

type MachineFile interface {
	// 分页获取机器文件信息列表
	GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据条件获取
	GetMachineFile(condition *entity.MachineFile, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineFile

	Save(entity *entity.MachineFile)

	Delete(id uint64)

	/**  sftp 相关操作 **/

	// 读取目录
	ReadDir(fid uint64, path string) []fs.FileInfo

	// 读取文件内容
	ReadFile(fileId uint64, path string) ([]byte, fs.FileInfo)

	// 写文件
	WriteFileContent(fileId uint64, path string, content []byte)

	// 文件上传
	UploadFile(la *model.LoginAccount, fileId uint64, path, filename string, content []byte)

	// 移除文件
	RemoveFile(fileId uint64, path string)
}

type machineFileAppImpl struct {
	machineFileRepo repository.MachineFile
	machineRepo     repository.Machine
	msgApp          sysApplication.Msg
}

// 实现类单例
var MachineFileApp MachineFile = &machineFileAppImpl{
	machineRepo:     persistence.MachineDao,
	machineFileRepo: persistence.MachineFileDao,
	msgApp:          sysApplication.MsgApp,
}

// 分页获取机器脚本信息列表
func (m *machineFileAppImpl) GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return m.machineFileRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (m *machineFileAppImpl) GetMachineFile(condition *entity.MachineFile, cols ...string) error {
	return m.machineFileRepo.GetMachineFile(condition, cols...)
}

// 根据id获取
func (m *machineFileAppImpl) GetById(id uint64, cols ...string) *entity.MachineFile {
	return m.machineFileRepo.GetById(id, cols...)
}

// 保存机器文件配置
func (m *machineFileAppImpl) Save(entity *entity.MachineFile) {
	biz.NotNil(m.machineRepo.GetById(entity.MachineId, "Name"), "该机器不存在")

	if entity.Id != 0 {
		model.UpdateById(entity)
	} else {
		model.Insert(entity)
	}
}

// 根据id删除
func (m *machineFileAppImpl) Delete(id uint64) {
	m.machineFileRepo.Delete(id)
}

func (m *machineFileAppImpl) ReadDir(fid uint64, path string) []fs.FileInfo {
	path, machineId := m.checkAndReturnPathMid(fid, path)
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	sftpCli := m.getSftpCli(machineId)
	fis, err := sftpCli.ReadDir(path)
	biz.ErrIsNilAppendErr(err, "读取目录失败: %s")
	return fis
}

func (m *machineFileAppImpl) ReadFile(fileId uint64, path string) ([]byte, fs.FileInfo) {
	path, machineId := m.checkAndReturnPathMid(fileId, path)
	sftpCli := m.getSftpCli(machineId)
	// 读取文件内容
	fc, err := sftpCli.Open(path)
	biz.ErrIsNilAppendErr(err, "打开文件失败：%s")
	defer fc.Close()

	fileInfo, _ := fc.Stat()
	biz.IsTrue(!fileInfo.IsDir(), "该文件为目录")
	dataByte, err := ioutil.ReadAll(fc)
	if err != nil && err != io.EOF {
		panic(biz.NewBizErr("读取文件内容失败"))
	}
	return dataByte, fileInfo
}

// 写文件内容
func (m *machineFileAppImpl) WriteFileContent(fileId uint64, path string, content []byte) {
	_, machineId := m.checkAndReturnPathMid(fileId, path)

	sftpCli := m.getSftpCli(machineId)
	f, err := sftpCli.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_RDWR)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	defer f.Close()

	fi, _ := f.Stat()
	biz.IsTrue(!fi.IsDir(), "该路径不是文件")
	f.Write(content)
}

// 上传文件
func (m *machineFileAppImpl) UploadFile(la *model.LoginAccount, fileId uint64, path, filename string, content []byte) {
	path, machineId := m.checkAndReturnPathMid(fileId, path)
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	sftpCli := m.getSftpCli(machineId)
	createfile, err := sftpCli.Create(path + filename)
	biz.ErrIsNilAppendErr(err, "创建文件失败: %s")
	defer createfile.Close()

	createfile.Write(content)
	// 保存消息并发送文件上传成功通知
	m.msgApp.CreateAndSend(la, ws.SuccessMsg("文件上传", fmt.Sprintf("[%s]文件已成功上传至[%s]", filename, path)))

}

// 删除文件
func (m *machineFileAppImpl) RemoveFile(fileId uint64, path string) {
	path, machineId := m.checkAndReturnPathMid(fileId, path)

	sftpCli := m.getSftpCli(machineId)
	file, err := sftpCli.Open(path)
	biz.ErrIsNilAppendErr(err, "打开文件失败: %s")
	fi, _ := file.Stat()
	if fi.IsDir() {
		err = sftpCli.RemoveDirectory(path)
	} else {
		err = sftpCli.Remove(path)
	}
	biz.ErrIsNilAppendErr(err, "删除文件失败: %s")
}

// 获取sftp client
func (m *machineFileAppImpl) getSftpCli(machineId uint64) *sftp.Client {
	return MachineApp.GetCli(machineId).GetSftpCli()
}

// 校验并返回实际可访问的文件path
func (m *machineFileAppImpl) checkAndReturnPathMid(fid uint64, inputPath string) (string, uint64) {
	biz.IsTrue(fid != 0, "文件id不能为空")
	mf := m.GetById(uint64(fid))
	biz.NotNil(mf, "文件不存在")
	if inputPath != "" {
		// 接口传入的地址需为配置路径的子路径
		biz.IsTrue(strings.HasPrefix(inputPath, mf.Path), "无权访问该目录或文件")
		return inputPath, mf.MachineId
	} else {
		return mf.Path, mf.MachineId
	}
}
