package service

import (
	"net/netip"
	"sync"
	"time"
)

var publicIPv4Services = []string{"https://api4.ipify.org", "https://ipv4.icanhazip.com", "https://v4.api.ipinfo.io/ip"}
var publicIPv6Services = []string{"https://api6.ipify.org", "https://ipv6.icanhazip.com", "https://v6.api.ipinfo.io/ip"}

type publicIPCache struct {
	mu          sync.Mutex
	ipv4, ipv6  string
	nextRefresh time.Time
	refreshing  bool
}

// snapshot never waits on external services. A single background lookup refreshes
// stale values; temporary failures keep the last valid address and retry sooner.
func (c *publicIPCache) snapshot(now time.Time, lookup func(string) string) (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ipv4 == "" {
		c.ipv4 = "N/A"
	}
	if c.ipv6 == "" {
		c.ipv6 = "N/A"
	}
	if !c.refreshing && !now.Before(c.nextRefresh) {
		c.refreshing = true
		go c.refresh(now, lookup)
	}
	return c.ipv4, c.ipv6
}

func lookupPublicIP(services []string, ipv6 bool, lookup func(string) string) string {
	for _, endpoint := range services {
		ip, err := netip.ParseAddr(lookup(endpoint))
		if err == nil && ip.Zone() == "" && ip.Is6() == ipv6 && !ip.Is4In6() {
			return ip.String()
		}
	}
	return ""
}

func (c *publicIPCache) refresh(now time.Time, lookup func(string) string) {
	ipv4 := lookupPublicIP(publicIPv4Services, false, lookup)
	ipv6 := lookupPublicIP(publicIPv6Services, true, lookup)
	c.mu.Lock()
	defer c.mu.Unlock()
	if ipv4 != "" {
		c.ipv4 = ipv4
	}
	if ipv6 != "" {
		c.ipv6 = ipv6
	}
	retry := 5 * time.Minute
	if ipv4 == "" || ipv6 == "" {
		retry = time.Minute
	}
	c.nextRefresh = now.Add(retry)
	c.refreshing = false
}
