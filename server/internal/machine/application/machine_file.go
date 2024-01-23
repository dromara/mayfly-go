package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"os"
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

	/**  sftp 相关操作 **/

	// 创建目录
	MkDir(fid uint64, path string) (*mcm.MachineInfo, error)

	// 创建文件
	CreateFile(fid uint64, path string) (*mcm.MachineInfo, error)

	// 读取目录
	ReadDir(fid uint64, path string) ([]fs.FileInfo, error)

	// 获取指定目录内容大小
	GetDirSize(fid uint64, path string) (string, error)

	// 获取文件stat
	FileStat(fid uint64, path string) (string, error)

	// 读取文件内容
	ReadFile(fileId uint64, path string) (*sftp.File, *mcm.MachineInfo, error)

	// 写文件
	WriteFileContent(fileId uint64, path string, content []byte) (*mcm.MachineInfo, error)

	// 文件上传
	UploadFile(fileId uint64, path, filename string, reader io.Reader) (*mcm.MachineInfo, error)

	// 移除文件
	RemoveFile(fileId uint64, path ...string) (*mcm.MachineInfo, error)

	Copy(fileId uint64, toPath string, paths ...string) (*mcm.MachineInfo, error)

	Mv(fileId uint64, toPath string, paths ...string) (*mcm.MachineInfo, error)

	Rename(fileId uint64, oldname string, newname string) (*mcm.MachineInfo, error)
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

func (m *machineFileAppImpl) ReadDir(fid uint64, path string) ([]fs.FileInfo, error) {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	_, sftpCli, err := m.GetMachineSftpCli(fid, path)
	if err != nil {
		return nil, err
	}
	return sftpCli.ReadDir(path)
}

func (m *machineFileAppImpl) GetDirSize(fid uint64, path string) (string, error) {
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

func (m *machineFileAppImpl) FileStat(fid uint64, path string) (string, error) {
	mcli, err := m.GetMachineCli(fid, path)
	if err != nil {
		return "", err
	}
	return mcli.Run(fmt.Sprintf("stat -L %s", path))
}

func (m *machineFileAppImpl) MkDir(fid uint64, path string) (*mcm.MachineInfo, error) {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	mi, sftpCli, err := m.GetMachineSftpCli(fid, path)
	if err != nil {
		return nil, err
	}

	sftpCli.MkdirAll(path)
	return mi, err
}

func (m *machineFileAppImpl) CreateFile(fid uint64, path string) (*mcm.MachineInfo, error) {
	mi, sftpCli, err := m.GetMachineSftpCli(fid, path)
	if err != nil {
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
func (m *machineFileAppImpl) WriteFileContent(fileId uint64, path string, content []byte) (*mcm.MachineInfo, error) {
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
func (m *machineFileAppImpl) UploadFile(fileId uint64, path, filename string, reader io.Reader) (*mcm.MachineInfo, error) {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
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

// 删除文件
func (m *machineFileAppImpl) RemoveFile(fileId uint64, path ...string) (*mcm.MachineInfo, error) {
	mcli, err := m.GetMachineCli(fileId, path...)
	if err != nil {
		return nil, err
	}
	minfo := mcli.Info

	// 优先使用命令删除（速度快），sftp需要递归遍历删除子文件等
	res, err := mcli.Run(fmt.Sprintf("rm -rf %s", strings.Join(path, " ")))
	if err == nil {
		return minfo, nil
	}
	logx.Errorf("使用命令rm删除文件失败: %s", res)

	sftpCli, err := mcli.GetSftpCli()
	if err != nil {
		return minfo, err
	}

	for _, p := range path {
		err = sftpCli.RemoveAll(p)
		if err != nil {
			break
		}
	}
	return minfo, err
}

func (m *machineFileAppImpl) Copy(fileId uint64, toPath string, paths ...string) (*mcm.MachineInfo, error) {
	mcli, err := m.GetMachineCli(fileId, paths...)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("cp -r %s %s", strings.Join(paths, " "), toPath))
	if err != nil {
		return mi, errors.New(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Mv(fileId uint64, toPath string, paths ...string) (*mcm.MachineInfo, error) {
	mcli, err := m.GetMachineCli(fileId, paths...)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("mv %s %s", strings.Join(paths, " "), toPath))
	if err != nil {
		return mi, errorx.NewBiz(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Rename(fileId uint64, oldname string, newname string) (*mcm.MachineInfo, error) {
	mi, sftpCli, err := m.GetMachineSftpCli(fileId, newname)
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
