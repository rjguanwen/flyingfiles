// 文件服务端，根据客户端请求向客户端传输数据
package main

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/patrickmn/go-cache"
	. "github.com/rjguanwen/flyingfiles/src/myfileutils"
	"github.com/rjguanwen/flyingfiles/src/myutil"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var splitSize = int64(1024 * 1024 * 10) // 拆分文件大小
var sessionCache *cache.Cache

func init() {
	// 初始化缓存，（默认有效时长，清理过期元素间隔）
	sessionCache = cache.New(20*time.Minute, 10*time.Minute)
}

func init() {
	// 设置逻辑处理器数量
	runtime.GOMAXPROCS(4)
}

func main() {
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
		port   = "9090"
		remote = ":" + port //此方式本地与非本地都可访问
	)

	fmt.Println("file server init... (Ctrl-C Stop)")

	lis, err := net.Listen("tcp", remote)
	defer lis.Close()

	if err != nil {
		log.Error("Port 9090 Listen Error: ", remote)
		os.Exit(-1)
	}

	// 循环监听客户端的链接，并作出响应
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Error("Client Connect Error: ", err.Error())
			// os.Exit(0)
			continue
		}
		log.Info("===> Connect LocalAddr:", conn.LocalAddr())
		log.Info("===> Connect RemoteAddr:", conn.RemoteAddr())

		//调用文件接收方法
		go dealRequest(conn)
	}
}

func dealRequest(con net.Conn) {
	log.Info("=== 开始 == dealRequest ...")
	defer func() { //异常处理
		err := recover()
		if err != nil {
			log.Errorf("dealRequest error: %v ", err)
			return
		}
	}()

	var (
		reqFlag string
		data    = make([]byte, 1024*1024) //用于保存接收的数据的切片
		//by           []byte
		//databuf      = bytes.NewBuffer(by) //数据缓冲变量
	)
	defer con.Close()
	log.Info("Create Connection: ", con.RemoteAddr())
	length, err := con.Read(data) // 读取客户端请求数据
	if err != nil {
		log.Error("Error:", err)
		return
	}
	// 解析请求数据
	// 1、首先解析请求标志位，明确是哪类请求
	// 2、根据不同的标志位，开启不同的处理程序
	//    1）如果是文件请求，则寻找相应文件，生成文件摘要并返回文件摘要信息，同时生成会话信息，将会话ID写回客户端
	//	  2）如果是子数据文件请求，解析令牌并核对，通过后将子文件数据流写回客户端
	reqFlag = string(data[0]) //获取请求标志位
	reqFlagInt, _ := strconv.Atoi(reqFlag)
	// 根据请求标志位，进行不同响应
	if reqFlagInt == myutil.Request4File {
		// 如果是文件请求
		// 生成32位随机码，作为sessionId
		sessionId := myutil.RandStringBytesMaskImprSrc(myutil.SessionIdLength)
		log.Info("sessionId ===>", sessionId)
		requestFileName := string(data[1:length]) // 请求的文件名
		log.Info("获取的请求文件名为：", requestFileName)
		//判断数据文件是否存在
		if !IsOutFileExist(requestFileName) {
			con.Write([]byte(strconv.Itoa(myutil.FileNotFound))) // 向客户端返回信息，文件不存在
			log.Error("请求的文件不存在：", requestFileName)
			return
		}
		// ---------------------------
		//
		// 判断是否有权限
		//
		// ---------------------------
		// 生成数据文件发送信息 FileFlyInfo
		ffi, err := GenFileFilyInfo(requestFileName, splitSize)
		if err != nil {
			log.Error("Error:", err)
			return
		}
		//组织session内容 [客户端IP，请求文件名，当前时间], 并存入缓存
		remoteIP := strings.Split(con.RemoteAddr().String(), ":")[0]
		sessionContent := myutil.GeneratSessionContent(remoteIP, requestFileName)
		sessionCache.Set(sessionId, sessionContent, 30*time.Minute) // 将会话内容存入缓存，有效期30分钟
		// 写入sessionId及文件摘要信息，发送到客户端
		responseStr := strconv.Itoa(myutil.FileReady) + sessionId + ffi.ToString()
		log.Info("生成响应字符串：", responseStr)
		con.Write([]byte(responseStr))
	} else if reqFlagInt == myutil.Request4SplitFile { // 如果是对子数据文件请求
		// 获取请求包内容并核对令牌
		splitFileReqPkg := string(data[1:length])
		log.Info("客户端发送过来的请求包：", splitFileReqPkg)
		sfrp := myutil.StringToSFRP(splitFileReqPkg)
		rSessionId := sfrp.SessionID
		sessionContentGet, found := sessionCache.Get(rSessionId)
		if !found {
			//con.Write([]byte(strconv.Itoa(myutil.NoPermission)))
			con.Write(NewRespDataHead(myutil.NoPermission, 0).ToByte())
			log.Error("缓存中会话标识未找到，sessionId:", rSessionId)
			return
		}
		// 类型断言
		sessionContent, ok := sessionContentGet.(myutil.SessionContent)
		if !ok {
			//con.Write([]byte(strconv.Itoa(myutil.ServerError)))
			con.Write(NewRespDataHead(myutil.ServerError, 0).ToByte())
			log.Error("SessionContent类型转换错误:", ok)
			return
		}

		fileNameC := sfrp.FileName          // 从客户端传入的文件名
		splitFileSeq_c := sfrp.SplitFileSEQ // 从客户端传入的子文件序号
		splitFileBegin := sfrp.Begin        // 子文件数据开始位置
		splitFileEnd := sfrp.End            // 子文件数据结束位置
		remoteIPC := strings.Split(con.RemoteAddr().String(), ":")[0]
		fileNameS := sessionContent.FileName // 服务端缓存的文件名
		remoteIPS := sessionContent.RemoteIp // 服务端缓存的客户端 IP 地址
		if fileNameC != fileNameS {          // 文件名称不一致
			//con.Write([]byte(strconv.Itoa(myutil.TokenError)))
			con.Write(NewRespDataHead(myutil.TokenError, 0).ToByte())
			log.Error("文件名与令牌不一致，拒绝服务！", ok)
			return
		}
		if remoteIPC != remoteIPS { // IP 地址名称不一致
			//con.Write([]byte(strconv.Itoa(myutil.TokenError)))
			con.Write(NewRespDataHead(myutil.TokenError, 0).ToByte())
			log.Error("IP地址不一致，怀疑令牌被伪造，拒绝服务！", ok)
			return
		}
		// 令牌检查通过，从指定子文件读取数据传输给客户端
		sendSplitFile2Client(con, fileNameC, splitFileSeq_c, splitFileBegin, splitFileEnd, rSessionId)
	} else { // 如果请求类型未知
		//con.Write([]byte(strconv.Itoa(myutil.RequestError)))
		con.Write(NewRespDataHead(myutil.RequestError, 0).ToByte())
		log.Error("请求类型错误，reqFlagInt:", reqFlagInt)
		return
	}
	log.Info("=== 正常结束 == dealRequest !!!")
}

// 将客户端请求的数据文件片段传送到客户端
func sendSplitFile2Client(con net.Conn, fileName string, splitFileSeq int, splitFileBegin int64, splitFileEnd int64, sessionId string) {
	// 文件读取缓冲区大小
	fileReadBufSize := 1024 * 100

	filePath := path.Join(AbsPath("file_store/out/"), fileName) // 获取文件绝对地址

	//打开待发送文件，准备发送文件数据
	file, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		log.Error("文件打开错误：", err)
		return
	}

	file.Seek(splitFileBegin, 0)         //设定读取文件的起始位置
	buf := make([]byte, fileReadBufSize) //创建用于保存读取文件数据的切片
	var msg = make([]byte, 1024)         //创建读取客户端返回信息的切片
	// 计算并写入响应头
	con.Write(NewRespDataHead(myutil.SplitFileData, splitFileEnd-splitFileBegin).ToByte())
	//读取数据并发送
	var sendDtaTolNum int = 0 // 记录发送成功的数据量
	for i := splitFileBegin; i < splitFileEnd; i += int64(fileReadBufSize) {
		length, err := file.Read(buf) //读取数据到切片中
		if err != nil {
			log.Error("File Read Error:", err)
		}
		//判断读取的数据长度与切片的长度是否相等，如果不相等，表明文件读取已到末尾
		if length == fileReadBufSize {
			//判断此次读取的数据是否在当前协程读取的数据范围内，如果超出，则去除多余数据, 否则全部发送到客户端
			if int64(i)+int64(fileReadBufSize) >= splitFileEnd {
				sendDataNum, err := con.Write(buf[:fileReadBufSize-int((int64(i)+int64(fileReadBufSize)-splitFileEnd))])
				if err != nil {
					log.Error("sFile Write Error:", err)
					return
				}
				sendDtaTolNum += sendDataNum
			} else {
				sendDataNum, err := con.Write(buf)
				if err != nil {
					log.Error("sFile Write Error:", err)
					return
				}
				sendDtaTolNum += sendDataNum
			}
		} else {
			sendDataNum, err := con.Write(buf[:length])
			if err != nil {
				log.Error("sFile Write Error:", err)
				return
			}
			sendDtaTolNum += sendDataNum
		}
	}

	log.Debugf("%s_%d 数据发送完成，等待客户端反馈！", fileName, splitFileSeq)
	//文件发送完成，等待客端反馈接收完成
	lengthMsg, err := con.Read(msg) //获取客户端返回信息
	if err != nil {
		log.Error("读取客户端返回信息错误：", lengthMsg, err)
		return
	}
	clientFlag := string(msg[:lengthMsg]) //获取请求标志位
	log.Info(fileName+"- "+strconv.Itoa(splitFileSeq)+"，子文件数据发送完成：", clientFlag)
}
