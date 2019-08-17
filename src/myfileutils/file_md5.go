package myfileutils

import (
	"crypto/md5"
	"encoding/hex"
	log "github.com/cihub/seelog"
	"io"
	"os"
)

// 计算文件的 md5 码
func HashFileMd5(filePath string) (md5Str string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("计算文件md5码过程中，文件打开错误%s: %v", filePath, err)
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		log.Errorf("计算文件md5码过程错误%s: %v", filePath, err)
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5Str = hex.EncodeToString(hashInBytes)
	return md5Str, nil
}
