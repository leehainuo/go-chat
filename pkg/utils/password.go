package utils

import (
	"crypto/md5"
	"encoding/hex"
)

// EncryptPassword 计算存储用密码摘要：MD5(salt + password)，salt 为固定常量。
//
// 安全提示：MD5 用于密码存储强度不足（GPU 可离线高速爆破），固定 salt 无法抵御
// 预计算彩虹表。设计评审 DR-4 已确认"接受"该风险（学习项目约束）；
// 若后续需要真实部署，仅需替换本函数实现（如 bcrypt/argon2id），调用方不受影响。
func EncryptPassword(salt, password string) string {
	sum := md5.Sum([]byte(salt + password))
	return hex.EncodeToString(sum[:])
}
