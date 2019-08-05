package runtime_test

import (
	"CuteEVM01/Out/common"
	"CuteEVM01/runtime"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)
var (
    FileName string = "C:\\Users\\ZQ\\Downloads\\aaa_sol_AddTest.bin"    //这是我们需要打开的文件，当然你也可以把它定义到从某个配置文件来获取变量。
)
func TestExampleExecute(t *testing.T) {
	buf, err := ioutil.ReadFile(FileName) //将整个文件的内容读到一个字节切片中。
	fmt.Println("利用反射得到的文件读取结果类型是",reflect.TypeOf(buf))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "文件错误: %s\n", err)
	}else{
		fmt.Printf("智能合约的字节码是%s\n", string(buf))
	}
	str:= string(buf)
	content01 := str[68 : len(str)-1]
	ret, _, err := runtime.Execute(common.Hex2Bytes(content01), nil, nil)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(ret)
	//fmt.Println(len("6060604052600a8060106000396000f360606040526008565b00"))
	// Output:
	// [96 96 96 64 82 96 8 86 91 0]
}
