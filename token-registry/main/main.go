package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/rs/zerolog/log"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

const (
	SAVE_PATH_ASSETLIST           = "dist/assetlist.json"
	SAVE_PATH_COSMOS_ASSETLIST    = "dist/cosmos-assetlist.json"
	SAVE_PATH_OFFICIAL_ERC20S     = "token-registry/official_erc20s.json"
	SAVE_PATH_OFFICIAL_BANK_COINS = "token-registry/official_bank_coins.json"
)

func main() {
	rootPath, err := tokenregistry.FindRootPath()
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Create the dist directory if it does not exist
	distDirPath := path.Join(rootPath, "dist")
	if _, err := os.Stat(distDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(distDirPath, 0755); err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}

	// Create dist/assetlist.json
	assetList := tokenregistry.NibiruAssetList()
	savePath := path.Join(rootPath, SAVE_PATH_ASSETLIST)
	prettyBz, err := json.MarshalIndent(assetList, "", "  ")
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	perm := os.FileMode(0666) // All can read and write
	err = os.WriteFile(savePath, prettyBz, perm)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Create token-registry/official_erc20s.json
	savePath = path.Join(rootPath, SAVE_PATH_OFFICIAL_ERC20S)
	prettyBz, err = tokenregistry.ParseOfficialSaveBz(tokenregistry.ERC20S)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	err = os.WriteFile(savePath, prettyBz, perm)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Create token-registry/official_bank_coins.json
	savePath = path.Join(rootPath, SAVE_PATH_OFFICIAL_BANK_COINS)
	prettyBz, err = tokenregistry.ParseOfficialSaveBz(tokenregistry.BANK_COINS)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	err = os.WriteFile(savePath, prettyBz, perm)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	// Create dist/cosmos-assetlist.json
	savePath = path.Join(rootPath, SAVE_PATH_COSMOS_ASSETLIST)
	saveBz := tokenregistry.PointImagesToCosmosChainRegistry(prettyBz)
	err = os.WriteFile(savePath, saveBz, perm)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	fmt.Printf("✅ Generation complete!\n")
	fmt.Printf(
		"File \"%v\" contains ERC20s for the Nibiru web app\n",
		SAVE_PATH_OFFICIAL_ERC20S,
	)
	fmt.Printf(
		"File \"%v\" contains Bank Coins for the Nibiru web app\n",
		SAVE_PATH_OFFICIAL_BANK_COINS,
	)
	fmt.Printf(
		"File \"%v\" contains the asset list using images only from the Nibiru repo\n",
		SAVE_PATH_ASSETLIST,
	)
	fmt.Printf(
		"File \"%v\" contains the asset list for the cosmos/chain-registry\n",
		SAVE_PATH_COSMOS_ASSETLIST,
	)
	fmt.Println("You can submit a PR to cosmos/chain-registry using " +
		SAVE_PATH_COSMOS_ASSETLIST +
		" as the file chain-registry/nibiru/assetlist.json",
	)
}
