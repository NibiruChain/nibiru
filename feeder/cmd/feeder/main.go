package main

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/NibiruChain/nibiru/feeder"
)

func main() {
	app.SetPrefixes(app.AccountAddressPrefix)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Debug().Msg("fetching configuration")

	config := feeder.GetConfig()

	log.Debug().Msg("connecting the feeder")
	f, err := feeder.Dial(config)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.Info().Msg("running the feeder")
	f.Run()
}
