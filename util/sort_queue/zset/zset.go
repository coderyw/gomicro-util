// Package sort_queue
// @Author: yinwei
// @File: zset
// @Version: 1.0.0
// @Date: 2025/1/24 13:34

package zset

import (
	"fmt"
	"math/rand"
	"sync"
)

const (
	maxLevel = 32   // 跳表的最大层数
	p        = 0.25 // 跳表的概率因子，用于随机生成层数
)

type Z struct {
	Key   string
	Score float64
}

// ZSet 有序集合
type ZSet struct {
	header *zskiplistNode     // 跳表头节点
	tail   *zskiplistNode     // 跳表尾节点
	length int                // 跳表长度（成员数量）
	level  int                // 跳表当前最大层数
	dict   map[string]float64 // 字典，存储成员和分数的映射关系
	mu     sync.RWMutex
}

// 跳表节点
type zskiplistNode struct {
	key      string            // 成员键
	score    float64           // 成员分数
	backward *zskiplistNode    // 后退指针，用于逆序遍历
	level    []*zskiplistLevel // 层数组，每层包含前进指针和跨度
}

// 跳表层
type zskiplistLevel struct {
	forward *zskiplistNode // 前进指针，指向下一个节点
	span    int            // 跨度，表示当前节点到下一个节点的距离
}

// 创建节点
func createNode(level int, key string, score float64) *zskiplistNode {
	node := &zskiplistNode{
		key:   key,
		score: score,
		level: make([]*zskiplistLevel, level),
	}
	for i := 0; i < level; i++ {
		node.level[i] = &zskiplistLevel{}
	}
	return node
}

// NewZSet 初始化有序集合
//
// Returns:
//   - *ZSet 返回有序集合实例
func NewZSet() *ZSet {
	zsl := &ZSet{
		header: createNode(maxLevel, "", 0), // 创建头节点
		level:  1,
		length: 0,
		dict:   make(map[string]float64),
	}
	for i := 0; i < maxLevel; i++ {
		zsl.header.level[i].forward = nil // 初始化每层的前进指针
	}
	zsl.header.backward = nil // 头节点的后退指针为空
	zsl.tail = nil            // 尾节点初始为空
	return zsl
}

// 随机生成层数
func randomLevel() int {
	level := 1
	for rand.Float64() < p && level < maxLevel {
		level++
	}
	return level
}

// ZAdd  添加或更新成员
//
// Parameters:
//   - key: string key
//   - score: float64 有序集合分数
//
// Returns:
func (zsl *ZSet) ZAdd(key string, score float64) {
	zsl.mu.Lock() // 写锁
	defer zsl.mu.Unlock()
	// 如果成员已存在，更新分数
	if oldScore, exists := zsl.dict[key]; exists {
		if oldScore == score {
			return // 分数未变化，直接返回
		}
		zsl.ZRem(key) // 先删除旧节点
	}

	// 生成随机层数
	level := randomLevel()
	if level > zsl.level {
		// 如果新节点的层数大于当前跳表的最大层数，更新跳表的层数
		for i := zsl.level; i < level; i++ {
			zsl.header.level[i].forward = nil
		}
		zsl.level = level
	}

	// 创建新节点
	node := createNode(level, key, score)
	update := make([]*zskiplistNode, maxLevel) // 用于记录每层的插入位置
	rank := make([]int, maxLevel)              // 用于记录每层的排名

	// 查找插入位置
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		if i == zsl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		// 在当前层查找插入位置
		for x.level[i].forward != nil && (x.level[i].forward.score < score || (x.level[i].forward.score == score && x.level[i].forward.key < key)) {
			rank[i] += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x // 记录每层的插入位置
	}

	// 插入节点
	for i := 0; i < level; i++ {
		node.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = node
		node.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	// 更新未触及的层
	for i := level; i < zsl.level; i++ {
		update[i].level[i].span++
	}

	// 更新后退指针
	if update[0] != zsl.header {
		node.backward = update[0]
	}
	if node.level[0].forward != nil {
		node.level[0].forward.backward = node
	} else {
		zsl.tail = node // 如果新节点是最后一个节点，更新尾节点
	}

	// 更新字典和长度
	zsl.dict[key] = score
	zsl.length++
}

// 删除节点（内部方法，不暴露给外部）
func (zsl *ZSet) removeNode(key string, score float64) {
	update := make([]*zskiplistNode, maxLevel) // 用于记录每层的删除位置
	x := zsl.header

	// 查找节点，并记录每层的删除位置
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (x.level[i].forward.score < score || (x.level[i].forward.score == score && x.level[i].forward.key < key)) {
			x = x.level[i].forward
		}
		update[i] = x // 记录每层的删除位置
	}

	x = x.level[0].forward
	if x != nil && x.key == key && x.score == score {
		zsl.deleteNode(x, update) // 直接删除节点
		delete(zsl.dict, key)     // 从字典中移除
		zsl.length--              // 更新长度
	}
}

// ZRem 删除成员
//
// Parameters:
//   - key: string 成员key
//
// Returns:
//   - bool true=成员存在且删除成功，false=成员不存在
func (zsl *ZSet) ZRem(key string) bool {
	zsl.mu.Lock() // 写锁
	defer zsl.mu.Unlock()

	score, exists := zsl.dict[key]
	if !exists {
		return false // 成员不存在
	}

	zsl.removeNode(key, score) // 直接调用内部方法
	return true
}

// 删除节点
func (zsl *ZSet) deleteNode(x *zskiplistNode, update []*zskiplistNode) {
	for i := 0; i < zsl.level; i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].forward = x.level[i].forward
			update[i].level[i].span += x.level[i].span - 1
		} else {
			update[i].level[i].span--
		}
	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		zsl.tail = x.backward // 如果删除的是尾节点，更新尾节点
	}
	for zsl.level > 1 && zsl.header.level[zsl.level-1].forward == nil {
		zsl.level-- // 如果最高层为空，降低跳表层数
	}
}

// ZScore 获取成员的分数
//
// Parameters:
//   - key: string 成员key
//
// Returns:
//   - float64 分数
//   - bool 成员是否存在
func (zsl *ZSet) ZScore(key string) (float64, bool) {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	score, exists := zsl.dict[key]
	return score, exists
}

// ZCard 获取有序集合的长度
//
// Returns:
//   - int 集合长度
func (zsl *ZSet) ZCard() int {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	return zsl.length
}

// ZRange 按排名范围获取成员
//
// Parameters:
//   - start: int 排名开始，从0（包含）
//   - stop: int 排名结束（包含）
//
// Returns:
//   - []string
func (zsl *ZSet) ZRange(start, stop int) []string {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	if start < 0 {
		start += zsl.length
	}
	if stop < 0 {
		stop += zsl.length
	}
	if start > stop || start >= zsl.length || stop < 0 {
		return nil
	}

	result := make([]string, 0)
	x := zsl.header.level[0].forward
	for i := 0; x != nil && i <= stop; i++ {
		if i >= start {
			result = append(result, x.key)
		}
		x = x.level[0].forward
	}
	return result
}

// ZRangeWithScore 按排名范围获取成员返回携带分数
//
// Parameters:
//   - start: int 排名开始，从0（包含）
//   - stop: int 排名结束（包含）
//
// Returns:
//   - []Z 返回 Z 类型切片
func (zsl *ZSet) ZRangeWithScore(start, stop int) []Z {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	if start < 0 {
		start += zsl.length
	}
	if stop < 0 {
		stop += zsl.length
	}
	if start > stop || start >= zsl.length || stop < 0 {
		return nil
	}

	result := make([]Z, 0)
	x := zsl.header.level[0].forward
	for i := 0; x != nil && i <= stop; i++ {
		if i >= start {
			result = append(result, Z{
				Key:   x.key,
				Score: x.score,
			})
		}
		x = x.level[0].forward
	}
	return result
}

// ZRevRange 按排名范围获取成员（降序）
//
// Parameters:
//   - start: int 排名开始，从0（包含）
//   - stop: int 排名结束（包含）
//
// Returns:
//   - []string
func (zsl *ZSet) ZRevRange(start, stop int) []string {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	if start < 0 {
		start += zsl.length
	}
	if stop < 0 {
		stop += zsl.length
	}
	if start > stop || start >= zsl.length || stop < 0 {
		return nil
	}

	result := make([]string, 0)
	x := zsl.tail
	for i := zsl.length - 1; x != nil && i >= start; i-- {
		if i <= stop {
			result = append(result, x.key)
		}
		x = x.backward
	}
	return result
}

// ZRevRangeWithScore 按排名范围获取成员（降序）
//
// Parameters:
//   - start: int 排名开始，从0（包含）
//   - stop: int 排名结束（包含）
//
// Returns:
//   - []Z
func (zsl *ZSet) ZRevRangeWithScore(start, stop int) []Z {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	if start < 0 {
		start += zsl.length
	}
	if stop < 0 {
		stop += zsl.length
	}
	if start > stop || start >= zsl.length || stop < 0 {
		return nil
	}

	result := make([]Z, 0)
	x := zsl.tail
	for i := zsl.length - 1; x != nil && i >= start; i-- {
		if i <= stop {
			result = append(result, Z{
				Key:   x.key,
				Score: x.score,
			})
		}
		x = x.backward
	}
	return result
}

// ZRangeByScore 按分数范围获取成员
//
// Parameters:
//   - min: float64 最小分数
//   - max: float64 最大分数
//
// Returns:
//   - []string 返回成员列表
func (zsl *ZSet) ZRangeByScore(min, max float64) []string {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	result := make([]string, 0)
	x := zsl.header.level[0].forward
	for x != nil && x.score < min {
		x = x.level[0].forward
	}
	for x != nil && x.score <= max {
		result = append(result, x.key)
		x = x.level[0].forward
	}
	return result
}

// ZRangeByScoreWithScore 按分数范围获取成员
//
// Parameters:
//   - min: float64 最小分数
//   - max: float64 最大分数
//
// Returns:
//   - []Z 返回成员列表
func (zsl *ZSet) ZRangeByScoreWithScore(min, max float64) []Z {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	result := make([]Z, 0)
	x := zsl.header.level[0].forward
	for x != nil && x.score < min {
		x = x.level[0].forward
	}
	for x != nil && x.score <= max {
		result = append(result, Z{
			Key:   x.key,
			Score: x.score,
		})
		x = x.level[0].forward
	}
	return result
}

// ZRevRangeByScore  按分数范围获取成员（降序）
//
// Parameters:
//   - min: float64 最小分数
//   - max: float64 最大分数
//
// Returns:
//   - []string 返回成员列表
func (zsl *ZSet) ZRevRangeByScore(min, max float64) []string {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	result := make([]string, 0)
	x := zsl.tail
	for x != nil && x.score > max {
		x = x.backward
	}
	for x != nil && x.score >= min {
		result = append(result, x.key)
		x = x.backward
	}
	return result
}

// ZRevRangeByScoreWithScore  按分数范围获取成员（降序）
//
// Parameters:
//   - min: float64 最小分数
//   - max: float64 最大分数
//
// Returns:
//   - []Z 返回成员列表
func (zsl *ZSet) ZRevRangeByScoreWithScore(min, max float64) []Z {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	result := make([]Z, 0)
	x := zsl.tail
	for x != nil && x.score > max {
		x = x.backward
	}
	for x != nil && x.score >= min {
		result = append(result, Z{
			Key:   x.key,
			Score: x.score,
		})
		x = x.backward
	}
	return result
}

// ZRank 获取成员的排名（升序）
//
// Parameters:
//   - key: string 元素key
//
// Returns:
//   - int 返回成员的排名（升序）,-1表示元素不存在
func (zsl *ZSet) ZRank(key string) int {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	score, exists := zsl.dict[key]
	if !exists {
		return -1
	}

	rank := 0
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (x.level[i].forward.score < score || (x.level[i].forward.score == score && x.level[i].forward.key < key)) {
			rank += x.level[i].span
			x = x.level[i].forward
		}
		if x.level[i].forward != nil && x.level[i].forward.key == key {
			return rank
		}
	}
	return -1
}

// ZRevRank 获取成员的排名（降序）
//
// Parameters:
//   - key: string 元素key
//
// Returns:
//   - int 返回成员的排名（降序）,-1表示元素不存在
func (zsl *ZSet) ZRevRank(key string) int {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	rank := zsl.ZRank(key)
	if rank == -1 {
		return -1
	}
	return zsl.length - rank - 1
}

// ZCount 统计分数范围内的成员数量
//
// Parameters:
//   - min: float64
//   - max: float64
//
// Returns:
//   - int
func (zsl *ZSet) ZCount(min, max float64) int {
	zsl.mu.RLock() // 读锁
	defer zsl.mu.RUnlock()
	count := 0
	x := zsl.header.level[0].forward
	for x != nil && x.score < min {
		x = x.level[0].forward
	}
	for x != nil && x.score <= max {
		count++
		x = x.level[0].forward
	}
	return count
}

// ZRemRangeByRank 按排名范围删除成员,升序
//
// Parameters:
//   - start: int
//   - stop: int
//
// Returns:
//   - int
func (zsl *ZSet) ZRemRangeByRank(start, stop int, callback ...func(key string, score float64)) int {
	zsl.mu.Lock() // 写锁
	defer zsl.mu.Unlock()
	if start < 0 {
		start += zsl.length
	}
	if stop < 0 {
		stop += zsl.length
	}
	if start > stop || start >= zsl.length || stop < 0 {
		return 0
	}

	removed := 0
	update := make([]*zskiplistNode, maxLevel) // 用于记录每层的删除位置
	x := zsl.header

	// 查找起始位置，并记录每层的删除位置
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && removed < start {
			removed += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x // 记录每层的删除位置
	}

	// 移动到起始位置
	x = x.level[0].forward

	// 遍历并删除排名在 [start, stop] 范围内的节点
	for x != nil && removed <= stop {
		next := x.level[0].forward // 记录下一个节点
		zsl.deleteNode(x, update)  // 直接删除节点
		delete(zsl.dict, x.key)    // 从字典中移除
		zsl.length--               // 更新长度
		removed++

		// 调用回调函数，处理当前节点
		if len(callback) > 0 && callback[0] != nil {
			callback[0](x.key, x.score)
		}

		x = next // 移动到下一个节点
	}

	return removed - start
}

// ZRemRangeByScore 按分数范围删除成员
//
// Parameters:
//   - min: float64
//   - max: float64
//
// Returns:
//   - int
func (zsl *ZSet) ZRemRangeByScore(min, max float64, callback ...func(key string, score float64)) int {
	zsl.mu.Lock() // 写锁
	defer zsl.mu.Unlock()
	removed := 0
	update := make([]*zskiplistNode, maxLevel) // 用于记录每层的删除位置
	x := zsl.header

	// 查找分数范围内的第一个节点，并记录每层的删除位置
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && x.level[i].forward.score < min {
			x = x.level[i].forward
		}
		update[i] = x // 记录每层的删除位置
	}

	// 移动到第一个分数 >= min 的节点
	x = x.level[0].forward

	// 遍历并删除分数在 [min, max] 范围内的节点
	for x != nil && x.score <= max {
		next := x.level[0].forward // 记录下一个节点
		zsl.deleteNode(x, update)  // 直接删除节点
		delete(zsl.dict, x.key)    // 从字典中移除
		zsl.length--               // 更新长度
		removed++

		// 调用回调函数，处理当前节点
		if len(callback) > 0 && callback[0] != nil {
			callback[0](x.key, x.score)
		}

		x = next // 移动到下一个节点
	}

	return removed
}

// 打印有序集合中的所有成员（升序或降序）
func (zsl *ZSet) Print(reverse bool) {
	zsl.mu.RLock()
	defer zsl.mu.RUnlock()
	if reverse {
		// 降序打印
		x := zsl.tail
		for x != nil {
			fmt.Printf("BusinessKey: %s, Score: %.2f\n", x.key, x.score)
			x = x.backward
		}
	} else {
		// 升序打印
		x := zsl.header.level[0].forward
		for x != nil {
			fmt.Printf("BusinessKey: %s, Score: %.2f\n", x.key, x.score)
			x = x.level[0].forward
		}
	}
}
