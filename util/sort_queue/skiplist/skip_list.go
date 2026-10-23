// Package sort_queue
// @Author: yinwei
// @File: skip_list
// @Version: 1.0.0
// @Date: 2025/1/23 10:23

package skiplist

import (
	"fmt"
	"math/rand"
)

const maxLevel = 32
const pFactor = 0.25

type SkipListNode struct {
	value   int64
	key     string
	forward []*SkipListNode
}

type SkipList struct {
	head  *SkipListNode
	level int
}

func NewSkipList() SkipList {
	return SkipList{&SkipListNode{-1, "", make([]*SkipListNode, maxLevel)}, 0}
}

func (SkipList) randomLevel() int {
	lv := 1
	for lv < maxLevel && rand.Float64() < pFactor {
		lv++
	}
	return lv
}

func (s *SkipList) Search(target int64) bool {
	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		// 找到第 i 层小于且最接近 target 的元素
		for curr.forward[i] != nil && curr.forward[i].value < target {
			curr = curr.forward[i]
		}
	}
	curr = curr.forward[0]
	// 检测当前元素的值是否等于 target
	return curr != nil && curr.value == target
}

func (s *SkipList) Add(num int64, key string) {
	update := make([]*SkipListNode, maxLevel)
	for i := range update {
		update[i] = s.head
	}
	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		// 找到第 i 层小于且最接近 num 的元素
		for curr.forward[i] != nil && curr.forward[i].value < num {
			curr = curr.forward[i]
		}
		update[i] = curr
	}
	lv := s.randomLevel()
	s.level = max(s.level, lv)
	newNode := &SkipListNode{num, key, make([]*SkipListNode, lv)}
	for i, node := range update[:lv] {
		// 对第 i 层的状态进行更新，将当前元素的 forward 指向新的节点
		newNode.forward[i] = node.forward[i]
		node.forward[i] = newNode
	}
}

func (s *SkipList) Remove(num int64) bool {
	update := make([]*SkipListNode, maxLevel)
	curr := s.head
	for i := s.level - 1; i >= 0; i-- {
		// 找到第 i 层小于且最接近 num 的元素
		for curr.forward[i] != nil && curr.forward[i].value < num {
			curr = curr.forward[i]
		}
		update[i] = curr
	}
	curr = curr.forward[0]
	// 如果值不存在则返回 false
	if curr == nil || curr.value != num {
		return false
	}
	for i := 0; i < s.level && update[i].forward[i] == curr; i++ {
		// 对第 i 层的状态进行更新，将 forward 指向被删除节点的下一跳
		update[i].forward[i] = curr.forward[i]
	}
	// 更新当前的 level
	for s.level > 1 && s.head.forward[s.level-1] == nil {
		s.level--
	}
	return true
}

func max(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func (sl *SkipList) Print() {
	for i := sl.level - 1; i >= 0; i-- {
		current := sl.head.forward[i]
		fmt.Printf("Level %d: ", i)
		for current != nil {
			fmt.Printf("%d -> ", current.value)
			current = current.forward[i]
		}
		fmt.Println("nil")
	}
}
