// Package xraycompat contains the small, deterministic compatibility layer
// used while reading old panel JSON.  It deliberately operates on a copy of
// the document: a failed conversion always returns the original bytes.
package xraycompat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// NormalizeConfig upgrades a stored Xray template.  It is idempotent and
// preserves fields this panel does not know about.  The returned JSON is the
// original byte string when no change is needed.
func NormalizeConfig(raw string) (string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return raw, false, nil
	}
	var doc any
	if err := decode(raw, &doc); err != nil {
		return raw, false, err
	}
	cfg, ok := doc.(map[string]any)
	if !ok {
		return raw, false, errors.New("xray config must be a JSON object")
	}
	changed := false
	if values, ok := cfg["inbounds"].([]any); ok {
		for _, item := range values {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			settings, _ := obj["settings"].(string)
			stream, _ := obj["streamSettings"].(string)
			// Templates normally carry objects. Keep support for the model's
			// legacy JSON-text fields when a caller gives us either form.
			if rawSettings, ok := obj["settings"].(map[string]any); ok {
				settings, _ = encode(rawSettings)
			}
			if rawStream, ok := obj["streamSettings"].(map[string]any); ok {
				stream, _ = encode(rawStream)
			}
			ns, nstream, c, err := NormalizeInbound(stringValue(obj["protocol"]), settings, stream)
			if err != nil {
				return raw, false, err
			}
			if c {
				for _, field := range []struct{ key, before, after string }{{"settings", settings, ns}, {"streamSettings", stream, nstream}} {
					if field.before == field.after {
						continue
					}
					if _, text := obj[field.key].(string); text {
						obj[field.key] = field.after
					} else {
						var value any
						if err := decode(field.after, &value); err != nil {
							return raw, false, err
						}
						obj[field.key] = value
					}
				}
				changed = true
			}
		}
	}
	if values, ok := cfg["outbounds"].([]any); ok {
		for _, item := range values {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if c, err := normalizeOutbound(obj); err != nil {
				return raw, false, err
			} else {
				changed = changed || c
			}
		}
	}
	if !changed {
		return raw, false, nil
	}
	out, err := encode(doc)
	if err != nil {
		return raw, false, err
	}
	return out, true, nil
}

// NormalizeInbound converts only fields whose Xray meaning is stable across
// versions.  A removed cipher or unsafe combination returns an error instead
// of silently changing the user's access credentials.
func NormalizeInbound(protocol, settings, stream string) (string, string, bool, error) {
	changed := false
	ns := settings
	if strings.TrimSpace(settings) != "" {
		var obj map[string]any
		if err := decode(settings, &obj); err != nil {
			return settings, stream, false, err
		}
		if strings.EqualFold(protocol, "vmess") {
			if clients, ok := obj["clients"].([]any); ok {
				for _, raw := range clients {
					if c, ok := raw.(map[string]any); ok {
						if v, ok := c["security"].(string); ok && (strings.EqualFold(v, "none") || strings.EqualFold(v, "zero")) {
							c["security"] = "auto"
							changed = true
						}
					}
				}
			}
		}
		if strings.EqualFold(protocol, "shadowsocks") {
			if method := stringValue(obj["method"]); method == "none" || method == "plain" {
				return settings, stream, false, fmt.Errorf("shadowsocks cipher %q was removed by Xray; choose an AEAD cipher before upgrading", method)
			}
		}
		if strings.EqualFold(protocol, "wireguard") {
			if _, exists := obj["workers"]; exists {
				delete(obj, "workers")
				changed = true
			}
		}
		if strings.EqualFold(protocol, "hysteria") || strings.EqualFold(protocol, "hysteria2") {
			if v, exists := obj["version"]; exists {
				if numberValue(v) != 2 {
					return settings, stream, false, fmt.Errorf("hysteria settings version %v is not supported; Xray requires version 2", v)
				}
			} else {
				obj["version"] = 2
				changed = true
			}
		}
		if changed {
			ns, _ = encode(obj)
		}
	}
	nstream, c, err := NormalizeStream(stream)
	if err != nil {
		return settings, stream, false, err
	}
	return ns, nstream, changed || c, nil
}

// NormalizeStream upgrades transport/finalmask JSON. It accepts an empty
// stream (as many old inbounds do) and rejects malformed objects atomically.
func NormalizeStream(raw string) (string, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return raw, false, nil
	}
	var doc map[string]any
	if err := decode(raw, &doc); err != nil {
		return raw, false, err
	}
	changed := false
	if tls := stringMap(doc["tlsSettings"]); tls != nil {
		if insecure, _ := tls["allowInsecure"].(bool); insecure {
			return raw, false, errors.New("tlsSettings.allowInsecure was removed by Xray; configure pinnedPeerCertSha256 or verifyPeerCertByName before upgrading")
		}
		if pins, ok := tls["pinnedPeerCertSha256"].([]any); ok {
			values := make([]string, 0, len(pins))
			for _, pin := range pins {
				value, ok := pin.(string)
				if !ok || strings.TrimSpace(value) == "" || strings.Contains(value, ",") {
					return raw, false, errors.New("tlsSettings.pinnedPeerCertSha256 must contain non-empty certificate hashes")
				}
				values = append(values, value)
			}
			tls["pinnedPeerCertSha256"] = strings.Join(values, ",")
			changed = true
		}
	}
	if _, network := doc["network"]; !network {
		if method, ok := doc["method"]; ok {
			doc["network"] = method
			delete(doc, "method")
			changed = true
		}
	}
	if xhttp, ok := doc["xhttpSettings"].(map[string]any); ok {
		if err := validateSessionLength(xhttp); err != nil {
			return raw, false, err
		}
		changed = liftXHTTP(xhttp) || changed
	}
	if reality := stringMap(doc["realitySettings"]); reality != nil {
		if dest, ok := reality["dest"]; ok {
			if _, exists := reality["target"]; !exists {
				reality["target"] = dest
			}
			delete(reality, "dest")
			changed = true
		}
	}
	if sock := stringMap(doc["sockopt"]); sock != nil {
		if v, ok := sock["V6Only"]; ok {
			if _, exists := sock["v6only"]; !exists {
				sock["v6only"] = v
			}
			delete(sock, "V6Only")
			changed = true
		}
	}
	if sec := strings.ToLower(stringValue(doc["security"])); sec == "reality" {
		if fm, ok := doc["finalmask"].(map[string]any); ok {
			if masks, _ := fm["tcp"].([]any); len(masks) > 0 {
				return raw, false, errors.New("xray REALITY cannot be combined with a TCP finalmask")
			}
		}
	}
	if hs, ok := doc["hysteriaSettings"].(map[string]any); ok {
		if v, exists := hs["version"]; exists {
			if numberValue(v) != 2 {
				return raw, false, fmt.Errorf("hysteria stream version %v is not supported; Xray requires version 2", v)
			}
		} else {
			hs["version"] = 2
			changed = true
		}
	}
	if fm, ok := doc["finalmask"].(map[string]any); ok {
		c, err := normalizeFinalmask(fm)
		if err != nil {
			return raw, false, err
		}
		changed = changed || c
	}
	if !changed {
		return raw, false, nil
	}
	out, err := encode(doc)
	if err != nil {
		return raw, false, err
	}
	return out, true, nil
}

// NormalizeFinalMask is exported for hosts, links and subscription caches
// that store the finalmask object without its surrounding stream settings.
func NormalizeFinalMask(raw string) (string, bool, error) {
	var fm map[string]any
	if err := decode(raw, &fm); err != nil {
		return raw, false, err
	}
	c, err := normalizeFinalmask(fm)
	if err != nil {
		return raw, false, err
	}
	if !c {
		return raw, false, nil
	}
	out, err := encode(fm)
	if err != nil {
		return raw, false, err
	}
	return out, true, nil
}

func normalizeOutbound(obj map[string]any) (bool, error) {
	changed := false
	if ps, ok := obj["proxySettings"].(map[string]any); ok {
		if tag := stringValue(ps["tag"]); tag != "" {
			stream, _ := obj["streamSettings"].(map[string]any)
			if stream == nil {
				stream = map[string]any{}
				obj["streamSettings"] = stream
			}
			sock, _ := stream["sockopt"].(map[string]any)
			if sock == nil {
				sock = map[string]any{}
				stream["sockopt"] = sock
			}
			if stringValue(sock["dialerProxy"]) == "" {
				sock["dialerProxy"] = tag
			}
		}
		delete(obj, "proxySettings")
		changed = true
	}
	stream, _ := obj["streamSettings"].(map[string]any)
	if stream == nil && (strings.EqualFold(stringValue(obj["protocol"]), "freedom") || strings.EqualFold(stringValue(obj["protocol"]), "wireguard")) {
		stream = map[string]any{}
		obj["streamSettings"] = stream
	}
	if stream != nil {
		raw, _ := encode(stream)
		normalized, c, err := NormalizeStream(raw)
		if err != nil {
			return false, err
		}
		if c {
			_ = decode(normalized, &stream)
			obj["streamSettings"] = stream
			changed = true
		}
	}
	proto := strings.ToLower(stringValue(obj["protocol"]))
	settings, _ := obj["settings"].(map[string]any)
	if proto == "freedom" && settings != nil {
		if sock, _ := stream["sockopt"].(map[string]any); sock != nil {
			if _, ok := sock["addressPortStrategy"]; ok {
				delete(sock, "addressPortStrategy")
				changed = true
			}
		}
		// Match Xray's own resolution order: a non-AsIs root target wins
		// over settings.targetStrategy/domainStrategy and socket defaults.
		strategy := stringValue(obj["targetStrategy"])
		if strategy == "" || strings.EqualFold(strategy, "AsIs") {
			strategy = stringValue(settings["targetStrategy"])
			if strategy == "" {
				strategy = stringValue(settings["domainStrategy"])
			}
		}
		if strategy != "" && !strings.EqualFold(strategy, "AsIs") {
			sock := stringMap(stream["sockopt"])
			if sock == nil {
				sock = map[string]any{}
				stream["sockopt"] = sock
			}
			sock["domainStrategy"] = strategy
		}
		for _, key := range []string{"targetStrategy", "domainStrategy"} {
			if _, exists := settings[key]; exists {
				delete(settings, key)
				changed = true
			}
		}
		if _, exists := obj["targetStrategy"]; exists {
			delete(obj, "targetStrategy")
			changed = true
		}
	}
	if proto == "wireguard" && settings != nil {
		if _, exists := settings["workers"]; exists {
			delete(settings, "workers")
			changed = true
		}
		legacy := stringValue(settings["domainStrategy"])
		if legacy != "" {
			delete(settings, "domainStrategy")
			sock, _ := stream["sockopt"].(map[string]any)
			if sock == nil {
				sock = map[string]any{}
				if stream == nil {
					stream = map[string]any{}
					obj["streamSettings"] = stream
				}
				stream["sockopt"] = sock
			}
			if stringValue(sock["domainStrategy"]) == "" || strings.EqualFold(stringValue(sock["domainStrategy"]), "AsIs") {
				sock["domainStrategy"] = legacy
			}
			if stringValue(obj["targetStrategy"]) == "" || strings.EqualFold(stringValue(obj["targetStrategy"]), "AsIs") {
				obj["targetStrategy"] = legacy
			}
			changed = true
		}
		if dns, ok := settings["remoteDNS"].([]any); ok && len(dns) == 1 && stringValue(dns[0]) == "local" {
			delete(settings, "remoteDNS")
			if stringValue(obj["targetStrategy"]) == "" {
				obj["targetStrategy"] = "ForceIP"
			}
			changed = true
		}
		if dns, ok := settings["remoteDNS"].([]any); ok {
			for _, server := range dns {
				if _, err := netip.ParseAddr(stringValue(server)); err != nil {
					return false, errors.New("wireguard remoteDNS must contain IP addresses; migrate local to targetStrategy")
				}
			}
		}
	}
	if proto == "dns" && settings != nil {
		changed = normalizeDNS(settings) || changed
	}
	return changed, nil
}

// InboundRuntimeStream projects panel transport defaults to a server listener.
// Client-only defaults stay in the stored row and subscription output. They
// must not make a server socket dial remote UDP ports or spawn a download leg.
func InboundRuntimeStream(raw string) (string, error) {
	normalized, _, err := NormalizeStream(raw)
	if err != nil || strings.TrimSpace(normalized) == "" {
		return raw, err
	}
	var stream map[string]any
	if err := decode(normalized, &stream); err != nil {
		return raw, err
	}
	changed := false
	if xhttp := stringMap(stream["xhttpSettings"]); xhttp != nil {
		for _, obj := range []map[string]any{xhttp, stringMap(xhttp["extra"])} {
			for _, key := range []string{"xmux", "downloadSettings", "scMinPostsIntervalMs", "uplinkChunkSize", "noGRPCHeader"} {
				if _, exists := obj[key]; exists {
					delete(obj, key)
					changed = true
				}
			}
		}
	}
	if fm := stringMap(stream["finalmask"]); fm != nil {
		if masks, ok := fm["udp"].([]any); ok {
			kept := make([]any, 0, len(masks))
			for _, item := range masks {
				m := stringMap(item)
				if strings.EqualFold(stringValue(m["type"]), "udphop") && stringValue(stringMap(m["settings"])["mode"]) == "intervalRemote" {
					changed = true
					continue
				}
				kept = append(kept, item)
			}
			if changed {
				fm["udp"] = kept
			}
		}
	}
	if !changed {
		return normalized, nil
	}
	return encode(stream)
}

func normalizeDNS(s map[string]any) bool {
	mode, hasMode := s["nonIPQuery"]
	types, hasTypes := s["blockTypes"]
	changed := false
	if rules, exists := s["rules"]; (hasMode || hasTypes) && (!exists || rules == nil) {
		rules := []any{}
		if hasTypes {
			rule := map[string]any{"action": "drop", "qType": formatQTypes(types)}
			if stringValue(mode) != "drop" && stringValue(mode) != "skip" {
				rule["action"] = "return"
				rule["rCode"] = 5
			}
			if rule["qType"] != "" {
				rules = append(rules, rule)
			}
		}
		rules = append(rules, map[string]any{"action": "hijack", "qType": "1,28"})
		action := "direct"
		if stringValue(mode) == "drop" {
			action = "drop"
		}
		if stringValue(mode) == "reject" || stringValue(mode) == "" {
			action = "return"
		}
		fallback := map[string]any{"action": action}
		if action == "return" {
			fallback["rCode"] = 5
		}
		rules = append(rules, fallback)
		s["rules"] = rules
	}
	if hasMode || hasTypes {
		delete(s, "nonIPQuery")
		delete(s, "blockTypes")
		changed = true
	}
	if rules, ok := s["rules"].([]any); ok {
		for _, item := range rules {
			if rule, ok := item.(map[string]any); ok {
				if n, ok := rule["qType"].(json.Number); ok && n.String() == "0" {
					rule["qType"] = "0"
					changed = true
				}
			}
		}
	}
	return changed
}

func formatQTypes(v any) string {
	if values, ok := v.([]any); ok {
		out := make([]string, 0, len(values))
		for _, item := range values {
			out = append(out, fmt.Sprint(item))
		}
		return strings.Join(out, ",")
	}
	return fmt.Sprint(v)
}

func normalizeFinalmask(fm map[string]any) (bool, error) {
	changed := false
	if qp, ok := fm["quicParams"].(map[string]any); ok {
		if hop, exists := qp["udpHop"]; exists {
			h := stringMap(hop)
			if strings.TrimSpace(stringValue(h["ports"])) == "" {
				return false, errors.New("finalmask quicParams.udpHop requires ports")
			}
			if mode, exists := h["mode"]; exists && mode != "intervalRemote" {
				return false, errors.New("legacy udpHop mode conflicts with intervalRemote")
			}
			if ports, exists := h["remotePorts"]; exists && !reflect.DeepEqual(ports, h["ports"]) {
				return false, errors.New("legacy udpHop ports conflict with remotePorts")
			}
			if value, exists := fm["udp"]; exists {
				if _, valid := value.([]any); !valid {
					return false, errors.New("finalmask udp must be an array to migrate udpHop")
				}
			}
			masks, _ := fm["udp"].([]any)
			set := make(map[string]any, len(h)+1)
			for key, value := range h {
				set[key] = value
			}
			delete(set, "ports")
			set["mode"], set["remotePorts"] = "intervalRemote", h["ports"]
			fm["udp"] = append(masks, map[string]any{"type": "udphop", "settings": set})
			delete(qp, "udpHop")
			if len(qp) == 0 {
				delete(fm, "quicParams")
			}
			changed = true
		}
	}
	if masks, ok := fm["udp"].([]any); ok {
		var hop any
		kept := make([]any, 0, len(masks))
		for _, m := range masks {
			mm := stringMap(m)
			typ := strings.ToLower(stringValue(mm["type"]))
			if typ == "salamander" && len(stringValue(stringMap(mm["settings"])["password"])) < 4 {
				return false, errors.New("salamander finalmask password must be at least 4 bytes")
			}
			if typ == "xdns" {
				c, err := normalizeXdns(stringMap(mm["settings"]))
				if err != nil {
					return false, err
				}
				if c {
					changed = true
				}
			}
			if typ == "udphop" {
				if hop != nil && !reflect.DeepEqual(hop, m) {
					return false, errors.New("multiple UDP hop masks have different settings; choose one configuration before upgrading")
				}
				hop = m
			} else {
				kept = append(kept, m)
			}
		}
		if hop != nil {
			kept = append(kept, hop)
			if !reflect.DeepEqual(kept, masks) {
				changed = true
			}
			fm["udp"] = kept
		}
	}
	if masks, ok := fm["tcp"].([]any); ok {
		for _, m := range masks {
			mm := stringMap(m)
			if strings.EqualFold(stringValue(mm["type"]), "xmc") {
				st := stringMap(mm["settings"])
				profiles, _ := st["profiles"].([]any)
				if len(profiles) == 0 {
					return false, errors.New("XMC finalmask needs complete Minecraft profiles (username, UUID, texturesValue and texturesSignature)")
				}
				for _, entry := range profiles {
					profile := stringMap(entry)
					_, err := uuid.Parse(stringValue(profile["uuid"]))
					if err != nil || !xmcUsername.MatchString(stringValue(profile["username"])) || stringValue(profile["texturesValue"]) == "" || stringValue(profile["texturesSignature"]) == "" {
						return false, errors.New("XMC finalmask profile is incomplete or invalid")
					}
				}
			}
		}
	}
	return changed, nil
}

var xmcUsername = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

func normalizeXdns(s map[string]any) (bool, error) {
	if s == nil {
		return false, errors.New("xdns settings must be an object")
	}
	changed := false
	if ds, ok := s["domains"].([]any); ok {
		out := make([]any, 0, len(ds))
		for _, v := range ds {
			switch x := v.(type) {
			case string:
				n, typ := strings.TrimSpace(x), 16
				if i := strings.LastIndex(n, ":"); i > 0 {
					if t, ok := map[string]int{"a": 1, "aaaa": 28, "txt": 16}[strings.ToLower(n[i+1:])]; ok {
						n, typ = n[:i], t
					}
				}
				if strings.Contains(n, ":") {
					return false, fmt.Errorf("xdns domain %q has an unknown record type", x)
				}
				n = strings.Trim(n, ".")
				if n == "" {
					return false, errors.New("xdns domain name must not be empty")
				}
				out = append(out, map[string]any{"names": []any{n}, "types": []any{typ}, "edns0": 1232})
				changed = true
			case map[string]any:
				if name, ok := x["name"]; ok {
					legacyName, valid := name.(string)
					if !valid || strings.TrimSpace(legacyName) == "" {
						return false, errors.New("legacy xdns domain name must be a non-empty string")
					}
					names, exists := x["names"]
					if !exists {
						x["names"] = []any{legacyName}
					} else if values, ok := names.([]any); ok {
						found := false
						for _, value := range values {
							found = found || value == legacyName
						}
						if !found {
							x["names"] = append(values, legacyName)
						}
					} else {
						return false, errors.New("xdns domain names must be an array to migrate name")
					}
					delete(x, "name")
					changed = true
				}
				out = append(out, x)
			default:
				out = append(out, v)
			}
		}
		s["domains"] = out
	}
	if rs, ok := s["resolvers"].([]any); ok {
		out := make([]any, 0, len(rs))
		for _, v := range rs {
			switch x := v.(type) {
			case string:
				if domains, exists := s["domains"]; exists {
					if _, valid := domains.([]any); !valid {
						return false, errors.New("xdns domains must be an array to migrate legacy resolvers")
					}
				}
				parts := strings.SplitN(x, "+udp://", 2)
				if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
					return false, errors.New("legacy xdns resolver must use name[:type]+udp://address")
				}
				legacyDomain := map[string]any{"domains": []any{parts[0]}}
				if _, err := normalizeXdns(legacyDomain); err != nil {
					return false, err
				}
				domain := legacyDomain["domains"].([]any)[0]
				domains, _ := s["domains"].([]any)
				present := false
				domainJSON, _ := encode(domain)
				for _, existing := range domains {
					existingJSON, _ := encode(existing)
					if existingJSON == domainJSON {
						present = true
					}
				}
				if !present {
					s["domains"] = append(domains, domain)
				}
				out = append(out, map[string]any{"addrs": []any{"udp://" + parts[1]}})
				changed = true
			case map[string]any:
				if _, current := x["addrs"]; current {
					out = append(out, x)
					continue
				}
				if _, legacy := x["type"]; legacy {
					st := stringMap(x["settings"])
					addr := stringValue(st["addr"])
					kind := strings.ToLower(stringValue(x["type"]))
					if addr != "" && (kind == "udp" || kind == "tcp") {
						x["addrs"] = []any{kind + "://" + addr}
						delete(x, "type")
						delete(st, "addr")
						if len(st) == 0 {
							delete(x, "settings")
						}
						out = append(out, x)
						changed = true
					} else {
						return false, errors.New("legacy xdns resolver requires udp/tcp and settings.addr")
					}
				} else {
					out = append(out, x)
				}
			default:
				out = append(out, v)
			}
		}
		s["resolvers"] = out
	}
	return changed, nil
}
func liftXHTTP(x map[string]any) bool {
	if x == nil {
		return false
	}
	c := false
	for _, p := range [][2]string{{"sessionPlacement", "sessionIDPlacement"}, {"sessionKey", "sessionIDKey"}} {
		if v, ok := x[p[0]]; ok {
			if _, exists := x[p[1]]; !exists {
				x[p[1]] = v
			}
			delete(x, p[0])
			c = true
		}
	}
	if extra := stringMap(x["extra"]); extra != nil {
		c = liftXHTTP(extra) || c
	}
	return c
}

func validateSessionLength(x map[string]any) error {
	if extra := stringMap(x["extra"]); extra != nil {
		if err := validateSessionLength(extra); err != nil {
			return err
		}
	}
	value, exists := x["sessionIDLength"]
	if !exists || value == "" {
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case json.Number:
		raw = v.String()
	default:
		return errors.New("xhttp sessionIDLength must be a positive int32 length or range")
	}
	parts := strings.Split(raw, "-")
	if len(parts) > 2 {
		return errors.New("xhttp sessionIDLength must be a positive int32 length or range")
	}
	for _, part := range parts {
		if n, err := strconv.ParseInt(part, 10, 32); err != nil || n <= 0 {
			return errors.New("xhttp sessionIDLength must be a positive int32 length or range")
		}
	}
	return nil
}
func decode(raw string, v any) error {
	d := json.NewDecoder(bytes.NewReader([]byte(raw)))
	d.UseNumber()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	if obj, ok := v.(*map[string]any); ok && *obj == nil {
		return errors.New("JSON value must be an object")
	}
	return nil
}
func encode(v any) (string, error)   { b, e := json.Marshal(v); return string(b), e }
func stringMap(v any) map[string]any { m, _ := v.(map[string]any); return m }
func stringValue(v any) string       { s, _ := v.(string); return s }
func numberValue(v any) int {
	switch n := v.(type) {
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
