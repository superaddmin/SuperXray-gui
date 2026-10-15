package xray

import (
	"bytes"
	"encoding/json"

	"github.com/superaddmin/SuperXray-gui/v2/util/json_util"
)

// InboundConfig represents an Xray inbound configuration.
// It defines how Xray accepts incoming connections including protocol, port, and settings.
type InboundConfig struct {
	Listen         json_util.RawMessage       `json:"listen"` // listen cannot be an empty string
	Port           int                        `json:"port"`
	Protocol       string                     `json:"protocol"`
	Settings       json_util.RawMessage       `json:"settings"`
	StreamSettings json_util.RawMessage       `json:"streamSettings"`
	Tag            string                     `json:"tag"`
	Sniffing       json_util.RawMessage       `json:"sniffing"`
	Extra          map[string]json.RawMessage `json:"-"`
}

func (c *InboundConfig) UnmarshalJSON(data []byte) error {
	type alias InboundConfig
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"listen", "port", "protocol", "settings", "streamSettings", "tag", "sniffing"} {
		delete(fields, key)
	}
	decoded.Extra = fields
	*c = InboundConfig(decoded)
	return nil
}

func (c InboundConfig) MarshalJSON() ([]byte, error) {
	type alias InboundConfig
	data, err := json.Marshal(alias(c))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	for key, value := range c.Extra {
		if _, known := fields[key]; !known {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

// Equals compares two InboundConfig instances for deep equality.
func (c *InboundConfig) Equals(other *InboundConfig) bool {
	if !bytes.Equal(c.Listen, other.Listen) {
		return false
	}
	if c.Port != other.Port {
		return false
	}
	if c.Protocol != other.Protocol {
		return false
	}
	if !bytes.Equal(c.Settings, other.Settings) {
		return false
	}
	if !bytes.Equal(c.StreamSettings, other.StreamSettings) {
		return false
	}
	if c.Tag != other.Tag {
		return false
	}
	if !bytes.Equal(c.Sniffing, other.Sniffing) {
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
