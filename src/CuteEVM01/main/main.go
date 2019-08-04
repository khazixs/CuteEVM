/*用于捕获本地的solcjs生成的字节码，并调用虚拟机对外接口，获取虚拟机返回结果并展示出来*/
package main

import (
	"CuteEVM01"
	"fmt"
)

func main() {
	//fmt.Println("Hello World!!!")
	var EvmObject vm.EVM
	var ContextObject vm.Context


	fmt.Println(ContextObject)
	fmt.Println(EvmObject)
}