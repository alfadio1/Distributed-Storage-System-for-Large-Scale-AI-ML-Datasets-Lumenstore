package client

import "os"

func getFileInfo(filePath string) (os.FileInfo, error) {
	return os.Stat(filePath)
}
