package main

import (
	"bufio"
	"fmt"
	log "github.com/cihub/seelog"
	. "github.com/rjguanwen/flyingfiles/src/myfileutils"
	"github.com/rjguanwen/flyingfiles/src/myutil"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"
)

const (
	prefix_debug   = "DEBUG:"
	prefix_info    = "INFO:"
	prefix_warning = "WARNING:"
	prefix_error   = "ERROR:"
)

func init() {
	// 设置逻辑处理器数量
	runtime.GOMAXPROCS(4)

}

func main() {
	//log.Debug(Request4File)
	testConfig := `
<seelog type="sync">
	<outputs formatid="main">
		<console/>
	</outputs>
	<formats>
		<format id="main" format="%Date/%Time [%Level] %RelFile:%Line %Msg%n"/>
	</formats>
</seelog>`

	logger, _ := log.LoggerFromConfigAsBytes([]byte(testConfig))
	log.ReplaceLogger(logger)

	var (
		host          = "127.0.0.1"               //服务端IP
		port          = "9090"                    //服务端端口
		remote        = host + ":" + port         //构造连接串
		fileName      = "weibo_data.txt"          // 请求数据文件名
		mergeFileName = "download_weibo_data.txt" //本地保存数据文件名
		//fileName      = "机器学习.zip"          // 请求数据文件名
		//mergeFileName = "download_机器学习.zip" //本地保存数据文件名
		//fileName      = "hyr.mkv"          // 请求数据文件名
		//mergeFileName = "download_hyr.mkv" //本地保存数据文件名
	)

	//获取参数信息。
	//参数顺序：
	// 1：请求数据文件名
	// 2：本地保存数据文件名
	for index, sargs := range os.Args {
		switch index {
		case 1:
			fileName = sargs
			mergeFileName = sargs
		case 2:
			mergeFileName = sargs
		}
	}

	fmt.Printf("请输入服务端IP: ")
	reader := bufio.NewReader(os.Stdin)
	ipdata, _, _ := reader.ReadLine()

	host = string(ipdata)
	//host = "127.0.0.1"
	remote = host + ":" + port
	beginTime := time.Now().Unix()

	// 请求与服务器连接
	con, err := net.Dial("tcp", remote)
	if err != nil {
		log.Debug("服务器连接失败.")
		os.Exit(-1)
		return
	}
	//log.Debug("连接已建立.文件请求发送中...")
	//log.Debug("客户端请求包：", strconv.Itoa(myutil.Request4File)+fileName)
	in, err := con.Write([]byte(strconv.Itoa(myutil.Request4File) + fileName)) //向服务器发送数据文件请求
	if err != nil {
		log.Errorf("向服务器发送数据错误: %d", in)
		os.Exit(0)
	}
	var msg = make([]byte, 1024*100) //创建读取服务端信息的切片
	lengthh, err := con.Read(msg)    //获取服务器返回信息
	if err != nil {
		log.Errorf("读取服务器数据错误.", lengthh)
		os.Exit(0)
	}
	// 关闭链接
	con.Close()
	//log.Debug("接收到的数据长度==>", lengthh)
	recvFlag := string(msg[0:1])
	//log.Debug("==>", string(msg[:]))
	if recvFlag == strconv.Itoa(myutil.FileReady) {
		// 文件已就绪
		sessionId := string(msg[1 : myutil.SessionIdLength+1])
		//log.Debug("sessionId===>>", sessionId)
		recvData := string(msg[myutil.SessionIdLength+1 : lengthh])
		//log.Debug("服务端返回信息：", recvData)
		//解析返回的数据,将其转化为文件摘要信息对象
		ffi := StringToFFI(recvData)
		fileSize := ffi.Size
		//md5 := ffi.MD5
		splitFils := ffi.SplitFiles
		splitFileSize := ffi.SplitFileSize
		// 将文件摘要信息写入摘要文件
		WriteFileConfigYAML(fileName, ffi)

		// 组织子文件开始结束位置数组，方便下载
		var sFilesTaskList = make([]DownloadTask, splitFils)
		var begin, end int64
		// 计算每个子文件的开始与结束位置
		for i := 0; i < splitFils; i++ {
			begin = splitFileSize * int64(i)
			if i != splitFils-1 {
				end = splitFileSize * (int64(i) + 1)
			} else {
				end = fileSize
			}
			tmpDownloadTask := DownloadTask{
				FileName: fileName,
				Seq:      i,
				Begin:    begin,
				End:      end,
			}
			sFilesTaskList[i] = tmpDownloadTask
		}

		// 采用工作池的方式开展下载任务
		// - 将每个子文件下载看做一个任务，放入任务管道
		// - 根据指定的最大协程数创建工作协程
		// - 每个工作协程到循环到任务管道里面获取任务并处理之（下载子文件）
		// - 每完成一个任务，工作协程将下载结果写入结果信号管道
		// - 当任务管道中无法获取任务时，工作协程认为工作已经完成并将退出信号写入退出信号管道
		// - 通过循环取退出信号管道来确定所有工作协程均执行完毕
		maxRoutineNum := 10            // 同一文件的并发下载协程最大为10
		if maxRoutineNum > splitFils { // 如果子文件个数小于最大协程数，则修改最大协程数为子文件个数
			maxRoutineNum = splitFils
		}
		taskch := make(chan DownloadTask, 20)    //任务管道
		resch := make(chan string, 100)          //结果信号管道
		exitch := make(chan bool, maxRoutineNum) //退出信号管道
		// 向任务管道中写入需要下载的子文件编号，每个编号对应一个下载任务
		go func() {
			//for i := 0; i < splitFils; i++ {
			//	taskch <- i
			//}
			for _, task := range sFilesTaskList {
				taskch <- task
			}
			close(taskch)
		}()

		for i := 0; i < maxRoutineNum; i++ { //启动goroutine做任务
			go doDownloadTasks(remote, fileName, i, sessionId, taskch, resch, exitch)
		}

		go func() { //等待goroutine结束
			for i := 0; i < maxRoutineNum; i++ {
				<-exitch
			}
			close(resch)  //任务处理完成关闭结果管道，不然range报错
			close(exitch) //关闭退出管道
		}()

		//打印协程执行信息，本质是通过resch与上面的close(resch)配合来等待作业协程完成工作
		for res := range resch {
			log.Debug("子文件下载协程====>> ", res)
		}
		//------------------------------ 方法结束 ------------------------------

		// 合并文件并完成校验
		isOK, err := MergeSplitFileAndCheck(fileName, ffi, sFilesTaskList)
		if err != nil {
			log.Errorf("文件合并出现问题（%s）：%v ", fileName, err)
		}
		if isOK {
			log.Debugf("文件下载成功完成：%s ", fileName)
		}
	} else if recvFlag == strconv.Itoa(myutil.FileNotFound) {
		// 文件未找到
	} else if recvFlag == strconv.Itoa(myutil.ServerError) {
		// 服务器内部错误
	} else if recvFlag == strconv.Itoa(myutil.NoPermission) {
		// 没有权限
	} else {
		// 未定义的错误！！！
	}

	endTime := time.Now().Unix()

	tot := endTime - beginTime
	log.Infof("总计耗时：%d 分 %d 秒 ", tot/60, tot%60)
	log.Debug("---", mergeFileName)

}

// 下载任务
func doDownloadTasks(remote string, fileName string, fileSEQ int, sessionId string, taskch chan DownloadTask, resch chan string, exitch chan bool) {
	defer func() { //异常处理
		err := recover()
		if err != nil {
			log.Errorf("downloadSplitFile（%s: %d） error: %v ", fileName, fileSEQ, err)
			return
		}
	}()
	for task := range taskch { //  处理任务
		seq := task.Seq
		begin := task.Begin
		end := task.End
		isSuccess := downloadSplitFile(remote, fileName, seq, begin, end, sessionId)
		var result string
		if isSuccess {
			result = "==>" + fileName + ": " + strconv.Itoa(seq) + "<== has download."
		} else {
			result = "==>" + fileName + ": " + strconv.Itoa(seq) + "<== download error!!!"
		}
		resch <- result
	}
	exitch <- true //处理完发送退出信号
}

// 请求子数据文件
func downloadSplitFile(remote string, fileName string, fileSEQ int, begin int64, end int64, sessionId string) (isSuccess bool) {

	var data = make([]byte, 1024*100) //创建读取服务端信息的切片

	// 请求与服务器连接
	con, err := net.Dial("tcp", remote)
	if err != nil {
		log.Errorf("子文件（%d）请求服务器连接失败！", fileSEQ)
		return
	}
	defer con.Close()
	log.Debugf("连接已建立.子数据文件（%d）请求发送中...", fileSEQ)

	// 构建请求包，并发送到服务端
	sfrp := myutil.SplitFileRequestPackage{
		SessionID:    sessionId,
		FileName:     fileName,
		SplitFileSEQ: fileSEQ,
		Begin:        begin,
		End:          end,
	}

	_, err = con.Write([]byte(strconv.Itoa(myutil.Request4SplitFile) + sfrp.ToString())) //向服务器发送数据文件请求
	if err != nil {
		log.Errorf("子文件（%d）向服务器发送数据请求失败： %v ", fileSEQ, err)
		return
	}
	//log.Debug("开始创建子文件...: ", fileSEQ)
	// 创建子文件
	tmpFileDir := path.Join(AbsPath("file_store/in/"), fileName+"_info")
	if !IsFileExist(tmpFileDir) { // 如果目录不存在，则创建
		err = os.MkdirAll(tmpFileDir, os.ModePerm) // 创建子文件夹
		if err != nil {
			log.Error("MkDir Error:", err)
			return
		}
	}
	tmpFilePath := path.Join(tmpFileDir, fileName+"_"+strconv.Itoa(fileSEQ)) // 子文件路径
	tmpFile, err := os.Create(tmpFilePath)                                   // 创建子文件
	if err != nil {
		log.Error("子文件创建错误:", err)
		return
	}
	defer tmpFile.Close()

	var isRespHeadRecived = false // 响应头是否已收到

	var respFlagInt int
	var dataTolLength int64
	var dataHasWrote int64 = 0
	var tmpRespHeadBytes = make([]byte, 0) // 用来缓存响应头，以防止数据被传输过程中被截断
	//var tmpRespHeadBytesOldLength int
	var tmpDataBytes = make([]byte, 0)
	for { // 循环接收服务端发送回的数据
		lengthh, err := con.Read(data) //获取服务器返回信息
		if err != nil {
			log.Debug("读取服务器数据长度：", lengthh)
			log.Error("读取服务器数据错误：", err)
			return
		}
		if !isRespHeadRecived { // 如果尚未收到响应头
			tmpRespHeadBytes = append(tmpRespHeadBytes, data[:lengthh]...) //将接收到的数据拼接到 tmpRespHeadBytes
			tmpRespHeadBytesNewLength := len(tmpRespHeadBytes)             // 拼接后的 tmpRespHeadBytes 长度

			if tmpRespHeadBytesNewLength >= ConstRDHLength {
				rdh := RespDataHeadFromBtye(tmpRespHeadBytes[:ConstRDHLength])
				respFlagInt = int(rdh.RespFlag)
				dataTolLength = rdh.DataLength

				tmpDataBytes = tmpRespHeadBytes[ConstRDHLength:tmpRespHeadBytesNewLength]

				isRespHeadRecived = true
			} else {
				log.Debug("警告：接收到的splitFile响应数据长度为：", tmpRespHeadBytesNewLength)
				continue
			}
		} else {
			tmpDataBytes = data[:lengthh]
		}
		tmpDataBytesLength := len(tmpDataBytes)

		if respFlagInt == myutil.SplitFileData { // 子文件数据
			tmpFile.Write(tmpDataBytes[:tmpDataBytesLength]) // 向文件写入数据
			dataHasWrote += int64(tmpDataBytesLength)
			if dataHasWrote == dataTolLength { // 接收到的数据等于发送的数据
				break
			}
			if dataHasWrote > dataTolLength { // 如果发送此种情况，说明有大问题了！
				log.Errorf("严重错误：%s_%d 接收到的数据 >> 发送数据！", fileName, fileSEQ)
				return
			}
		} else if respFlagInt == myutil.NoPermission { // 没有权限，一般是令牌过期或令牌为伪造
			log.Error("没有权限，一般是令牌过期或令牌为伪造：", respFlagInt)
			return
		} else if respFlagInt == myutil.ServerError { // 服务器错误
			log.Error("服务器错误：", respFlagInt)
			return
		} else if respFlagInt == myutil.TokenError { // 令牌校验不通过，一般为文件名或IP地址被篡改
			log.Error("令牌校验不通过，一般为文件名或IP地址被篡改：", respFlagInt)
			return
		} else if respFlagInt == myutil.RequestError { // 请求错误，请求码未知
			log.Error("请求错误，请求码未知：", respFlagInt)
			return
		} else if respFlagInt == myutil.SplitFileNotFound { // 所请求的子文件未找到
			log.Error("所请求的子文件未找到：", respFlagInt)
			return
		} else {
			log.Error("服务器返回标志位无法识别：", respFlagInt)
			return
		}
	}
	// 将子文件数据接收完成信息告知服务端
	con.Write([]byte("OK!"))
	log.Debug(fileName + "-" + strconv.Itoa(fileSEQ) + " 数据接收完成！")
	isSuccess = true
	return
}
