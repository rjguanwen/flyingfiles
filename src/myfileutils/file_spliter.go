package myfileutils

import (
	"os"
	"path"
	"path/filepath"
	"rjguanwen.cn/flyingfiles/src/fflog"
	"strconv"
	"sync"
	"time"
)

func init() {
	// 设置逻辑处理器数量
	//runtime.GOMAXPROCS(runtime.NumCPU())
}

//根据数据文件名来拆分数据文件
func SplitFileByFileNameSize(fileName string, splitSize int64) (fsi FileSummaryInfo, err error) {
	filePath := path.Join(AbsPath("file_store/out/"), fileName)
	fsi, err = SplitFileBySize(filePath, splitSize)
	return
}

// 按照指定的大小分割文件，文件大小至少为 1024
// 返回分割后的文件名数组
func SplitFileBySize(filePath string, splitSize int64) (fsi FileSummaryInfo, err error) {
	// 开始时间，用于统计耗时
	beginTime := time.Now().Unix()

	//计算源文件可以被分割为几个子文件
	fl, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil {
		fflog.Errorln("Read File Error:", err)
		return
	}
	stat, err := fl.Stat() //获取文件状态
	if err != nil {
		fflog.Errorln("File Stat Error:", err)
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
	// 设置等待协程数量，以保证协程同步
	// 每个子文件有一个协程进行处理
	var wg sync.WaitGroup
	wg.Add(int(splitFileNum))
	// 文件读取缓冲区大小
	fileReadBufSize := 1024 * 100
	var begin, end int64
	for i := 0; int64(i) < splitFileNum; i++ {
		// 读取开始位置
		begin = splitSize * int64(i)
		// 计算读取结束位置
		if i == int(splitFileNum)-1 {
			end = fileSize
		} else {
			end = begin + splitSize
		}
		// 启动协程，读取并生成临时文件
		go createSplitFile(filePath, fileReadBufSize, i, begin, end, &wg)
	}
	wg.Wait()
	// 文件拆分结束时间，用于统计耗时
	splitTime := time.Now().Unix()

	// 获取文件摘要信息
	paths, fileName := filepath.Split(filePath)    // 获取源文件路径及源文件名称
	sFileDir := path.Join(paths, fileName+"_info") // 获取子文件夹路径
	md5, err := HashFileMd5(filePath)              // 获取源文件 MD5 码
	if err != nil {
		fflog.Errorln("Get File MD5 Error:", err)
	}
	var sFileMD5s []string = make([]string, splitFileNum)
	// 获取子文件 MD5 码
	for i := 0; i < int(splitFileNum); i++ {
		sFilePath := path.Join(sFileDir, fileName+"_"+strconv.Itoa(i))
		sFileMD5s[i], err = HashFileMd5(sFilePath) // 获取子文件 MD5 码
		if err != nil {
			fflog.Errorln("Get sFile MD5 Error:", err)
		}
	}

	fsi = FileSummaryInfo{
		FileName:      fileName,
		Size:          fileSize,
		MD5:           md5,
		SplitFiles:    int(splitFileNum),
		SplitFilesMD5: sFileMD5s,
	}
	// 将文件摘要信息写入
	configFile := path.Join(sFileDir, fileName+"_config")
	WriteConfigYAML(configFile, fsi)
	// 结束时间，用于统计耗时
	endTime := time.Now().Unix()
	spt := splitTime - beginTime
	fflog.Infof("文件拆分耗时：%d 分 %d 秒 \n", spt/60, spt%60)
	sit := endTime - splitTime
	fflog.Infof("摘要信息生成耗时：%d 分 %d 秒 \n", sit/60, sit%60)
	return
}

// 分割文件
// 根据指定的大小，将文件进行分割
func createSplitFile(filePath string, fileReadBufSize int, sFileNum int, begin int64, end int64, wg *sync.WaitGroup) int {
	defer func() { //异常处理
		err := recover()
		if err != nil {
			fflog.Errorf("createSplitFile （%s: %d） error: %v \n", filePath, sFileNum, err)
			return
		}
	}()

	//打开源文件，准备提取数据，生成切分文件
	file, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		fflog.Errorln(filePath+"，file opn error!", err)
		return 0
	}
	file.Seek(begin, 0)                  //设定读取文件的位置
	buf := make([]byte, fileReadBufSize) //创建用于保存读取文件数据的切片
	// 创建子文件
	paths, fileName := filepath.Split(filePath)    // 获取源文件路径及源文件名称
	sFileDir := path.Join(paths, fileName+"_info") // 子文件夹路径
	//fmt.Println("====>", sFileDir)
	err = os.MkdirAll(sFileDir, os.ModePerm) // 创建子文件夹
	if err != nil {
		fflog.Errorln("MkDir Error:", err)
		return 0
	}
	sFilePath := path.Join(sFileDir, fileName+"_"+strconv.Itoa(sFileNum)) // 子文件路径
	//fmt.Println("====>", sFileNum)
	//fmt.Println("====>", sFilePath)
	sFile, err := os.Create(sFilePath) // 创建子文件
	if err != nil {
		fflog.Errorln("SplitFile Create Error:", err)
		return 0
	}
	defer sFile.Close()

	//读取并保存数据到子文件
	var saveDtaTolNum int = 0 // 记录保存成功的数据量
	for i := begin; i < end; i += int64(fileReadBufSize) {
		length, err := file.Read(buf) //读取数据到切片中
		if err != nil {
			fflog.Errorln("File Read Error:", err)
		}

		//判断读取的数据长度与切片的长度是否相等，如果不相等，表明文件读取已到末尾
		if length == fileReadBufSize {
			//判断此次读取的数据是否在当前协程读取的数据范围内，如果超出，则去除多余数据, 否则全部写入子文件
			if int64(i)+int64(fileReadBufSize) >= end {
				saveDataNum, err := sFile.Write(buf[:fileReadBufSize-int((int64(i)+int64(fileReadBufSize)-end))])
				if err != nil {
					fflog.Errorln("sFile Write Error:", err)
					return 0
				}
				saveDtaTolNum += saveDataNum
			} else {
				saveDataNum, err := sFile.Write(buf)
				if err != nil {
					fflog.Errorln("sFile Write Error:", err)
					return 0
				}
				saveDtaTolNum += saveDataNum
			}
		} else {
			saveDataNum, err := sFile.Write(buf[:length])
			if err != nil {
				fflog.Errorln("sFile Write Error:", err)
				return 0
			}
			saveDtaTolNum += saveDataNum
		}
	}
	// 计数+1
	wg.Done()
	fflog.Infoln("sFile " + strconv.Itoa(sFileNum) + " has write data " + strconv.Itoa(saveDtaTolNum))
	// 返回子文件写入的数据量
	return saveDtaTolNum
}
