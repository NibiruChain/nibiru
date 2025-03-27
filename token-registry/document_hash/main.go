// /
package main

/*
Usage:
```
go run token-registry/document_hash/main.go
```
*/

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

const (
	SAVE_PATH_ASSETLIST           = "dist/assetlist.json"
	SAVE_PATH_COSMOS_ASSETLIST    = "dist/cosmos-assetlist.json"
	SAVE_PATH_OFFICIAL_ERC20S     = "token-registry/official_erc20s.json"
	SAVE_PATH_OFFICIAL_BANK_COINS = "token-registry/official_bank_coins.json"
)

type FileAndHash struct {
	Filename string
	// Hex-encdoed sha256 hash of the document specified by "filename". This
	// hash can be used to verify the document didn't change.
	// Optionally used for the bank.MetaData.URIHash. See
	// "cosmos-sdk/x/bank/types/bank.pb.go".
	URIHash string `json:"uri_hash"`
}

func main() {
	rootPath, err := tokenregistry.FindRootPath()
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Create the dist directory if it does not exist
	imgDirPath := path.Join(rootPath, "token-registry", "img")
	if _, err := os.Stat(imgDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(imgDirPath, 0755); err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}

	// Read token-registry/img contents
	entries, err := os.ReadDir(imgDirPath)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	output := []FileAndHash{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		fname := entry.Name()
		fileExt := filepath.Ext(fname)
		switch fileExt {
		case ".png", ".svg":
			imgPath := path.Join(imgDirPath, fname)
			imgBz, err := os.ReadFile(imgPath)
			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			// Produce the sha256 checksum
			computedHash := sha256.Sum256(imgBz)
			output = append(output, FileAndHash{
				Filename: fname,
				URIHash:  fmt.Sprintf("%x", computedHash),
			})
		default:
		}
	}

	prettyBz, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Error().Msg(err.Error())
		// Fall back to the debug string of the struct
		fmt.Printf("output: %+s", output)
		return
	}

	fmt.Printf("Computed sha256 checksums for the image files in the \"token-registry/img\" directory")
	fmt.Println(string(prettyBz))
}
