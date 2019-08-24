package config

import (
	"encoding/json"
	"flag"
	"strings"

	"github.com/Strum355/log"

	"github.com/spf13/viper"
)

func Load() error {
	InitDefaults()
	initFlags()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

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

func PrintSettings() {
	// Print settings with secrets redacted
	settings := viper.AllSettings()
	settings["windlass"].(map[string]interface{})["secret"] = "[redacted]"

	out, _ := json.MarshalIndent(settings, "", "\t")
	log.Debug("config:\n%s", string(out))
}
