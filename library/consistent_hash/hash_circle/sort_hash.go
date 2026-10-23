package hash_circle

import (
	"hash/fnv"
	"slices"
	"strings"
	"sync"

	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// 初始化环长
const initHashCircleCap = 200

type SortConsistentHash struct {
	sync.RWMutex
	// nodes 在 hash 之后升序排列数组
	hashCircle []*nodeInfo
	// node 对应 在 hashCircle上的索引
	mp        map[string]struct{}
	totalHash int64
	//hashCircleStart int
}

type nodeInfo struct {
	Node     string
	NodeHash int64
}

func NewSortHash() *SortConsistentHash {
	return &SortConsistentHash{
		RWMutex:    sync.RWMutex{},
		hashCircle: make([]*nodeInfo, 0, initHashCircleCap),
		mp:         make(map[string]struct{}, initHashCircleCap),
	}
}

// 在 hash 环上添加信元素 safe
func (s *SortConsistentHash) Add(node string) {
	s.Lock()
	defer s.Unlock()
	// 不允许重复添加
	if _, ok := s.mp[node]; ok {
		return
	}
	s.sortAdd(node)

}

func (s *SortConsistentHash) Remove(node string) {
	s.Lock()
	defer s.Unlock()
	hsmu := hashSum(node)

	idx := slices.IndexFunc(s.hashCircle, func(info *nodeInfo) bool {
		return info.NodeHash == hsmu
	})
	if idx < 0 {
		return
	}
	s.startIndexCul(-hsmu)
	s.hashCircle = append(s.hashCircle[:idx], s.hashCircle[idx+1:]...)
	delete(s.mp, node)

}

// 复制的方式添加元素 not safe
//func (s *SortConsistentHash) copyAdd(node string) {
//	hsum := hashSum(node)
//	// 找到插入位置
//	slices.IndexFunc(s.hashCircle, func(info *nodeInfo) bool {
//
//	})
//	idx := sort.Search(len(s.hashCircle), func(i int) bool {
//
//		return s.hashCircle[i].NodeHash >= hsum
//	})
//
//	// 在 idx 处插入
//	s.hashCircle = append(s.hashCircle, nil)       // 扩展切片
//	copy(s.hashCircle[idx+1:], s.hashCircle[idx:]) // 将 idx 及后面元素右移一位
//	s.hashCircle[idx] = &nodeInfo{
//		Node:     node,
//		NodeHash: hsum,
//	}
//	//s.mp[node] = idx
//	s.startIndexCul(hsum)
//}

// 重新排序的方式添加元素 not safe
func (s *SortConsistentHash) sortAdd(node string) {
	hsum := hashSum(node)
	s.hashCircle = append(s.hashCircle, &nodeInfo{
		Node:     node,
		NodeHash: hsum,
	})
	slices.SortFunc(s.hashCircle, func(a, b *nodeInfo) int {
		if int(a.NodeHash-b.NodeHash) == 0 {
			return strings.Compare(strings.ToLower(a.Node), strings.ToLower(b.Node))
		}
		return int(a.NodeHash - b.NodeHash)
	})
	s.mp[node] = struct{}{}
	s.startIndexCul(hsum)
}

// 计算 hash 初始位置
func (s *SortConsistentHash) startIndexCul(hsum int64) {
	s.totalHash += hsum
	// 总共 hash 值， hash 环长度，取模，得到 hash 环的起始位置
	//s.hashCircleStart = int(s.totalHash) % len(s.hashCircle)
}

// Gets 获取 targetNode 分配的 arr
func (s *SortConsistentHash) Gets(targetNode string, arr []string) []string {
	s.RLock()
	defer s.RUnlock()
	// 获取节点在哈希环上的索引
	_, ok := s.mp[targetNode]
	if !ok {
		return nil
	}
	idx := slices.IndexFunc(s.hashCircle, func(info *nodeInfo) bool {
		return info.Node == targetNode
	})
	l := len(s.hashCircle) //hash 环长度
	log.Debug("Gets", zap.Any("s.hashCircle", s.hashCircle))
	var resp []string
	for i, v := range arr {
		nodeIdx := (i + int(s.totalHash)) % l
		log.Debug("gets", zap.Int("nodeIdx", nodeIdx), zap.Int("current", idx))
		if idx == nodeIdx { //
			resp = append(resp, v)
		}
	}
	return resp
}

func (s *SortConsistentHash) GetAll(arr []string) map[string][]string {
	s.RLock()
	defer s.RUnlock()
	resp := make(map[string][]string)
	l := len(s.hashCircle) //hash 环长度
	for i, v := range arr {
		nodeIdx := (i + int(s.totalHash)) % l
		if ar, ok := resp[s.hashCircle[nodeIdx].Node]; ok {
			ar = append(ar, v)
			resp[s.hashCircle[nodeIdx].Node] = ar
		} else {
			resp[s.hashCircle[nodeIdx].Node] = []string{v}
		}

	}
	return resp
}

func (r *SortConsistentHash) Members() []string {
	r.RLock()
	defer r.RUnlock()
	mem := make([]string, 0, len(r.hashCircle))
	for _, k := range r.hashCircle {
		mem = append(mem, k.Node)
	}
	return mem
}

// 转 hash，然后排序 not safe
func (s *SortConsistentHash) sort(arr []string) []int64 {
	var sortArr = make([]int64, 0, len(arr))
	for _, v := range arr {
		sortArr = append(sortArr, hashSum(v))
	}
	slices.Sort(sortArr)
	return sortArr
}

func hashSum(str string) int64 {
	h := fnv.New32a()
	h.Write([]byte(str))
	return int64(h.Sum32())
}
