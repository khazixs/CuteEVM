package vm

import (
	"fmt"
	"math/big"

	"CuteEVM01/Out/common/math"
)

// Memory为ethereum虚拟机实现了一个简单的内存模型。
type Memory struct {
	store       []byte
	lastGasCost uint64
}

// NewMemory返回一个新的内存模型。
func NewMemory() *Memory {
	return &Memory{}
}

// Set 设置偏移量（sets offset） + 大小 to value
func (m *Memory) Set(offset, size uint64, value []byte) {
	//可能偏移量大于0，大小等于0。这是因为当大小为0时，calcMemSize (common.go)可能返回0 (NO-OP)_无指令
	if size > 0 {
		// 存储长度不能小于偏移量+大小。
		// 在设置内存之前，应该调整存储的大小
		if offset+size > uint64(len(m.store)) {
			panic("invalid memory: store empty")
		}
		copy(m.store[offset:offset+size], value)
	}
}

// Set32（）将从偏移量开始的32字节设置为val的值，左填充0到32字节。
func (m *Memory) Set32(offset uint64, val *big.Int) {
	//存储长度不能小于偏移量+大小。
	//在设置内存之前，应该调整存储的大小
	if offset+32 > uint64(len(m.store)) {
		panic("invalid memory: store empty")
	}
	// 填充0占位
	copy(m.store[offset:offset+32], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	// 填充相关位
	math.ReadBits(val, m.store[offset:offset+32])
}

// Resize 调整size大小
func (m *Memory) Resize(size uint64) {
	if uint64(m.Len()) < size {
		m.store = append(m.store, make([]byte, size-uint64(m.Len()))...)
	}
}

// Get 返回offset+size为一个新切片
func (m *Memory) Get(offset, size int64) (cpy []byte) {
	if size == 0 {
		return nil
	}

	if len(m.store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, m.store[offset:offset+size])

		return
	}

	return
}

// GetPtr 返回 offset + size
func (m *Memory) GetPtr(offset, size int64) []byte {
	if size == 0 {
		return nil
	}

	if len(m.store) > int(offset) {
		return m.store[offset : offset+size]
	}

	return nil
}

// Len 返回切片长度
func (m *Memory) Len() int {
	return len(m.store)
}

// Data 返回切片
func (m *Memory) Data() []byte {
	return m.store
}

// Print 打印堆栈.
func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
