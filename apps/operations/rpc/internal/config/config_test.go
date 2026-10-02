package config

import "testing"

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
			t.Fatal("unsafe OperationsQuery configuration accepted")
		}
	}
}
