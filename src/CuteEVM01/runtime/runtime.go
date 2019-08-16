package runtime

import (

	"fmt"
	"math"
	"math/big"
	"time"

	"CuteEVM01"
	"CuteEVM01/Out/common"
	"CuteEVM01/Out/core/rawdb"
	"CuteEVM01/Out/core/state"
	"CuteEVM01/Out/crypto"
	"CuteEVM01/Out/params"
)

// Config是一个关于EVM运行的特定配置标志的基本类型
type Config struct {
	ChainConfig *params.ChainConfig
	Difficulty  *big.Int
	Origin      common.Address
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        *big.Int
	GasLimit    uint64
	GasPrice    *big.Int
	Value       *big.Int
	Debug       bool
	EVMConfig   vm.Config

	State     *state.StateDB
	GetHashFn func(n uint64) common.Hash
}

// 设置配置的默认值
func setDefaults(cfg *Config) {
	if cfg.ChainConfig == nil {
		cfg.ChainConfig = &params.ChainConfig{
			ChainID:        big.NewInt(1),
			HomesteadBlock: new(big.Int),
			DAOForkBlock:   new(big.Int),
			DAOForkSupport: false,
			EIP150Block:    new(big.Int),
			EIP155Block:    new(big.Int),
			EIP158Block:    new(big.Int),
		}
	}

	if cfg.Difficulty == nil {
		cfg.Difficulty = new(big.Int)
	}
	if cfg.Time == nil {
		cfg.Time = big.NewInt(time.Now().Unix())
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = math.MaxUint64
	}
	if cfg.GasPrice == nil {
		cfg.GasPrice = new(big.Int)
	}
	if cfg.Value == nil {
		cfg.Value = new(big.Int)
	}
	if cfg.BlockNumber == nil {
		cfg.BlockNumber = new(big.Int)
	}
	if cfg.GetHashFn == nil {
		cfg.GetHashFn = func(n uint64) common.Hash {
			return common.BytesToHash(crypto.Keccak256([]byte(new(big.Int).SetUint64(n).String())))
		}
	}
}

// Execute在执行期间使用输入作为调用数据执行代码
// 它返回EVM的返回值、新状态，如果失败则返回一个错误
//
// execution临时在内存中设置执行给定代码的环境。它确保它在之后恢复到原来的状态。
func Execute(code, input []byte, cfg *Config) ([]byte, *state.StateDB, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
	}
	var (
		address = common.BytesToAddress([]byte("contract"))
		vmEnv   = NewEnv(cfg)
		sender  = vm.AccountRef(cfg.Origin)
	)
	fmt.Println("origin是",cfg.Origin)
	cfg.State.CreateAccount(address)
	//设置receiver(the executing contract)的执行代码。
	cfg.State.SetCode(address, code)
	// 使用给定的配置调用代码
	ret, _, err := vmEnv.Call(
		sender,
		common.BytesToAddress([]byte("contract")),
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	fmt.Println("db是",cfg.State)
	fmt.Println("value是",cfg.Value)
	return ret, cfg.State, err
}

// Create使用EVM Create方法执行代码
func Create(input []byte, cfg *Config) ([]byte, common.Address, uint64, error) {
	if cfg == nil {
		cfg = new(Config)
	}
	setDefaults(cfg)

	if cfg.State == nil {
		cfg.State, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
	}
	var (
		vmEnv  = NewEnv(cfg)
		sender = vm.AccountRef(cfg.Origin)
	)

	// 使用给定的配置调用代码。
	code, address, leftOverGas, err := vmEnv.Create(
		sender,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return code, address, leftOverGas, err
}

// Call执行给定合约地址上的代码 他将返回EVM的返回值和执行失败的错误
//
// Call, 不像 Execute, 需要一个配置文件（config）还需要设置State字段
func Call(address common.Address, input []byte, cfg *Config) ([]byte, uint64, error) {
	setDefaults(cfg)

	vmEnv := NewEnv(cfg)

	sender := cfg.State.GetOrNewStateObject(cfg.Origin)
	// 使用给定的配置调用代码
	ret, leftOverGas, err := vmEnv.Call(
		sender,
		address,
		input,
		cfg.GasLimit,
		cfg.Value,
	)
	return ret, leftOverGas, err
}
