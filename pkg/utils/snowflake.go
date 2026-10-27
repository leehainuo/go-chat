package utils

import (
	"strconv"
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	nodeOnce sync.Once
	node     *snowflake.Node
	nodeErr  error
)

func initNode() {
	nodeOnce.Do(func() {
		n, err := snowflake.NewNode(1)
		if err != nil {
			nodeErr = err
			return
		}
		node = n
	})
}

// NextId 生成全局唯一递增 ID（本地单机固定 node=1）。
func NextId() (int64, error) {
	initNode()
	if nodeErr != nil {
		return 0, nodeErr
	}
	return node.Generate().Int64(), nil
}

// NextIdStr 生成字符串形式 ID（避免 JS number 精度丢失）。
func NextIdStr() (string, error) {
	id, err := NextId()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
