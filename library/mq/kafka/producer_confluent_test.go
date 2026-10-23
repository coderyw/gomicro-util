package kafka

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_msgMarshal1(t *testing.T) {
	type args struct {
		obj interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "Int=1", args: args{obj: 1}, want: "1", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
		{name: "float32=232.343", args: args{obj: float32(232.343)}, want: "232.343", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
		{name: "float64=232.343323", args: args{obj: float64(232.343323)}, want: "232.343323", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
		{name: "string=\"ewr2342\"", args: args{obj: "ewr2342"}, want: "ewr2342", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
		{name: "array={1,2,3}", args: args{obj: []int{1, 2, 3}}, want: "[1,2,3]", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
		{name: "map={\"a\":34,\"b\":\"ee\"}", args: args{obj: map[string]interface{}{"a": 34, "b": "ee"}}, want: "{\"a\":34,\"b\":\"ee\"}", wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
			return err != nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := msgMarshal(tt.args.obj)
			if !tt.wantErr(t, err, fmt.Sprintf("msgMarshal1(%v)", tt.args.obj)) {
				return
			}
			assert.Equalf(t, tt.want, got, "msgMarshal1(%v)", tt.args.obj)
		})
	}
}
