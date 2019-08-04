package vm

import (
	"math/big"

	"CuteEVM01/Out/common"
)

// ContractRef 对合约函数返回对象的引用
type ContractRef interface {
	Address() common.Address
}

// AccountRef 实现 ContractRef接口.
//
// 帐户引用在EVM初始化期间使用，它的主要用途是获取地址。
// 由于从父合约(即调用者)中获取的缓存跳转目标是一个contractRef，因此很难删除该对象。
type AccountRef common.Address

// Address函数 将AccountRef转换为一个地址
func (ar AccountRef) Address() common.Address { return (common.Address)(ar) }

// Contract 表示状态数据库中的ethereum合约。 它包含合同代码，调用参数。Contract implements ContractRef
type Contract struct {
	// CallerAddress 是初始化本合约的调用方的结果。 然而，当“调用方法”被委托时，需要将该值初始化为调用方的调用方的值。
	CallerAddress common.Address
	caller        ContractRef
	self          ContractRef

	jumpdests map[common.Hash]bitvec // JUMPDEST分析结果汇总
	analysis  bitvec                 // jumpdests分析的本地缓存结果

	Code     []byte
	CodeHash common.Hash
	CodeAddr *common.Address
	Input    []byte

	Gas   uint64
	value *big.Int
}

// NewContract 返回执行EVM的新合约环境。
func NewContract(caller ContractRef, object ContractRef, value *big.Int, gas uint64) *Contract {
	c := &Contract{CallerAddress: caller.Address(), caller: caller, self: object}

	if parent, ok := caller.(*Contract); ok {
		// 如果可用则从父合约 重用 JUMPDEST 分析
		c.jumpdests = parent.jumpdests
	} else {
		c.jumpdests = make(map[common.Hash]bitvec)
	}

	// Gas 应该是一个指针，所以他可以通过run方法被安全的减少
	// 这个指针会脱离状态转换
	c.Gas = gas
	// 确保有一个值被设置
	c.value = value

	return c
}

func (c *Contract) validJumpdest(dest *big.Int) bool {
	udest := dest.Uint64()
	// PC不能超过len(code)，当然也不能大于63位
	// 在这种情况下，不必费心检查JUMPDEST。
	if dest.BitLen() >= 63 || udest >= uint64(len(c.Code)) {
		return false
	}
	// Only JUMPDESTs allowed for destinations
	if OpCode(c.Code[udest]) != JUMPDEST {
		return false
	}
	// 判断我们是否已经有一个合约哈希值
	if c.CodeHash != (common.Hash{}) {
		// Does parent context have the analysis?
		analysis, exist := c.jumpdests[c.CodeHash]
		if !exist {
			// 是否在父上下文中进行分析和保存
			// 我们不需要将它存储在c.analysis中
			analysis = codeBitmap(c.Code)
			c.jumpdests[c.CodeHash] = analysis
		}
		return analysis.codeSegment(udest)
	}
	// 我们没有代码哈希，很可能是一段initcode还没有处于trie状态。
	// 在这种情况下，我们进行分析，并将其保存在本地，因此我们不必为执行中的每个跳转指令重新计算它，
	// 但是，我们不将其保存在父上下文中
	if c.analysis == nil {
		c.analysis = codeBitmap(c.Code)
	}
	return c.analysis.codeSegment(udest)
}

// AsDelegate 将合约设置为 一个delegateCall调用 并且返回当前的合约（contract）(for chaining calls)
func (c *Contract) AsDelegate() *Contract {
	// NOTE: 调用者 必须, 一直是一个Contract.不应该发生调用者是Contract以外的东西。
	parent := c.caller.(*Contract)
	c.CallerAddress = parent.CallerAddress
	c.value = parent.value

	return c
}

// GetOp 返回合约字节数组中的第n个元素
func (c *Contract) GetOp(n uint64) OpCode {
	return OpCode(c.GetByte(n))
}

// GetByte 返回约字节数组中的第n个字节
func (c *Contract) GetByte(n uint64) byte {
	if n < uint64(len(c.Code)) {
		return c.Code[n]
	}

	return 0
}

// Caller 调用方返回合约的调用者。
//
//当契约是委托调用时，Caller将递归调用调用者，包括调用者的调用者的调用者。
func (c *Contract) Caller() common.Address {
	return c.CallerAddress
}

// UseGas 尝试使用和减少gas 并且在成功时返回true
func (c *Contract) UseGas(gas uint64) (ok bool) {
	if c.Gas < gas {
		return false
	}
	c.Gas -= gas
	return true
}

// Address 返回合约地址
func (c *Contract) Address() common.Address {
	return c.self.Address()
}

// Value 返回合约的value
func (c *Contract) Value() *big.Int {
	return c.value
}

// SetCallCode 设置合约的代码和返回的数据对象的地址
func (c *Contract) SetCallCode(addr *common.Address, hash common.Hash, code []byte) {
	c.Code = code
	c.CodeHash = hash
	c.CodeAddr = addr
}

// SetCodeOptionalHash 可以被是用来提供代码, 但是提供hash值也是可行的
// 如果没有提供hash，那么jumpdests的analysis将不会保存到parent Context中
func (c *Contract) SetCodeOptionalHash(addr *common.Address, codeAndHash *codeAndHash) {
	c.Code = codeAndHash.code
	c.CodeHash = codeAndHash.hash
	c.CodeAddr = addr
}
