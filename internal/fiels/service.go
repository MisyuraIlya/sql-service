package fiels

import (
	"os"
	"path/filepath"
	"strings"
)

type FileService struct{}

func NewFilesService() *FileService {
	return &FileService{}
}

func (fs *FileService) ListImages(folderPath string) ([]string, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
	}

	var images []string
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if allowedExts[ext] {
				images = append(images, entry.Name())
			}
		}
	}

	return images, nil
}
