package myfileutils

import (
	"bufio"
	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"os"
	"path"
	"strconv"
)

// 将子文件合并为数据文件
// 先进行子文件校验，然后合并，最后对合并后的数据文件校验
func MergeSplitFileAndCheck(fileName string, ffi FileFlyInfo, taskList []DownloadTask) (isOK bool, err error) {
	// 校验子文件下载是否完整
	isOK, err = checkSplitFiles(fileName, ffi, taskList) //首先校验子文件下载是否OK
	if !isOK {
		log.Error("子文件校验出错：", err)
		return
	}
	// 组织子文件路径列表，完成合并
	splitFileNum := ffi.SplitFiles
	var splitFilePaths []string = make([]string, splitFileNum)              // 子文件路径列表
	splitFileDir := path.Join(AbsPath("/file_store/in/"), fileName+"_info") // 获取子文件夹绝对路径
	for i := 0; i < int(splitFileNum); i++ {
		splitFilePaths[i] = path.Join(splitFileDir, fileName+"_"+strconv.Itoa(i))
	}
	targetFilePath := path.Join(AbsPath("/file_store/in/"), fileName)
	isOK, err = fileMerge(splitFilePaths, targetFilePath, false)
	if !isOK {
		log.Error("子文件合并出错：", err)
		return
	}
	// 校验目标文件MD5
	targetFileMD5, err := HashFileMd5(targetFilePath)
	if targetFileMD5 == ffi.MD5 {
		isOK = true
		log.Infof("数据文件（%s）校验通过！", fileName)
	}
	return
}

// 校验子文件，通过大小进行校验
func checkSplitFiles(fileName string, ffi FileFlyInfo, taskList []DownloadTask) (isOK bool, err error) {
	splitFileNum := ffi.SplitFiles
	// 获取文件大小
	splitFileDir := path.Join(AbsPath("/file_store/in/"), fileName+"_info") // 获取子文件夹路径
	var sFileSizes []int64 = make([]int64, splitFileNum)
	// 获取子文件大小
	for i := 0; i < int(splitFileNum); i++ {
		sFilePath := path.Join(splitFileDir, fileName+"_"+strconv.Itoa(i))
		fl, err := os.OpenFile(sFilePath, os.O_RDWR, 0666)
		if err != nil {
			log.Error("Read File Error:", err)
			break
		}
		stat, err := fl.Stat() //获取文件状态
		if err != nil {
			log.Error("File Stat Error:", err)
			break
		}
		// 获取文件大小
		fileSize := stat.Size()
		sFileSizes[i] = fileSize
		fl.Close()
	}
	isSameSize := true
	for i, tmpTask := range taskList { // 循环比较每个子文件的大小否一致
		if tmpTask.End-tmpTask.Begin != sFileSizes[i] {
			isSameSize = false
			break
		}
	}
	if isSameSize {
		isOK = true
	}
	return isOK, nil
}

// 将一系列文件合并为一个文件
func fileMerge(sourceFileList []string, targetFilePath string, removeSourceFiles bool) (isOK bool, err error) {
	// 打开目标文件
	targetFile, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Error("Can not open file %s: %v", targetFilePath, err)
		return false, err
	}
	bWriter := bufio.NewWriter(targetFile)

	readBuffer := make([]byte, 1024)        // 文件读取数据缓冲
	for _, sfPath := range sourceFileList { // 循环目标文件
		fp, err := os.Open(sfPath)
		if err != nil {
			fmt.Printf("Can not open file %s: %v", sfPath, err)
			return false, err
		}
		bReader := bufio.NewReader(fp)
		for {
			readCount, readErr := bReader.Read(readBuffer)
			if readErr == io.EOF {
				break
			} else {
				bWriter.Write(readBuffer[:readCount])
			}
		}
		bWriter.Flush()
		fp.Close() // 关闭文件
	}
	targetFile.Close() //关闭目标文件
	// 如果需要删除文件，则循环删除
	if removeSourceFiles {
		for _, sfPath := range sourceFileList {
			err := os.Remove(sfPath)
			if err != nil {
				//如果删除失败则输出错误详细信息
				log.Error("文件合并完成，删除文件时发生错误：%s: %v", sfPath, err)
				return false, err
			}
		}
	}
	isOK = true
	return isOK, nil
}
