//定义了EVM运行环境结构体，并实现 转账处理 这些比较高级的，跟交易本身有关的功能
package core

import (
	"math/big"

	"CuteEVM01"
	"CuteEVM01/Out/common"
	"CuteEVM01/Out/consensus"
	"CuteEVM01/Out/core/types"
)

// ChainContext支持从当前区块链中检索要在事务处理期间使用的头和一致参数。
type ChainContext interface {
	// Engine检索链的一致引擎。
	Engine() consensus.Engine

	// GetHeader返回与它们的散列对应的散列
	GetHeader(common.Hash, uint64) *types.Header
}
//补充代码~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
type Message interface {
	From() common.Address
	//FromFrontier() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() uint64
	Value() *big.Int

	Nonce() uint64
	CheckNonce() bool
	Data() []byte
}
// NewEVMContext创建一个用于EVM的新上下文。
func NewEVMContext(msg Message, header *types.Header, chain ChainContext, author *common.Address) vm.Context {
	// 如果我们没有显式的作者(即没有挖掘)，从标题中提取
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}
	return vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(header, chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int).SetUint64(header.Time),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		GasLimit:    header.GasLimit,
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn返回一个GetHashFunc，它根据数字检索头散列
func GetHashFn(ref *types.Header, chain ChainContext) func(n uint64) common.Hash {
	var cache map[uint64]common.Hash

	return func(n uint64) common.Hash {
		// 如果还没有散列缓存，创建一个
		if cache == nil {
			cache = map[uint64]common.Hash{
				ref.Number.Uint64() - 1: ref.ParentHash,
			}
		}
		// 尝试从缓存中完成请求
		if hash, ok := cache[n]; ok {
			return hash
		}
		// 不缓存，迭代块并缓存散列
		for header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); header != nil; header = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1) {
			cache[header.Number.Uint64()-1] = header.ParentHash
			if n == header.Number.Uint64()-1 {
				return header.ParentHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer检查地址账户中是否有足够的资金进行转账。
// 这并不包括必要的gas帐户，使转让有效。
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer使用给定的Db从发送方减去金额并向接收方添加金额
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

