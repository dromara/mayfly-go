package ssh

import (
	"fmt"
	"mayfly-go/cli/cmd/shared"
	"mayfly-go/cli/i18n"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: i18n.T(i18n.MsgSshLsShort),
	Long:  i18n.T(i18n.MsgSshLsLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		dirPath, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")
		longFormat, _ := cmd.Flags().GetBool("long")

		if machineId == 0 || dirPath == "" {
			return shared.Fail(i18n.MsgSshLsRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		if longFormat {
			result, err := apiClient.RunMachineCmd(machineId, authCert, fmt.Sprintf("ls -la %s", dirPath))
			if err != nil {
				return shared.Fail(i18n.MsgSshLsFailed, err)
			}
			if shared.JsonOutput {
				shared.PrintJSONSuccess(result)
				return nil
			}
			if output, ok := result["output"]; ok {
				fmt.Print(output)
			}
			return nil
		}

		result, err := apiClient.ListDir(machineId, authCert, dirPath)
		if err != nil {
			return shared.Fail(i18n.MsgSshLsFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

var catCmd = &cobra.Command{
	Use:   "cat",
	Short: i18n.T(i18n.MsgSshCatShort),
	Long:  i18n.T(i18n.MsgSshCatLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		filePath, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || filePath == "" {
			return shared.Fail(i18n.MsgSshCatRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		content, err := apiClient.ReadRemoteFile(machineId, authCert, filePath)
		if err != nil {
			return shared.Fail(i18n.MsgSshCatFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"path": filePath, "content": content})
			return nil
		}

		fmt.Print(content)
		return nil
	},
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: i18n.T(i18n.MsgSshDownloadShort),
	Long:  i18n.T(i18n.MsgSshDownloadLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		remotePath, _ := cmd.Flags().GetString("path")
		localPath, _ := cmd.Flags().GetString("output")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || remotePath == "" {
			return shared.Fail(i18n.MsgSshDownloadRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		data, err := apiClient.DownloadFile(machineId, authCert, remotePath)
		if err != nil {
			return shared.Fail(i18n.MsgSshDownloadFailed, err)
		}

		if localPath == "" {
			localPath = filepath.Base(remotePath)
		} else if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			localPath = filepath.Join(localPath, filepath.Base(remotePath))
		}

		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return shared.Fail(i18n.MsgSshDownloadWriteFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{
				"remote": remotePath,
				"local":  localPath,
				"size":   len(data),
			})
			return nil
		}

		fmt.Printf("%s %s -> %s (%d bytes)\n", i18n.T(i18n.MsgSshDownloadOk), remotePath, localPath, len(data))
		return nil
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: i18n.T(i18n.MsgSshUploadShort),
	Long:  i18n.T(i18n.MsgSshUploadLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		remoteDir, _ := cmd.Flags().GetString("path")
		localFile, _ := cmd.Flags().GetString("file")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || remoteDir == "" || localFile == "" {
			return shared.Fail(i18n.MsgSshUploadRequired, nil)
		}

		content, err := os.ReadFile(localFile)
		if err != nil {
			return shared.Fail(i18n.MsgSshUploadReadFailed, err)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err = resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		filename := filepath.Base(localFile)
		if err := apiClient.UploadFile(machineId, authCert, remoteDir, filename, content); err != nil {
			return shared.Fail(i18n.MsgSshUploadFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{
				"local":  localFile,
				"remote": remoteDir + "/" + filename,
				"size":   len(content),
			})
			return nil
		}

		fmt.Printf("%s %s -> %s/%s (%d bytes)\n", i18n.T(i18n.MsgSshUploadOk), localFile, remoteDir, filename, len(content))
		return nil
	},
}

var writeCmd = &cobra.Command{
	Use:   "write",
	Short: i18n.T(i18n.MsgSshWriteShort),
	Long:  i18n.T(i18n.MsgSshWriteLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		filePath, _ := cmd.Flags().GetString("path")
		content, _ := cmd.Flags().GetString("content")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || filePath == "" || content == "" {
			return shared.Fail(i18n.MsgSshWriteRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		if err := apiClient.WriteRemoteFile(machineId, authCert, filePath, content); err != nil {
			return shared.Fail(i18n.MsgSshWriteFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"path": filePath, "written": true})
			return nil
		}

		fmt.Printf("%s %s\n", i18n.T(i18n.MsgSshWriteOk), filePath)
		return nil
	},
}

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: i18n.T(i18n.MsgSshRmShort),
	Long:  i18n.T(i18n.MsgSshRmLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		path, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || path == "" {
			return shared.Fail(i18n.MsgSshRmRequired, nil)
		}

		// Dry-run 模式：显示将执行的操作
		if shared.DryRun {
			result := map[string]interface{}{
				"dry_run":       true,
				"operation":     "rm",
				"would_execute": "rm -rf " + path,
				"target":        map[string]interface{}{"machine_id": machineId, "path": path},
			}
			if shared.JsonOutput {
				shared.PrintJSONSuccess(result)
			} else {
				fmt.Printf("[DRY-RUN] %s: rm -rf %s\n", i18n.T(i18n.MsgSshRmShort), path)
			}
			return nil
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		if err := apiClient.RemoveRemoteFile(machineId, authCert, []string{path}); err != nil {
			return shared.Fail(i18n.MsgSshRmFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{
				"operation": "rm",
				"path":      path,
				"removed":   true,
				"target":    map[string]interface{}{"machine_id": machineId, "path": path},
			})
			return nil
		}

		fmt.Printf("%s %s\n", i18n.T(i18n.MsgSshRmOk), path)
		return nil
	},
}

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: i18n.T(i18n.MsgSshKillShort),
	Long:  i18n.T(i18n.MsgSshKillLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		pid, _ := cmd.Flags().GetInt("pid")

		if machineId == 0 || pid <= 0 {
			return shared.Fail(i18n.MsgSshKillRequired, nil)
		}

		// Dry-run 模式：显示将执行的操作
		if shared.DryRun {
			result := map[string]interface{}{
				"dry_run":       true,
				"operation":     "kill",
				"would_execute": fmt.Sprintf("kill -9 %d", pid),
				"target":        map[string]interface{}{"machine_id": machineId, "pid": pid},
			}
			if shared.JsonOutput {
				shared.PrintJSONSuccess(result)
			} else {
				fmt.Printf("[DRY-RUN] %s: kill -9 %d\n", i18n.T(i18n.MsgSshKillShort), pid)
			}
			return nil
		}

		apiClient := shared.NewClient(cmd)
		if err := apiClient.KillProcess(machineId, pid); err != nil {
			return shared.Fail(i18n.MsgSshKillFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{
				"operation": "kill",
				"pid":       pid,
				"killed":    true,
				"target":    map[string]interface{}{"machine_id": machineId, "pid": pid},
			})
			return nil
		}

		fmt.Printf("%s PID %d\n", i18n.T(i18n.MsgSshKillOk), pid)
		return nil
	},
}

var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: i18n.T(i18n.MsgSshCpShort),
	Long:  i18n.T(i18n.MsgSshCpLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		src, _ := cmd.Flags().GetString("src")
		dst, _ := cmd.Flags().GetString("dst")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || src == "" || dst == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		_, err = apiClient.RunMachineCmd(machineId, authCert, fmt.Sprintf("cp -r %s %s", src, dst))
		if err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"src": src, "dst": dst, "copied": true})
			return nil
		}

		fmt.Printf("%s: %s -> %s\n", i18n.T(i18n.MsgSshCpOk), src, dst)
		return nil
	},
}

var mvCmd = &cobra.Command{
	Use:   "mv",
	Short: i18n.T(i18n.MsgSshMvShort),
	Long:  i18n.T(i18n.MsgSshMvLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		src, _ := cmd.Flags().GetString("src")
		dst, _ := cmd.Flags().GetString("dst")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || src == "" || dst == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		_, err = apiClient.RunMachineCmd(machineId, authCert, fmt.Sprintf("mv %s %s", src, dst))
		if err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"src": src, "dst": dst, "moved": true})
			return nil
		}

		fmt.Printf("%s: %s -> %s\n", i18n.T(i18n.MsgSshMvOk), src, dst)
		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: i18n.T(i18n.MsgSshRenameShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		path, _ := cmd.Flags().GetString("path")
		newName, _ := cmd.Flags().GetString("name")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || path == "" || newName == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		dir := path[:strings.LastIndex(path, "/")+1]
		_, err = apiClient.RunMachineCmd(machineId, authCert, fmt.Sprintf("mv %s %s%s", path, dir, newName))
		if err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"path": path, "newName": newName, "renamed": true})
			return nil
		}

		fmt.Printf("%s: %s -> %s\n", i18n.T(i18n.MsgSshRenameOk), path, newName)
		return nil
	},
}

var mkdirCmd = &cobra.Command{
	Use:   "mkdir",
	Short: i18n.T(i18n.MsgSshMkdirShort),
	Long:  i18n.T(i18n.MsgSshMkdirLong),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		path, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || path == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		if err := apiClient.CreateRemoteFile(machineId, authCert, path, true); err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"path": path, "created": true})
			return nil
		}

		fmt.Printf("%s: %s\n", i18n.T(i18n.MsgSshMkdirOk), path)
		return nil
	},
}

var touchCmd = &cobra.Command{
	Use:   "touch",
	Short: i18n.T(i18n.MsgSshTouchShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		path, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || path == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		_, err = apiClient.RunMachineCmd(machineId, authCert, fmt.Sprintf("touch %s", path))
		if err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(map[string]interface{}{"path": path, "created": true})
			return nil
		}

		fmt.Printf("%s: %s\n", i18n.T(i18n.MsgSshTouchOk), path)
		return nil
	},
}

var statCmd = &cobra.Command{
	Use:   "stat",
	Short: i18n.T(i18n.MsgSshStatShort),
	RunE: func(cmd *cobra.Command, args []string) error {
		machineId, _ := cmd.Flags().GetUint64("id")
		path, _ := cmd.Flags().GetString("path")
		authCert, _ := cmd.Flags().GetString("auth-cert")

		if machineId == 0 || path == "" {
			return shared.Fail(i18n.MsgSshOpRequired, nil)
		}

		apiClient := shared.NewClient(cmd)
		authCert, err := resolveAuthCert(apiClient, machineId, authCert)
		if err != nil {
			return shared.Fail(i18n.MsgSshExecNotFound, err)
		}

		result, err := apiClient.GetFileStat(machineId, authCert, path)
		if err != nil {
			return shared.Fail(i18n.MsgSshOpFailed, err)
		}

		if shared.JsonOutput {
			shared.PrintJSONSuccess(result)
			return nil
		}

		shared.PrintResult(result)
		return nil
	},
}

func init() {
	Cmd.AddCommand(lsCmd)
	Cmd.AddCommand(catCmd)
	Cmd.AddCommand(downloadCmd)
	Cmd.AddCommand(uploadCmd)
	Cmd.AddCommand(writeCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(killCmd)
	Cmd.AddCommand(cpCmd)
	Cmd.AddCommand(mvCmd)
	Cmd.AddCommand(renameCmd)
	Cmd.AddCommand(mkdirCmd)
	Cmd.AddCommand(touchCmd)
	Cmd.AddCommand(statCmd)

	// ssh ls
	lsCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	lsCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	lsCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	lsCmd.Flags().BoolP("long", "l", false, "Long format (permissions, size, time)")
	lsCmd.MarkFlagRequired("id")
	lsCmd.MarkFlagRequired("path")

	// ssh cat
	catCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	catCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	catCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	catCmd.MarkFlagRequired("id")
	catCmd.MarkFlagRequired("path")

	// ssh download
	downloadCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	downloadCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	downloadCmd.Flags().StringP("output", "o", "", i18n.T(i18n.MsgSshFlagOutput))
	downloadCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	downloadCmd.MarkFlagRequired("id")
	downloadCmd.MarkFlagRequired("path")

	// ssh upload
	uploadCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	uploadCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagRemoteDir))
	uploadCmd.Flags().String("file", "", i18n.T(i18n.MsgSshFlagLocalFile))
	uploadCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	uploadCmd.MarkFlagRequired("id")
	uploadCmd.MarkFlagRequired("file")

	// ssh write
	writeCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	writeCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	writeCmd.Flags().String("content", "", i18n.T(i18n.MsgSshFlagContent))
	writeCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	writeCmd.MarkFlagRequired("id")
	writeCmd.MarkFlagRequired("path")
	writeCmd.MarkFlagRequired("content")

	// ssh rm
	rmCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	rmCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	rmCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	rmCmd.MarkFlagRequired("id")
	rmCmd.MarkFlagRequired("path")

	// ssh kill
	killCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	killCmd.Flags().Int("pid", 0, i18n.T(i18n.MsgSshFlagPid))
	killCmd.MarkFlagRequired("id")
	killCmd.MarkFlagRequired("pid")

	// ssh cp / mv
	for _, c := range []*cobra.Command{cpCmd, mvCmd} {
		c.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
		c.Flags().String("src", "", i18n.T(i18n.MsgSshFlagSrc))
		c.Flags().String("dst", "", i18n.T(i18n.MsgSshFlagDst))
		c.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
		c.MarkFlagRequired("id")
		c.MarkFlagRequired("src")
		c.MarkFlagRequired("dst")
	}

	// ssh rename
	renameCmd.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
	renameCmd.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
	renameCmd.Flags().String("name", "", i18n.T(i18n.MsgSshFlagName))
	renameCmd.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
	renameCmd.MarkFlagRequired("id")
	renameCmd.MarkFlagRequired("path")
	renameCmd.MarkFlagRequired("name")

	// ssh mkdir / touch / stat
	for _, c := range []*cobra.Command{mkdirCmd, touchCmd, statCmd} {
		c.Flags().Uint64("id", 0, i18n.T(i18n.MsgSshFlagId))
		c.Flags().String("path", "", i18n.T(i18n.MsgSshFlagPath))
		c.Flags().String("auth-cert", "", i18n.T(i18n.MsgSshFlagAuthCert))
		c.MarkFlagRequired("id")
		c.MarkFlagRequired("path")
	}

	// 动态补全
	shared.RegisterIdCompletion(lsCmd, "machine")
	shared.RegisterIdCompletion(catCmd, "machine")
	shared.RegisterIdCompletion(downloadCmd, "machine")
	shared.RegisterIdCompletion(uploadCmd, "machine")
	shared.RegisterIdCompletion(writeCmd, "machine")
	shared.RegisterIdCompletion(rmCmd, "machine")
	shared.RegisterIdCompletion(killCmd, "machine")
	shared.RegisterIdCompletion(cpCmd, "machine")
	shared.RegisterIdCompletion(mvCmd, "machine")
	shared.RegisterIdCompletion(renameCmd, "machine")
	shared.RegisterIdCompletion(mkdirCmd, "machine")
	shared.RegisterIdCompletion(touchCmd, "machine")
	shared.RegisterIdCompletion(statCmd, "machine")
}
