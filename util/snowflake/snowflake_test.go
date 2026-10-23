// Package snowflake
// @Author: yinwei
// @File: snowflake_test.go
// @Version: 1.0.0
// @Date: 2023/11/21 10:21

package snowflake

import "testing"

func TestNextIdByNodeBits(t *testing.T) {
	type args struct {
		timestamp uint64
		nodeId    uint64
		sequence  uint64
		nodeBits  uint
	}
	tests := []struct {
		name    string
		args    args
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotId, gotAfterSequence, gotAfterTimeStamp, err := NextIdByNodeBits(tt.args.timestamp, tt.args.nodeId, tt.args.sequence, tt.args.nodeBits)
			if (err != nil) != tt.wantErr {
				t.Errorf("NextIdByNodeBits() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("gotId=%v,gotAfterSequence=%v,gotAfterTimeStamp=%v", gotId, gotAfterSequence, gotAfterTimeStamp)
		})
	}
}
