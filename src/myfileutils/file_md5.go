package myfileutils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// 计算文件的 md5 码
func HashFileMd5(filePath string) (md5Str string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()
	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return
	}
	hashInBytes := hash.Sum(nil)[:16]
	md5Str = hex.EncodeToString(hashInBytes)
	return
}
