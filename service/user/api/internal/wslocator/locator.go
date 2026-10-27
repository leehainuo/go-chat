package wslocator

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"go-chat/service/user/api/internal/consts"
)

const virtualNodesPerAddr = 160

type vnode struct {
	hash uint32
	addr string
}

// Locator 一致性哈希 WS 节点定位器。
// 节点列表来自 etcd prefix（realtime 服务注册目录），后台 Watch 变更自动重建哈希环，
// 解决固定取模在节点扩缩容时大量连接被打散重连的问题。
type Locator struct {
	mu     sync.RWMutex
	ring   []vnode
	client *clientv3.Client
	prefix string
}

// NewLocator 构造定位器。endpoints 为空或连接失败时环为空：
// GetWsServerUri 返回空串、登录流程不受影响（phase-07 realtime 上线前的降级约定）。
func NewLocator(endpoints []string, prefix string) *Locator {
	l := &Locator{prefix: prefix}
	if len(endpoints) == 0 {
		return l
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
		Logger:      zap.NewNop(),
	})
	if err != nil {
		logx.Errorf("[wslocator] 连接 etcd 失败: %v，哈希环为空，GetWsServerUri 将返回空串", err)
		return l
	}
	l.client = cli
	l.reload()
	go l.watch()
	return l
}

func (l *Locator) reload() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := l.client.Get(ctx, l.prefix, clientv3.WithPrefix())
	if err != nil {
		logx.Errorf("[wslocator] 拉取节点失败 prefix=%s: %v", l.prefix, err)
		return
	}
	addrs := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Value))
	}
	l.rebuild(addrs)
}

func (l *Locator) watch() {
	rch := l.client.Watch(context.Background(), l.prefix, clientv3.WithPrefix())
	for range rch {
		l.reload()
	}
}

func (l *Locator) rebuild(addrs []string) {
	ring := make([]vnode, 0, len(addrs)*virtualNodesPerAddr)
	for _, addr := range addrs {
		for i := 0; i < virtualNodesPerAddr; i++ {
			ring = append(ring, vnode{
				hash: fnv32a(fmt.Sprintf("%s#%d", addr, i)),
				addr: addr,
			})
		}
	}
	sort.Slice(ring, func(i, j int) bool { return ring[i].hash < ring[j].hash })

	l.mu.Lock()
	l.ring = ring
	l.mu.Unlock()
}

// GetWsServerUri 对 userId 做一致性哈希，返回 "ws://addr/ws/chat"；环为空返回空串。
func (l *Locator) GetWsServerUri(userId int64) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.ring) == 0 {
		return ""
	}
	h := fnv32a(strconv.FormatInt(userId, 10))
	idx := sort.Search(len(l.ring), func(i int) bool { return l.ring[i].hash >= h })
	if idx == len(l.ring) {
		idx = 0
	}
	return "ws://" + l.ring[idx].addr + consts.WsChatPath
}

func fnv32a(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
