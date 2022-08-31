package main

import "github.com/NibiruChain/nibiru/feeder/pkg/config"

func main() {
	config := config.MustGet()

	panic(config)
}
