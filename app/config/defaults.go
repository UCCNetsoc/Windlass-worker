package config

import (
	"net"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func InitDefaults() {
	viper.SetDefault("http.port", "9786")
	viper.SetDefault("http.address", getFQDN())

	// Consul settings
	viper.SetDefault("consul.host", "127.0.0.1:8500")
	viper.SetDefault("consul.token", "") // ACL token
	viper.SetDefault("consul.path", "windlass")

	// Vault settings
	viper.SetDefault("vault.enabled", false) // If enabled, gets dynamic secret to access Consul from Vault
	viper.SetDefault("vault.token", "")
}

// TODO: Add random number to 'unknown' state
func getFQDN() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}

	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return hostname
			}
			fqdn := hosts[0]
			return strings.TrimSuffix(fqdn, ".") // return fqdn without trailing dot
		}
	}
	return hostname
}
