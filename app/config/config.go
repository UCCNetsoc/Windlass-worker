package config

import (
	"encoding/json"
	"flag"
	"strings"

	log "github.com/UCCNetworkingSociety/Windlass-worker/utils/logging"

	"github.com/spf13/viper"
)

func Load() error {
	InitDefaults()
	initFlags()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	printSettings()
	return nil
}

func initFlags() {
	var port string
	flag.StringVar(&port, "port", "9786", "sets the port the worker listens on")
	flag.Parse()

	if port != "" {
		viper.Set("http.port", port)
	}
}

func printSettings() {
	// Print settings with secrets redacted
	settings := viper.AllSettings()
	out, _ := json.MarshalIndent(settings, "", "\t")
	log.Debug("config:\n%s", string(out))
}
