package xray

import (
	"bytes"
	"encoding/json"

	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
)

// Config represents the complete Xray configuration structure.
// It contains all sections of an Xray config file including inbounds, outbounds, routing, etc.
type Config struct {
	Env              json_util.RawMessage       `json:"env"`
	LogConfig        json_util.RawMessage       `json:"log"`
	RouterConfig     json_util.RawMessage       `json:"routing"`
	DNSConfig        json_util.RawMessage       `json:"dns"`
	InboundConfigs   []InboundConfig            `json:"inbounds"`
	OutboundConfigs  json_util.RawMessage       `json:"outbounds"`
	Transport        json_util.RawMessage       `json:"transport"`
	Policy           json_util.RawMessage       `json:"policy"`
	API              json_util.RawMessage       `json:"api"`
	Stats            json_util.RawMessage       `json:"stats"`
	Reverse          json_util.RawMessage       `json:"reverse"`
	FakeDNS          json_util.RawMessage       `json:"fakedns"`
	Observatory      json_util.RawMessage       `json:"observatory"`
	BurstObservatory json_util.RawMessage       `json:"burstObservatory"`
	Metrics          json_util.RawMessage       `json:"metrics"`
	Extra            map[string]json.RawMessage `json:"-"`
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type alias Config
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	known := map[string]bool{"env": true, "log": true, "routing": true, "dns": true, "inbounds": true, "outbounds": true, "transport": true, "policy": true, "api": true, "stats": true, "reverse": true, "fakedns": true, "observatory": true, "burstObservatory": true, "metrics": true}
	decoded.Extra = make(map[string]json.RawMessage)
	for key, value := range all {
		if !known[key] {
			decoded.Extra[key] = value
		}
	}
	*c = Config(decoded)
	return nil
}

func (c Config) MarshalJSON() ([]byte, error) {
	type alias Config
	data, err := json.Marshal(alias(c))
	if err != nil {
		return nil, err
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	for key, value := range c.Extra {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return json.Marshal(out)
}

// Equals compares two Config instances for deep equality.
func (c *Config) Equals(other *Config) bool {
	if c == nil || other == nil {
		return c == other
	}
	if len(c.InboundConfigs) != len(other.InboundConfigs) {
		return false
	}
	for i, inbound := range c.InboundConfigs {
		if !inbound.Equals(&other.InboundConfigs[i]) {
			return false
		}
	}
	if !bytes.Equal(c.LogConfig, other.LogConfig) {
		return false
	}
	if !bytes.Equal(c.RouterConfig, other.RouterConfig) {
		return false
	}
	if !bytes.Equal(c.DNSConfig, other.DNSConfig) {
		return false
	}
	if !bytes.Equal(c.OutboundConfigs, other.OutboundConfigs) {
		return false
	}
	if !bytes.Equal(c.Transport, other.Transport) {
		return false
	}
	if !bytes.Equal(c.Policy, other.Policy) {
		return false
	}
	if !bytes.Equal(c.API, other.API) {
		return false
	}
	if !bytes.Equal(c.Stats, other.Stats) {
		return false
	}
	if !bytes.Equal(c.Reverse, other.Reverse) {
		return false
	}
	if !bytes.Equal(c.FakeDNS, other.FakeDNS) {
		return false
	}
	if !bytes.Equal(c.Metrics, other.Metrics) {
		return false
	}
	if !bytes.Equal(c.Env, other.Env) {
		return false
	}
	if !bytes.Equal(c.Observatory, other.Observatory) {
		return false
	}
	if !bytes.Equal(c.BurstObservatory, other.BurstObservatory) {
		return false
	}
	if len(c.Extra) != len(other.Extra) {
		return false
	}
	for key, value := range c.Extra {
		if !bytes.Equal(value, other.Extra[key]) {
			return false
		}
	}
	return true
}
