package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRpcServerCfg_Unmarshal(t *testing.T) {

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{name: "Parse yaml", data: `
rpc_server:
  listen_on: ":9804"
  server:
    auths:
      - auth_type: 'basic'
        auth_endpoint:
          - 'ca-delay-center.DelayCenterService.*'
        auth_secret:
          - 'fwfwefwifwef'
        auth_version: '1.0.0'
        force_auth: true
      - auth_type: 'basic'
        auth_endpoint:
          - 'ca-delay-center.TestService.*'
        auth_secret:
          - 'fwfwefwifwef'
        auth_version: '1.0.0'
        force_auth: false
  client:
    auths:
      - auth_type: 'basic'
        auth_endpoint:
          - 'ca-delay-center.DelayCenterService.*'
        auth_secret: 'fwfwefwifwef'
        auth_version: '1.0.0'`, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ServerConfig{}
			err := yaml.Unmarshal([]byte(tt.data), &s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("%+v\n", s)
		})
	}
}
