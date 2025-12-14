package util

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateDirectory 创建目录
func CreateDirectory(path string) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CreateFile 创建文件
func CreateFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := CreateDirectory(dir); err != nil {
			return nil, err
		}
	}
	
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// RemoveFile 删除文件
func RemoveFile(path string) error {
	return os.Remove(path)
}

// RenameFile 重命名文件
func RenameFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// ReadFile 读取文件
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile 写入文件
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := CreateDirectory(dir); err != nil {
			return err
		}
	}
	
	return os.WriteFile(path, data, 0644)
}

// GzipReader gzip读取器包装
type GzipReader struct {
	*gzip.Reader
}

// NewGzipReader 创建gzip读取器
func NewGzipReader(r io.Reader) (*GzipReader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &GzipReader{Reader: gz}, nil
}

// CleanFilename 清理文件名，移除非法字符
func CleanFilename(filename string) string {
	// Windows不允许的字符
	illegalChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\"}
	
	for _, char := range illegalChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}
	
	// 移除前后空格和点
	filename = strings.Trim(filename, " .")
	
	// 如果文件名为空或为空字符串，返回默认名称
	if filename == "" {
		return "untitled"
	}
	
	return filename
}

// EnsureExt 确保文件有指定的扩展名
func EnsureExt(filename, ext string) string {
	if !strings.HasSuffix(strings.ToLower(filename), strings.ToLower(ext)) {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		filename = filename + ext
	}
	return filename
}

// ReplaceFileExtension 替换文件扩展名
func ReplaceFileExtension(filename, newExt string) string {
	ext := filepath.Ext(filename)
	if ext != "" {
		filename = filename[:len(filename)-len(ext)]
	}
	
	if !strings.HasPrefix(newExt, ".") {
		newExt = "." + newExt
	}
	
	return filename + newExt
}

// GetTempPath 获取临时文件路径
func GetTempPath(originalPath string) string {
	ext := filepath.Ext(originalPath)
	base := originalPath[:len(originalPath)-len(ext)]
	return base + ".tmp"
}

// FormatBytes 格式化字节数为可读字符串
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化时长
func FormatDuration(seconds int) string {
	if seconds < 0 {
		return "00:00"
	}
	
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

// SanitizePath 清理路径
func SanitizePath(path string) string {
	// 统一使用正斜杠
	path = strings.ReplaceAll(path, "\\", "/")
	
	// 移除重复的斜杠
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	
	return path
}

// GetRelativePath 获取相对路径
func GetRelativePath(basePath, targetPath string) (string, error) {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return "", err
	}
	return SanitizePath(rel), nil
}

// IsDirectory 判断路径是否为目录
func IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}