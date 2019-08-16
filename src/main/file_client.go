package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path"
	"rjguanwen.cn/flyingfiles/src/fflog"
	"rjguanwen.cn/flyingfiles/src/myfileutils"
	"rjguanwen.cn/flyingfiles/src/myutil"
	"runtime"
	"strconv"
	"time"
)

func init() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.GOMAXPROCS(4)
}

func init() {
	// 设置逻辑处理器数量
	runtime.GOMAXPROCS(4)
}

func main() {
	//fflog.Infoln(Request4File)
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
		fflog.Infoln("服务器连接失败.")
		os.Exit(-1)
		return
	}
	//fflog.Infoln("连接已建立.文件请求发送中...")
	//fflog.Infoln("客户端请求包：", strconv.Itoa(myutil.Request4File)+fileName)
	in, err := con.Write([]byte(strconv.Itoa(myutil.Request4File) + fileName)) //向服务器发送数据文件请求
	if err != nil {
		fmt.Printf("向服务器发送数据错误: %d\n", in)
		os.Exit(0)
	}
	var msg = make([]byte, 1024*100) //创建读取服务端信息的切片
	lengthh, err := con.Read(msg)    //获取服务器返回信息
	if err != nil {
		fmt.Printf("读取服务器数据错误.\n", lengthh)
		os.Exit(0)
	}
	// 关闭链接
	con.Close()
	//fflog.Infoln("接收到的数据长度==>", lengthh)
	recvFlag := string(msg[0:1])
	//fflog.Infoln("==>", string(msg[:]))
	if recvFlag == strconv.Itoa(myutil.FileReady) {
		// 文件已就绪
		sessionId := string(msg[1 : myutil.SessionIdLength+1])
		//fflog.Infoln("sessionId===>>", sessionId)
		recvData := string(msg[myutil.SessionIdLength+1 : lengthh])
		//fflog.Infoln("服务端返回信息：", recvData)
		//解析返回的数据,将其转化为文件摘要信息对象
		fsi := myfileutils.StringToFSI(recvData)
		//fileSize := fsi.Size
		//md5 := fsi.MD5
		splitFils := fsi.SplitFiles
		//splitFilesMD5 := fsi.SplitFilesMD5
		// 将文件摘要信息写入摘要文件
		myfileutils.WriteFileConfigYAML(fileName, fsi)
		// ------------------------------ 方法一 ------------------------------
		// 下面的写法会导致大文件一次创建了太多的协程，生成了太多的链接，超出系统限制的情况时有发生，修改之
		////为每个子文件创建一个协程
		//var wg sync.WaitGroup
		//wg.Add(splitFils)
		//for i := 0; i < splitFils; i++ {
		//	// 下载每一个子文件
		//	go downloadSplitFile(remote, fileName, i, sessionId, &wg) // 下载子文件的协程
		//}
		//wg.Wait()
		// ------------------------------ 方法二 ------------------------------
		// 下面的写法按批次顺序执行，存在某批次中的某个协程挂起或运行缓慢，导致后续批次无法继续执行的风险,修改之
		//// 每次最多起10个协程，如果子文件个数超过10个，则按批次进行
		//maxRoutineNum := 10
		//if splitFils <= maxRoutineNum { // 如果子文件数小于最大协程数
		//	var wg sync.WaitGroup
		//	wg.Add(splitFils)
		//	for i := 0; i < splitFils; i++ {
		//		// 下载每一个子文件
		//		go downloadSplitFile(remote, fileName, i, sessionId, &wg) // 下载子文件的协程
		//	}
		//	wg.Wait()
		//} else {
		//	splitFilesGroups := splitFils / maxRoutineNum
		//	if splitFils%maxRoutineNum != 0 {
		//		splitFilesGroups += 1
		//	}
		//	for i := 0; i < splitFilesGroups; i++ { //循环处理每个组
		//		var thisLoopNum int // 本次循环需要处理的子文件个数
		//		var wg sync.WaitGroup
		//		if i == splitFilesGroups-1 {
		//			thisLoopNum = splitFils % maxRoutineNum
		//		} else {
		//			thisLoopNum = maxRoutineNum
		//		}
		//		wg.Add(thisLoopNum)
		//		for j := 0; j < thisLoopNum; j++ {
		//			fileSeq := i*maxRoutineNum + j
		//			// 下载每一个子文件
		//			go downloadSplitFile(remote, fileName, fileSeq, sessionId, &wg) // 下载子文件的协程
		//		}
		//		wg.Wait() // 通过 Wait，控制分批执行
		//	}
		//}
		// ------------------------------ 方法三 ------------------------------
		// 采用工作池的方式重写
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
		taskch := make(chan int, 20)             //任务管道
		resch := make(chan string, 100)          //结果信号管道
		exitch := make(chan bool, maxRoutineNum) //退出信号管道
		// 向任务管道中写入需要下载的子文件编号，每个编号对应一个下载任务
		go func() {
			for i := 0; i < splitFils; i++ {
				taskch <- i
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
			fflog.Debugln("子文件下载协程====>> ", res)
		}
		//------------------------------ 方法结束 ------------------------------

		// 合并文件并完成校验
		isOK, err := myfileutils.MergeSplitFileAndCheck(fileName, fsi)
		if err != nil {
			fflog.Errorf("文件合并出现问题（%s）：%v \n", fileName, err)
		}
		if isOK {
			fflog.Infof("文件下载成功完成：%s \n", fileName)
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
	fmt.Printf("总计耗时：%d 分 %d 秒 \n", tot/60, tot%60)
	fflog.Infoln("---", mergeFileName)

}

// 下载任务
func doDownloadTasks(remote string, fileName string, fileSEQ int, sessionId string, taskch chan int, resch chan string, exitch chan bool) {
	defer func() { //异常处理
		err := recover()
		if err != nil {
			fflog.Errorf("downloadSplitFile（%s: %d） error: %v \n", fileName, fileSEQ, err)
			return
		}
	}()
	for i := range taskch { //  处理任务
		isSuccess := downloadSplitFile(remote, fileName, i, sessionId)
		var result string
		if isSuccess {
			result = "==>" + fileName + ": " + strconv.Itoa(i) + "<== has download."
		} else {
			result = "==>" + fileName + ": " + strconv.Itoa(i) + "<== download error!!!"
		}
		resch <- result
	}
	exitch <- true //处理完发送退出信号
}

// 请求子数据文件
func downloadSplitFile(remote string, fileName string, fileSEQ int, sessionId string) (isSuccess bool) {
	//func downloadSplitFile(remote string, fileName string, fileSEQ int, sessionId string, wg *sync.WaitGroup) (isSuccess bool){
	//	defer wg.Done()

	var data = make([]byte, 1024*100) //创建读取服务端信息的切片

	// 请求与服务器连接
	con, err := net.Dial("tcp", remote)
	if err != nil {
		fflog.Errorf("子文件（%d）请求服务器连接失败！\n", fileSEQ)
		return
	}
	defer con.Close()
	fflog.Infof("连接已建立.子数据文件（%d）请求发送中...\n", fileSEQ)

	// 构建请求包，并发送到服务端
	sfrp := myutil.SplitFileRequestPackage{
		SessionID:    sessionId,
		FileName:     fileName,
		SplitFileSEQ: fileSEQ,
	}

	//fflog.Infoln("子文件数据请求=====>>", strconv.Itoa(myutil.Request4SplitFile)+sfrp.ToString())

	_, err = con.Write([]byte(strconv.Itoa(myutil.Request4SplitFile) + sfrp.ToString())) //向服务器发送数据文件请求
	if err != nil {
		fflog.Errorf("子文件（%d）向服务器发送数据请求失败： %v \n", fileSEQ, err)
		return
	}
	//fflog.Debugln("开始创建子文件...: ", fileSEQ)
	// 创建子文件
	tmpFileDir := path.Join(myfileutils.AbsPath("file_store/in/"), fileName+"_info")
	if !myfileutils.IsFileExist(tmpFileDir) { // 如果目录不存在，则创建
		err = os.MkdirAll(tmpFileDir, os.ModePerm) // 创建子文件夹
		if err != nil {
			fflog.Errorln("MkDir Error:", err)
			return
		}
	}
	tmpFilePath := path.Join(tmpFileDir, fileName+"_"+strconv.Itoa(fileSEQ)) // 子文件路径
	tmpFile, err := os.Create(tmpFilePath)                                   // 创建子文件
	if err != nil {
		fflog.Errorln("子文件创建错误:", err)
		return
	}
	defer tmpFile.Close()

	//fflog.Debugln("开始子文件数据接收...: ", fileSEQ)
	//overFlag, overFlagLength := myutil.GetSplitFileOverFlag(sessionId) // 子文件传输完成标志
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
			fflog.Infoln("读取服务器数据长度：", lengthh)
			fflog.Errorln("读取服务器数据错误：", err)
			return
		}
		if !isRespHeadRecived { // 如果尚未收到响应头
			//fflog.myTrace.Printf(":: %s_%d 判断响应头开始...", fileName, fileSEQ)
			//tmpRespHeadBytesOldLength = len(tmpRespHeadBytes) // 拼接前的 tmpRespHeadBytes 长度
			tmpRespHeadBytes = append(tmpRespHeadBytes, data[:lengthh]...) //将接收到的数据拼接到 tmpRespHeadBytes
			tmpRespHeadBytesNewLength := len(tmpRespHeadBytes)             // 拼接后的 tmpRespHeadBytes 长度
			//fflog.Errorln("--------->>>>", tmpRespHeadBytes)
			if tmpRespHeadBytesNewLength >= myfileutils.ConstRDHLength {
				//fflog.Errorln("==========>>>>", tmpRespHeadBytes)
				rdh := myfileutils.RespDataHeadFromBtye(tmpRespHeadBytes[:myfileutils.ConstRDHLength])
				//fflog.myTrace.Printf(":: %s_%d 响应头---> %s", fileName, fileSEQ, rdh.ToString())
				respFlagInt = int(rdh.RespFlag)
				dataTolLength = rdh.DataLength
				//data = data[myfileutils.ConstRDHLength-tmpRespHeadBytesOldLength: lengthh] // 剔除掉响应头
				//lengthh = len(data)
				//if lengthh > myfileutils.ConstRDHLength-tmpRespHeadBytesOldLength {
				//	tmpDataBytes = data[myfileutils.ConstRDHLength-tmpRespHeadBytesOldLength: lengthh] // 本次接收数据中的内容部分
				//}
				tmpDataBytes = tmpRespHeadBytes[myfileutils.ConstRDHLength:tmpRespHeadBytesNewLength]

				//fmt.Printf("--------（%s_%d）总长度 %d\n", fileName, fileSEQ, lengthh)
				//fmt.Printf("--------（%s_%d）tmpOld长度 %d\n", fileName, fileSEQ, tmpRespHeadBytesOldLength)
				//fmt.Printf("--------（%s_%d）tmpBytes长度 %d\n", fileName, fileSEQ, len(tmpDataBytes))
				//fmt.Printf("--------（%s_%d）tmpBytes内容 %d\n", fileName, fileSEQ, tmpDataBytes)

				//fflog.myTrace.Printf(":: %s_%d -----> %d", fileName, fileSEQ, lengthh)
				//fflog.myTrace.Printf(":: %s_%d 判断响应头已获取...", fileName, fileSEQ)
				isRespHeadRecived = true
			} else {
				//fflog.myTrace.Printf(":: %s_%d 判断响应头未获取...", fileName, fileSEQ)
				fflog.Warningln("警告：接收到的splitFile响应数据长度为：", tmpRespHeadBytesNewLength)
				continue
			}
		} else {
			tmpDataBytes = data[:lengthh]
		}
		tmpDataBytesLength := len(tmpDataBytes)

		if respFlagInt == myutil.SplitFileData { // 子文件数据
			//if fileSEQ == 13{
			//	fflog.myTrace.Printf("%s_%d 数据开始，数据长度：\n", fileName, fileSEQ, lengthh)
			//}
			tmpFile.Write(tmpDataBytes[:tmpDataBytesLength]) // 向文件写入数据
			dataHasWrote += int64(tmpDataBytesLength)
			if dataHasWrote == dataTolLength { // 接收到的数据等于发送的数据
				//fflog.myTrace.Printf("%s_%d 数据接收成功完成！\n", fileName, fileSEQ)
				break
			}
			if dataHasWrote > dataTolLength { // 如果发送此种情况，说明有大问题了！
				fflog.Errorf("严重错误：%s_%d 接收到的数据 >> 发送数据！\n", fileName, fileSEQ)
				return
			}
		} else if respFlagInt == myutil.NoPermission { // 没有权限，一般是令牌过期或令牌为伪造
			fflog.Errorln("没有权限，一般是令牌过期或令牌为伪造：", respFlagInt)
			return
		} else if respFlagInt == myutil.ServerError { // 服务器错误
			fflog.Errorln("服务器错误：", respFlagInt)
			return
		} else if respFlagInt == myutil.TokenError { // 令牌校验不通过，一般为文件名或IP地址被篡改
			fflog.Errorln("令牌校验不通过，一般为文件名或IP地址被篡改：", respFlagInt)
			return
		} else if respFlagInt == myutil.RequestError { // 请求错误，请求码未知
			fflog.Errorln("请求错误，请求码未知：", respFlagInt)
			return
		} else if respFlagInt == myutil.SplitFileNotFound { // 所请求的子文件未找到
			fflog.Errorln("所请求的子文件未找到：", respFlagInt)
			return
		} else {
			fflog.Errorln("服务器返回标志位无法识别：", respFlagInt)
			return
		}
	}
	//fflog.myTrace.Printf("------->> %s_%d 数据接收完成并跳出 for 循环！\n", fileName, fileSEQ)
	// 将子文件数据接收完成信息告知服务端
	con.Write([]byte("OK!"))
	fflog.Infoln(fileName + "-" + strconv.Itoa(fileSEQ) + " 数据接收完成！")
	isSuccess = true
	return
}
