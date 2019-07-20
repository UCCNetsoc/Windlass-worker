package main

import (
	"net/http"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/config"

	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"
	"github.com/UCCNetworkingSociety/Windlass-worker/utils/must"
	"github.com/go-chi/chi"
)

func main() {
	must.Do(func() error {
		return config.LoadConfig()
	})

	must.Do(func() error {
		return connections.EstablishConnections()
	})

	r := chi.NewRouter()
	http.ListenAndServe(":"+viper.GetString("http.port"), r)
}
