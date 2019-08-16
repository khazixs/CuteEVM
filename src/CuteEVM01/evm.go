//定义了EVM结构体，提供Create和Call方法，作为虚拟机的入口，分别对应创建合约和执行合约代码
package vm

import (
	"math/big"
	"sync/atomic"
	"time"

	"CuteEVM01/Out/common"
	"CuteEVM01/Out/crypto"
	"CuteEVM01/Out/params"
)

// emptyCodeHash是由create使用以确保已部署的合约地址不允许部署(relevant after the account abstraction).
var emptyCodeHash = crypto.Keccak256Hash(nil)

type (
	// CanTransferFunc是传输保护函数的签名
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc是传递函数的签名
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int)
	// GetHashFunc返回区块链中的第n个块的散列值，并由BLOCKHASH EVM op代码使用。
	GetHashFunc func(uint64) common.Hash
)
// run运行给定的合约，并负责使用回退字节码解释器运行预编译。
func run(evm *EVM, contract *Contract, input []byte, readOnly bool) ([]byte, error) {
	if contract.CodeAddr != nil {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(p, input, contract)
		}
	}
	for _, interpreter := range evm.interpreters {
		if interpreter.CanRun(contract.Code) {
			if evm.interpreter != interpreter {
				// 确保解释器指针在返回时被设置回当前值
				defer func(i Interpreter) {
					evm.interpreter = i
				}(evm.interpreter)
				evm.interpreter = interpreter
			}
			return interpreter.Run(contract, input, readOnly)
		}
	}
	return nil, ErrNoCompatibleInterpreter
}

// Context为EVM提供附加信息。一旦提供，就不应该修改。
type Context struct {
	// CanTransfer返回帐户是否包含足够的ether来传输值
	CanTransfer CanTransferFunc
	// Transfer 转移 ether 从一个账户 到另外一个
	Transfer TransferFunc
	// GetHash返回与n对应的散列值
	GetHash GetHashFunc

	// Message 信息
	Origin   common.Address // 为ORIGIN提供信息
	GasPrice *big.Int       // 为GASPRICE提供信息

	// Block 信息
	Coinbase     common.Address // 为COINBASE提供信息
	GasLimit     uint64         // 为GASLIMIT提供信息
	BlockNumber  *big.Int       // 为NUMBER提供信息
	Time         *big.Int       // 为TIME提供信息
	Difficulty   *big.Int
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int)
	// 为DIFFICULTY提供信息
}

// EVM是Ethereum虚拟机的基本对象，它提供了必要的工具，可以在给定的状态下使用所提供的上下文运行合约。
// 应该注意的是，通过任何调用生成的任何错误都应该被认为是一个revert-state-and-consume-all-gas的操作
//不检查特定的错误。解释器确保生成的任何错误都被认为是错误代码.
// EVM不应该被重用，也不是线程安全的。
type EVM struct {
	// Context提供了附加的区块链相关信息
	Context
	// StateDB 提供对基础状态的访问
	StateDB StateDB
	// Depth 是当前调用堆栈的调用深度
	depth int

	// chainConfig 包含当前链的信息
	chainConfig *params.ChainConfig
	// chainRules 包含当前时代的链规则
	chainRules params.Rules
	// virtual machine configuration options用于初始化虚拟机
	vmConfig Config
	// 全局(to this Context)ethereum虚拟机，在tx执行过程中使用。
	interpreters []Interpreter
	interpreter  Interpreter
	// abort用于中止EVM调用操作
	// 注意:必须自动设置
	abort int32
	//callGasTemp保存当前调用可用的gas。这是必要的，因为可用gas是根据63/64规则在gasCall*中计算的，然后应用到opCall*中。
	callGasTemp uint64
}

// NewEVM 返回一个新的虚拟机对象. 返回的虚拟机不是线程安全所以应该只被使用一次
func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:      ctx,
		StateDB:      statedb,
		vmConfig:     vmConfig,
		chainConfig:  chainConfig,
		chainRules:   chainConfig.Rules(ctx.BlockNumber),
		interpreters: make([]Interpreter, 0, 1),
	}

	if chainConfig.IsEWASM(ctx.BlockNumber) {
		// 由EVM-C和cart PRs实现
		// if vmConfig.EWASMInterpreter != "" {
		//  extIntOpts := strings.Split(vmConfig.EWASMInterpreter, ":")
		//  path := extIntOpts[0]
		//  options := []string{}
		//  if len(extIntOpts) > 1 {
		//    options = extIntOpts[1..]
		//  }
		//  evm.interpreters = append(evm.interpreters, NewEVMVCInterpreter(evm, vmConfig, options))
		// } else {
		// 	evm.interpreters = append(evm.interpreters, NewEWASMInterpreter(evm, vmConfig))
		// }
		panic("No supported ewasm interpreter yet.")
	}

	// vmConfig。EVMInterpreter将被EVM-c使用，它不会在这里被检查，因为我们总是希望将内置的EVM作为故障转移选项。
	evm.interpreters = append(evm.interpreters, NewEVMInterpreter(evm, vmConfig))
	evm.interpreter = evm.interpreters[0]

	return evm
}

//Cancel取消任何正在运行的EVM操作。这可以并发调用，并且可以安全地调用多次。
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Cancelled returns true if Cancel has been called
func (evm *EVM) Cancelled() bool {
	return atomic.LoadInt32(&evm.abort) == 1
}

// Interpreter returns the current interpreter
func (evm *EVM) Interpreter() Interpreter {
	return evm.interpreter
}

// 调用使用给定的输入作为参数执行与addr关联的合约。
// 它还处理所需的任何必要的value转移，并采取必要的步骤来创建帐户，并在执行错误或value转移失败时反转状态。
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// 如果我们试图在调用深度限制之上执行，则会失败
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// 如果我们试图转移超过可用余额，就会失败
	if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	if !evm.StateDB.Exist(addr) {
		precompiles := PrecompiledContractsHomestead
		if evm.ChainConfig().IsByzantium(evm.BlockNumber) {
			precompiles = PrecompiledContractsByzantium
		}
		if precompiles[addr] == nil && evm.ChainConfig().IsEIP158(evm.BlockNumber) && value.Sign() == 0 {
			// 调用一个不存在的帐户，不做任何事情，但是ping the tracer
			if evm.vmConfig.Debug && evm.depth == 0 {
				_ = evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				_ = evm.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			return nil, gas, nil
		}
		evm.StateDB.CreateAccount(addr)
	}
	evm.Transfer(evm.StateDB, caller.Address(), to.Address(), value)
	// 初始化一个新的合约，并设置EVM要使用的字节码码
	// 合约只是这个执行上下文的作用域环境.
	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// 即使帐户没有代码，我们也需要继续，因为它可能是预编译的
	start := time.Now()

	// 在调试模式下捕获Tracer启动/结束事件
	if evm.vmConfig.Debug && evm.depth == 0 {
		_ = evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)

		defer func() { // 参数的延迟计算
			_ = evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
		}()
	}
	ret, err = run(evm, contract, input, false)

	//当EVM返回错误或设置上面的创建代码时，我们将恢复到快照并且消耗掉剩余的任何gas
	// 此外，当我们处在HomeStead指令集下，这也计算代码存储gas错误。
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode使用给定的输入作为参数执行与addr关联的合约。
// 它还处理所需的任何必要的值传输，并采取必要的步骤来创建帐户，当出现值传递失败和执行错误时，反转状态state
//
// CallCode与Call的不同之处在于，它以调用者作为上下文执行给定的地址的代码
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}


	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	contract := NewContract(caller, to, value, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall使用给定的输入作为参数执行与addr关联的合约。它会在执行错误时反转状态。
//
// delegateCall与CallCode的不同之处在于，它使用调用者作为上下文执行给定的地址代码，并将调用者设置为调用者的调用者。
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	//初始化一个新合约，并初始化委托值
	contract := NewContract(caller, to, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall使用给定的输入作为参数执行与addr关联的合约，同时不允许在调用期间对状态进行任何修改。
// 试图执行此类修改的OpCode将导致异常而不是执行修改。
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)

	contract := NewContract(caller, to, new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// We do an AddBalance of zero here, just in order to trigger a touch.
	// This doesn't matter on Mainnet, where all empties are gone at the time of Byzantium,
	// but is the correct thing to do and matters on other networks, in tests, and potential
	// future scenarios
	evm.StateDB.AddBalance(addr, bigZero)

	//当EVM返回错误或设置上面的创建代码时，我们将恢复到快照并且消耗掉剩余的任何gas
	// 此外，当我们处在HomeStead指令集下，这也计算代码存储gas错误。
	ret, err = run(evm, contract, input, true)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

type codeAndHash struct {
	code []byte
	hash common.Hash
}

func (c *codeAndHash) Hash() common.Hash {
	if c.hash == (common.Hash{}) {
		c.hash = crypto.Keccak256Hash(c.code)
	}
	return c.hash
}

// create 使用代码作为部署代码创建一个新合约。
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *big.Int, address common.Address) ([]byte, common.Address, uint64, error) {
	// 深度检查执行
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !evm.CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}
	nonce := evm.StateDB.GetNonce(caller.Address())
	evm.StateDB.SetNonce(caller.Address(), nonce+1)

	// 我确定在指定的地址没有现成的合同
	contractHash := evm.StateDB.GetCodeHash(address)
	if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != emptyCodeHash) {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// 在该状态上创建一个新帐户
	snapshot := evm.StateDB.Snapshot()
	evm.StateDB.CreateAccount(address)
	if evm.ChainConfig().IsEIP158(evm.BlockNumber) {
		evm.StateDB.SetNonce(address, 1)
	}
	evm.Transfer(evm.StateDB, caller.Address(), address, value)

	contract := NewContract(caller, AccountRef(address), value, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, address, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		_ = evm.vmConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.code, gas, value)
	}
	start := time.Now()

	ret, err := run(evm, contract, nil, false)

	// 检查是否超过了最大代码大小
	maxCodeSizeExceeded := evm.ChainConfig().IsEIP158(evm.BlockNumber) && len(ret) > params.MaxCodeSize
	//如果合约创建运行成功且没有返回错误，则计算存储代码所需的gas。
	//如果代码不能存储由于没有足够的gas设置一个错误，并让它处理的错误检查条件如下。
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || (err != nil && (evm.ChainConfig().IsHomestead(evm.BlockNumber) || err != ErrCodeStoreOutOfGas)) {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// 当err仍然为空时，如果合约代码大小超过最大值，则分配err。
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	if evm.vmConfig.Debug && evm.depth == 0 {
		_ = evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, address, contract.Gas, err

}

// Create使用代码作为部署代码创建一个新合约
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), evm.StateDB.GetNonce(caller.Address()))
	return evm.create(caller, &codeAndHash{code: code}, gas, value, contractAddr)
}

// Create2 使用代码作为部署代码创建一个新合约
//
// Create2和Create之间的区别是 Create2 使用 sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// 而不是通常的 sender-and-nonce-hash 作为初始化合约的地址.
func (evm *EVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	codeAndHash := &codeAndHash{code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), common.BigToHash(salt), codeAndHash.Hash().Bytes())
	return evm.create(caller, codeAndHash, gas, endowment, contractAddr)
}

// ChainConfig 环境的chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }
