/**
 * @Author: yinwei
 * @Description: 雪花算法ID生成器，提供分布式唯一ID生成功能
 * @File: snowflake
 * @Version: 1.0.0
 * @Date: 2023/11/1 09:58
 */
// Package snowflake 雪花算法ID生成器工具包
// 提供分布式系统中唯一ID生成功能，支持自定义节点位数
package snowflake

import (
	"fmt"
)

const (
	// nodeSequenceBitsSum 节点ID和序列号可容纳的总位数
	nodeSequenceBitsSum = uint(22)

	epoch          int64 = 1                            // 设置起始时间（这里一般是项目上线时间）
	timestampBits        = uint(42)                     // 时间戳占42位（扩展了1位以增加容量）
	nodeBits             = uint(2)                      // 节点ID占2位（最多支持4个节点）
	sequenceBits         = uint(20)                     // 序列号占20位（扩展了8位以提高并发能力）
	maxleftBits          = uint(22)                     // 除去时间戳，剩余可用位数
	nodeMax              = uint64(1<<nodeBits) - 1      // 节点ID标识最大值（2^2-1 = 3）
	sequenceMax          = uint64(1<<sequenceBits) - 1  // 序列号最大值（2^20-1 = 1048575）
	timestampMax         = uint64(1<<timestampBits) - 1 // 时间戳最大值（2^42-1）
	nodeShift            = sequenceBits                 // 节点ID左移位数（20位）
	timestampShift       = sequenceBits + nodeBits      // 时间戳左移位数（22位）
)

var (
	// SNOWFLAKE_ERROR_TIMESTAMP_OVER 雪花算法时间戳耗尽错误
	SNOWFLAKE_ERROR_TIMESTAMP_OVER = fmt.Errorf("雪花算法时间戳耗尽")
	// SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT 节点ID超出限制错误
	SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT = fmt.Errorf("node节点超出限制")
	// SNOWFLAKE_ERROR_NODE_TOO_FEW 节点数太少错误
	SNOWFLAKE_ERROR_NODE_TOO_FEW = fmt.Errorf("node节点数太少")
)

// 雪花算法ID结构说明：
//
// 标准雪花算法数据划分：
// 0 | 00000000 00000000 00000000 00000000 00000000 0 |     00000     |    00000    | 00000000 0000
//   |                 41bit时间戳                     |数据中心id 5bit |机器id 5bit  |    序列号12bit
//
// 本系统重新划分的数据结构：
// |0  00000000 00000000 00000000 00000000 00000000 0 |     00         |    00000000 00000000 0000
//   |                 42bit时间戳                       |  节点ID 2bit    |    序列号20bit
//
// 主要改进：
// 1. 数据中心ID和机器ID合并为节点ID（2bit），最多支持4个节点
// 2. 序列号扩容到20bit，大幅提高单毫秒内的并发处理能力
// 3. 时间戳扩容到42bit，延长可用时间范围
// 4. 生成的ID为无符号int64数字，从1开始递增

// NextId 生成下一个雪花ID（使用默认节点位数配置）
// 使用默认的节点bits配置（2bit），只支持4个节点
// 参数约束：
//   - 0 < nodeBits < maxleftBits
//   - 0 <= timestamp <= timestampMax
//   - 0 <= nodeId <= nodeMax
//   - 1 <= sequence <= sequenceMax（序列号从1开始，确保ID从1开始）
//
// 参数：
//   - timestamp: 当前时间戳
//   - nodeId: 节点ID
//   - sequence: 当前序列号
//
// 返回值：
//   - id: 生成的雪花ID
//   - afterSequence: 更新后的序列号
//   - afterTimeStamp: 更新后的时间戳
//   - err: 错误信息
func NextId(timestamp, nodeId, sequence uint64) (id, afterSequence, afterTimeStamp uint64, err error) {
	// 序列号必须从1开始，确保生成的ID不为0
	if sequence == 0 {
		sequence = 1
	}
	// 如果序列号超过最大值，重置为1并增加时间戳
	if sequence > sequenceMax {
		sequence = 1
		timestamp += 1
	}
	// 检查时间戳是否超出最大范围
	if timestamp > timestampMax {
		return 0, 0, 0, SNOWFLAKE_ERROR_TIMESTAMP_OVER
	}

	// 组合生成最终的ID：时间戳 | 节点ID | 序列号
	val := ((timestamp) << timestampShift) | (nodeId << nodeShift) | sequence

	// 计算下一次调用时的序列号
	sequence = (sequence + 1) & sequenceMax
	// 如果下次序列号为0，说明当前时间戳的序列号已用完，需要进入下一个时间戳
	if sequence == 0 {
		sequence = 1
		timestamp += 1
	}
	return val, sequence, timestamp, nil
}

// NextIdByNodeBits 生成下一个雪花ID（支持自定义节点位数）
// 允许自定义节点ID的位数大小，从而调整节点数量和序列号的容量平衡
// 参数约束：
//   - 0 < nodeBits < maxleftBits（节点位数必须在有效范围内）
//   - 0 <= timestamp <= timestampMax（时间戳不能超出最大值）
//   - 0 <= nodeId <= nodeMax（节点ID不能超出当前位数支持的最大值）
//   - 1 <= sequence <= sequenceMax（序列号从1开始，确保ID从1开始）
//
// 参数：
//   - timestamp: 当前时间戳
//   - nodeId: 节点ID
//   - sequence: 当前序列号
//   - nodeBits: 节点ID占用的位数
//
// 返回值：
//   - id: 生成的雪花ID
//   - afterSequence: 更新后的序列号
//   - afterTimeStamp: 更新后的时间戳
//   - err: 错误信息
func NextIdByNodeBits(timestamp, nodeId, sequence uint64, nodeBits uint) (id, afterSequence, afterTimeStamp uint64, err error) {
	// 检查节点位数是否超出最大可用位数
	if nodeBits >= maxleftBits {
		return 0, 0, 0, SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT
	}
	// 检查节点位数是否为0（至少需要1位）
	if nodeBits == 0 {
		return 0, 0, 0, SNOWFLAKE_ERROR_NODE_TOO_FEW
	}

	// 根据自定义节点位数计算相关参数
	nodeMax := uint64(1<<nodeBits) - 1 // 当前节点位数支持的最大节点ID
	if nodeId > nodeMax {
		return 0, 0, 0, SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT
	}

	// 重新计算序列号位数（总可用位数 - 节点位数）
	sequenceBits := maxleftBits - nodeBits
	// 序列号最大值
	sequenceMax := uint64(1<<sequenceBits) - 1
	// 节点ID左移位数
	nodeShift := sequenceBits
	// 时间戳左移位数
	timestampShift := sequenceBits + nodeBits

	// 序列号必须从1开始，确保生成的ID不为0
	if sequence == 0 {
		sequence = 1
	}
	// 如果序列号超过最大值，重置为1并增加时间戳
	if sequence > sequenceMax {
		sequence = 1
		timestamp += 1
	}
	// 检查时间戳是否超出最大范围
	if timestamp > timestampMax {
		return 0, 0, 0, SNOWFLAKE_ERROR_TIMESTAMP_OVER
	}

	// 组合生成最终的ID：时间戳 | 节点ID | 序列号
	val := ((timestamp) << timestampShift) | (nodeId << nodeShift) | sequence

	// 计算下一次调用时的序列号
	sequence = (sequence + 1) & sequenceMax
	// 如果下次序列号为0，说明当前时间戳的序列号已用完，需要进入下一个时间戳
	if sequence == 0 {
		sequence = 1
		timestamp += 1
	}
	return val, sequence, timestamp, nil
}

//func main() {
//
//	fmt.Println(sequenceMask)
//	fmt.Println(1048576 & sequenceMask)
//
//	//return
//	var (
//		t, node, seq uint64
//		id           uint64
//	)
//	for {
//		id, seq, t, _ = NextId(t, node, seq)
//		fmt.Printf("id=%d,t=%v,seq=%v\n", id, t, seq)
//		time.Sleep(10 * time.Millisecond)
//	}
//}
