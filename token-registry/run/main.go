package main

import (
	"fmt"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

func main() {
	assetList := tokenregistry.NibiruAssetList()
	fmt.Println("FOO!!")
	fmt.Println(assetList)
}
