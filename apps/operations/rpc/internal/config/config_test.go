package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestFaultInjectionYAMLRemainsOptionalAndLoadsRules(t *testing.T) {
	const base = `Name: operations.rpc
ListenOn: 127.0.0.1:9101
Mode: dev
Mongo:
  Url: mongodb://mongo:27017
  Db: test
ServiceAuth:
  Enable: true
  Token: synthetic-test-token
  ExtraTokens: []
DeliveryObservation:
  Enabled: false
  AckMode: NoAck
Query:
  MaxLimit: 200
  DefaultLimit: 50
`
	for _, injected := range []bool{false, true} {
		yaml := base
		if injected {
			yaml += `FaultInjection:
  Enabled: true
  Rules:
    "665f1c0000000000000000ab": delivery_timeout
`
		}
		var c Config
		if err := conf.LoadFromYamlBytes([]byte(yaml), &c); err != nil {
			t.Fatal(err)
		}
		if err := c.Validate(); err != nil {
			t.Fatal(err)
		}
		if c.FaultInjection.Enabled != injected {
			t.Fatalf("enabled=%v want=%v", c.FaultInjection.Enabled, injected)
		}
		if injected && c.FaultInjection.Rules["665f1c0000000000000000ab"] != "delivery_timeout" {
			t.Fatal("fault rules not loaded from YAML")
		}
	}
}

func TestValidateRequiresAuthenticatedReadOnlyConfiguration(t *testing.T) {
	valid := Config{}
	valid.Mongo.Url = "mongodb://mongo:27017"
	valid.Mongo.Db = "yllmis-im"
	valid.ServiceAuth.Enable = true
	valid.ServiceAuth.Token = "test-only"
	valid.Query.DefaultLimit = 50
	valid.Query.MaxLimit = 200
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}

	for _, mutate := range []func(*Config){
		func(c *Config) { c.ServiceAuth.Enable = false },
		func(c *Config) { c.ServiceAuth.Token = "" },
		func(c *Config) { c.Mongo.Url = "" },
		func(c *Config) { c.Query.DefaultLimit = 201 },
	} {
		c := valid
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Fatal("unsafe ObservationQuery configuration accepted")
		}
	}
}

func TestObservationOnlyConfigurationDoesNotRequireDomainServices(t *testing.T) {
	var c Config
	err := conf.LoadFromYamlBytes([]byte(`Name: operations.rpc
ListenOn: 127.0.0.1:9100
Mongo:
  Url: mongodb://mongo:27017
  Db: test
ServiceAuth:
  Enable: true
  Token: synthetic-token
  ExtraTokens: []
Query:
  MaxLimit: 200
  DefaultLimit: 50
DeliveryObservation:
  Enabled: false
  AckMode: NoAck
`), &c)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
