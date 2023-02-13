package utils

import "encoding/binary"

func Bytes2Int8(bytes []byte) int8 {
	return int8(Byte2Uint16(bytes))
}

func Bytes2Int(bytes []byte) int {
	return int(Byte2Uint64(bytes))
}

func Bytes2Int64(bytes []byte) int64 {
	return int64(Byte2Uint64(bytes))
}

func Byte2Uint64(bytes []byte) uint64 {
	return binary.LittleEndian.Uint64(bytes)
}

func Byte2Uint32(bytes []byte) uint32 {
	return binary.LittleEndian.Uint32(bytes)
}

func Byte2Uint16(bytes []byte) uint16 {
	return binary.LittleEndian.Uint16(bytes)
}
