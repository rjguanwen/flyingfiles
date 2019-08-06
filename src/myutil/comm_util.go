package myutil

import (
	"bytes"
	"encoding/binary"
)

//整形转换成字节
func IntToBytes(n int) []byte {
	x := int32(n)

	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}

// 组织子文件传输结束标准
func GetSplitFileOverFlag(sessionId string) (overFlag []byte, length int) {
	overFlag = []byte("<<--splitFileSendOver=" + sessionId + "=")
	length = len(overFlag)
	return
}
