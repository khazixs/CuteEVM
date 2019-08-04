package vm

import (
	"math/big"

	"CuteEVM01/Out/common"
	"CuteEVM01/Out/common/math"
)

// calcMemSize64 计算需要的内存大小, 并且返回大小和该结果是否溢出unit64
func calcMemSize64(off, l *big.Int) (uint64, bool) {
	if !l.IsUint64() {
		return 0, true
	}
	return calcMemSize64WithUint(off, l.Uint64())
}

// calcMemSize64WithUint 计算需要的内存大小, 并且返回大小和该结果是否溢出unit64
// 与calcMemSize64是相同的定义, 但长度是uint64
func calcMemSize64WithUint(off *big.Int, length64 uint64) (uint64, bool) {
	// 如果长度是零, 内存大小 总是零, 与offset无关
	if length64 == 0 {
		return 0, false
	}
	// 检查offset是否溢出
	if !off.IsUint64() {
		return 0, true
	}
	offset64 := off.Uint64()
	val := offset64 + length64
	// 如果value值 < 它的任何一部分，然后溢出
	return val, val < offset64
}

// getData根据起始和大小从数据中返回一个切片，并使用0填充到大小。这个函数是溢出安全的。
func getData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return common.RightPadBytes(data[start:end], int(size))
}

//getDataBig根据起始和大小从数据中返回一个切片，并使用0填充到大小。这个函数是溢出安全的。
func getDataBig(data []byte, start *big.Int, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := math.BigMin(start, dlen)
	e := math.BigMin(new(big.Int).Add(s, size), dlen)
	return common.RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

// bigUint64返回被转换为uint64的整数，并返回它是否在进程中溢出。
func bigUint64(v *big.Int) (uint64, bool) {
	return v.Uint64(), !v.IsUint64()
}

// toWordSize内存扩展所需的上限字节数大小
func toWordSize(size uint64) uint64 {
	if size > math.MaxUint64-31 {
		return math.MaxUint64/32 + 1
	}

	return (size + 31) / 32
}

func allZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}
