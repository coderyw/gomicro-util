// Package snowflake
// @Author: yinwei
// @File: snowflake_rvs
// @Version: 1.0.0
// @Date: 2023/11/22 10:04

// 雪花算法改版

package snowflake

import "fmt"

////
//// 改版雪花id，会将时间戳定义到中间位置，节点放在首位，如下
//// 00        |0  00000000 00000000 00000000 00000000 00000000 0 |  00000000 00000000 0000
////
//// 节点ID 2bit|                 42bit时间戳                       |   序列号20bit
//// 时间戳默认42bit，的uint64;每次递增序列号，如果序列号超过长度则会从头开始，然后时间戳+1
//// 节点id，和序列号为变长，具体根据业务需求定义，但是，这两个最长都不会超过 22bit
//// 业务需求可传入节点id的bit数，nodeBits,然后回生产序列化的bit，leftBits-nodeBits
//// 这种定义即是按着节点id，进行划分id范围，比如节点nodeBits=4,即0<=nodeId<16,将节点划分16份，
//// 每份获取到的id不一样，并且，nodeId下的id大小会是根据nodeId大小来递增，nodeId0<nodeId1<...<nodeId15
////
//// 适用：数据库分表，每个表id分布式id，并且每个表id根据表不同而递增，对于分页查询友好

// NextRvsIdByNodeBits
// 自定义node节点数的bits大小
// 0< nodeBits < maxleftBits
// 0<= timestamp <= timestampMax
// 0<= nodeId <= nodeMax
// 1<= sequence <= sequenceMax 序列号从1开始，这样保证id会从1开始
func NextRvsIdByNodeBits(timestamp, nodeId, sequence uint64, nodeBits uint) (id, afterSequence, afterTimeStamp uint64, err error) {
	if nodeBits >= maxleftBits {
		return 0, 0, 0, SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT
	}
	//if nodeBits == 0 { //bits=1表示只有一个节点
	//	return 0, 0, 0, SNOWFLAKE_ERROR_NODE_TOO_FEW
	//}
	nodeMax := uint64(1<<nodeBits) - 1
	if nodeId > nodeMax {
		return 0, 0, 0, SNOWFLAKE_ERROR_NODE_EXCEEDS_LIMIT
	}
	// 重新计算 序列号bit
	sequenceBits := maxleftBits - nodeBits
	// 序列号 最大值
	sequenceMax := uint64(1<<sequenceBits) - 1
	// node左移位数
	timestampShift := sequenceBits
	nodeShift := timestampShift + timestampBits

	if sequence == 0 { //序列化必须从1开始
		sequence = 1
	}
	if sequence > sequenceMax { //序列化超过，重新开始序列化，时间戳+1
		sequence = 1
		timestamp += 1
	}
	if timestamp > timestampMax {
		return 0, 0, 0, SNOWFLAKE_ERROR_TIMESTAMP_OVER
	}

	val := (nodeId << nodeShift) | ((timestamp) << timestampShift) | sequence
	sequence = (sequence + 1) & sequenceMax //计算下次序列号
	if sequence == 0 {                      //下次序列号=0，表明已经用完了，需要重新开始新一轮
		sequence = 1
		timestamp += 1
	}
	return val, sequence, timestamp, nil
}

func GetNodeIdById(id uint64, nodeBits uint) int {
	nodeNum := int(1<<nodeBits) - 1
	if nodeBits < 1 {
		panic(fmt.Sprintf("nodeBits length error %v", nodeNum))
	}
	var (
		maxId uint64
		minId uint64
	)
	for i, j := 0, nodeNum; i < j; i, j = i+1, j-1 {
		maxId, _, _, _ = NextRvsIdByNodeBits(0, uint64(i+1), 1, nodeBits)
		if id < maxId {
			return i
		}
		minId, _, _, _ = NextRvsIdByNodeBits(0, uint64(j), 1, nodeBits)
		if id >= minId {
			return j
		}
	}
	return 0
}
