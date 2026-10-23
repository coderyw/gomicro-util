package mul_tenant

import (
	"testing"

	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/model/config"
)

func TestStartListen(t *testing.T) {
	type args struct {
		cfg config.RegistryCfg
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "ok", args: args{cfg: config.RegistryCfg{
			Type: "nacos",
			Hosts: []string{
				"dev-nacos-internal-NLB-664ef66d616ee60a.elb.ap-southeast-1.amazonaws.com:8848",
			},
			Account:     "nacos",
			Pwd:         "hU6csGhudotd0HBCqzrW",
			NameId:      "",
			EnableIstio: false,
		}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StartListen(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("StartListen() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListenChange(t *testing.T) {
	err := StartListen(config.RegistryCfg{
		Type: "nacos",
		Hosts: []string{
			"dev-nacos-internal-NLB-664ef66d616ee60a.elb.ap-southeast-1.amazonaws.com:8848",
		},
		Account:     "nacos",
		Pwd:         "hU6csGhudotd0HBCqzrW",
		NameId:      "",
		EnableIstio: false,
	})
	if err != nil {
		t.Fatalf("StartListen() error = %v", err)
	}
	<-single.OsKillSingle()

}
