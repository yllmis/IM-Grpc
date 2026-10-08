// Package faultinject contains deterministic, read-only fault injection for
// non-production ObservationQuery environments. It never mutates IM data.
package faultinject

import (
	"fmt"
	"strings"
)

// Scenario is intentionally small: every value changes only the response of a
// matching read query, so normal requests keep the existing code path.
type Scenario string

const (
	QueryTimeout          Scenario = "query_timeout"
	DeliveryEmpty         Scenario = "delivery_empty"
	DeliveryTimeout       Scenario = "delivery_timeout"
	ReceiverOffline       Scenario = "receiver_offline"
	AckTimeout            Scenario = "ack_timeout"
	UnsupportedCapability Scenario = "unsupported_capability"
)

var allowedScenarios = map[Scenario]struct{}{
	QueryTimeout:          {},
	DeliveryEmpty:         {},
	DeliveryTimeout:       {},
	ReceiverOffline:       {},
	AckTimeout:            {},
	UnsupportedCapability: {},
}

// Config is loaded from the ObservationQuery YAML. Enabled defaults to false;
// an empty Rules map therefore has zero runtime effect.
type Config struct {
	Enabled bool              `json:",optional"`
	Rules   map[string]string `json:",optional"`
}

func (c Config) ScenarioFor(key string) (Scenario, bool) {
	if !c.Enabled {
		return "", false
	}
	raw, ok := c.Rules[strings.TrimSpace(key)]
	if !ok {
		return "", false
	}
	return Scenario(strings.TrimSpace(raw)), true
}

func (c Config) Validate(mode string) error {
	if !c.Enabled {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(mode), "pro") || strings.EqualFold(strings.TrimSpace(mode), "production") {
		return fmt.Errorf("fault injection must be disabled in production mode")
	}
	if len(c.Rules) > 50 {
		return fmt.Errorf("fault injection rules exceed maximum of 50")
	}
	for key, raw := range c.Rules {
		if strings.TrimSpace(key) == "" || len(key) > 128 {
			return fmt.Errorf("fault injection rule key must be 1..128 characters")
		}
		scenario := Scenario(strings.TrimSpace(raw))
		if _, ok := allowedScenarios[scenario]; !ok {
			return fmt.Errorf("unsupported fault injection scenario %q", raw)
		}
	}
	return nil
}
