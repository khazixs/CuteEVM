//虚拟机的调度器，开始真正的解析执行合约代码
package vm

import (
	"fmt"
	"hash"
	"sync/atomic"

	"CuteEVM01/Out/common"
	"CuteEVM01/Out/common/math"
	"CuteEVM01/Out/params"
)

// Config是解释器的配置选项
type Config struct {
	Debug                   bool   // debug标志位
	Tracer                  Tracer // Opcode记录——即一个接口（示踪物）
	NoRecursion             bool   // 禁止 call, callcode, delegate call and create等函数
	EnablePreimageRecording bool   // SHA3/keccak 原像记录可用与否标志位

	JumpTable [256]operation // EVM指令表，如果未设置，将自动填充

	EWASMInterpreter string // 外部EWASM解释器选项
	EVMInterpreter   string // 外部EVM解释器选项
}

// 解释器用于运行基于Ethereum的合约，并将使用传递的环境查询外部源以获取状态信息。
// 解释器将根据传递的配置运行字节码VM。
type Interpreter interface {
	// 运行循环并使用给定的输入数据计算合约的字节码，并返回返回字节片，如果发生错误，则返回一个错误。
	Run(contract *Contract, input []byte, static bool) ([]byte, error)
	// CanRun（）告诉当前解释器是否可以运行作为参数传递的合约。这意味着调用者可以做如下事情:
	// ```golang
	// for _, interpreter := range interpreters {
	//   if interpreter.CanRun(contract.code) {
	//     interpreter.Run(contract.code, input)
	//   }
	// }
	// ```
	CanRun([]byte) bool
}

// keccakState包装sha3.state。
// 除了常用的哈希方法之外，它还支持Read从哈希状态获取可变数量的数据。
// Read比Sum快，因为它不复制内部状态，但也修改内部状态。
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// EVMInterpreter 表示一个EVM解释器
type EVMInterpreter struct {
	evm      *EVM
	cfg      Config
	gasTable params.GasTable

	intPool *intPool

	hasher    keccakState // Keccak256 hasher 实例可以跨操作码共享
	hasherBuf common.Hash // Keccak256 hasher 可以跨操作码共享结果数组

	readOnly   bool   // 是否抛出状态修改标志
	returnData []byte // 最后一个调用的返回数据，以便后续重用
}

// NewEVMInterpreter 返回解释器的新实例
func NewEVMInterpreter(evm *EVM, cfg Config) *EVMInterpreter {
	//我们使用STOP指令查看是否初始化了跳转表。如果不是，我们将设置默认跳转表。
	if !cfg.JumpTable[STOP].valid {
		switch {
		case evm.ChainConfig().IsConstantinople(evm.BlockNumber):
			cfg.JumpTable = constantinopleInstructionSet
		case evm.ChainConfig().IsByzantium(evm.BlockNumber):
			cfg.JumpTable = byzantiumInstructionSet
		case evm.ChainConfig().IsHomestead(evm.BlockNumber):
			cfg.JumpTable = homesteadInstructionSet
		default:
			cfg.JumpTable = frontierInstructionSet
		}
	}

	return &EVMInterpreter{
		evm:      evm,
		cfg:      cfg,
		gasTable: evm.ChainConfig().GasTable(evm.BlockNumber),
	}
}

// 运行循环并使用给定的输入数据计算合约的字节码，并返回返回字节片，如果发生错误，则返回一个错误。
//
// 需要注意的是，解释器返回的任何错误都应该被认为是一个“revert-and-consume-all-gas”的操作，
// 但errExecutionReverted除外，它意味着“revert-and-keep-gas-left”。
func (in *EVMInterpreter) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	if in.intPool == nil {
		in.intPool = poolOfIntPools.get()
		defer func() {
			poolOfIntPools.put(in.intPool)
			in.intPool = nil
		}()
	}

	// Increment the call depth which is restricted to 1024
	//增加调用深度（原调用深度被限制为1024）
	in.evm.depth++
	defer func() { in.evm.depth-- }()

	// 确保只有在还没有设置readOnly时才设置readOnly.
	// 这也确保了没有为子调用删除readOnly标志。
	if readOnly && !in.readOnly {
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	// 重置前一个调用的返回数据。保存旧缓冲区并不重要
	// 因为每次返回调用都会返回新的数据。
	in.returnData = nil

	// 果没有代码，就不要费心执行。
	if len(contract.Code) == 0 {
		return nil, nil
	}

	var (
		op    OpCode        // 当前opcode
		mem   = NewMemory() // bound memory 内存范围、大小
		stack = newstack()  // 为本次操作服务的堆栈
		// 出于优化的原因，我们使用uint64作为程序计数器。
		// 理论上有可能超过2^64。YP将程序计数器定义为uint256。实际上就不那么可行了。
		pc   = uint64(0) // 程序计数器
		cost uint64
		// 示踪器（Tracer）使用的副本
		pcCopy  uint64 // needed for the deferred Tracer
		gasCopy uint64 // 用于Tracer中指令在执行前记录剩余气体
		logged  bool   // deferred Tracer should ignore already logged steps
		res     []byte // OpCode执行函数的结果
	)
	contract.Input = input

	// Reclaim the stack as an int pool when the execution stops
	//当执行停止时，回收作为int pool的堆栈
	defer func() { in.intPool.put(stack.data...) }()

	if in.cfg.Debug {
		defer func() {
			if err != nil {
				if !logged {
					_ = in.cfg.Tracer.CaptureState(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
				} else {
					_ = in.cfg.Tracer.CaptureFault(in.evm, pcCopy, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
				}
			}
		}()
	}
	// 解释器主运行循环(上下文)。此循环将一直运行，
	// 直到执行显式的STOP、RETURN或self - destruct（自销毁），
	// 或者在执行某个操作时发生错误，或者直到父上下文设置done标志。
	for atomic.LoadInt32(&in.evm.abort) == 0 {
		if in.cfg.Debug {
			// 捕获用于跟踪的执行前值。
			logged, pcCopy, gasCopy = false, pc, contract.Gas
		}

		//从跳转表获取操作并验证堆栈，以确保有足够的堆栈空间可用来执行该操作。
		op = contract.GetOp(pc)
		operation := in.cfg.JumpTable[op]
		if !operation.valid {
			return nil, fmt.Errorf("invalid opcode 0x%x", int(op))
		}
		// 验证堆栈
		if sLen := stack.len(); sLen < operation.minStack {
			return nil, fmt.Errorf("stack underflow (%d <=> %d)", sLen, operation.minStack)
		} else if sLen > operation.maxStack {
			return nil, fmt.Errorf("stack limit reached %d (%d)", sLen, operation.maxStack)
		}
		// 如果操作有效，则强制执行和写入限制
		if in.readOnly && in.evm.chainRules.IsByzantium {
			// 如果解释器在只读模式下运行，请确保没有执行任何状态修改操作。
			// 调用操作的第三个堆栈项是值。将值从一个帐户转移到其他帐户意味着状态被修改，并且应该返回一个错误。
			if operation.writes || (op == CALL && stack.Back(2).Sign() != 0) {
				return nil, errWriteProtection
			}
		}
		// 汽油静态（固定）部分
		if !contract.UseGas(operation.constantGas) {
			return nil, ErrOutOfGas
		}

		var memorySize uint64
		// 计算新的内存大小并扩展内存以适应操作
		// 在评估动态汽油部分之前，需要进行内存检查，以检测计算溢出
		if operation.memorySize != nil {
			memSize, overflow := operation.memorySize(stack)
			if overflow {
				return nil, errGasUintOverflow
			}
			// memory被扩展到32字节. Gas 也是通过字节计算
			if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
				return nil, errGasUintOverflow
			}
		}
		// 动态汽油部分
		// 如果没有足够的气体可用,消耗气体并返回一个错误。
		// 显式地设置了成本，以便捕获状态延迟方法可以获得适当的成本
		if operation.dynamicGas != nil {
			cost, err = operation.dynamicGas(in.gasTable, in.evm, contract, stack, mem, memorySize)
			if err != nil || !contract.UseGas(cost) {
				return nil, ErrOutOfGas
			}
		}
		if memorySize > 0 {
			mem.Resize(memorySize)
		}

		if in.cfg.Debug {
			_ = in.cfg.Tracer.CaptureState(in.evm, pc, op, gasCopy, cost, mem, stack, contract, in.evm.depth, err)
			logged = true
		}

		// 执行操作
		res, err = operation.execute(&pc, in, contract, mem, stack)
		// verifyPool是一个构建标志。 池验证确保完整性，通过将值与默认值进行比较，得到整数池的值。
		if verifyPool {
			verifyIntegerPool(in.intPool)
		}
		// 如果操作清除返回数据(例如，它有返回数据)
		// 将最后一个返回值设置为操作的结果。
		if operation.returns {
			in.returnData = res
		}

		switch {
		case err != nil:
			return nil, err
		case operation.reverts:
			return res, errExecutionReverted
		case operation.halts:
			return res, nil
		case !operation.jumps:
			pc++
		}
	}
	return nil, nil
}

// CanRun告诉作为参数传递的合约是否可以由当前解释器运行。
func (in *EVMInterpreter) CanRun(code []byte) bool {
	return true
}
