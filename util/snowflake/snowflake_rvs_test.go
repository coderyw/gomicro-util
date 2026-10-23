// Package snowflake
// @Author: yinwei
// @File: snowflake_rvs_test.go
// @Version: 1.0.0
// @Date: 2023/11/22 10:21

package snowflake

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestNextRvsIdByNodeBits(t *testing.T) {
	type args struct {
		timestamp uint64
		nodeId    uint64
		sequence  uint64
		nodeBits  uint
	}
	tests := []struct {
		name    string
		args    args
		wantId  uint64
		wantErr bool
	}{
		{name: "nodeBits-4,nodeId-0", args: args{
			timestamp: 0,
			nodeId:    0,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-1", args: args{
			timestamp: 0,
			nodeId:    1,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-2", args: args{
			timestamp: 0,
			nodeId:    2,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-3", args: args{
			timestamp: 0,
			nodeId:    3,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-4", args: args{
			timestamp: 0,
			nodeId:    4,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-5", args: args{
			timestamp: 0,
			nodeId:    5,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-6", args: args{
			timestamp: 0,
			nodeId:    6,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-7", args: args{
			timestamp: 0,
			nodeId:    7,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-8", args: args{
			timestamp: 0,
			nodeId:    8,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-16", args: args{
			timestamp: 0,
			nodeId:    16,
			sequence:  0,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-0,max序列号", args: args{
			timestamp: 0,
			nodeId:    1,
			sequence:  uint64(1<<18) - 1,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-4,nodeId-15", args: args{
			timestamp: 0,
			nodeId:    15,
			sequence:  uint64(1<<18) - 2,
			nodeBits:  4,
		}, wantErr: false},
		{name: "nodeBits-0,nodeId-15", args: args{
			timestamp: uint64(time.Now().Unix()),
			nodeId:    0,
			sequence:  1,
			nodeBits:  0,
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotId, gotAfterSequence, gotAfterTimeStamp, err := NextRvsIdByNodeBits(tt.args.timestamp, tt.args.nodeId, tt.args.sequence, tt.args.nodeBits)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextRvsIdByNodeBits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("gotId= %v ,  gotAfterSequence=%v,gotAfterTimeStamp=%v", gotId, gotAfterSequence, gotAfterTimeStamp)
			fmt.Println(uint64(1<<64-1) - gotId)
		})
	}
}

func TestNextRvsIdByNodeBitsFor(t *testing.T) {
	node := uint64(0)
	seq := uint64(0)
	ts := uint64(0)
	nodeBits := 4
	var (
		got uint64
		err error
	)
	for {
		got, seq, ts, err = NextRvsIdByNodeBits(ts, node, seq, uint(nodeBits))
		if err != nil {
			panic(err)
		}
		fmt.Println(got)
		time.Sleep(time.Millisecond * 100)
	}

}

func TestGetNodeIdById(t *testing.T) {
	type args struct {
		id       uint64
		nodeBits uint
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "bits4-node0", args: args{
			id:       1,
			nodeBits: 4,
		}, want: 0},
		{name: "bits4-node2", args: args{
			id:       2305843009213696600,
			nodeBits: 4,
		}, want: 2},
		{name: "bits4-node14", args: args{
			id:       16140901064495857675,
			nodeBits: 4,
		}, want: 14},
		{name: "bits4-node15-maxUint64", args: args{
			id:       math.MaxUint64,
			nodeBits: 4,
		}, want: 15},
		{name: "bits4-node8", args: args{
			id:       9223372036854775810,
			nodeBits: 4,
		}, want: 8},
		{name: "bits4-node7", args: args{
			id:       8070450532247928858,
			nodeBits: 4,
		}, want: 7},
		{name: "bits3-node0", args: args{
			id:       1233,
			nodeBits: 3,
		}, want: 0},
		{name: "bits3-node3", args: args{
			id:       8070450532247928858,
			nodeBits: 3,
		}, want: 3},
		{name: "bits1-node0", args: args{
			id:       1,
			nodeBits: 1,
		}, want: 0},
		{name: "bits1-node1-maxUin64", args: args{
			id:       math.MaxUint64,
			nodeBits: 1,
		}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNodeIdById(tt.args.id, tt.args.nodeBits); got != tt.want {
				t.Errorf("GetNodeIdById() = %v, want %v", got, tt.want)
			}
		})
	}
}
