package fiels

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type FileService struct{}

func (service *FileService) FindFile(dto FindFileDto) (string, error) {
	if dto.FileName != "" {
		return recursiveFinder(dto.FileName)
	} else {
		return pathFinder(dto.Path)
	}
}

func recursiveFinder(fileName string) (string, error) {
	var foundPath string

	errFound := errors.New("file found")

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == fileName {
			foundPath = path
			return errFound
		}
		return nil
	})

	if err != nil && err != errFound {
		return "", err
	}
	if foundPath != "" {
		return foundPath, nil
	}
	return "", fmt.Errorf("file %s not found", fileName)
}

// pathFinder checks if the file exists at the given path.
func pathFinder(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("path %s does not exist", path)
	} else if err != nil {
		return "", err
	}
	return path, nil
}

// Example usage
func main() {
	service := FileService{}

	// Example 1: Search by file name
	dto1 := FindFileDto{FileName: "target.txt"}
	path1, err := service.FindFile(dto1)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Found file at:", path1)
	}

	// Example 2: Check specific file path
	dto2 := FindFileDto{Path: "./somefile.txt"}
	path2, err := service.FindFile(dto2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("File exists at:", path2)
	}
}
