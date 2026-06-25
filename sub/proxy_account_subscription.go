package sub

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/database/model"
)

type proxyAccount struct {
	User    string
	Pass    string
	SubID   string
	Email   string
	Remark  string
	Enabled bool
}

func isProxyAccountProtocol(protocol model.Protocol) bool {
	return protocol == model.HTTP || protocol == model.Mixed
}

func proxyAccountsBySubID(inbound *model.Inbound, subId string) ([]proxyAccount, error) {
	accounts, topLevelSubID, err := proxyAccountsFromInbound(inbound)
	if err != nil {
		return nil, err
	}

	subId = strings.TrimSpace(subId)
	if len(accounts) == 0 {
		if topLevelSubID == subId {
			return []proxyAccount{{SubID: topLevelSubID, Enabled: true}}, nil
		}
		return nil, nil
	}

	matched := make([]proxyAccount, 0, len(accounts))
	for _, account := range accounts {
		if !account.Enabled {
			continue
		}
		effectiveSubID := proxyAccountSubID(account, topLevelSubID)
		if effectiveSubID != subId {
			continue
		}
		account.SubID = effectiveSubID
		matched = append(matched, account)
	}
	return matched, nil
}

func proxyAccountsFromInbound(inbound *model.Inbound) ([]proxyAccount, string, error) {
	var settings map[string]any
	if err := json.Unmarshal([]byte(inbound.Settings), &settings); err != nil {
		return nil, "", err
	}
	topLevelSubID := strings.TrimSpace(stringFromAny(settings["subId"]))
	rawAccounts, _ := settings["accounts"].([]any)
	accounts := make([]proxyAccount, 0, len(rawAccounts))
	for _, rawAccount := range rawAccounts {
		accountMap, ok := rawAccount.(map[string]any)
		if !ok {
			continue
		}
		accounts = append(accounts, proxyAccount{
			User:    firstString(accountMap, "user", "username"),
			Pass:    firstString(accountMap, "pass", "password"),
			SubID:   firstString(accountMap, "subId", "subID", "sub_id"),
			Email:   firstString(accountMap, "email"),
			Remark:  firstString(accountMap, "remark", "comment", "name"),
			Enabled: proxyAccountEnabled(accountMap),
		})
	}
	return accounts, topLevelSubID, nil
}

func proxyAccountSubID(account proxyAccount, topLevelSubID string) string {
	if strings.TrimSpace(account.SubID) != "" {
		return strings.TrimSpace(account.SubID)
	}
	return strings.TrimSpace(topLevelSubID)
}

func proxyAccountEnabled(account map[string]any) bool {
	for _, key := range []string{"enable", "enabled"} {
		if value, ok := account[key].(bool); ok {
			return value
		}
	}
	return true
}

func proxyAccountUsesAuth(account proxyAccount) bool {
	return account.User != "" || account.Pass != ""
}

func proxyAccountUserinfo(account proxyAccount) string {
	if !proxyAccountUsesAuth(account) {
		return ""
	}
	if account.Pass == "" {
		return encodeUserinfo(account.User) + "@"
	}
	return fmt.Sprintf("%s:%s@", encodeUserinfo(account.User), encodeUserinfo(account.Pass))
}

func (s *SubService) genHTTPProxyLink(inbound *model.Inbound, account proxyAccount) string {
	if inbound.Protocol != model.HTTP {
		return ""
	}
	link := fmt.Sprintf("http://%s%s:%d", proxyAccountUserinfo(account), s.resolveInboundAddress(inbound), inbound.Port)
	return buildLinkWithParams(link, nil, s.genProxyAccountRemark(inbound, account, ""))
}

func (s *SubService) genSocks5ProxyLink(inbound *model.Inbound, account proxyAccount) string {
	if inbound.Protocol != model.Mixed {
		return ""
	}
	link := fmt.Sprintf("socks5://%s%s:%d", proxyAccountUserinfo(account), s.resolveInboundAddress(inbound), inbound.Port)
	return buildLinkWithParams(link, nil, s.genProxyAccountRemark(inbound, account, ""))
}

func (s *SubService) genProxyAccountRemark(inbound *model.Inbound, account proxyAccount, extra string) string {
	label := strings.TrimSpace(account.Remark)
	if label == "" {
		label = strings.TrimSpace(account.Email)
	}
	return s.genRemark(inbound, label, extra)
}

func firstString(source map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(stringFromAny(source[key])); value != "" {
			return value
		}
	}
	return ""
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}
