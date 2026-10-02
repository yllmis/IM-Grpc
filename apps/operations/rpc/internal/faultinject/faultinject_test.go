package faultinject

import "testing"

func TestConfigDefaultsToDisabled(t *testing.T) {
	c := Config{Rules: map[string]string{"msg": string(QueryTimeout)}}
	if _, ok := c.ScenarioFor("msg"); ok {
		t.Fatal("disabled fault injection must not affect queries")
	}
}

func TestConfigRejectsProductionAndUnknownScenario(t *testing.T) {
	tests := []struct {
		name string
		mode string
		cfg  Config
	}{
		{name: "production", mode: "pro", cfg: Config{Enabled: true}},
		{name: "unknown", mode: "dev", cfg: Config{Enabled: true, Rules: map[string]string{"msg": "delete_data"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(tc.mode); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestConfigResolvesScenarioOnlyWhenEnabled(t *testing.T) {
	c := Config{Enabled: true, Rules: map[string]string{"msg_fault": string(DeliveryEmpty)}}
	scenario, ok := c.ScenarioFor("msg_fault")
	if !ok || scenario != DeliveryEmpty {
		t.Fatalf("unexpected scenario: %q %v", scenario, ok)
	}
}
