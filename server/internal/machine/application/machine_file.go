package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mayfly-go/internal/machine/config"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/bytex"
	"mayfly-go/pkg/utils/collx"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

type MachineFileOpParam struct {
	Ua           *model.LoginAccount
	MachineId    uint64 `json:"machineId" binding:"required" form:"machineId"`
	Protocol     int    `json:"protocol" binding:"required" form:"protocol"`
	AuthCertName string `json:"authCertName"  binding:"required" form:"authCertName"` // 授权凭证
	Path         string `json:"path" form:"path"`                                     // 文件路径
}

type MachineFile interface {
	base.App[*entity.MachineFile]

	// 分页获取机器文件信息列表
	GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 根据条件获取
	GetMachineFile(condition *entity.MachineFile, cols ...string) error

	Save(ctx context.Context, entity *entity.MachineFile) error

	// 获取机器cli
	GetMachineCli(authCertName string) (*mcm.Cli, error)

	GetRdpFilePath(ua *model.LoginAccount, path string) string

	/**  sftp 相关操作 **/

	// 创建目录
	MkDir(ctx context.Context, opParam *MachineFileOpParam) (*mcm.MachineInfo, error)

	// 创建文件
	CreateFile(ctx context.Context, opParam *MachineFileOpParam) (*mcm.MachineInfo, error)

	// 读取目录
	ReadDir(ctx context.Context, opParam *MachineFileOpParam) ([]fs.FileInfo, error)

	// 获取指定目录内容大小
	GetDirSize(ctx context.Context, opParam *MachineFileOpParam) (string, error)

	// 获取文件stat
	FileStat(ctx context.Context, opParam *MachineFileOpParam) (string, error)

	// 读取文件内容
	ReadFile(ctx context.Context, opParam *MachineFileOpParam) (*sftp.File, *mcm.MachineInfo, error)

	// 写文件
	WriteFileContent(ctx context.Context, opParam *MachineFileOpParam, content []byte) (*mcm.MachineInfo, error)

	// 文件上传
	UploadFile(ctx context.Context, opParam *MachineFileOpParam, filename string, reader io.Reader) (*mcm.MachineInfo, error)

	UploadFiles(ctx context.Context, opParam *MachineFileOpParam, basePath string, fileHeaders []*multipart.FileHeader, paths []string) (*mcm.MachineInfo, error)

	// 移除文件
	RemoveFile(ctx context.Context, opParam *MachineFileOpParam, path ...string) (*mcm.MachineInfo, error)

	Copy(ctx context.Context, opParam *MachineFileOpParam, toPath string, path ...string) (*mcm.MachineInfo, error)

	Mv(ctx context.Context, opParam *MachineFileOpParam, toPath string, path ...string) (*mcm.MachineInfo, error)

	Rename(ctx context.Context, opParam *MachineFileOpParam, newname string) (*mcm.MachineInfo, error)
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
	return m.GetByCond(model.NewModelCond(condition).Columns(cols...))
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

func (m *machineFileAppImpl) ReadDir(ctx context.Context, opParam *MachineFileOpParam) ([]fs.FileInfo, error) {
	path := opParam.Path
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	// 如果是rdp，则直接读取本地文件
	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		dirs, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		return collx.ArrayMap[fs.DirEntry, fs.FileInfo](dirs, func(val fs.DirEntry) fs.FileInfo {
			fi, _ := val.Info()
			return fi
		}), nil
	}

	_, sftpCli, err := m.GetMachineSftpCli(opParam)
	if err != nil {
		return nil, err
	}
	return sftpCli.ReadDir(path)
}

func (m *machineFileAppImpl) GetDirSize(ctx context.Context, opParam *MachineFileOpParam) (string, error) {
	path := opParam.Path

	if opParam.Protocol == entity.MachineProtocolRdp {
		dirPath := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)

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

	mcli, err := m.GetMachineCli(opParam.AuthCertName)
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

func (m *machineFileAppImpl) FileStat(ctx context.Context, opParam *MachineFileOpParam) (string, error) {
	path := opParam.Path
	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		stat, err := os.Stat(path)
		return fmt.Sprintf("%v", stat), err
	}

	mcli, err := m.GetMachineCli(opParam.AuthCertName)
	if err != nil {
		return "", err
	}
	return mcli.Run(fmt.Sprintf("stat -L %s", path))
}

func (m *machineFileAppImpl) MkDir(ctx context.Context, opParam *MachineFileOpParam) (*mcm.MachineInfo, error) {
	path := opParam.Path
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		os.MkdirAll(path, os.ModePerm)
		return nil, nil
	}

	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
	if err != nil {
		return nil, err
	}

	sftpCli.MkdirAll(path)
	return mi, err
}

func (m *machineFileAppImpl) CreateFile(ctx context.Context, opParam *MachineFileOpParam) (*mcm.MachineInfo, error) {
	path := opParam.Path
	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		file, err := os.Create(path)
		defer file.Close()
		return nil, err
	}

	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
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

func (m *machineFileAppImpl) ReadFile(ctx context.Context, opParam *MachineFileOpParam) (*sftp.File, *mcm.MachineInfo, error) {
	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
	if err != nil {
		return nil, nil, err
	}

	// 读取文件内容
	fc, err := sftpCli.Open(opParam.Path)
	return fc, mi, err
}

// 写文件内容
func (m *machineFileAppImpl) WriteFileContent(ctx context.Context, opParam *MachineFileOpParam, content []byte) (*mcm.MachineInfo, error) {
	path := opParam.Path
	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		file, err := os.Create(path)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		file.Write(content)
		return nil, err
	}

	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
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
func (m *machineFileAppImpl) UploadFile(ctx context.Context, opParam *MachineFileOpParam, filename string, reader io.Reader) (*mcm.MachineInfo, error) {
	path := opParam.Path
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	if opParam.Protocol == entity.MachineProtocolRdp {
		path = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), path)
		file, err := os.Create(path + filename)
		defer file.Close()
		if err != nil {
			return nil, err
		}
		io.Copy(file, reader)
		return nil, nil
	}

	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
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

func (m *machineFileAppImpl) UploadFiles(ctx context.Context, opParam *MachineFileOpParam, basePath string, fileHeaders []*multipart.FileHeader, paths []string) (*mcm.MachineInfo, error) {
	if opParam.Protocol == entity.MachineProtocolRdp {
		baseFolder := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), basePath)

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
			m.MkDir(ctx, &MachineFileOpParam{
				MachineId: opParam.MachineId,
				Protocol:  opParam.Protocol,
				Path:      rdpDir,
			})

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
func (m *machineFileAppImpl) RemoveFile(ctx context.Context, opParam *MachineFileOpParam, path ...string) (*mcm.MachineInfo, error) {
	if opParam.Protocol == entity.MachineProtocolRdp {
		for _, pt := range path {
			pt = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), pt)
			os.RemoveAll(pt)
		}
		return nil, nil
	}

	mcli, err := m.GetMachineCli(opParam.AuthCertName)
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

func (m *machineFileAppImpl) Copy(ctx context.Context, opParam *MachineFileOpParam, toPath string, path ...string) (*mcm.MachineInfo, error) {
	if opParam.Protocol == entity.MachineProtocolRdp {
		for _, pt := range path {
			srcPath := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), pt)
			targetPath := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), toPath+pt)

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

	mcli, err := m.GetMachineCli(opParam.AuthCertName)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("cp -r %s %s", strings.Join(path, " "), toPath))
	if err != nil {
		return mi, errors.New(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Mv(ctx context.Context, opParam *MachineFileOpParam, toPath string, path ...string) (*mcm.MachineInfo, error) {
	if opParam.Protocol == entity.MachineProtocolRdp {
		for _, pt := range path {
			// 获取文件名
			filename := filepath.Base(pt)
			if !strings.HasSuffix(toPath, "/") {
				toPath += "/"
			}

			srcPath := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), pt)
			targetPath := m.GetRdpFilePath(contextx.GetLoginAccount(ctx), toPath+filename)
			os.Rename(srcPath, targetPath)
		}
		return nil, nil
	}

	mcli, err := m.GetMachineCli(opParam.AuthCertName)
	if err != nil {
		return nil, err
	}

	mi := mcli.Info
	res, err := mcli.Run(fmt.Sprintf("mv %s %s", strings.Join(path, " "), toPath))
	if err != nil {
		return mi, errorx.NewBiz(res)
	}
	return mi, err
}

func (m *machineFileAppImpl) Rename(ctx context.Context, opParam *MachineFileOpParam, newname string) (*mcm.MachineInfo, error) {
	oldname := opParam.Path
	if opParam.Protocol == entity.MachineProtocolRdp {
		oldname = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), oldname)
		newname = m.GetRdpFilePath(contextx.GetLoginAccount(ctx), newname)
		return nil, os.Rename(oldname, newname)
	}

	mi, sftpCli, err := m.GetMachineSftpCli(opParam)
	if err != nil {
		return nil, err
	}
	return mi, sftpCli.Rename(oldname, newname)
}

// 获取文件机器cli
func (m *machineFileAppImpl) GetMachineCli(authCertName string) (*mcm.Cli, error) {
	return m.machineApp.GetCliByAc(authCertName)
}

// 获取文件机器 sftp cli
func (m *machineFileAppImpl) GetMachineSftpCli(opParam *MachineFileOpParam) (*mcm.MachineInfo, *sftp.Client, error) {
	mcli, err := m.GetMachineCli(opParam.AuthCertName)
	if err != nil {
		return nil, nil, err
	}

	sftpCli, err := mcli.GetSftpCli()
	if err != nil {
		return nil, nil, err
	}

	return mcli.Info, sftpCli, nil
}

func (m *machineFileAppImpl) GetRdpFilePath(ua *model.LoginAccount, path string) string {
	return fmt.Sprintf("%s/%s%s", config.GetMachine().GuacdFilePath, ua.Username, path)
}
