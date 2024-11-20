package base

import (
	"embed"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// SQLStatement 结构体用于存储解析后的 SQL 语句及其注释
type SQLStatement struct {
	Comment string
	SQL     string
}

var sqlMap = make(map[string]string)

func RegisterSql(fs embed.FS) error {
	return walkDir(fs, ".", func(fp string, data []byte) error {
		if filepath.Ext(fp) != ".sql" {
			return nil
		}

		fileNameWithExt := path.Base(fp)
		sqls, err := parseSQL(string(data))
		if err != nil {
			return err
		}

		filename := strings.TrimSuffix(fileNameWithExt, path.Ext(fileNameWithExt))
		for _, sql := range sqls {
			sqlMap[filename+"."+strings.TrimSpace(sql.Comment)] = strings.TrimSpace(sql.SQL)
		}

		return nil
	})
}

func GetSQL(filename, stmt string) string {
	return sqlMap[filename+"."+stmt]
}

// walkDir 递归遍历目录
func walkDir(fsys fs.FS, path string, callback func(filePath string, data []byte) error) error {
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			// 递归遍历子目录
			if err := walkDir(fsys, entryPath, callback); err != nil {
				return err
			}
		} else {
			// 读取文件内容
			data, err := fs.ReadFile(fsys, entryPath)
			if err != nil {
				return err
			}
			if err := callback(entryPath, data); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseSQL 解析带有注释的 SQL 语句
func parseSQL(sql string) ([]SQLStatement, error) {
	var statements []SQLStatement
	lines := strings.Split(sql, "\n")

	var currentComment string
	var currentSQL string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "--") {
			// 处理单行注释
			if currentSQL != "" {
				statements = append(statements, SQLStatement{Comment: currentComment, SQL: strings.TrimRight(currentSQL, " ")})
				currentComment = ""
				currentSQL = ""
			}
			currentComment += strings.TrimPrefix(trimmedLine, "--") + "\n"
			continue
		}

		if trimmedLine == "" {
			continue
		}

		currentSQL += line + " "
		if strings.HasSuffix(trimmedLine, ";") {
			statements = append(statements, SQLStatement{Comment: currentComment, SQL: strings.TrimRight(currentSQL, " ")})
			currentComment = ""
			currentSQL = ""
		}
	}

	// 处理最后一段未结束的 SQL 语句
	if currentSQL != "" {
		statements = append(statements, SQLStatement{Comment: currentComment, SQL: strings.TrimRight(currentSQL, " ")})
	}

	return statements, nil
}
