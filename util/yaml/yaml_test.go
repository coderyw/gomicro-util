/**
 * @Author: yinwei
 * @Description:
 * @File: yaml_test.go
 * @Version: 1.0.0
 * @Date: 2023/10/25 19:13
 */
package yaml

import (
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
	"testing"
)

func TestYamlLoad(t *testing.T) {
	type args struct {
		path string
		obj  interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "parse", args: args{
			path: "./cfg.yaml",
			obj:  &config.ServerConfig{},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := YamlLoad(tt.args.path, tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("YamlLoad() error = %v, wantErr %v", err, tt.wantErr)
			}
			log.Info("parse", zap.Any("ogj", tt.args.obj))
		})
	}
}
