package app

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"

	_ "github.com/NibiruChain/nibiru/contrib/swagger/statik"
)

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}
