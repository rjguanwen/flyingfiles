// 自定义的文件操作相关的工具类
package myfileutils

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/spf13/viper"
	"os"
	"path"
)

// 结构体，文件发送信息
type FileFlyInfo struct {
	FileName      string // 文件名称
	Size          int64  // 文件大小
	MD5           string // MD5码
	SplitFiles    int    // 子文件个数
	SplitFileSize int64  // 子文件大小
}

// 将结构体转换为string
func (ffi *FileFlyInfo) ToString() (ffiStr string) {
	ffiJSON, _ := json.Marshal(ffi)
	ffiStr = string(ffiJSON)
	return
}

// 将字符串转为 FileFlyInfo 结构体
func StringToFFI(ffiStr string) (ffi FileFlyInfo) {
	json.Unmarshal([]byte(ffiStr), &ffi)
	return
}

// 根据指定路径与指定文件名，读取 YAML 配置文件信息
func ReadConfigYAML(configPath string, configName string) (hasFsi bool, ffi FileFlyInfo, err error) {
	v := viper.New()
	v.SetConfigName(configName) //指定配置文件的文件名称(不需要制定配置文件的扩展名)
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath) //设置配置文件的搜索目录
	//v.AddConfigPath(".")    // 设置配置文件和可执行二进制文件在用一个目录
	//判断是否存在相应相应文件
	if !IsFileExist(path.Join(configPath, configName+".yaml")) {
		hasFsi = false
		return
	}
	hasFsi = true
	err = v.ReadInConfig() // 根据以上配置读取加载配置文件
	if err != nil {
		log.Error("读取配置文件错误：", err) // 读取配置文件失败致命错误
	}

	ffi = FileFlyInfo{
		FileName:      v.GetString(`FileName`),
		Size:          v.GetInt64(`Size`),
		MD5:           v.GetString(`MD5`),
		SplitFiles:    v.GetInt(`SplitFiles`),
		SplitFileSize: v.GetInt64(`SplitFileSize`),
	}
	return
}

func WriteFileConfigYAML(fileName string, ffi FileFlyInfo) {
	configDir := path.Join(AbsPath("file_store/in/"), fileName+"_info") //配置文件所在目录
	if !IsFileExist(configDir) {                                        // 如果目录不存在，则创建
		err := os.MkdirAll(configDir, os.ModePerm) // 创建子文件夹
		if err != nil {
			log.Error("MkDir configDir Error:", err)
			return
		}
	}
	configFile := "file_store/in/" + fileName + "_info/" + fileName + "_config"
	WriteConfigYAML(configFile, ffi)
}

// 将文件摘要信息写入指定的配置文件
func WriteConfigYAML(configFile string, ffi FileFlyInfo) {
	v := viper.New()
	v.SetConfigFile(configFile)
	//v.SetConfigName(configName)
	v.SetConfigType("yaml")

	v.SetDefault("FileName", ffi.FileName)
	v.SetDefault("Size", ffi.Size)
	v.SetDefault("SplitFiles", ffi.SplitFiles)
	v.SetDefault("MD5", ffi.MD5)
	v.SetDefault("SplitFileSize", ffi.SplitFileSize)

	if err := v.WriteConfigAs(configFile + ".yaml"); err != nil {
		log.Error(err)
	}
}

// 获取绝对路径
func AbsPath(relFilepath string) string {
	// 获取绝对路径
	rootDir, err := os.Getwd()
	if err != nil {
		log.Error("获取文件绝对路径失败：", err)
	}
	absPath := path.Join(rootDir, relFilepath)
	return absPath
}

// 判断文件是否存在
func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// 判断要输出的数据文件是否存在
func IsOutFileExist(fileName string) bool {
	fileAbsPath := path.Join(AbsPath("file_store/out/"), fileName) // 文件绝对路径
	return IsFileExist(fileAbsPath)
}

// 下载任务
type DownloadTask struct {
	FileName string // 文件名称
	Seq      int    // 序号
	Begin    int64  // 开始位置
	End      int64  // 结束位置
}
