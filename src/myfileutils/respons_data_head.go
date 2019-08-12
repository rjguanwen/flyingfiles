package myfileutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"rjguanwen.cn/flyingfiles/src/myutil"
)

const (
	ConstRespFlagLenght = 4
	ConstSendDataLength = 8
	ConstRDHLength      = ConstRespFlagLenght + ConstSendDataLength
)

// Head结构，包含响应标志和数据长度
type RespDataHead struct {
	RespFlag   int32 //1个字节，响应类型
	DataLength int64 //存放数据的长度（4个字节）
}

// 初始化一个RespDataHead包
func NewRespDataHead(respFlag int32, dataLength int64) *RespDataHead {
	rdh := &RespDataHead{}
	rdh.RespFlag = respFlag
	rdh.DataLength = dataLength
	return rdh
}

// 将 RespDataHead 结构体转换为 string
func (rdh *RespDataHead) ToByte() []byte {
	var bb bytes.Buffer
	bb.Write(myutil.IntToBytes(int(rdh.RespFlag)))
	bb.Write(myutil.Int64ToBytes(rdh.DataLength))
	return bb.Bytes()
}

// 将字符串转为 RespDataHead 结构体
func RespDataHeadFromBtye(rdhBytes []byte) (rdh *RespDataHead) {
	if len(rdhBytes) < ConstRespFlagLenght+ConstSendDataLength {
		return
	} else {
		respFlag := myutil.BytesToInt(rdhBytes[0:ConstRespFlagLenght])
		dataLength := myutil.BytesToInt64(rdhBytes[ConstRespFlagLenght : ConstRespFlagLenght+ConstSendDataLength])
		rdh = NewRespDataHead(int32(respFlag), dataLength)
	}
	return
}

// 将 RespDataHead 结构体转换为 string
func (rdh *RespDataHead) ToString() string {
	rdhJSON, _ := json.Marshal(rdh)
	return string(rdhJSON)
}

func CheckErr(err error) {
	if err != nil {
		fmt.Println("err occur: ", err)
		os.Exit(1)
	}
}
