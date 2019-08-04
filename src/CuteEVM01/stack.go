package vm

import (
	"fmt"
	"math/big"
)

//堆栈是用于基本堆栈操作的对象。弹出到堆栈中的项预计将被更改和修改。堆栈不负责添加新初始化的对象。
type Stack struct {
	data []*big.Int
}

func newstack() *Stack {
	return &Stack{data: make([]*big.Int, 0, 1024)}
}

//数据返回底层的big.Int数组。
func (st *Stack) Data() []*big.Int {
	return st.data
}

func (st *Stack) push(d *big.Int) {
	//注意:在baseChec中检查了push limit (1024)k
	//stackItem := new(big.Int).Set(d)
	//st.data = append(st.data, stackItem)
	st.data = append(st.data, d)
}
func (st *Stack) pushN(ds ...*big.Int) {
	st.data = append(st.data, ds...)
}

func (st *Stack) pop() (ret *big.Int) {
	ret = st.data[len(st.data)-1]
	st.data = st.data[:len(st.data)-1]
	return
}

func (st *Stack) len() int {
	return len(st.data)
}

func (st *Stack) swap(n int) {
	st.data[st.len()-n], st.data[st.len()-1] = st.data[st.len()-1], st.data[st.len()-n]
}

func (st *Stack) dup(pool *intPool, n int) {
	st.push(pool.get().Set(st.data[st.len()-n]))
}

func (st *Stack) peek() *big.Int {
	return st.data[st.len()-1]
}

// 返回堆栈中的第n项
func (st *Stack) Back(n int) *big.Int {
	return st.data[st.len()-n-1]
}

func (st *Stack) require(n int) error {
	if st.len() < n {
		return fmt.Errorf("stack underflow (%d <=> %d)", len(st.data), n)
	}
	return nil
}

// Print打印当前堆栈中的内容
func (st *Stack) Print() {
	fmt.Println("### stack ###")
	if len(st.data) > 0 {
		for i, val := range st.data {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}
