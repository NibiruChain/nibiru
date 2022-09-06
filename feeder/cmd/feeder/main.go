package main

import (
	"github.com/NibiruChain/nibiru/feeder/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/app"
)

func main() {
	app.SetPrefixes(app.AccountAddressPrefix)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Debug().Msg("fetching configuration")

	conf := config.Get()

	log.Debug().Msg("connecting the feeder")
	f, err := conf.DialFeeder()
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.Info().Msg("running the feeder")
	f.Run()
}
