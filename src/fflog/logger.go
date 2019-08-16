package fflog

import (
	"io"
	"log"
	"os"
)

var (
	myDebug       *log.Logger // 记录所有日志
	myFileDebug   *log.Logger // 记录所有日志
	myInfo        *log.Logger // 重要的信息
	myFileInfo    *log.Logger // 重要的信息
	myWarning     *log.Logger // 需要注意的信息
	myFileWarning *log.Logger // 需要注意的信息
	myError       *log.Logger // 非常严重的问题
)

const (
	prefix_debug   = "DEBUG:"
	prefix_info    = "INFO:"
	prefix_warning = "WARNING:"
	prefix_error   = "ERROR:"
)

func init() {

	file_debug, err0 := os.OpenFile("logs/flyingfiles_debug.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err0 != nil {
		log.Fatalln("Failed to open trace log file:", err0)
	}
	myDebug = log.New(io.MultiWriter(file_debug, os.Stdout),
		prefix_debug,
		log.Ldate|log.Ltime|log.Lshortfile)
	myFileDebug = log.New(file_debug,
		prefix_debug,
		log.Ldate|log.Ltime|log.Lshortfile)

	file_info, err1 := os.OpenFile("logs/flyingfiles_info.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err1 != nil {
		log.Fatalln("Failed to open info log file:", err1)
	}
	myInfo = log.New(io.MultiWriter(file_info, os.Stdout),
		prefix_info,
		log.Ldate|log.Ltime|log.Lshortfile)
	myFileInfo = log.New(file_info,
		prefix_info,
		log.Ldate|log.Ltime|log.Lshortfile)

	file_warning, err2 := os.OpenFile("logs/flyingfiles_warning.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err2 != nil {
		log.Fatalln("Failed to open warning log file:", err2)
	}
	myWarning = log.New(io.MultiWriter(file_warning, os.Stdout),
		prefix_warning,
		log.Ldate|log.Ltime|log.Lshortfile)
	myFileWarning = log.New(file_warning,
		prefix_warning,
		log.Ldate|log.Ltime|log.Lshortfile)

	file_error, err3 := os.OpenFile("logs/flyingfiles_error.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err3 != nil {
		log.Fatalln("Failed to open error log file:", err3)
	}
	myError = log.New(io.MultiWriter(file_error, os.Stderr),
		prefix_error,
		log.Ldate|log.Ltime|log.Lshortfile)
}

func Debugf(format string, v ...interface{}) {
	myDebug.Printf(format, v)
}
func Debugln(v ...interface{}) {
	myDebug.Println(v)
}

func Infof(format string, v ...interface{}) {
	myInfo.Printf(format, v)
	myFileDebug.SetPrefix(prefix_info)
	myFileDebug.Printf(format, v)
}
func Infoln(v ...interface{}) {
	myInfo.Println(v)
	myFileDebug.SetPrefix(prefix_info)
	myFileDebug.Println(v)
}

func Warningf(format string, v ...interface{}) {
	myWarning.Printf(format, v)
	myFileInfo.SetPrefix(prefix_warning)
	myFileInfo.Printf(format, v)
	myFileDebug.SetPrefix(prefix_warning)
	myFileDebug.Printf(format, v)
}
func Warningln(v ...interface{}) {
	myWarning.Println(v)
	myFileInfo.SetPrefix(prefix_warning)
	myFileInfo.Println(v)
	myFileDebug.SetPrefix(prefix_warning)
	myFileDebug.Println(v)
}

func Errorf(format string, v ...interface{}) {
	myError.Printf(format, v)
	myFileWarning.SetPrefix(prefix_error)
	myFileWarning.Printf(format, v)
	myFileInfo.SetPrefix(prefix_error)
	myFileInfo.Printf(format, v)
	myFileDebug.SetPrefix(prefix_error)
	myFileDebug.Printf(format, v)
}

func Errorln(v ...interface{}) {
	myError.Println(v)
	myFileWarning.SetPrefix(prefix_error)
	myFileWarning.Println(v)
	myFileInfo.SetPrefix(prefix_error)
	myFileInfo.Println(v)
	myFileDebug.SetPrefix(prefix_error)
	myFileDebug.Println(v)
}
