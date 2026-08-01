package kfm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Tar 函数：创建 tar 归档
func Tar(sourceDir string, destFile string) error {
	// 创建目标文件
	file, err := os.Create(destFile)
	if err != nil {
		return fmt.Errorf("创建目标 tar 文件失败: %w", err)
	}
	defer file.Close()
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()
	// 遍历源目录，将文件添加到 tar 归档
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "") // 基于文件信息创建 tar header
		if err != nil {
			return fmt.Errorf("tar：创建 tar header 失败: %w", err)
		}
		// tar 归档中，文件路径需要相对于归档根目录，这里使用相对路径
		header.Name, err = filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("tar：获取相对路径失败: %w", err)
		}
		if header.Name == "." { // 根目录相对路径是 ".", 需要修正为空字符串
			header.Name = ""
		}
		if err := tarWriter.WriteHeader(header); err != nil { // 写入 header
			return fmt.Errorf("写入 tar header 失败: %w", err)
		}
		if info.IsDir() { // 如果是目录，header 已经写入，无需写入内容
			return nil
		}
		// 打开文件并写入文件内容 添加缓冲写入以提高性能
		srcFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("tar：打开源文件失败: %w", err)
		}
		defer srcFile.Close()
		buf := make([]byte, 32*1024) // 32KB buffer
		_, err = io.CopyBuffer(tarWriter, srcFile, buf)
		if err != nil {
			return fmt.Errorf("tar：复制文件内容到 tar 失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("tar：遍历源目录失败: %w", err)
	}
	return nil
}

// TarDecompress 函数：解压 tar 归档到指定目录
func TarDecompress(tarFile string, destDir string) error {
	file, err := os.Open(tarFile) // 打开 tar 文件
	if err != nil {
		return fmt.Errorf("打开 tar 文件失败: %w", err)
	}
	defer file.Close()
	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next() // 读取下一个 header
		if err == io.EOF {              // 文件结束
			break
		}
		if err != nil {
			return fmt.Errorf("读取 tar header 失败: %w", err)
		}
		targetPath := filepath.Join(destDir, header.Name) // 构建目标路径
		// 防止 Zip Slip 路径穿越攻击
		if !strings.HasPrefix(filepath.Clean(targetPath)+string(os.PathSeparator), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("tarDecompress: illegal file path: %s", header.Name)
		}
		if header.Typeflag == tar.TypeDir { // 如果是目录，创建目录
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("tarDecompress: 创建目录失败: %w", err)
			}
			continue
		}
		// 创建文件
		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("tarDecompress: 创建目标文件失败: %w", err)
		}

		// 复制文件内容
		if _, err := io.Copy(outFile, tarReader); err != nil {
			_ = outFile.Close()
			return fmt.Errorf("tarDecompress: 复制文件内容失败: %w", err)
		}
		_ = outFile.Close()
	}
	return nil
}

// Gzip 函数：压缩数据
func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf) // 创建 gzip writer，写入到 buffer
	_, err := gzipWriter.Write(data)   // 将原始数据写入 gzip writer 进行压缩
	if err != nil {
		return nil, fmt.Errorf("gzip write error: %w", err)
	}
	err = gzipWriter.Close() // **重要**: 关闭 writer，完成 gzip 流
	if err != nil {
		return nil, fmt.Errorf("gzip close error: %w", err)
	}
	return buf.Bytes(), nil // 返回 buffer 中的压缩数据
}

// GzipDecompress 函数：解压缩数据
func GzipDecompress(compressedData []byte) ([]byte, error) {
	buf := bytes.NewReader(compressedData) // 从压缩数据创建 reader
	gzipReader, err := gzip.NewReader(buf) // 创建 gzip reader，从 buffer 读取压缩数据
	if err != nil {
		return nil, fmt.Errorf("gzipDecompress: gzip reader creation error: %w", err)
	}
	defer gzipReader.Close()                        // 确保 reader 在使用后关闭
	decompressedData, err := io.ReadAll(gzipReader) // 读取所有解压缩后的数据
	if err != nil {
		return nil, fmt.Errorf("gzipDecompress: gzip read error: %w", err)
	}
	return decompressedData, nil // 返回解压缩后的数据
}

// Zstd 函数：使用 zstd 压缩数据
func Zstd(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zstdWriter, err := zstd.NewWriter(&buf) // 创建 zstd writer，写入到 buffer
	if err != nil {
		return nil, fmt.Errorf("zstd writer 创建失败: %w", err)
	}
	defer zstdWriter.Close()        // 确保 writer 关闭
	_, err = zstdWriter.Write(data) // 将原始数据写入 zstd writer 进行压缩
	if err != nil {
		return nil, fmt.Errorf("zstd 写入错误: %w", err)
	}
	err = zstdWriter.Close() // 完成 zstd 流
	if err != nil {
		return nil, fmt.Errorf("zstd 关闭错误: %w", err)
	}
	return buf.Bytes(), nil // 返回 buffer 中的压缩数据
}

// ZstdDecompress 函数：使用 zstd 解压缩数据
func ZstdDecompress(compressedData []byte) ([]byte, error) {
	buf := bytes.NewReader(compressedData) // 从压缩数据创建 reader
	zstdReader, err := zstd.NewReader(buf) // 创建 zstd reader，从 buffer 读取压缩数据
	if err != nil {
		return nil, fmt.Errorf("zstd reader 创建失败: %w", err)
	}
	defer zstdReader.Close()                        // 确保 reader 关闭
	decompressedData, err := io.ReadAll(zstdReader) // 读取所有解压缩后的数据
	if err != nil {
		return nil, fmt.Errorf("zstd 读取错误: %w", err)
	}
	return decompressedData, nil // 返回解压缩后的数据
}

// Lz4 函数：使用 lz4 压缩数据
func Lz4(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	lz4Writer := lz4.NewWriter(&buf) // 创建 lz4 writer，写入到 buffer
	defer lz4Writer.Close()          // 确保 writer 关闭
	_, err := lz4Writer.Write(data)  // 将原始数据写入 lz4 writer 进行压缩
	if err != nil {
		return nil, fmt.Errorf("lz4 写入错误: %w", err)
	}
	err = lz4Writer.Close() // 完成 lz4 流
	if err != nil {
		return nil, fmt.Errorf("lz4 关闭错误: %w", err)
	}
	return buf.Bytes(), nil // 返回 buffer 中的压缩数据
}

// Lz4Decompress 函数：使用 lz4 解压缩数据
func Lz4Decompress(compressedData []byte) ([]byte, error) {
	buf := bytes.NewReader(compressedData)         // 从压缩数据创建 reader
	lz4Reader := lz4.NewReader(buf)                // 创建 lz4 reader，从 buffer 读取压缩数据
	decompressedData, err := io.ReadAll(lz4Reader) // 读取所有解压缩后的数据
	if err != nil {
		return nil, fmt.Errorf("lz4 读取错误: %w", err)
	}
	return decompressedData, nil // 返回解压缩后的数据
}

// Snappy 函数：使用 snappy 压缩数据
func Snappy(data []byte) ([]byte, error) {
	compressedData := snappy.Encode(nil, data) // 使用 snappy.Encode 直接压缩，无需 Writer
	return compressedData, nil                 // snappy.Encode 返回压缩后的 []byte，以及 error (但目前 snappy.Encode 不直接返回 error)
}

// SnappyDecompress 函数：使用 snappy 解压缩数据
func SnappyDecompress(compressedData []byte) ([]byte, error) {
	decompressedData, err := snappy.Decode(nil, compressedData) // 使用 snappy.Decode 解压缩，无需 Reader
	if err != nil {
		return nil, fmt.Errorf("snappy 解压缩失败: %w", err)
	}
	return decompressedData, nil
}
