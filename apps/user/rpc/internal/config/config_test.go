package config

import (
	"github.com/zeromicro/go-zero/core/conf"
	"testing"
)

func TestReadQueryAuthIsOptionalAndFailsClosed(t *testing.T) {
	base := `Name: user.rpc
ListenOn: 127.0.0.1:10000
Mysql:
  DataSource: synthetic-test-dsn
Cache:
  - Host: localhost:6379
Redisx:
  Host: localhost:6379
JwtAuth:
  AccessSecret: synthetic-test-secret
  AccessExpire: 60
`
	var c Config
	if err := conf.LoadFromYamlBytes([]byte(base), &c); err != nil {
		t.Fatal(err)
	}
	if c.ReadQueryAuth.Enable {
		t.Fatal("legacy config unexpectedly enables query access")
	}
	if err := conf.LoadFromYamlBytes([]byte(base+`ReadQueryAuth:
  Enable: true
  Token: synthetic-domain-token
`), &c); err != nil {
		t.Fatal(err)
	}
	if !c.ReadQueryAuth.Enable || c.ReadQueryAuth.Validate() != nil {
		t.Fatal("cannot load domain auth")
	}
}
