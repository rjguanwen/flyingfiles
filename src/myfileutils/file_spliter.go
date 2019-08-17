package myfileutils

import (
	log "github.com/cihub/seelog"
	"os"
	"path"
)

// 生成文件发送信息
func GenFileFilyInfo(fileName string, splitSize int64) (ffi FileFlyInfo, err error) {
	filePath := path.Join(AbsPath("file_store/out/"), fileName)
	// 计算源文件可以被分割为几个子文件
	fl, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil {
		log.Error("Read File Error:", err)
		return
	}
	stat, err := fl.Stat() //获取文件状态
	if err != nil {
		log.Error("File Stat Error:", err)
	}
	var fileSize int64
	// 获取文件大小
	fileSize = stat.Size()
	fl.Close()
	// 子文件最小为1024
	if splitSize < 1024 {
		splitSize = 1024
	}
	// 计算子文件个数
	splitFileNum := fileSize / splitSize
	if fileSize > splitFileNum*splitSize {
		splitFileNum += 1
	}
	md5, err := HashFileMd5(filePath) // 获取源文件 MD5 码

	ffi = FileFlyInfo{
		FileName:      fileName,
		Size:          fileSize,
		MD5:           md5,
		SplitFiles:    int(splitFileNum),
		SplitFileSize: splitSize,
	}
	return
}
