package config

import (
	"strconv"
)

// atoi 字符串转整数
func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}