package main

import (
	vm "CuteEVM01"
	"CuteEVM01/Out/common"
	"CuteEVM01/Out/core"
	"CuteEVM01/Out/core/rawdb"
	"CuteEVM01/Out/core/state"
	"CuteEVM01/Out/params"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"time"
)
func main() {
	var (
		FileName string = "C:\\Users\\ZQ\\Downloads\\aaa_sol_AddTest.bin"    //这是我们需要打开的文件，当然你也可以把它定义到从某个配置文件来获取变量。
	)
	buf, err := ioutil.ReadFile(FileName) //将整个文件的内容读到一个字节切片中。
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "文件错误: %s\n", err)
	}else{
		fmt.Println(len(string(buf)))
	}
	str:= string(buf)


	chainCfg := &params.ChainConfig{
		ChainID:             big.NewInt(2),
		HomesteadBlock:      new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
	address := common.BytesToAddress([]byte("contract"))
	//配置虚拟机基本需求信息即部分链上信息
	context01 := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Origin:      common.Address{},
		GasPrice:    new(big.Int),
		Coinbase:    common.Address{},
		GasLimit:    math.MaxUint64,
		BlockNumber: new(big.Int),
		Time:        big.NewInt(time.Now().Unix()),
		Difficulty:  new(big.Int),
	}
	cfg01 := vm.Config{
		Debug:                   true,
		NoRecursion:             false,
		EnablePreimageRecording: true,
	}
	//生成虚拟机对象
	var evm01 = vm.NewEVM(context01, statedb, chainCfg, cfg01)
	sender  := vm.AccountRef(context01.Origin)
	//aaa := vm.ContractRef.Address(sender)
	//设置合约
	statedb.CreateAccount(address)
	//设置代码地址和代码
	statedb.SetCode(address, common.FromHex(str))
	//evm01.StateDB.AddBalance(sender.Address(),new(big.Int).SetInt64(10000))
	fmt.Println(evm01.StateDB.GetBalance(sender.Address()))
	value01 := new(big.Int).SetInt64(10)
	ret, _, err := evm01.Call(
		sender,
		common.BytesToAddress([]byte("contract")),
		nil,
		context01.GasLimit,
		value01,
	)
	if err!=nil{
		fmt.Println("err-->",err)
	}
	fmt.Println("ret-->",ret)
}
