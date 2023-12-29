package service

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/domain/service"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/structx"
)

// BinlogFile is the metadata of the MySQL binlog file.
type BinlogFile struct {
	Name string
	Size int64

	// Sequence is parsed from Name and is for the sorting purpose.
	Sequence       int64
	FirstEventTime time.Time
	Downloaded     bool
}

func newBinlogFile(name string, size int64) (*BinlogFile, error) {
	_, seq, err := ParseBinlogName(name)
	if err != nil {
		return nil, err
	}
	return &BinlogFile{Name: name, Size: size, Sequence: seq}, nil
}

var _ service.DbInstanceSvc = (*DbInstanceSvcImpl)(nil)

type DbInstanceSvcImpl struct {
	instanceId        uint64
	dbInfo            *dbm.DbInfo
	backupHistoryRepo repository.DbBackupHistory
	binlogHistoryRepo repository.DbBinlogHistory
}

func NewDbInstanceSvc(instance *entity.DbInstance, repositories *repository.Repositories) *DbInstanceSvcImpl {
	dbInfo := new(dbm.DbInfo)
	_ = structx.Copy(dbInfo, instance)
	return &DbInstanceSvcImpl{
		instanceId:        instance.Id,
		dbInfo:            dbInfo,
		backupHistoryRepo: repositories.BackupHistory,
		binlogHistoryRepo: repositories.BinlogHistory,
	}
}

type RestoreInfo struct {
	backupHistory   *entity.DbBackupHistory
	binlogHistories []*entity.DbBinlogHistory
	startPosition   int64
	targetPosition  int64
	targetTime      time.Time
}

func (ri *RestoreInfo) getBinlogFiles(binlogDir string) []string {
	files := make([]string, 0, len(ri.binlogHistories))
	for _, history := range ri.binlogHistories {
		files = append(files, filepath.Join(binlogDir, history.FileName))
	}
	return files
}

func (svc *DbInstanceSvcImpl) getBinlogFilePath(fileName string) string {
	return filepath.Join(getBinlogDir(svc.instanceId), fileName)
}

func (svc *DbInstanceSvcImpl) GetRestoreInfo(ctx context.Context, dbName string, targetTime time.Time) (*RestoreInfo, error) {
	binlogHistory, err := svc.binlogHistoryRepo.GetHistoryByTime(svc.instanceId, targetTime)
	if err != nil {
		return nil, err
	}
	position, err := getBinlogEventPositionAtOrAfterTime(ctx, svc.getBinlogFilePath(binlogHistory.FileName), targetTime)
	if err != nil {
		return nil, err
	}
	target := &entity.BinlogInfo{
		FileName: binlogHistory.FileName,
		Sequence: binlogHistory.Sequence,
		Position: position,
	}
	backupHistory, err := svc.backupHistoryRepo.GetLatestHistory(svc.instanceId, dbName, target)
	if err != nil {
		return nil, err
	}
	start := &entity.BinlogInfo{
		FileName: backupHistory.BinlogFileName,
		Sequence: backupHistory.BinlogSequence,
		Position: backupHistory.BinlogPosition,
	}
	binlogHistories, err := svc.binlogHistoryRepo.GetHistories(svc.instanceId, start, target)
	if err != nil {
		return nil, err
	}
	return &RestoreInfo{
		backupHistory:   backupHistory,
		binlogHistories: binlogHistories,
		startPosition:   backupHistory.BinlogPosition,
		targetPosition:  target.Position,
		targetTime:      targetTime,
	}, nil
}

func (svc *DbInstanceSvcImpl) Backup(ctx context.Context, backupHistory *entity.DbBackupHistory) (*entity.BinlogInfo, error) {
	dir := getDbBackupDir(backupHistory.DbInstanceId, backupHistory.DbBackupId)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}
	tmpFile := filepath.Join(dir, "backup.tmp")
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	args := []string{
		"--host", svc.dbInfo.Host,
		"--port", strconv.Itoa(svc.dbInfo.Port),
		"--user", svc.dbInfo.Username,
		"--password=" + svc.dbInfo.Password,
		"--add-drop-database",
		"--result-file", tmpFile,
		"--single-transaction",
		"--master-data=2",
		"--databases", backupHistory.DbName,
	}

	cmd := exec.CommandContext(ctx, mysqldumpPath(), args...)
	logx.Debug("backup database using mysqldump binary: ", cmd.String())
	if err := runCmd(cmd); err != nil {
		logx.Errorf("运行 mysqldump 程序失败: %v", err)
		return nil, errors.Wrap(err, "运行 mysqldump 程序失败")
	}

	logx.Debug("Checking dumped file stat", tmpFile)
	if _, err := os.Stat(tmpFile); err != nil {
		logx.Errorf("未找到备份文件: %v", err)
		return nil, errors.Wrapf(err, "未找到备份文件")
	}
	reader, err := os.Open(tmpFile)
	if err != nil {
		return nil, err
	}
	binlogInfo, err := readBinlogInfoFromBackup(reader)
	_ = reader.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "从备份文件中读取 binlog 信息失败")
	}
	fileName := filepath.Join(dir, fmt.Sprintf("%s.sql", backupHistory.Uuid))
	if err := os.Rename(tmpFile, fileName); err != nil {
		return nil, errors.Wrap(err, "备份文件改名失败")
	}

	return binlogInfo, nil
}

func (svc *DbInstanceSvcImpl) RestoreBackup(ctx context.Context, database, fileName string) error {
	args := []string{
		"--host", svc.dbInfo.Host,
		"--port", strconv.Itoa(svc.dbInfo.Port),
		"--database", database,
		"--user", svc.dbInfo.Username,
		"--password=" + svc.dbInfo.Password,
	}

	file, err := os.Open(fileName)
	if err != nil {
		return errors.Wrap(err, "打开备份文件失败")
	}
	defer func() {
		_ = file.Close()
	}()

	cmd := exec.CommandContext(ctx, mysqlPath(), args...)
	cmd.Stdin = file
	logx.Debug("恢复数据库: ", cmd.String())
	if err := runCmd(cmd); err != nil {
		logx.Errorf("运行 mysql 程序失败: %v", err)
		return errors.Wrap(err, "运行 mysql 程序失败")
	}
	return nil
}

func (svc *DbInstanceSvcImpl) Restore(ctx context.Context, task *entity.DbRestore) error {
	if task.PointInTime.IsZero() {
		backupHistory := &entity.DbBackupHistory{}
		err := svc.backupHistoryRepo.GetById(backupHistory, task.DbBackupHistoryId)
		if err != nil {
			return err
		}
		fileName := filepath.Join(getDbBackupDir(backupHistory.DbInstanceId, backupHistory.DbBackupId),
			fmt.Sprintf("%v.sql", backupHistory.Uuid))
		return svc.RestoreBackup(ctx, task.DbName, fileName)
	}

	if err := svc.FetchBinlogs(ctx, true); err != nil {
		return err
	}
	restoreInfo, err := svc.GetRestoreInfo(ctx, task.DbName, task.PointInTime)
	if err != nil {
		return err
	}
	fileName := filepath.Join(getDbBackupDir(restoreInfo.backupHistory.DbInstanceId, restoreInfo.backupHistory.DbBackupId),
		fmt.Sprintf("%s.sql", restoreInfo.backupHistory.Uuid))

	if err := svc.RestoreBackup(ctx, task.DbName, fileName); err != nil {
		return err
	}
	return svc.ReplayBinlogToDatabase(ctx, task.DbName, task.DbName, restoreInfo)
}

// Download binlog files on server.
func (svc *DbInstanceSvcImpl) downloadBinlogFilesOnServer(ctx context.Context, binlogFilesOnServerSorted []*BinlogFile, downloadLatestBinlogFile bool) error {
	if len(binlogFilesOnServerSorted) == 0 {
		logx.Debug("No binlog file found on server to download")
		return nil
	}
	if err := os.MkdirAll(getBinlogDir(svc.instanceId), os.ModePerm); err != nil {
		return errors.Wrapf(err, "创建 binlog 目录失败: %q", getBinlogDir(svc.instanceId))
	}
	latestBinlogFileOnServer := binlogFilesOnServerSorted[len(binlogFilesOnServerSorted)-1]
	for _, fileOnServer := range binlogFilesOnServerSorted {
		isLatest := fileOnServer.Name == latestBinlogFileOnServer.Name
		if isLatest && !downloadLatestBinlogFile {
			continue
		}
		binlogFilePath := filepath.Join(getBinlogDir(svc.instanceId), fileOnServer.Name)
		logx.Debug("Downloading binlog file from MySQL server.", logx.String("path", binlogFilePath), logx.Bool("isLatest", isLatest))
		if err := svc.downloadBinlogFile(ctx, fileOnServer, isLatest); err != nil {
			logx.Error("下载 binlog 文件失败", logx.String("path", binlogFilePath), logx.String("error", err.Error()))
			return errors.Wrapf(err, "下载 binlog 文件失败: %q", binlogFilePath)
		}
	}
	return nil
}

// Parse the first binlog eventTs of a local binlog file.
func parseLocalBinlogFirstEventTime(ctx context.Context, filePath string) (eventTime time.Time, parseErr error) {
	args := []string{
		// Local binlog file path.
		filePath,
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
	}
	cmd := exec.CommandContext(ctx, mysqlbinlogPath(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return time.Time{}, err
	}

	if err := cmd.Start(); err != nil {
		return time.Time{}, err
	}
	defer func() {
		_ = cmd.Cancel()
		if err := cmd.Wait(); err != nil && parseErr != nil && stderr.Len() > 0 {
			parseErr = errors.Wrap(parseErr, stderr.String())
		}
	}()

	for s := bufio.NewScanner(pr); ; s.Scan() {
		line := s.Text()
		eventTimeParsed, found, err := parseBinlogEventTimeInLine(line)
		if err != nil {
			return time.Time{}, errors.Wrap(err, "解析 binlog 文件失败")
		}
		if found {
			return eventTimeParsed, nil
		}
	}
	return time.Time{}, errors.New("解析 binlog 文件失败")
}

// getBinlogDir gets the binlogDir.
func getBinlogDir(instanceId uint64) string {
	return filepath.Join(
		config.Conf.Db.BackupPath,
		fmt.Sprintf("instance-%d", instanceId),
		"binlog")
}

func getDbInstanceBackupRoot(instanceId uint64) string {
	return filepath.Join(
		config.Conf.Db.BackupPath,
		fmt.Sprintf("instance-%d", instanceId))
}

func getDbBackupDir(instanceId, backupId uint64) string {
	return filepath.Join(
		config.Conf.Db.BackupPath,
		fmt.Sprintf("instance-%d", instanceId),
		fmt.Sprintf("backup-%d", backupId))
}

var singleFlightGroup singleflight.Group

// FetchBinlogs downloads binlog files from startingFileName on server to `binlogDir`.
func (svc *DbInstanceSvcImpl) FetchBinlogs(ctx context.Context, downloadLatestBinlogFile bool) error {
	latestDownloaded := false
	_, err, _ := singleFlightGroup.Do(strconv.FormatUint(svc.instanceId, 10), func() (interface{}, error) {
		latestDownloaded = downloadLatestBinlogFile
		err := svc.fetchBinlogs(ctx, downloadLatestBinlogFile)
		return nil, err
	})

	if downloadLatestBinlogFile && !latestDownloaded {
		_, err, _ = singleFlightGroup.Do(strconv.FormatUint(svc.instanceId, 10), func() (interface{}, error) {
			err := svc.fetchBinlogs(ctx, true)
			return nil, err
		})
	}
	return err
}

// fetchBinlogs downloads binlog files from startingFileName on server to `binlogDir`.
func (svc *DbInstanceSvcImpl) fetchBinlogs(ctx context.Context, downloadLatestBinlogFile bool) error {
	// Read binlog files list on server.
	binlogFilesOnServerSorted, err := svc.GetSortedBinlogFilesOnServer(ctx)
	if err != nil {
		return err
	}
	if len(binlogFilesOnServerSorted) == 0 {
		logx.Debug("No binlog file found on server to download")
		return nil
	}
	latest, ok, err := svc.binlogHistoryRepo.GetLatestHistory(svc.instanceId)
	if err != nil {
		return err
	}
	binlogFileName := ""
	latestSequence := int64(-1)
	earliestSequence := int64(-1)
	if ok {
		latestSequence = latest.Sequence
		binlogFileName = latest.FileName
	} else {
		earliest, err := svc.backupHistoryRepo.GetEarliestHistory(svc.instanceId)
		if err != nil {
			return err
		}
		earliestSequence = earliest.BinlogSequence
		binlogFileName = earliest.BinlogFileName
	}
	indexHistory := -1
	for i, file := range binlogFilesOnServerSorted {
		if latestSequence == file.Sequence {
			indexHistory = i + 1
			break
		}
		if earliestSequence == file.Sequence {
			indexHistory = i
			break
		}
	}
	if indexHistory < 0 {
		return errors.New(fmt.Sprintf("在数据库服务器上未找到 binlog 文件 %q", binlogFileName))
	}
	if indexHistory > len(binlogFilesOnServerSorted)-1 {
		indexHistory = len(binlogFilesOnServerSorted) - 1
	}
	binlogFilesOnServerSorted = binlogFilesOnServerSorted[indexHistory:]

	if err := svc.downloadBinlogFilesOnServer(ctx, binlogFilesOnServerSorted, downloadLatestBinlogFile); err != nil {
		return err
	}
	for i, fileOnServer := range binlogFilesOnServerSorted {
		if !fileOnServer.Downloaded {
			break
		}
		history := &entity.DbBinlogHistory{
			CreateTime:     time.Now(),
			FileName:       fileOnServer.Name,
			FileSize:       fileOnServer.Size,
			Sequence:       fileOnServer.Sequence,
			FirstEventTime: fileOnServer.FirstEventTime,
			DbInstanceId:   svc.instanceId,
		}
		if i == len(binlogFilesOnServerSorted)-1 {
			if err := svc.binlogHistoryRepo.Upsert(ctx, history); err != nil {
				return err
			}
		} else {
			if err := svc.binlogHistoryRepo.Insert(ctx, history); err != nil {
				return err
			}
		}
	}

	return nil
}

// Syncs the binlog specified by `meta` between the instance and local.
// If isLast is true, it means that this is the last binlog file containing the targetTs event.
// It may keep growing as there are ongoing writes to the database. So we just need to check that
// the file size is larger or equal to the binlog file size we queried from the MySQL server earlier.
func (svc *DbInstanceSvcImpl) downloadBinlogFile(ctx context.Context, binlogFileToDownload *BinlogFile, isLast bool) error {
	tempBinlogPrefix := filepath.Join(getBinlogDir(svc.instanceId), "tmp-")
	args := []string{
		binlogFileToDownload.Name,
		"--read-from-remote-server",
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		"--host", svc.dbInfo.Host,
		"--port", strconv.Itoa(svc.dbInfo.Port),
		"--user", svc.dbInfo.Username,
		"--raw",
		// With --raw this is a prefix for the file names.
		"--result-file", tempBinlogPrefix,
	}

	cmd := exec.CommandContext(ctx, mysqlbinlogPath(), args...)
	// We cannot set password as a flag. Otherwise, there is warning message
	// "mysqlbinlog: [Warning] Using a password on the command line interface can be insecure."
	if svc.dbInfo.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("MYSQL_PWD=%s", svc.dbInfo.Password))
	}

	logx.Debug("Downloading binlog files using mysqlbinlog:", cmd.String())
	binlogFilePathTemp := tempBinlogPrefix + binlogFileToDownload.Name
	defer func() {
		_ = os.Remove(binlogFilePathTemp)
	}()
	if err := runCmd(cmd); err != nil {
		logx.Errorf("运行 mysqlbinlog 程序失败: %v", err)
		return errors.Wrap(err, "运行 mysqlbinlog 程序失败")
	}

	logx.Debug("Checking downloaded binlog file stat", logx.String("path", binlogFilePathTemp))
	binlogFileTempInfo, err := os.Stat(binlogFilePathTemp)
	if err != nil {
		logx.Error("未找到 binlog 文件", logx.String("path", binlogFilePathTemp), logx.String("error", err.Error()))
		return errors.Wrapf(err, "未找到 binlog 文件: %q", binlogFilePathTemp)
	}
	if !isLast && binlogFileTempInfo.Size() != binlogFileToDownload.Size {
		logx.Error("Downloaded archived binlog file size is not equal to size queried on the MySQL server earlier.",
			logx.String("binlog", binlogFileToDownload.Name),
			logx.Int64("sizeInfo", binlogFileToDownload.Size),
			logx.Int64("downloadedSize", binlogFileTempInfo.Size()),
		)
		return errors.Errorf("下载的 binlog 文件 %q 与服务上的文件大小不一致 %d != %d", binlogFilePathTemp, binlogFileTempInfo.Size(), binlogFileToDownload.Size)
	}

	binlogFilePath := svc.getBinlogFilePath(binlogFileToDownload.Name)
	if err := os.Rename(binlogFilePathTemp, binlogFilePath); err != nil {
		return errors.Wrapf(err, "binlog 文件更名失败: %q -> %q", binlogFilePathTemp, binlogFilePath)
	}
	firstEventTime, err := parseLocalBinlogFirstEventTime(ctx, binlogFilePath)
	if err != nil {
		return err
	}
	binlogFileToDownload.FirstEventTime = firstEventTime
	binlogFileToDownload.Downloaded = true

	return nil
}

// GetSortedBinlogFilesOnServer returns the information of binlog files in ascending order by their numeric extension.
func (svc *DbInstanceSvcImpl) GetSortedBinlogFilesOnServer(_ context.Context) ([]*BinlogFile, error) {
	conn, err := svc.dbInfo.Conn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	query := "SHOW BINARY LOGS"
	columns, rows, err := conn.Query(query)
	if err != nil {
		return nil, errors.Wrapf(err, "SQL 语句 %q 执行失败", query)
	}
	findFileName := false
	findFileSize := false
	for _, column := range columns {
		switch column.Name {
		case "Log_name":
			findFileName = true
		case "File_size":
			findFileSize = true
		}
	}
	if !findFileName || !findFileSize {
		return nil, errors.Errorf("SQL 语句 %q 执行结果解析失败", query)
	}

	var binlogFiles []*BinlogFile

	for _, row := range rows {
		name, nameOk := row["Log_name"].(string)
		size, sizeOk := row["File_size"].(uint64)
		if !nameOk || !sizeOk {
			return nil, errors.Errorf("SQL 语句 %q 执行结果解析失败", query)
		}

		binlogFile, err := newBinlogFile(name, int64(size))
		if err != nil {
			return nil, errors.Wrapf(err, "SQL 语句 %q 执行结果解析失败", query)
		}
		binlogFiles = append(binlogFiles, binlogFile)
	}

	return sortBinlogFiles(binlogFiles), nil
}

var regexpBinlogInfo = regexp.MustCompile("CHANGE MASTER TO MASTER_LOG_FILE='([^.]+).([0-9]+)', MASTER_LOG_POS=([0-9]+);")

func readBinlogInfoFromBackup(reader io.Reader) (*entity.BinlogInfo, error) {
	matching := false
	r := bufio.NewReader(reader)
	const maxMatchRow = 100
	for i := 0; i < maxMatchRow; i++ {
		row, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if !matching {
			if row == "-- Position to start replication or point-in-time recovery from\n" {
				matching = true
			} else {
				continue
			}
		}
		res := regexpBinlogInfo.FindStringSubmatch(row)
		if res == nil {
			continue
		}
		seq, err := strconv.ParseInt(res[2], 10, 64)
		if err != nil {
			return nil, err
		}
		pos, err := strconv.ParseInt(res[3], 10, 64)
		if err != nil {
			return nil, err
		}

		return &entity.BinlogInfo{
			FileName: fmt.Sprintf("%s.%s", res[1], res[2]),
			Sequence: seq,
			Position: pos,
		}, nil
	}
	return nil, errors.New("备份文件中未找到 binlog 信息")
}

// Use command like mysqlbinlog --start-datetime=targetTs binlog.000001 to parse the first binlog event position with timestamp equal or after targetTs.
func getBinlogEventPositionAtOrAfterTime(ctx context.Context, filePath string, targetTime time.Time) (position int64, parseErr error) {
	args := []string{
		// Local binlog file path.
		filePath,
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		// Tell mysqlbinlog to suppress the BINLOG statements for row events, which reduces the unneeded output.
		"--base64-output=DECODE-ROWS",
		// Instruct mysqlbinlog to start output only after encountering the first binlog event with timestamp equal or after targetTime.
		"--start-datetime", formatDateTime(targetTime),
	}
	cmd := exec.CommandContext(ctx, mysqlbinlogPath(), args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	defer func() {
		_ = cmd.Cancel()
		if err := cmd.Wait(); err != nil && parseErr != nil && stderr.Len() > 0 {
			parseErr = errors.Wrap(errors.New(stderr.String()), parseErr.Error())
		}
	}()

	for s := bufio.NewScanner(pr); ; s.Scan() {
		line := s.Text()
		posParsed, found, err := parseBinlogEventPosInLine(line)
		if err != nil {
			return 0, errors.Wrap(err, "binlog 文件解析失败")
		}
		// When invoking mysqlbinlog with --start-datetime, the first valid event will always be FORMAT_DESCRIPTION_EVENT which should be skipped.
		if found && posParsed != 4 {
			return posParsed, nil
		}
	}
	return 0, errors.Errorf("在 %v 之后没有 binlog 事件", targetTime)
}

// replayBinlog replays the binlog for `originDatabase` from `startBinlogInfo.Position` to `targetTs`, read binlog from `binlogDir`.
func (svc *DbInstanceSvcImpl) replayBinlog(ctx context.Context, originalDatabase, targetDatabase string, restoreInfo *RestoreInfo) (replayErr error) {
	const (
		// Variable lower_case_table_names related.

		// LetterCaseOnDiskLetterCaseCmp stores table and database names using the letter case specified in the CREATE TABLE or CREATE DATABASE statement.
		// Name comparisons are case-sensitive.
		LetterCaseOnDiskLetterCaseCmp = 0
		// LowerCaseOnDiskLowerCaseCmp stores table names in lowercase on disk and name comparisons are not case-sensitive.
		LowerCaseOnDiskLowerCaseCmp = 1
		// LetterCaseOnDiskLowerCaseCmp stores table and database names are stored on disk using the letter case specified in the CREATE TABLE or CREATE DATABASE statement, but MySQL converts them to lowercase on lookup.
		// Name comparisons are not case-sensitive.
		LetterCaseOnDiskLowerCaseCmp = 2
	)

	caseVariable := "lower_case_table_names"
	identifierCaseSensitive, err := svc.getServerVariable(ctx, caseVariable)
	if err != nil {
		return err
	}

	identifierCaseSensitiveValue, err := strconv.Atoi(identifierCaseSensitive)
	if err != nil {
		return err
	}

	var originalDBName string
	switch identifierCaseSensitiveValue {
	case LetterCaseOnDiskLetterCaseCmp:
		originalDBName = originalDatabase
	case LowerCaseOnDiskLowerCaseCmp:
		originalDBName = strings.ToLower(originalDatabase)
	case LetterCaseOnDiskLowerCaseCmp:
		originalDBName = strings.ToLower(originalDatabase)
	default:
		return errors.Errorf("参数 %s 的值 %s 不符合预期: [%d, %d, %d] ", caseVariable, identifierCaseSensitive, 0, 1, 2)
	}

	// Extract the SQL statements from the binlog and replay them to the pitrDatabase via the mysql client by pipe.
	mysqlbinlogArgs := []string{
		// Verify checksum binlog events.
		"--verify-binlog-checksum",
		// Disable binary logging.
		"--disable-log-bin",
		// Create rewrite rules for databases when playing back from logs written in row-based format, so that we can apply the binlog to PITR database instead of the original database.
		"--rewrite-db", fmt.Sprintf("%s->%s", originalDBName, targetDatabase),
		// List entries for just this database. It's applied after the --rewrite-db option, so we should provide the rewritten database, i.e., pitrDatabase.
		"--database", targetDatabase,
		// Decode binary log from first event with position equal to or greater than argument.
		"--start-position", fmt.Sprintf("%d", restoreInfo.startPosition),
		// 	Stop decoding binary log at first event with position equal to or greater than argument.
		"--stop-position", fmt.Sprintf("%d", restoreInfo.targetPosition),
	}

	mysqlbinlogArgs = append(mysqlbinlogArgs, restoreInfo.getBinlogFiles(getBinlogDir(svc.instanceId))...)

	mysqlArgs := []string{
		"--host", svc.dbInfo.Host,
		"--port", strconv.Itoa(svc.dbInfo.Port),
		"--user", svc.dbInfo.Username,
	}

	if svc.dbInfo.Password != "" {
		// The --password parameter of mysql/mysqlbinlog does not support the "--password PASSWORD" format (split by space).
		// If provided like that, the program will hang.
		mysqlArgs = append(mysqlArgs, fmt.Sprintf("--password=%s", svc.dbInfo.Password))
	}

	mysqlbinlogCmd := exec.CommandContext(ctx, mysqlbinlogPath(), mysqlbinlogArgs...)
	mysqlCmd := exec.CommandContext(ctx, mysqlPath(), mysqlArgs...)
	logx.Debug("Start replay binlog commands.",
		logx.String("mysqlbinlog", mysqlbinlogCmd.String()),
		logx.String("mysql", mysqlCmd.String()))
	defer func() {
		if replayErr == nil {
			logx.Debug("Replayed binlog successfully.")
		}
	}()

	mysqlRead, err := mysqlbinlogCmd.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "创建 mysqlbinlog 输出管道失败")
	}
	defer func() {
		_ = mysqlRead.Close()
	}()

	var mysqlbinlogErr, mysqlErr strings.Builder
	mysqlbinlogCmd.Stderr = &mysqlbinlogErr
	mysqlCmd.Stderr = &mysqlErr
	mysqlCmd.Stdout = os.Stdout
	mysqlCmd.Stdin = mysqlRead

	if err := mysqlbinlogCmd.Start(); err != nil {
		return errors.Wrap(err, "启动 mysqlbinlog 程序失败")
	}
	defer func() {
		if err := mysqlbinlogCmd.Wait(); err != nil {
			if replayErr != nil {
				replayErr = errors.Wrap(replayErr, "运行 mysqlbinlog 程序失败")
			} else {
				replayErr = errors.Errorf("运行 mysqlbinlog 程序失败: %s", mysqlbinlogErr.String())
			}
		}
	}()
	if err := mysqlCmd.Start(); err != nil {
		return errors.Wrap(err, "启动 mysql 程序失败")
	}
	if err := mysqlCmd.Wait(); err != nil {
		return errors.Errorf("运行 mysql 程序失败: %s", mysqlbinlogErr.String())
	}

	return nil
}

// ReplayBinlogToDatabase replays the binlog of originDatabaseName to the targetDatabaseName.
func (svc *DbInstanceSvcImpl) ReplayBinlogToDatabase(ctx context.Context, originDatabaseName, targetDatabaseName string, restoreInfo *RestoreInfo) error {
	return svc.replayBinlog(ctx, originDatabaseName, targetDatabaseName, restoreInfo)
}

func (svc *DbInstanceSvcImpl) getServerVariable(_ context.Context, varName string) (string, error) {
	conn, err := svc.dbInfo.Conn()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	query := fmt.Sprintf("SHOW VARIABLES LIKE '%s'", varName)
	_, rows, err := conn.Query(query)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", sql.ErrNoRows
	}

	var varNameFound, value string
	varNameFound = rows[0]["Variable_name"].(string)
	if varName != varNameFound {
		return "", errors.Errorf("未找到数据库参数 %s", varName)
	}
	value = rows[0]["Value"].(string)
	return value, nil
}

// CheckBinlogEnabled checks whether binlog is enabled for the current instance.
func (svc *DbInstanceSvcImpl) CheckBinlogEnabled(ctx context.Context) error {
	value, err := svc.getServerVariable(ctx, "log_bin")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ON" {
		return errors.Errorf("数据库未启用 binlog")
	}
	return nil
}

// CheckBinlogRowFormat checks whether the binlog format is ROW.
func (svc *DbInstanceSvcImpl) CheckBinlogRowFormat(ctx context.Context) error {
	value, err := svc.getServerVariable(ctx, "binlog_format")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ROW" {
		return errors.Errorf("binlog 格式 %s 不是行模式", value)
	}
	return nil
}

func runCmd(cmd *exec.Cmd) error {
	var stderr strings.Builder
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return errors.New(stderr.String())
	}
	return nil
}

func (svc *DbInstanceSvcImpl) execute(database string, sql string) error {
	args := []string{
		"--host", svc.dbInfo.Host,
		"--port", strconv.Itoa(svc.dbInfo.Port),
		"--user", svc.dbInfo.Username,
		"--password=" + svc.dbInfo.Password,
		"--execute", sql,
	}
	if len(database) > 0 {
		args = append(args, database)
	}

	cmd := exec.Command(mysqlPath(), args...)
	logx.Debug("execute sql using mysql binary: ", cmd.String())
	if err := runCmd(cmd); err != nil {
		logx.Errorf("运行 mysql 程序失败: %v", err)
		return errors.Wrap(err, "运行 mysql 程序失败")
	}
	return nil
}

// sortBinlogFiles will sort binlog files in ascending order by their numeric extension.
// For mysql binlog, after the serial number reaches 999999, the next serial number will not return to 000000, but 1000000,
// so we cannot directly use string to compare lexicographical order.
func sortBinlogFiles(binlogFiles []*BinlogFile) []*BinlogFile {
	var sorted []*BinlogFile
	sorted = append(sorted, binlogFiles...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Sequence < sorted[j].Sequence
	})
	return sorted
}

func parseBinlogEventTimeInLine(line string) (eventTs time.Time, found bool, err error) {
	// The target line starts with string like "#220421 14:49:26 server id 1"
	if !strings.Contains(line, "server id") {
		return time.Time{}, false, nil
	}
	if strings.Contains(line, "end_log_pos 0") {
		// https://github.com/mysql/mysql-server/blob/8.0/client/mysqlbinlog.cc#L1209-L1212
		// Fake events with end_log_pos=0 could be generated and we need to ignore them.
		return time.Time{}, false, nil
	}
	fields := strings.Fields(line)
	// fields should starts with ["#220421", "14:49:26", "server", "id", "1", "end_log_pos", "34794"]
	if len(fields) < 7 ||
		(len(fields[0]) != 7 || fields[2] != "server" || fields[3] != "id" || fields[5] != "end_log_pos") {
		return time.Time{}, false, errors.Errorf("found unexpected mysqlbinlog output line %q when parsing binlog event timestamp", line)
	}
	datetime, err := time.ParseInLocation("060102 15:04:05", fmt.Sprintf("%s %s", fields[0][1:], fields[1]), time.Local)
	if err != nil {
		return time.Time{}, false, err
	}
	return datetime, true, nil
}

func parseBinlogEventPosInLine(line string) (pos int64, found bool, err error) {
	// The mysqlbinlog output will contains a line starting with "# at 35065", which is the binlog event's start position.
	if !strings.HasPrefix(line, "# at ") {
		return 0, false, nil
	}
	// This is the line containing the start position of the binlog event.
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return 0, false, errors.Errorf("unexpected mysqlbinlog output line %q when parsing binlog event start position", line)
	}
	pos, err = strconv.ParseInt(fields[2], 10, 0)
	if err != nil {
		return 0, false, err
	}
	return pos, true, nil
}

// ParseBinlogName parses the numeric extension and the binary log base name by using split the dot.
// Examples:
//   - ("binlog.000001") => ("binlog", 1)
//   - ("binlog000001") => ("", err)
func ParseBinlogName(name string) (string, int64, error) {
	s := strings.Split(name, ".")
	if len(s) != 2 {
		return "", 0, errors.Errorf("failed to parse binlog extension, expecting two parts in the binlog file name %q but got %d", name, len(s))
	}
	seq, err := strconv.ParseInt(s[1], 10, 0)
	if err != nil {
		return "", 0, errors.Wrapf(err, "failed to parse the sequence number %s", s[1])
	}
	return s[0], seq, nil
}

// formatDateTime formats the timestamp to the local time string.
func formatDateTime(t time.Time) string {
	t = t.Local()
	return fmt.Sprintf("%d-%d-%d %d:%d:%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func mysqlPath() string {
	return config.Conf.Db.MysqlUtil.Mysql
}

func mysqldumpPath() string {
	return config.Conf.Db.MysqlUtil.MysqlDump
}

func mysqlbinlogPath() string {
	return config.Conf.Db.MysqlUtil.MysqlBinlog
}
