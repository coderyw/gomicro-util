package hash_circle

import (
	"github.com/stathat/consistent"
)

func NewRandomConsistentHash(replicas int) *RandomConsistentHash {
	c := consistent.New()
	c.NumberOfReplicas = replicas
	return &RandomConsistentHash{c: c}
}

type RandomConsistentHash struct {
	c *consistent.Consistent
}

func (r *RandomConsistentHash) Add(node string) {
	r.c.Add(node)
}

func (r *RandomConsistentHash) Remove(node string) {
	r.c.Remove(node)
}

func (r *RandomConsistentHash) Gets(targetNode string, arr []string) []string {
	vals := []string{}
	for _, val := range arr {
		// 忽略错误，错误是在哈希环为 0 时候才会出现的
		id, _ := r.c.Get(val)
		// 计算当前任务 val，所落在哪个服务上执行。如果等于当前服务 id，则可以返回
		if id == targetNode {
			vals = append(vals, val)
		}
	}
	return vals
}

func (r *RandomConsistentHash) Members() []string {
	return r.c.Members()
}
