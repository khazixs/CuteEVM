package vm

import (
	"math/big"

	"CuteEVM01/Out/common"
	"CuteEVM01/Out/core/types"
)

// StateDB是一个用于完整状态查询的EVM数据库。
type 	StateDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetCommittedState(common.Address, common.Hash) common.Hash
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist报告给定帐户是否处于状态。 值得注意的是，suicided accounts的情况也应如此。
	Exist(common.Address) bool
	// 返回给定帐户是否为空。空根据EIP161被定义为(balance = nonce = code = 0)。
	Empty(common.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) error
}

// CallContext为EVM调用合约提供了一个基本接口。EVM依赖于为执行子调用和初始化新的EVM合约而实现的上下文。
type CallContext interface {
	// 调用另一个智能合约
	Call(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// 使用他人的合约代码并在我们自己的上下文中执行
	CallCode(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// 与CallCode相同，只是发送方和值从父范围传播到子范围
	DelegateCall(env *EVM, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error)
	// 创建一个新的智能合约
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error)
}
