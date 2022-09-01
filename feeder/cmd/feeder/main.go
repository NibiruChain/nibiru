package main

import (
	"github.com/NibiruChain/nibiru/feeder"
	"log"
)

func main() {
	config := feeder.GetConfig()

	f, err := feeder.Dial(config)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	log.Printf("running feeder")
	f.Run()
}
