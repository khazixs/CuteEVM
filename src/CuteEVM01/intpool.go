package vm

import (
	"math/big"
	"sync"
)

var checkVal = big.NewInt(-42)

const poolLimit = 256

//intPool是一个大整数池，可以为所有大整数操作重用。
type intPool struct {
	pool *Stack
}

func newIntPool() *intPool {
	return &intPool{pool: newstack()}
}

// get从池中检索一个大int数，如果池是空的，则分配一个int数。
// 注意，返回的int值是任意的，不会为零!
func (p *intPool) get() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop()
	}
	return new(big.Int)
}

// getZero从池中检索一个大int，将其设置为零或分配
// 如果池是空的，就换一个新的。
func (p *intPool) getZero() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop().SetUint64(0)
	}
	return new(big.Int)
}

// put返回分配给池的一个大int，稍后由get调用重用。
// 注意，按原样保存的值;既不输出也不输出0 !
func (p *intPool) put(is ...*big.Int) {
	if len(p.pool.data) > poolLimit {
		return
	}
	for _, i := range is {
		// verifyPool是一个构建标志。池验证通过将值与默认值进行比较，确保整数池的完整性。
		if verifyPool {
			i.Set(checkVal)
		}
		p.pool.push(i)
	}
}

// intPool的默认容量
const poolDefaultCap = 25

// intPoolPool管理一个intpool池。
type intPoolPool struct {
	pools []*intPool
	lock  sync.Mutex
}

var poolOfIntPools = &intPoolPool{
	pools: make([]*intPool, 0, poolDefaultCap),
}

// get正在寻找要返回的可用池。
func (ipp *intPoolPool) get() *intPool {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(poolOfIntPools.pools) > 0 {
		ip := ipp.pools[len(ipp.pools)-1]
		ipp.pools = ipp.pools[:len(ipp.pools)-1]
		return ip
	}
	return newIntPool()
}

// 使用get分配一个池。
func (ipp *intPoolPool) put(ip *intPool) {
	ipp.lock.Lock()
	defer ipp.lock.Unlock()

	if len(ipp.pools) < cap(ipp.pools) {
		ipp.pools = append(ipp.pools, ip)
	}
}
