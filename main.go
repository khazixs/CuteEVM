package main

import "fmt"

func main() {
	var n int
	var nextNum int
	var tag bool
	fmt.Println(calcu())
	fmt.Println(n)
	fmt.Println(nextNum)
	fmt.Println(tag)

	// 这里我们定义了一个可以存储整数类型的带缓冲通道
	// 缓冲区大小为2
	//var chans []chan int
	//var ch chan int
	//for i:=0;i<10;i++{
	//	ch = make(chan int, 2)
	//	chans = append(chans, ch)
	//	}
	//if len(chans)!=0 {
	//	chans[0]<-1
	//	chans[0]<-3
	//	chans[2]<-3
	//	chans[2]<-3
	//	chans[3]<-3
	//}
	//// 因为 ch 是带缓冲的通道，我们可以同时发送两个数据
	//// 而不用立刻需要去同步读取数据
	//
	//// 获取这两个数据
	//fmt.Println(<-chans[0])
	//fmt.Println(<-chans[2])
	//fmt.Println(<-chans[2])
	//fmt.Println(<-chans[3])
	//fmt.Println(<-chans[0])
}

func calcu() int {

	return 0
}