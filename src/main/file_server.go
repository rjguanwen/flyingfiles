// 文件服务端，根据客户端请求向客户端传输数据
package main

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net"
	"os"
	"path"
	"rjguanwen.cn/flyingfiles/src/myfileutils"
	"rjguanwen.cn/flyingfiles/src/mylog"
	"rjguanwen.cn/flyingfiles/src/myutil"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var splitSize = int64(1024 * 1024 * 5)
var sessionCache *cache.Cache

func init(){
	// 初始化缓存，（默认有效时长，清理过期元素间隔）
	sessionCache = cache.New(20 * time.Minute, 10 * time.Minute)
}

func init(){
	// 设置逻辑处理器数量
	runtime.GOMAXPROCS(4)
}

func main(){
	var (
		port = "9090"
		remote = ":" + port //此方式本地与非本地都可访问
	)

	fmt.Println("file server init... (Ctrl-C Stop)")

	lis, err := net.Listen("tcp", remote)
	defer lis.Close()

	if err != nil {
		mylog.MyInfo.Println("Port 9090 Listen Error: ", remote)
		os.Exit(-1)
	}

	// 循环监听客户端的链接，并作出响应
	for {
		conn, err := lis.Accept()
		if err != nil {
			mylog.MyInfo.Println("Client Connect Error: ", err.Error())
			// os.Exit(0)
			continue
		}
		mylog.MyInfo.Println("===> Connect LocalAddr:", conn.LocalAddr())
		mylog.MyInfo.Println("===> Connect RemoteAddr:", conn.RemoteAddr())

		//调用文件接收方法
		go dealRequest(conn)
	}
}

func dealRequest(con net.Conn){
	mylog.MyInfo.Println("=== 开始 == dealRequest ...")
	var (
		reqFlag	string
		data         = make([]byte, 1024 * 1024) //用于保存接收的数据的切片
		//by           []byte
		//databuf      = bytes.NewBuffer(by) //数据缓冲变量
	)
	defer con.Close()
	mylog.MyInfo.Println("Create Connection: ", con.RemoteAddr())
	length, err := con.Read(data) // 读取客户端请求数据
	if err != nil {
		mylog.MyError.Println("Error:", err)
		return
	}
	// 解析请求数据
	// 1、首先解析请求标志位，明确是哪类请求
	// 2、根据不同的标志位，开启不同的处理程序
	//    1）如果是文件请求，则寻找相应文件，生成文件摘要并返回文件摘要信息，同时生成会话信息，将会话ID写回客户端
	//	  2）如果是子数据文件请求，解析令牌并核对，通过后将子文件数据流写回客户端
	reqFlag = string(data[0]) //获取请求标志位
	mylog.MyInfo.Println("reqFlag:", reqFlag)
	reqFlagInt, _ := strconv.Atoi(reqFlag)
	mylog.MyInfo.Println("reqFlagInt:", reqFlagInt)
	// 根据请求标志位，进行不同响应
	if reqFlagInt == myutil.Request4File {
		// 如果是文件请求
		// 生成32位随机码，作为sessionId
		sessionId := myutil.RandStringBytesMaskImprSrc(myutil.SessionIdLength)
		mylog.MyInfo.Println("sessionId ===>", sessionId)
		mylog.MyInfo.Println("准备获取文件名...")
		requestFileName := string(data[1:length]) // 请求的文件名
		mylog.MyInfo.Println("获取的请求文件名为：", requestFileName)
		//判断数据文件是否存在
		if !myfileutils.IsOutFileExist(requestFileName){
			con.Write([]byte(strconv.Itoa(myutil.FileNotFound))) // 向客户端返回信息，文件不存在
			mylog.MyError.Println("请求的文件不存在：", requestFileName)
			return
		}
		// ---------------------------
		//
		// 判断是否有权限
		//
		// ---------------------------
		// 判断文件是否已就绪（就绪是指子文件及文件摘要已生成）
		ready, fsi := myfileutils.IsFileReady(requestFileName)
		mylog.MyInfo.Println("文件是否已就绪：", ready)
		if (ready) {
			mylog.MyInfo.Println("获取到文件摘要信息：", fsi)
		} else { // 如果文件未就绪，则准备文件拆分并生成文件摘要
			mylog.MyInfo.Println("文件未就绪，开始文件拆分及摘要信息生成...")
			fsi, err = myfileutils.SplitFileByFileNameSize(requestFileName, splitSize)
			if err != nil {
				mylog.MyError.Println("文件拆分错误！", err)
				con.Write([]byte(strconv.Itoa(myutil.ServerError)))
				mylog.MyError.Println("请求的文件不存在：", requestFileName)
				return
			}
			mylog.MyInfo.Println("文件已就绪！")
			mylog.MyInfo.Println("返回文件摘要信息：", fsi)
		}
		//组织session内容 [客户端IP，请求文件名，当前时间], 并存入缓存
		remoteIP := strings.Split(con.RemoteAddr().String(), ":")[0]
		sessionContent := myutil.GeneratSessionContent(remoteIP, requestFileName)
		sessionCache.Set(sessionId, sessionContent, 30 * time.Minute) // 将会话内容存入缓存，有效期30分钟
		// 写入sessionId及文件摘要信息，发送到客户端
		responseStr := strconv.Itoa(myutil.FileReady) + sessionId + fsi.ToString()
		mylog.MyInfo.Println("生成响应字符串：", responseStr)
		con.Write([]byte(responseStr))
	} else if reqFlagInt == myutil.Request4SplitFile {// 如果是对子数据文件请求
		// 获取请求包内容并核对令牌
		splitFileReqPkg := string(data[1: length])
		mylog.MyInfo.Println("客户端发送过来的请求包：",splitFileReqPkg)
		sfrp := myutil.StringToSFRP(splitFileReqPkg)
		rSessionId := sfrp.SessionID
		sessionContentGet, found := sessionCache.Get(rSessionId)
		if !found {
			con.Write([]byte(strconv.Itoa(myutil.NoPermission)))
			mylog.MyError.Println("缓存中会话标识未找到，sessionId:", rSessionId)
			return
		}
		// 类型断言
		sessionContent, ok := sessionContentGet.(myutil.SessionContent)
		if !ok {
			con.Write([]byte(strconv.Itoa(myutil.ServerError)))
			mylog.MyError.Println("SessionContent类型转换错误:", ok)
			return
		}

		fileNameC := sfrp.FileName	// 从客户端传入的文件名
		splitFileSeq_c := sfrp.SplitFileSEQ		// 从客户端传入的子文件序号
		remoteIPC := strings.Split(con.RemoteAddr().String(), ":")[0]
		fileNameS := sessionContent.FileName	// 服务端缓存的文件名
		remoteIPS := sessionContent.RemoteIp	// 服务端缓存的客户端 IP 地址
		if fileNameC != fileNameS { // 文件名称不一致
			con.Write([]byte(strconv.Itoa(myutil.TokenError)))
			mylog.MyError.Println("文件名与令牌不一致，拒绝服务！", ok)
			return
		}
		if remoteIPC != remoteIPS { // IP 地址名称不一致
			con.Write([]byte(strconv.Itoa(myutil.TokenError)))
			mylog.MyError.Println("IP地址不一致，怀疑令牌被伪造，拒绝服务！", ok)
			return
		}
		// 令牌检查通过，从指定子文件读取数据传输给客户端
		sendSplitFile2Client(con, fileNameC, splitFileSeq_c, rSessionId)
	} else { // 如果请求类型未知
		con.Write([]byte(strconv.Itoa(myutil.RequestError)))
		mylog.MyError.Println("请求类型错误，reqFlagInt:", reqFlagInt)
		return
	}
	mylog.MyInfo.Println("=== 正常结束 == dealRequest !!!")
}

// 读取子文件数据传送到客户端
// 首先检查所选子文件是否存在，如果不存在则向客户端返回异常
// 如果文件存在，则读取文件内容，向客户端发送
func sendSplitFile2Client(con net.Conn, fileName string, splitFileSeq int, sessionId string) {
	splitFileDirPath := path.Join(myfileutils.AbsPath("file_store/out/"), fileName + "_info")
	splitFilePath := path.Join(splitFileDirPath, fileName + "_" + strconv.Itoa(splitFileSeq))
	if !myfileutils.IsFileExist(splitFilePath) { // 如果子文件不存在
		con.Write([]byte(strconv.Itoa(myutil.SplitFileNotFound)))
		mylog.MyError.Println("客户端请求的子文件不存在：", splitFilePath)
		return
	}
	// 开始读取并传输子文件数据到客户端
	con.Write([]byte(strconv.Itoa(myutil.SplitFileData)))

	//打开待发送文件，准备发送文件数据
	file, err := os.OpenFile(splitFilePath, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		mylog.MyError.Println("文件打开错误：", err)
		return
	}
	fileStat, err := file.Stat() //获取文件状态
	if err != nil {
		mylog.MyError.Println("获取文件状态错误：", err)
		return
	}
	var fileSize int64 = fileStat.Size() // 文件大小

	var bufsize = 1024 * 50    //单次发送数据的大小
	buf := make([]byte, bufsize) //创建用于保存读取文件数据的切片

	var sendDtaTolNum int = 0 //记录发送成功的数据量（Byte）

	var begin, end = int64(0), fileSize // 文件读取的开始与结束
	var msg = make([]byte, 1024)  //创建读取客户端返回信息的切片

	//读取并发送数据
	for i := begin; int64(i) < end; i += int64(bufsize) {
		length, err := file.Read(buf) //读取数据到切片中
		if err != nil {
			mylog.MyError.Println("读取文件错误：", err)
			return
		}

		//判断读取的数据长度与切片的长度是否相等，如果不相等，表明文件读取已到末尾
		if length == bufsize {
			sendDataNum, err := con.Write(buf)
			//sendDataPkgNum++
			if err != nil {
				mylog.MyError.Println("向服务器发送数据错误: %d\n", sendDataNum)
				return
			}
			sendDtaTolNum += sendDataNum
		} else {
			sendDataNum, err := con.Write(buf[:length])
			//sendDataPkgNum++
			if err != nil {
				mylog.MyError.Println("向服务器发送数据错误: %d\n", sendDataNum)
				return
			}
			sendDtaTolNum += sendDataNum
		}
	}
	//文件发送完成，通知客户端，并等待客端反馈接收完成
	overFlag, _ := myutil.GetSplitFileOverFlag(sessionId)
	mylog.MyInfo.Println("======>", splitFileSeq, "-", string(overFlag))
	con.Write(overFlag)
	mylog.MyInfo.Println("============>")
	lengthMsg, err := con.Read(msg) //获取客户端返回信息
	if err != nil {
		mylog.MyError.Println("读取客户端返回信息错误：", lengthMsg, err)
		return
	}
	clientFlag := string(msg[:lengthMsg]) //获取请求标志位
	mylog.MyInfo.Println(fileName + "- " + strconv.Itoa(splitFileSeq) + "，子文件数据发送完成：", clientFlag)
}