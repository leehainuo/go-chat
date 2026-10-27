package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// RandomCaptcha 生成 n 位数字验证码，使用 crypto/rand 保证随机性。
func RandomCaptcha(n int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", n, v.Int64()), nil
}
