package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	getFolderSize "github.com/markthree/go-get-folder-size/src"
	"github.com/schollz/progressbar/v3"
)

type FileInfo struct {
	Name           string
	Size           int64
	Path           string
	LastAccessTime time.Time
}

func prettyPrintSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	var i int
	for i = 0; size >= 1024 && i < len(units)-1; i++ {
		size /= 1024
	}
	return fmt.Sprintf("%d%s", size, units[i])
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <root directory>")
		return
	}

	root := os.Args[1]
	fileList := []FileInfo{}

	excludeDirs := []string{
		".git", "node_modules", "vendor",
		"bin", "obj", "packages", "dist",
		"build", "out", "target", "release",
		"logs", "tmp", "temp", "cache", "data",
		"backup", "backups", "uploads", "download",
		"downloads", "media", "assets", "static",
		"test", "tests", "testing", "example",
		"examples", "doc", "docs", "__pycache__",
		"Lib", "lib", "Library", "bin", "obj",
		".vs", ".vscode", ".idea", ".gitignore",
		"Usuarios", "ProgramData",
		"Program Files (x86)", "Windows", "Windows.old",
		"Archivos de programa", "Archivos de programa (x86)",
		"Windows Defender Advanced Threat Protection",
		"Program Files", "System Volume Information",
		"Packages",
	}

	totalSize := getFolderSize.LooseParallel(root)

	bar := progressbar.DefaultBytes(
		totalSize,
		"Scanning files...",
	)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
		}
		if info.IsDir() {
			for _, dir := range excludeDirs {
				if strings.EqualFold(filepath.Base(path), dir) {
					return filepath.SkipDir
				}
			}
		} else {
			// Check if the file has been modified within the last 1.5 years
			stat, err := os.Stat(path)
			if err != nil {
				return filepath.SkipDir
			}
			fileTime := stat.Sys().(*syscall.Win32FileAttributeData).LastAccessTime
			accesTime := time.Unix(0, fileTime.Nanoseconds())
			cutoffTime := time.Now().AddDate(-1, -6, 0) // 1.5 years ago
			if !accesTime.After(cutoffTime) {
				fileList = append(fileList, FileInfo{
					Name:           info.Name(),
					Size:           info.Size(),
					Path:           path,
					LastAccessTime: accesTime,
				})
			}
			bar.Add64(info.Size())
		}

		return nil
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].Size > fileList[j].Size
	})

	file, err := os.Create("file_sizes.csv")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"Name", "Size", "Path", "Last Access Time"}
	writer.Write(header)

	for _, file := range fileList {
		// Remove the root directory from the file path
		relPath, err := filepath.Rel(root, file.Path)
		if err != nil {
			fmt.Println(err)
			continue
		}
		row := []string{file.Name, prettyPrintSize(file.Size), relPath, file.LastAccessTime.Format("2006-01-02")}
		writer.Write(row)
	}

	fmt.Println("CSV file saved successfully.")

	var totalSelectedSize int64
	for _, file := range fileList {
		totalSelectedSize += file.Size
	}
	fmt.Println("Total size:", prettyPrintSize(totalSelectedSize))
}
