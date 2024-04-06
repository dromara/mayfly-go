package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/bytex"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

type MachineFile interface {
	base.App[*entity.MachineFile]

	// 分页获取机器文件信息列表
	GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 根据条件获取
	GetMachineFile(condition *entity.MachineFile, cols ...string) error

	Save(ctx context.Context, entity *entity.MachineFile) error

	// 获取文件关联的机器信息，主要用于记录日志使用
	// GetMachine(fileId uint64) *mcm.Info

	// 检查文件路径，并返回机器id
	GetMachineCli(fileId uint64, path ...string) (*mcm.Cli, error)

	GetRdpFilePath(MachineId uint64, path string) string

	/**  sftp 相关操作 **/

	// 创建目录
	MkDir(fid uint64, path string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error)

	// 创建文件
	CreateFile(fid uint64, path string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error)

	// 读取目录
	ReadDir(fid uint64, opForm *form.ServerFileOptionForm) ([]fs.FileInfo, error)

	// 获取指定目录内容大小
	GetDirSize(fid uint64, opForm *form.ServerFileOptionForm) (string, error)

	// 获取文件stat
	FileStat(opForm *form.ServerFileOptionForm) (string, error)

	// 读取文件内容
	ReadFile(fileId uint64, path string) (*sftp.File, *mcm.MachineInfo, error)

	// 写文件
	WriteFileContent(fileId uint64, path string, content []byte, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error)

	// 文件上传
	UploadFile(fileId uint64, path, filename string, reader io.Reader, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error)

	UploadFiles(basePath string, fileHeaders []*multipart.FileHeader, paths []string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error)

	// 移除文件
	RemoveFile(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error)

	Copy(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error)

	Mv(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error)

	Rename(renameForm *form.MachineFileRename) (*mcm.MachineInfo, error)
}

type machineFileAppImpl struct {
	base.AppImpl[*entity.MachineFile, repository.MachineFile]

	machineApp Machine `inject:"MachineApp"`
}

// 注入MachineFileRepo
func (m *machineFileAppImpl) InjectMachineFileRepo(repo repository.MachineFile) {
	m.Repo = repo
}

// 分页获取机器脚本信息列表
func (m *machineFileAppImpl) GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.GetRepo().GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (m *machineFileAppImpl) GetMachineFile(condition *entity.MachineFile, cols ...string) error {
	return m.GetBy(condition, cols...)
}

// 保存机器文件配置
func (m *machineFileAppImpl) Save(ctx context.Context, mf *entity.MachineFile) error {
	_, err := m.machineApp.GetById(new(entity.Machine), mf.MachineId, "Name")
	if err != nil {
		return errorx.NewBiz("该机器不存在")
	}

	if mf.Id != 0 {
		return m.UpdateById(ctx, mf)
	}

	return m.Insert(ctx, mf)
}

func (m *machineFileAppImpl) ReadDir(fid uint64, opForm *form.ServerFileOptionForm) ([]fs.FileInfo, error) {
	if !strings.HasSuffix(opForm.Path, "/") {
		opForm.Path = opForm.Path + "/"
	}

	// 如果是rdp，则直接读取本地文件
	if opForm.Protocol == entity.MachineProtocolRdp {
		opForm.Path = m.GetRdpFilePath(opForm.MachineId, opForm.Path)
		return ioutil.ReadDir(opForm.Path)
	}

	_, sftpCli, err := m.GetMachineSftpCli(fid, opForm.Path)
	if err != nil {
		return nil, err
	}
	return sftpCli.ReadDir(opForm.Path)
}

func (m *machineFileAppImpl) GetDirSize(fid uint64, opForm *form.ServerFileOptionForm) (string, error) {
	path := opForm.Path

	if opForm.Protocol == entity.MachineProtocolRdp {
		dirPath := m.GetRdpFilePath(opForm.MachineId, path)

		// 递归计算目录下文件大小
		var totalSize int64
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			// 忽略目录本身
			if path != dirPath {
				totalSize += info.Size()
			}
			return nil
		})
		if err != nil {
			return "", err
		}

		return bytex.FormatSize(totalSize), nil
	}

	mcli, err := m.GetMachineCli(fid, path)
	if err != nil {
		return "", err
	}
	res, err := mcli.Run(fmt.Sprintf("du -sh %s", path))
	if err != nil {
		// 若存在目录为空，则可能会返回如下内容。最后一行即为真正目录内容所占磁盘空间大小
		//du: cannot access ‘/proc/19087/fd/3’: No such file or directory\n
		//du: cannot access ‘/proc/19087/fdinfo/3’: No such file or directory\n
		//18G     /\n
		if res == "" {
			return "", errorx.NewBiz("获取目录大小失败: %s", err.Error())
		}
		strs := strings.Split(res, "\n")
		res = strs[len(strs)-2]

		if !strings.Contains(res, "\t") {
			return "", errorx.NewBiz(res)
		}
	}
	// 返回 32K\t/tmp\n
	return strings.Split(res, "\t")[0], nil
}

func (m *machineFileAppImpl) FileStat(opForm *form.ServerFileOptionForm) (string, error) {
	if opForm.Protocol == entity.MachineProtocolRdp {
		path := m.GetRdpFilePath(opForm.MachineId, opForm.Path)
		stat, err := os.Stat(path)
		return fmt.Sprintf("%v", stat), err
	}

	mcli, err := m.GetMachineCli(opForm.FileId, opForm.Path)
	if err != nil {
		return "", err
	}
	return mcli.Run(fmt.Sprintf("stat -L %s", opForm.Path))
}

func (m *machineFileAppImpl) MkDir(fid uint64, path string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error) {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	if opForm.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(opForm.MachineId, path)
		os.MkdirAll(path, os.ModePerm)
		return nil, nil
	}

	mi, sftpCli, err := m.GetMachineSftpCli(fid, path)
	if err != nil {
		return nil, err
	}

	sftpCli.MkdirAll(path)
	return mi, err
}

func (m *machineFileAppImpl) CreateFile(fid uint64, path string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error) {
	mi, sftpCli, err := m.GetMachineSftpCli(fid, path)
	if err != nil {
		return nil, err
	}

	if opForm.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(opForm.MachineId, path)
		file, err := os.Create(path)
		defer file.Close()
		return nil, err
	}

	file, err := sftpCli.Create(path)
	if err != nil {
		return nil, errorx.NewBiz("创建文件失败: %s", err.Error())
	}
	defer file.Close()
	return mi, err
}

func (m *machineFileAppImpl) ReadFile(fileId uint64, path string) (*sftp.File, *mcm.MachineInfo, error) {
	mi, sftpCli, err := m.GetMachineSftpCli(fileId, path)
	if err != nil {
		return nil, nil, err
	}

	// 读取文件内容
	fc, err := sftpCli.Open(path)
	return fc, mi, err
}

// 写文件内容
func (m *machineFileAppImpl) WriteFileContent(fileId uint64, path string, content []byte, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error) {

	if opForm.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(opForm.MachineId, path)
		file, err := os.Create(path)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		file.Write(content)
		return nil, err
	}

	mi, sftpCli, err := m.GetMachineSftpCli(fileId, path)
	if err != nil {
		return nil, err
	}

	f, err := sftpCli.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_RDWR)
	if err != nil {
		return mi, err
	}
	defer f.Close()
	f.Write(content)
	return mi, err
}

// 上传文件
func (m *machineFileAppImpl) UploadFile(fileId uint64, path, filename string, reader io.Reader, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error) {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	if opForm.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(opForm.MachineId, path)
		file, err := os.Create(path + filename)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		io.Copy(file, reader)
		return nil, nil
	}

	mi, sftpCli, err := m.GetMachineSftpCli(fileId, path)
	if err != nil {
		return nil, err
	}

	createfile, err := sftpCli.Create(path + filename)
	if err != nil {
		return mi, err
	}
	defer createfile.Close()
	io.Copy(createfile, reader)
	return mi, err
}

func (m *machineFileAppImpl) UploadFiles(basePath string, fileHeaders []*multipart.FileHeader, paths []string, opForm *form.ServerFileOptionForm) (*mcm.MachineInfo, error) {
	if opForm.Protocol == entity.MachineProtocolRdp {
		baseFolder := m.GetRdpFilePath(opForm.MachineId, basePath)

		for i, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				return nil, err
			}
			defer file.Close()

			// 创建文件夹
			rdpBaseDir := basePath
			if !strings.HasSuffix(rdpBaseDir, "/") {
				rdpBaseDir = rdpBaseDir + "/"
			}
			rdpDir := filepath.Dir(rdpBaseDir + paths[i])
			m.MkDir(0, rdpDir, opForm)

			// 创建文件
			if !strings.HasSuffix(baseFolder, "/") {
				baseFolder = baseFolder + "/"
			}
			fileAbsPath := baseFolder + paths[i]
			createFile, err := os.Create(fileAbsPath)
			if err != nil {
				return nil, err
			}
			defer createFile.Close()

			// 复制文件内容
			io.Copy(createFile, file)
		}
	}

	return nil, nil
}

// 删除文件
func (m *machineFileAppImpl) RemoveFile(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error) {

	if opForm.Protocol == entity.MachineProtocolRdp {
		for _, pt := range opForm.Path {
			pt = m.GetRdpFilePath(opForm.MachineId, pt)
			os.RemoveAll(pt)
		}
		return nil, nil
	}

	mcli, err := m.GetMachineCli(opForm.FileId, opForm.Path...)
	if err != nil {
		return nil, err
	}
	minfo := mcli.Info

	// 优先使用命令删除（速度快），sftp需要递归遍历删除子文件等
	res, err := mcli.Run(fmt.Sprintf("rm -rf %s", strings.Join(opForm.Path, " ")))
	if err == nil {
		return minfo, nil
	}
	logx.Errorf("使用命令rm删除文件失败: %s", res)

	sftpCli, err := mcli.GetSftpCli()
	if err != nil {
		return minfo, err
	}

	for _, p := range opForm.Path {
		err = sftpCli.RemoveAll(p)
		if err != nil {
			break
		}
	}
	return minfo, err
}

func (m *machineFileAppImpl) Copy(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error) {
	if opForm.Protocol == entity.MachineProtocolRdp {
		for _, pt := range opForm.Path {
			srcPath := m.GetRdpFilePath(opForm.MachineId, pt)
			targetPath := m.GetRdpFilePath(opForm.MachineId, opForm.ToPath+pt)

			// 打开源文件
			srcFile, err := os.Open(srcPath)
			if err != nil {
				fmt.Println("Error opening source file:", err)
				return nil, err
			}
			// 创建目标文件
			destFile, err := os.Create(targetPath)
			if err != nil {
				fmt.Println("Error creating destination file:", err)
				return nil, err
			}
			io.Copy(destFile, srcFile)
		}
		return nil, nil
	}

	mcli, err := m.GetMachineCli(opForm.FileId, opForm.Path...)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("cp -r %s %s", strings.Join(opForm.Path, " "), opForm.ToPath))
	if err != nil {
		return mi, errors.New(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Mv(opForm *form.MachineFileOpForm) (*mcm.MachineInfo, error) {
	if opForm.Protocol == entity.MachineProtocolRdp {
		for _, pt := range opForm.Path {
			// 获取文件名
			filename := filepath.Base(pt)
			topath := opForm.ToPath
			if !strings.HasSuffix(topath, "/") {
				topath += "/"
			}

			srcPath := m.GetRdpFilePath(opForm.MachineId, pt)
			targetPath := m.GetRdpFilePath(opForm.MachineId, topath+filename)
			os.Rename(srcPath, targetPath)
		}
		return nil, nil
	}

	mcli, err := m.GetMachineCli(opForm.FileId, opForm.Path...)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("mv %s %s", strings.Join(opForm.Path, " "), opForm.ToPath))
	if err != nil {
		return mi, errorx.NewBiz(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Rename(renameForm *form.MachineFileRename) (*mcm.MachineInfo, error) {
	oldname := renameForm.Oldname
	newname := renameForm.Newname
	if renameForm.Protocol == entity.MachineProtocolRdp {
		oldname = m.GetRdpFilePath(renameForm.MachineId, renameForm.Oldname)
		newname = m.GetRdpFilePath(renameForm.MachineId, renameForm.Newname)
		return nil, os.Rename(oldname, newname)
	}

	mi, sftpCli, err := m.GetMachineSftpCli(renameForm.FileId, newname)
	if err != nil {
		return nil, err
	}
	return mi, sftpCli.Rename(oldname, newname)
}

// 获取文件机器cli
func (m *machineFileAppImpl) GetMachineCli(fid uint64, inputPath ...string) (*mcm.Cli, error) {
	mf, err := m.GetById(new(entity.MachineFile), fid)
	if err != nil {
		return nil, errorx.NewBiz("文件不存在")
	}

	for _, path := range inputPath {
		// 接口传入的地址需为配置路径的子路径
		if !strings.HasPrefix(path, mf.Path) {
			return nil, errorx.NewBiz("无权访问该目录或文件: %s", path)
		}
	}
	return m.machineApp.GetCli(mf.MachineId)
}

// 获取文件机器 sftp cli
func (m *machineFileAppImpl) GetMachineSftpCli(fid uint64, inputPath ...string) (*mcm.MachineInfo, *sftp.Client, error) {
	mcli, err := m.GetMachineCli(fid, inputPath...)
	if err != nil {
		return nil, nil, err
	}

	sftpCli, err := mcli.GetSftpCli()
	if err != nil {
		return nil, nil, err
	}

	return mcli.Info, sftpCli, nil
}

func (m *machineFileAppImpl) GetRdpFilePath(MachineId uint64, path string) string {
	return fmt.Sprintf("%s/%d%s", config.GetMachine().GuacdFilePath, MachineId, path)
}
