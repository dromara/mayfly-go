//go:build e2e

package mysql

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/suite"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/infrastructure/persistence"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	instanceIdTest          = 0
	backupIdTest            = 0
	dbNameBackupTest        = "test-backup-01"
	tableNameBackupTest     = "test-backup"
	tableNameRestorePITTest = "test-restore-pit"
	tableNameNoBackupTest   = "test-not-backup"
)

type DbInstanceSuite struct {
	suite.Suite
	repositories *repository.Repositories
	instanceSvc  *DbProgramMysql
	dbConn       *dbi.DbConn
}

func (s *DbInstanceSuite) SetupSuite() {
	if err := chdir("mayfly-go", "server"); err != nil {
		panic(err)
	}
	dbInfo := dbi.DbInfo{
		Type:     dbi.DbTypeMysql,
		Host:     "localhost",
		Port:     3306,
		Username: "test",
		Password: "test",
	}
	dbConn, err := dbInfo.Conn(dbi.GetMeta(dbi.DbTypeMysql))
	s.Require().NoError(err)
	s.dbConn = dbConn
	s.repositories = &repository.Repositories{
		Instance:       persistence.NewInstanceRepo(),
		Backup:         persistence.NewDbBackupRepo(),
		BackupHistory:  persistence.NewDbBackupHistoryRepo(),
		Restore:        persistence.NewDbRestoreRepo(),
		RestoreHistory: persistence.NewDbRestoreHistoryRepo(),
		Binlog:         persistence.NewDbBinlogRepo(),
		BinlogHistory:  persistence.NewDbBinlogHistoryRepo(),
	}
	s.instanceSvc = NewDbProgramMysql(s.dbConn)
	var extName string
	if runtime.GOOS == "windows" {
		extName = ".exe"
	}
	path := "db/mysql/bin"
	s.instanceSvc.mysqlBin = &config.MysqlBin{
		Path:            filepath.Join(path),
		MysqlPath:       filepath.Join(path, "mysql"+extName),
		MysqldumpPath:   filepath.Join(path, "mysqldump"+extName),
		MysqlbinlogPath: filepath.Join(path, "mysqlbinlog"+extName),
	}
	s.instanceSvc.backupPath = "db/backup"
}

func (s *DbInstanceSuite) TearDownSuite() {
	if s.dbConn != nil {
		s.dbConn.Close()
		s.dbConn = nil
	}
}

func (s *DbInstanceSuite) SetupTest() {
	sql := strings.Builder{}
	require := s.Require()
	sql.WriteString(fmt.Sprintf("drop database if exists `%s`;", dbNameBackupTest))
	sql.WriteString(fmt.Sprintf("create database `%s`;", dbNameBackupTest))
	require.NoError(s.instanceSvc.execute("", sql.String()))
}

func (s *DbInstanceSuite) TearDownTest() {
	require := s.Require()
	sql := fmt.Sprintf("drop database if exists `%s`", dbNameBackupTest)
	require.NoError(s.instanceSvc.execute("", sql))

	_ = os.RemoveAll(s.instanceSvc.getDbInstanceBackupRoot(instanceIdTest))
}

func (s *DbInstanceSuite) TestBackup() {
	history := &entity.DbBackupHistory{
		DbName: dbNameBackupTest,
		Uuid:   dbNameBackupTest,
	}
	history.Id = backupIdTest
	s.testBackup(history)
}

func (s *DbInstanceSuite) testBackup(backupHistory *entity.DbBackupHistory) {
	require := s.Require()
	binlogInfo, err := s.instanceSvc.Backup(context.Background(), backupHistory)
	require.NoError(err)

	fileName := filepath.Join(s.instanceSvc.getDbBackupDir(s.dbConn.Info.InstanceId, backupHistory.Id), dbNameBackupTest+".sql.gz")
	_, err = os.Stat(fileName)
	require.NoError(err)

	backupHistory.BinlogFileName = binlogInfo.FileName
	backupHistory.BinlogSequence = binlogInfo.Sequence
	backupHistory.BinlogPosition = binlogInfo.Position
}

func TestDbInstance(t *testing.T) {
	suite.Run(t, &DbInstanceSuite{})
}

func (s *DbInstanceSuite) TestRestoreDatabase() {
	backupHistory := &entity.DbBackupHistory{
		DbName: dbNameBackupTest,
		Uuid:   dbNameBackupTest,
	}

	s.createTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.testBackup(backupHistory)
	s.createTable(dbNameBackupTest, tableNameNoBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "")
	s.testRestore(backupHistory)
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "运行 mysql 程序失败")
}

func (s *DbInstanceSuite) TestRestorePontInTime() {
	backupHistory := &entity.DbBackupHistory{
		DbName: dbNameBackupTest,
		Uuid:   dbNameBackupTest,
	}

	s.createTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.testBackup(backupHistory)

	s.createTable(dbNameBackupTest, tableNameRestorePITTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "")
	time.Sleep(time.Second)

	// 首次恢复数据库
	firstTargetTime := time.Now()
	s.dropTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "运行 mysql 程序失败")
	s.createTable(dbNameBackupTest, tableNameNoBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "")

	s.testRestore(backupHistory)
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "运行 mysql 程序失败")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "运行 mysql 程序失败")

	s.testReplayBinlog(backupHistory, firstTargetTime)
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "运行 mysql 程序失败")

	s.testBackup(backupHistory)

	// 再次恢复数据库
	secondTargetTime := time.Now()
	s.dropTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "运行 mysql 程序失败")
	s.dropTable(dbNameBackupTest, tableNameRestorePITTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "运行 mysql 程序失败")
	s.createTable(dbNameBackupTest, tableNameNoBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "")

	s.testRestore(backupHistory)
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "运行 mysql 程序失败")

	s.testReplayBinlog(backupHistory, secondTargetTime)
	s.selectTable(dbNameBackupTest, tableNameBackupTest, "")
	s.selectTable(dbNameBackupTest, tableNameRestorePITTest, "")
	s.selectTable(dbNameBackupTest, tableNameNoBackupTest, "运行 mysql 程序失败")
}

func (s *DbInstanceSuite) testReplayBinlog(backupHistory *entity.DbBackupHistory, targetTime time.Time) {
	require := s.Require()
	binlogFilesOnServerSorted, err := s.instanceSvc.GetSortedBinlogFilesOnServer(context.Background())
	require.NoError(err)
	require.True(len(binlogFilesOnServerSorted) > 0, "binlog 文件不存在")
	for i, bf := range binlogFilesOnServerSorted {
		if bf.Name == backupHistory.BinlogFileName {
			binlogFilesOnServerSorted = binlogFilesOnServerSorted[i:]
			break
		}
		require.Less(i, len(binlogFilesOnServerSorted), "binlog 文件没找到")
	}
	err = s.instanceSvc.downloadBinlogFilesOnServer(context.Background(), binlogFilesOnServerSorted, true)
	require.NoError(err)

	binlogFileLast := binlogFilesOnServerSorted[len(binlogFilesOnServerSorted)-1]
	position, err := s.instanceSvc.GetBinlogEventPositionAtOrAfterTime(context.Background(), binlogFileLast.Name, targetTime)
	require.NoError(err)
	binlogHistories := make([]*entity.DbBinlogHistory, 0, 2)
	binlogHistoryBackup := &entity.DbBinlogHistory{
		FileName: backupHistory.BinlogFileName,
		Sequence: backupHistory.BinlogSequence,
	}
	binlogHistories = append(binlogHistories, binlogHistoryBackup)
	if binlogHistoryBackup.Sequence != binlogFileLast.Sequence {
		require.Equal(binlogFilesOnServerSorted[0].Sequence, binlogHistoryBackup.Sequence)
		binlogHistoryLast := &entity.DbBinlogHistory{
			FileName: binlogFileLast.Name,
			Sequence: binlogFileLast.Sequence,
		}
		binlogHistories = append(binlogHistories, binlogHistoryLast)
	}

	restoreInfo := &dbi.RestoreInfo{
		BackupHistory:   backupHistory,
		BinlogHistories: binlogHistories,
		StartPosition:   backupHistory.BinlogPosition,
		TargetPosition:  position,
		TargetTime:      targetTime,
	}
	err = s.instanceSvc.ReplayBinlog(context.Background(), dbNameBackupTest, dbNameBackupTest, restoreInfo)
	require.NoError(err)
}

func (s *DbInstanceSuite) testRestore(backupHistory *entity.DbBackupHistory) {
	require := s.Require()
	err := s.instanceSvc.RestoreBackupHistory(context.Background(), backupHistory.DbName, backupHistory.DbBackupId, backupHistory.Uuid)
	require.NoError(err)
}

func (s *DbInstanceSuite) selectTable(database, tableName, wantErr string) {
	require := s.Require()
	sql := fmt.Sprintf("select * from`%s`;", tableName)
	err := s.instanceSvc.execute(database, sql)
	if len(wantErr) > 0 {
		require.ErrorContains(err, wantErr)
		return
	}
	require.NoError(err)
}

func (s *DbInstanceSuite) createTable(database, tableName, wantErr string) {
	require := s.Require()
	sql := fmt.Sprintf("create table `%s`(id int);", tableName)
	err := s.instanceSvc.execute(database, sql)
	if len(wantErr) > 0 {
		require.ErrorContains(err, wantErr)
		return
	}
	require.NoError(err)
}

func (s *DbInstanceSuite) dropTable(database, tableName, wantErr string) {
	require := s.Require()
	sql := fmt.Sprintf("drop table `%s`;", tableName)
	err := s.instanceSvc.execute(database, sql)
	if len(wantErr) > 0 {
		require.ErrorContains(err, wantErr)
		return
	}
	require.NoError(err)
}

func chdir(projectName string, subdir ...string) error {
	subdir = append([]string{"/", projectName}, subdir...)
	suffix := filepath.Join(subdir...)
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if strings.HasSuffix(wd, suffix) {
			if err := os.Chdir(wd); err != nil {
				return err
			}
			return nil
		}
		upper := filepath.Join(wd, "..")
		if upper == wd {
			return errors.New(fmt.Sprintf("not found directory: %s", suffix[1:]))
		}
		wd = upper
	}
}
