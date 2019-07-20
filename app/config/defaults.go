package config

import (
	"github.com/spf13/viper"
)

func InitDefaults() {
	viper.SetDefault("http.port", "9786")
}
