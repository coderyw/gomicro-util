package jwt

import "testing"

func TestParseGrpcJwtToken(t *testing.T) {
	type args struct {
		tokenString  string
		secret       []byte
		claims       *AuthGrpcClaims
		skipValidate bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "jwt", args: args{
			tokenString:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJkYXRhIjpbeyJ1c2VyIjoiaHR0cC1jbGllbnQifV0sImlhdCI6MTc2ODgxNTA5MiwiZXhwIjoxNzY4NjYxOTk5LCJhdWQiOiIiLCJpc3MiOiJmcmFuY2lzIiwic3ViIjoiIn0.u52MEsBY3rE9Gvj_Wrg1cp6GCZOzXfPgpXVBJ0A3qYM",
			secret:       []byte("test-only-ee9aba1bd903913fa37a237e7b766e6f"),
			claims:       &AuthGrpcClaims{},
			skipValidate: false,
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ParseGrpcJwtToken(tt.args.tokenString, tt.args.secret, tt.args.claims, tt.args.skipValidate); (err != nil) != tt.wantErr {
				t.Errorf("ParseGrpcJwtToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
