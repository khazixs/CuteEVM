package main

import (
	vm "CuteEVM01"
	"CuteEVM01/Out/common"
	"CuteEVM01/Out/core/rawdb"
	"CuteEVM01/Out/core/state"
	"CuteEVM01/Out/params"
	"fmt"
	"math/big"
)
func main() {
	chainCfg := &params.ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      new(big.Int),
		ByzantiumBlock:      new(big.Int),
		ConstantinopleBlock: new(big.Int),
		DAOForkBlock:        new(big.Int),
		DAOForkSupport:      false,
		EIP150Block:         new(big.Int),
		EIP155Block:         new(big.Int),
		EIP158Block:         new(big.Int),
	}
	code := "6060604052600a8060106000396000f360606040526008565b00"
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
	sender := common.BytesToAddress([]byte("sender"))
	receiver := common.BytesToAddress([]byte("receiver"))
	statedb.CreateAccount(sender)
	statedb.SetCode(receiver, common.FromHex(code))
	context01 := vm.Context{
		Origin:      sender,
		GasPrice:    new(big.Int),
		Coinbase:    common.Address{},
		GasLimit:    1000000,
		BlockNumber: new(big.Int).SetUint64(1),
		Time:        new(big.Int).SetUint64(0),
		Difficulty:  big.NewInt(0x200000),
	}
	cfg01 := vm.Config{
		Debug:                   true,
		Tracer:                  nil,
		NoRecursion:             false,
		EnablePreimageRecording: true,
	}
	var evm01 = vm.NewEVM(context01, statedb, chainCfg, cfg01)
	var bytes []byte
	var accountRef vm.AccountRef
	fmt.Println("sender是",sender)
	accountRef = vm.AccountRef(sender)
	fmt.Println("accountRef是",accountRef)
	var contractRef vm.ContractRef
	fmt.Println(contractRef)
	fmt.Println(evm01)
	fmt.Println(accountRef)
	contract01:= vm.NewContract(contractRef,accountRef,new(big.Int),10000)
	bytes, _ = vm.Run(evm01, contract01, nil, false)
	fmt.Println(bytes)
}
