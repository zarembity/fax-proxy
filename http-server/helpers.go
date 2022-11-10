package http_server

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func generateFilename(path, uid, oldFilename string) string {
	extension := filepath.Ext(oldFilename)
	return fmt.Sprintf("%s/%s%s", path, uid, extension)
}

//todo дописать валидацию при необходимости
func validatePhoneNumber(phoneNumber string) bool {
	if phoneNumber == "" {
		return false
	}

	return true
}

func ContainsItemSlice(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func deleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Errorf("http server: deleteFile return error: %s", err)
	}
}
