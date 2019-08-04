package vm

import (
	"math/big"

	"CuteEVM01/Out/params"
)

// Gas costs
const (
	GasQuickStep   uint64 = 2
	GasFastestStep uint64 = 3
	GasFastStep    uint64 = 5
	GasMidStep     uint64 = 8
	GasSlowStep    uint64 = 10
	GasExtStep     uint64 = 20
)

// calcGas返回调用的实际气成本.
//
// The cost of gas was changed during the homestead price change HF. To allow for EIP150
// to be implemented. The returned gas is gas - base * 63 / 64.
//在homestead价格变动HF期间，燃气成本发生了变化。使EIP150得以实施。返回的气体为气基* 63 / 64。
func callGas(gasTable params.GasTable, availableGas, base uint64, callCost *big.Int) (uint64, error) {
	if gasTable.CreateBySuicide > 0 {
		availableGas = availableGas - base
		gas := availableGas - availableGas/64
		// 如果位长超过64位，我们知道新计算的EIP150的“gas”小于请求的量。因此，我们返回新的气体而不是返回一个错误。
		if !callCost.IsUint64() || gas < callCost.Uint64() {
			return gas, nil
		}
	}
	if !callCost.IsUint64() {
		return 0, errGasUintOverflow
	}

	return callCost.Uint64(), nil
}
