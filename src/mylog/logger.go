package mylog

import (
	"io"
	"log"
	"os"
)

var (
	MyTrace *log.Logger	// 记录所有日志
	MyInfo *log.Logger	// 重要的信息
	MyWarning *log.Logger	// 需要注意的信息
	MyError *log.Logger	// 非常严重的问题
)

func init(){

	file_trace, err0 := os.OpenFile("logs/Trace.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err0 != nil {
		log.Fatalln("Failed to open trace log file:", err0)
	}
	MyTrace = log.New(io.MultiWriter(file_trace, os.Stdout),
		"TRACE:",
		log.Ldate|log.Ltime|log.Lshortfile)

	file_info, err1 := os.OpenFile("logs/Info.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err1 != nil {
		log.Fatalln("Failed to open info log file:", err1)
	}
	MyInfo = log.New(io.MultiWriter(file_info, os.Stdout),
		"INFO:",
		log.Ldate|log.Ltime|log.Lshortfile)

	file_warning, err2 := os.OpenFile("logs/Warning.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err2 != nil {
		log.Fatalln("Failed to open warning log file:", err2)
	}
	MyWarning = log.New(io.MultiWriter(file_warning, os.Stdout),
		"WARNING:",
		log.Ldate|log.Ltime|log.Lshortfile)

	file_error, err3 := os.OpenFile("logs/Error.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err3 != nil {
		log.Fatalln("Failed to open error log file:", err3)
	}
	MyError = log.New(io.MultiWriter(file_error, os.Stderr),
		"ERROR:",
		log.Ldate|log.Ltime|log.Lshortfile)
}