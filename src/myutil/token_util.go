// 客户端 Token 相关功能
package myutil

import (
	"encoding/json"
	"time"
)

type SessionContent struct {
	RemoteIp 	string
	FileName	string
	Timestamp	int64
}

// 生成Token
func GeneratSessionContent(remoteIP string, fileName string) (token SessionContent){
	now := time.Now().Unix()
	token = SessionContent{
		RemoteIp: remoteIP,
		FileName: fileName,
		Timestamp: now,
	}
	return
}

// 客户端对子数据文件的请求包
type SplitFileRequestPackage struct{
	SessionID	string	// 会话ID
	FileName 	string	//文件名
	SplitFileSEQ	int	// 子数据文件序号
}

// 将 SplitFileRequestPackage 结构体转换为 string
func (sfrp *SplitFileRequestPackage) ToString() (sfrpStr string){
	sfrpJSON, _ := json.Marshal(sfrp)
	//fmt.Println(string(sfrpJSON))
	sfrpStr = string(sfrpJSON)
	return
}

// 将字符串转为 SplitFileRequestPackage 结构体
func StringToSFRP(sfrpStr string) (sfrp SplitFileRequestPackage){
	json.Unmarshal([]byte(sfrpStr), &sfrp)
	return
}

// 将 SessionContent 结构体转换为 string
func (sessionContent *SessionContent) ToString() (sessionContentStr string){
	scJSON, _ := json.Marshal(sessionContent)
	//fmt.Println(string(scJSON))
	sessionContentStr = string(scJSON)
	return
}

// 将字符串转为 SessionContent 结构体
func StringToSessionContent(sessionContentStr string) (sessionContent SessionContent){
	json.Unmarshal([]byte(sessionContentStr), &sessionContent)
	return
}