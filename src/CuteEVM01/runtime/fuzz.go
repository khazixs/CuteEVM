package runtime

// Fuzz是go-fuzz工具的基本入口点
// 这将为有效的可解析/可运行代码返回1, 为无效的OpCode返回0
func Fuzz(input []byte) int {
	_, _, err := Execute(input, input, &Config{
		GasLimit: 3000000,
	})
	// 无效的OpCode
	if err != nil && len(err.Error()) > 6 && string(err.Error()[:7]) == "invalid" {
		return 0
	}

	return 1
}
