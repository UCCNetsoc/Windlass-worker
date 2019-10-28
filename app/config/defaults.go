package config

import (
	"net"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func InitDefaults() {
	viper.SetDefault("http.port", "9786")
	viper.SetDefault("http.hostname", getFQDN())
	viper.SetDefault("http.address", getOutboundIP().String())

	viper.SetDefault("containerHost.type", "lxd")

	viper.SetDefault("lxd.baseImage", "c4fad4b4dce1") // sample image

	// Consul settings
	viper.SetDefault("consul.url", "127.0.0.1:8500")
	viper.SetDefault("consul.token", "") // ACL token
	viper.SetDefault("consul.path", "windlass")

	// Vault settings
	viper.SetDefault("vault.enabled", false) // If enabled, gets dynamic secret to access Consul from Vault maybe
	viper.SetDefault("vault.url", "127.0.0.1:8200")
	viper.SetDefault("vault.token", "")

	viper.SetDefault("windlass.secret", "")
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

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
