package core

import (
	"fmt"
	"math"
)

// GasPool跟踪一个块中执行操作时可用的gas量。零值是一个可用gas为零的池。
type GasPool uint64

// AddGas使gas可以执行.
func (gp *GasPool) AddGas(amount uint64) *GasPool {
	if uint64(*gp) > math.MaxUint64-amount {
		panic("gas pool pushed above uint64")
	}
	*(*uint64)(gp) += amount
	return gp
}

// 如果有足够的可用gas，则SubGas从池中扣除给定的数量，否则返回错误。
func (gp *GasPool) SubGas(amount uint64) error {
	if uint64(*gp) < amount {
		return ErrGasLimitReached
	}
	*(*uint64)(gp) -= amount
	return nil
}

// Gas()返回池中剩余的gas量。
func (gp *GasPool) Gas() uint64 {
	return uint64(*gp)
}

func (gp *GasPool) String() string {
	return fmt.Sprintf("%d", *gp)
}
