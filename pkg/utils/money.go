package utils

import "fmt"

// FenToYuan 将"分"金额格式化为"元"字符串（两位小数），支持负数（扣减流水）。
// 关键不变量：余额一律以"分"（int64）存储，仅在展示层转换为"元"。
func FenToYuan(fen int64) string {
	sign := ""
	if fen < 0 {
		sign = "-"
		fen = -fen
	}
	return fmt.Sprintf("%s%d.%02d", sign, fen/100, fen%100)
}
