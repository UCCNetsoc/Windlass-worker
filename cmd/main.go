package main

import (
	"net/http"

	log "github.com/UCCNetworkingSociety/Windlass-worker/utils/logging"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/api"
	"github.com/UCCNetworkingSociety/Windlass-worker/app/connections"

	"github.com/UCCNetworkingSociety/Windlass-worker/app/config"

	"github.com/spf13/viper"

	"github.com/UCCNetworkingSociety/Windlass-worker/utils/must"
	"github.com/go-chi/chi"
)

func main() {
	r := chi.NewRouter()

	must.Do(config.Load)

	must.Do(connections.EstablishConnections)
	defer connections.Close()

	config.PrintSettings()

	api.NewAPI(r).Init()
	log.Info("API server started")

	if err := http.ListenAndServe(":"+viper.GetString("http.port"), r); err != nil {
		log.Error(err, "error starting server")
	}
}
