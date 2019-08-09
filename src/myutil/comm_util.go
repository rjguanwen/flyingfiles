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

func Int64ToBytes(i int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

//// int64
//func Int64ToByte(num int64) []byte {
//	var buffer bytes.Buffer
//	err := binary.Write(&buffer, binary.BigEndian, num)
//	CheckErr(err)
//	return buffer.Bytes()
//}

// 组织子文件传输结束标准
func GetSplitFileOverFlag(sessionId string) (overFlag []byte, length int) {
	overFlag = []byte("<<====SplitFileSendOver=" + sessionId + "=")
	length = len(overFlag)
	return
}

// 快速合并字节数组
func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}
