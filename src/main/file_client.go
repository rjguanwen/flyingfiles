package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path"
	"rjguanwen.cn/flyingfiles/src/myfileutils"
	"rjguanwen.cn/flyingfiles/src/mylog"
	"rjguanwen.cn/flyingfiles/src/myutil"
	"runtime"
	"strconv"
	"sync"
	"time"
)


func init() {
	//runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.GOMAXPROCS(4)
}

func init(){
	// 设置逻辑处理器数量
	runtime.GOMAXPROCS(4)
}

func main() {
	//mylog.MyInfo.Println(Request4File)
	var (
		host   = "127.0.0.1"     //服务端IP
		port   = "9090"            //服务端端口
		remote = host + ":" + port //构造连接串
		fileName = "weibo_data.txt" // 请求数据文件名
		mergeFileName = "download_weibo_data.txt"   //本地保存数据文件名
		//fileName = "zhengbin.pst" // 请求数据文件名
		//mergeFileName = "download_zhengbin.pst"   //本地保存数据文件名
		//fileName = "数据驱动1008.pptx" // 请求数据文件名
		//mergeFileName = "download_数据驱动1008.pptx"   //本地保存数据文件名
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

	//fmt.Printf("请输入服务端IP: ")
	//reader := bufio.NewReader(os.Stdin)
	//ipdata, _, _ := reader.ReadLine()

	//host = string(ipdata)
	host = "127.0.0.1"
	remote = host + ":" + port
	beginTime := time.Now().Unix()

	// 请求与服务器连接
	con, err := net.Dial("tcp", remote)
	if err != nil {
		mylog.MyInfo.Println("服务器连接失败.")
		os.Exit(-1)
		return
	}
	mylog.MyInfo.Println("连接已建立.文件请求发送中...")
	mylog.MyInfo.Println("客户端请求包：", strconv.Itoa(myutil.Request4File) + fileName)
	in, err := con.Write([]byte(strconv.Itoa(myutil.Request4File) + fileName)) //向服务器发送数据文件请求
	if err != nil {
		fmt.Printf("向服务器发送数据错误: %d\n", in)
		os.Exit(0)
	}
	var msg = make([]byte, 1024 * 100)  //创建读取服务端信息的切片
	lengthh, err := con.Read(msg) //获取服务器返回信息
	if err != nil {
		fmt.Printf("读取服务器数据错误.\n", lengthh)
		os.Exit(0)
	}
	// 关闭链接
	con.Close()
	mylog.MyInfo.Println("接收到的数据长度==>", lengthh)
	recvFlag := string(msg[0: 1])
	mylog.MyInfo.Println("==>", string(msg[:]))
	if recvFlag == strconv.Itoa(myutil.FileReady){
		// 文件已就绪
		sessionId := string(msg[1: myutil.SessionIdLength + 1])
		mylog.MyInfo.Println("sessionId===>>", sessionId)
		recvData := string(msg[myutil.SessionIdLength + 1: lengthh])
		mylog.MyInfo.Println("服务端返回信息：",recvData)
		//解析返回的数据,将其转化为文件摘要信息对象
		fsi := myfileutils.StringToFSI(recvData)
		//fileSize := fsi.Size
		//md5 := fsi.MD5
		splitFils := fsi.SplitFiles
		//splitFilesMD5 := fsi.SplitFilesMD5
		//为每个子文件创建一个协程
		var wg sync.WaitGroup
		wg.Add(splitFils)
		for i := 0; i < splitFils; i++ {
			// 下载每一个子文件
			go downloadSplitFile(remote, fileName, i, sessionId, &wg) // 下载子文件的协程
		}
		wg.Wait()
		// 合并文件

		// 文件校验
	} else if recvFlag == strconv.Itoa(myutil.FileNotFound){
		// 文件未找到
	} else if recvFlag == strconv.Itoa(myutil.ServerError){
		// 服务器内部错误
	} else if recvFlag == strconv.Itoa(myutil.NoPermission){
		// 没有权限
	} else {
		// 未定义的错误！！！
	}

	endTime := time.Now().Unix()

	tot := endTime - beginTime
	fmt.Printf("总计耗时：%d 分 %d 秒 \n", tot/60, tot%60)
	mylog.MyInfo.Println("---", mergeFileName)

}

// 请求子数据文件
func downloadSplitFile(remote string, fileName string, fileSEQ int, sessionId string, wg *sync.WaitGroup){
	defer wg.Done()


	var data = make([]byte, 1024 * 100)  //创建读取服务端信息的切片

	// 请求与服务器连接
	con, err := net.Dial("tcp", remote)
	if err != nil {
		mylog.MyError.Println("服务器连接失败.")
		return
	}
	defer con.Close()
	mylog.MyInfo.Println("连接已建立.子数据文件请求发送中...")

	//databuf.WriteString(fileName)
	// 构建请求包，并发送到服务端
	sfrp := myutil.SplitFileRequestPackage {
		SessionID: sessionId,
		FileName: fileName,
		SplitFileSEQ: fileSEQ,
	}

	mylog.MyInfo.Println("子文件数据请求=====>>", strconv.Itoa(myutil.Request4SplitFile) + sfrp.ToString())
	mylog.MyInfo.Println("------------")
	in, err := con.Write([]byte(strconv.Itoa(myutil.Request4SplitFile) + sfrp.ToString())) //向服务器发送数据文件请求
	if err != nil {
		fmt.Printf("向服务器发送数据错误: %d\n", in)
		os.Exit(0)
	}
	// 创建子文件
	tmpFileDir := path.Join(myfileutils.AbsPath("file_store/in/"), fileName + "_info")
	if !myfileutils.IsFileExist(tmpFileDir) { // 如果目录不存在，则创建
		err = os.MkdirAll(tmpFileDir, os.ModePerm)    // 创建子文件夹
		if err != nil{
			mylog.MyError.Println("MkDir Error:" , err)
			return
		}
	}
	tmpFilePath := path.Join(tmpFileDir, fileName + "_" + strconv.Itoa(fileSEQ)) // 子文件路径
	tmpFile, err := os.Create(tmpFilePath) // 创建子文件
	if err != nil {
		mylog.MyError.Fatalln("子文件创建错误:" , err)
		return
	}
	defer tmpFile.Close()

	overFlag, overFlagLength := myutil.GetSplitFileOverFlag(sessionId) // 子文件传输完成标志
	j := 0 // 标记接收数据的次数
	var isTheEndData = false // 标记收到的数据是不是最后一段数据
	for { // 循环接收服务端发送回的数据
		lengthh, err := con.Read(data) //获取服务器返回信息
		// 判断接收到的数据是否含有
		if isSplitFileOver(overFlag, overFlagLength, data, lengthh){
			lengthh = lengthh - overFlagLength
			isTheEndData = true
		}
		// 接收到的数据包个数 + 1
		//receivedDataPkgNum++
		//recivDataLength += lengthh
		if err != nil {
			//mylog.MyInfo.Println(fileName + "-" + strconv.Itoa(fileSEQ) + " 接收数据长度：" + strconv.Itoa(recivDataLength), err)
			mylog.MyInfo.Println("读取服务器数据长度：", lengthh)
			mylog.MyError.Println("读取服务器数据错误：", err)
			//mylog.MyInfo.Println(fileName + "-" + strconv.Itoa(fileSEQ) + " 数据接收与消息返回次数：" + strconv.Itoa(receivedDataPkgNum) + "|" +  strconv.Itoa(sendMsgPkgNum) )
			return
		}
		if j == 0{ // 如果是第一次接收，需要判断服务器响应是否为文件内容
			respFlag := string(data[0]) //获取请求标志位
			mylog.MyInfo.Println("respFlag:", respFlag)
			respFlagInt, _ := strconv.Atoi(respFlag)
			mylog.MyInfo.Println("respFlagInt:", respFlagInt)
			// 根据请求标志位，进行不同响应
			if respFlagInt == myutil.SplitFileData { // 子文件数据
				//hasReceiveData = true
				tmpFile.Write(data[1: lengthh]) // 向文件写入数据
			} else if respFlagInt == myutil.NoPermission { // 没有权限，一般是令牌过期或令牌为伪造
				mylog.MyError.Println("没有权限，一般是令牌过期或令牌为伪造：", respFlag)
				return
			} else if respFlagInt == myutil.ServerError { // 服务器错误
				mylog.MyError.Println("服务器错误：", respFlag)
				return
			} else if respFlagInt == myutil.TokenError { // 令牌校验不通过，一般为文件名或IP地址被篡改
				mylog.MyError.Println("令牌校验不通过，一般为文件名或IP地址被篡改：", respFlag)
				return
			} else if respFlagInt == myutil.RequestError { // 请求错误，请求码未知
				mylog.MyError.Println("请求错误，请求码未知：", respFlag)
				return
			} else if respFlagInt == myutil.SplitFileNotFound { // 所请求的子文件未找到
				mylog.MyError.Println("所请求的子文件未找到：", respFlag)
				return
			} else {
				mylog.MyError.Println("服务器返回标志位无法识别：", respFlag)
				return
			}
		} else {
			tmpFile.Write(data[0: lengthh])
		}
		j++
		if isTheEndData { // 如果是最后一段数据，则跳出循环
			break
		}
	}
	// 将子文件数据接收完成信息告知服务端
	con.Write([]byte("OK!"))
	mylog.MyInfo.Println(fileName + "-" + strconv.Itoa(fileSEQ) + " 数据接收完成！")
}

// 根据接收到的 []byte 判断子文件数据传输是否完成
func isSplitFileOver(overFlag []byte, overFlagLength int, data []byte, dataLength int) bool{
	if dataLength < overFlagLength{
		return false
	} else {
		return bytes.Equal(overFlag, data[dataLength - overFlagLength: dataLength])
	}
}