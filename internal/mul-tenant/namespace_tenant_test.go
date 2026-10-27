package mul_tenant

import (
	"github.com/coderyw/gomicro-util/internal/testenv"
	"testing"

	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/model/config"
)

func TestStartListen(t *testing.T) {
	nc := testenv.Require(t, "GOMICRO_TEST_NACOS_ADDR", "GOMICRO_TEST_NACOS_PASSWORD")
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
				nc[0],
			},
			Account:     testenv.Get("GOMICRO_TEST_NACOS_ACCOUNT", "nacos"),
			Pwd:         nc[1],
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
	nc := testenv.Require(t, "GOMICRO_TEST_NACOS_ADDR", "GOMICRO_TEST_NACOS_PASSWORD")
	err := StartListen(config.RegistryCfg{
		Type: "nacos",
		Hosts: []string{
			nc[0],
		},
		Account:     testenv.Get("GOMICRO_TEST_NACOS_ACCOUNT", "nacos"),
		Pwd:         nc[1],
		NameId:      "",
		EnableIstio: false,
	})
	if err != nil {
		t.Fatalf("StartListen() error = %v", err)
	}
	<-single.OsKillSingle()

}
