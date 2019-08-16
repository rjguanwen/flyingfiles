// 自定义的文件操作相关的工具类
package myfileutils

import (
	"encoding/json"
	"github.com/rjguanwen/flyingfiles/src/fflog"
	"github.com/spf13/viper"
	"os"
	"path"
)

// 结构体，文件摘要
type FileSummaryInfo struct {
	FileName      string
	Size          int64
	MD5           string
	SplitFiles    int
	SplitFilesMD5 []string
}

// 将结构体转换为string
func (fsi *FileSummaryInfo) ToString() (fsiStr string) {
	fsiJSON, _ := json.Marshal(fsi)
	//fmt.Println(string(fsiJSON))
	fsiStr = string(fsiJSON)
	return
}

// 将字符串转为 FileSummaryInfo 结构体
func StringToFSI(fsiStr string) (fsi FileSummaryInfo) {
	json.Unmarshal([]byte(fsiStr), &fsi)
	return
}

// 读取指定数据文件的 YAML 配置文件信息
func ReadFileConfigYAML(fileName string) (hasFsi bool, fsi FileSummaryInfo, err error) {
	configPath := path.Join(AbsPath("file_store/out/"), fileName+"_info") // 子文件夹路径
	configName := fileName + "_config"
	//判断是否存在相应相应文件
	if !IsFileExist(path.Join(configPath, configName+".yaml")) {
		hasFsi = false
		return
	}
	hasFsi, fsi, err = ReadConfigYAML(configPath, configName)
	return
}

// 根据指定路径与指定文件名，读取 YAML 配置文件信息
func ReadConfigYAML(configPath string, configName string) (hasFsi bool, fsi FileSummaryInfo, err error) {
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
		fflog.Errorln("读取配置文件错误：", err) // 读取配置文件失败致命错误
	}

	fsi = FileSummaryInfo{
		FileName:      v.GetString(`FileName`),
		Size:          v.GetInt64(`Size`),
		MD5:           v.GetString(`MD5`),
		SplitFiles:    v.GetInt(`SplitFiles`),
		SplitFilesMD5: v.GetStringSlice(`SplitFilesMD5`),
	}
	return
}

func WriteFileConfigYAML(fileName string, fileInfo FileSummaryInfo) {
	configDir := path.Join(AbsPath("file_store/in/"), fileName+"_info") //配置文件所在目录
	if !IsFileExist(configDir) {                                        // 如果目录不存在，则创建
		err := os.MkdirAll(configDir, os.ModePerm) // 创建子文件夹
		if err != nil {
			fflog.Errorln("MkDir configDir Error:", err)
			return
		}
	}
	configFile := "file_store/in/" + fileName + "_info/" + fileName + "_config"
	WriteConfigYAML(configFile, fileInfo)
}

// 将文件摘要信息写入指定的配置文件
func WriteConfigYAML(configFile string, fileInfo FileSummaryInfo) {
	v := viper.New()
	v.SetConfigFile(configFile)
	//v.SetConfigName(configName)
	v.SetConfigType("yaml")

	v.SetDefault("FileName", fileInfo.FileName)
	v.SetDefault("Size", fileInfo.Size)
	v.SetDefault("SplitFiles", fileInfo.SplitFiles)
	v.SetDefault("MD5", fileInfo.MD5)
	v.SetDefault("SplitFilesMD5", fileInfo.SplitFilesMD5)

	if err := v.WriteConfigAs(configFile + ".yaml"); err != nil {
		fflog.Errorln(err)
	}
}

// 获取绝对路径
func AbsPath(relFilepath string) string {
	// 获取绝对路径
	rootDir, err := os.Getwd()
	if err != nil {
		fflog.Errorln("获取文件绝对路径失败：", err)
	}
	//fmt.Println("==>" + rootDir)
	absPath := path.Join(rootDir, relFilepath)
	//fmt.Println("==>" + absPath)
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

// 判断指定的文件是否已就绪
// 摘要文件已生成，并且相应子文件均存在，则认为文件已就绪
func IsFileReady(fileName string) (ready bool, fsi FileSummaryInfo) {
	ready = false
	hasFsi, fsi, err := ReadFileConfigYAML(fileName) // 读取数据文件的摘要信息
	//fmt.Println(hasFsi)
	//fmt.Println(fsi)
	//fmt.Println(err)
	if !hasFsi {
		return
	}
	if err != nil {
		fflog.Errorln(fileName+"摘要信息读取异常。", err)
		return
	}
	ready = true
	return
}
