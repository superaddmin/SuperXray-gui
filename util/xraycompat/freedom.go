package xraycompat

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RuntimeOutbounds projects Freedom rules onto the wire format shared by
// v26.10.21 and v26.11.9 without changing the template stored by the panel.
func RuntimeOutbounds(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var outbounds []json.RawMessage
	if err := json.Unmarshal(raw, &outbounds); err != nil {
		return nil, err
	}
	changed := false
	for i, outbound := range outbounds {
		var config struct {
			Protocol string
			Settings json.RawMessage
		}
		if err := json.Unmarshal(outbound, &config); err != nil {
			return nil, err
		}
		switch strings.ToLower(config.Protocol) {
		case "freedom", "direct":
		default:
			continue
		}
		if len(config.Settings) == 0 {
			continue
		}
		var settings struct{ FinalRules []json.RawMessage }
		if err := json.Unmarshal(config.Settings, &settings); err != nil {
			return nil, err
		}
		rulesChanged := false
		for j, rawRule := range settings.FinalRules {
			projected, err := runtimeFreedomRule(rawRule)
			if err != nil {
				return nil, fmt.Errorf("outbounds[%d].settings.finalRules[%d]: %w", i, j, err)
			}
			if projected != nil {
				settings.FinalRules[j] = projected
				rulesChanged = true
			}
		}
		if rulesChanged {
			rules, err := replaceObjectFields(config.Settings, map[string]any{"finalRules": settings.FinalRules})
			if err != nil {
				return nil, err
			}
			outbounds[i], err = replaceObjectFields(outbound, map[string]any{"settings": rules})
			if err != nil {
				return nil, err
			}
			changed = true
		}
	}
	if !changed {
		return raw, nil
	}
	return json.Marshal(outbounds)
}

func runtimeFreedomRule(raw json.RawMessage) (json.RawMessage, error) {
	// Xray uses Go's case-insensitive struct field matching. Decode the same
	// way so historical spellings and duplicate fields follow input order.
	var rule struct {
		Action            string
		Delay, BlockDelay *json.RawMessage
		LogLevel          string
	}
	if err := json.Unmarshal(raw, &rule); err != nil {
		return nil, err
	}
	switch strings.ToLower(rule.Action) {
	case "allow":
		if rule.Delay != nil || rule.LogLevel != "" {
			return nil, fmt.Errorf("delay and logLevel require action block")
		}
		if rule.BlockDelay != nil {
			return replaceObjectFields(raw, map[string]any{"blockDelay": nil})
		}
	case "block":
		switch strings.ToLower(rule.LogLevel) {
		case "", "none", "debug", "info", "warning", "error":
		default:
			return nil, fmt.Errorf("unsupported block logLevel")
		}
		if rule.Delay != nil {
			// New cores prefer a non-null blockDelay over delay; older
			// cores can only read blockDelay.
			delay := rule.BlockDelay
			if delay == nil {
				delay = rule.Delay
			}
			return replaceObjectFields(raw, map[string]any{"delay": nil, "blockDelay": delay})
		}
	}
	return nil, nil
}

// Replace only the selected fields. Unknown values stay as raw JSON, including
// large numbers. Remove every spelling of replaced fields before marshaling.
func replaceObjectFields(raw json.RawMessage, updates map[string]any) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	for field, value := range updates {
		for key := range object {
			if strings.EqualFold(key, field) {
				delete(object, key)
			}
		}
		if value != nil {
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			object[field] = encoded
		}
	}
	return json.Marshal(object)
}
