package config

import (
	"github.com/zeromicro/go-zero/core/conf"
	"testing"
)

func TestReadQueryAuthIsOptionalAndFailsClosed(t *testing.T) {
	base := `Name: im.rpc
ListenOn: 127.0.0.1:10002
Mongo:
  Url: mongodb://localhost:27017
  Db: test
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
