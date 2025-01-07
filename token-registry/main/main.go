package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/rs/zerolog/log"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

// findRootPath returns the absolute path of the repository root
// This is retrievable with: go list -m -f {{.Dir}}
func findRootPath() (string, error) {
	// rootPath, _ := exec.Command("go list -m -f {{.Dir}}").Output()
	// This returns the path to the root of the project.
	rootPathBz, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", err
	}
	rootPath := strings.Trim(string(rootPathBz), "\n")
	return rootPath, nil
}

const SAVE_PATH_ASSETLIST = "dist/assetlist.json"

func main() {
	assetList := tokenregistry.NibiruAssetList()

	prettyBz, err := json.MarshalIndent(assetList, "", "  ")
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	rootPath, err := findRootPath()
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	savePath := path.Join(rootPath, SAVE_PATH_ASSETLIST)

	// Create the dist directory if it does not exist
	distDirPath := path.Join(rootPath, "dist")
	if _, err := os.Stat(distDirPath); os.IsNotExist(err) {
		if err := os.Mkdir(distDirPath, 0755); err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}

	perm := os.FileMode(0666) // All can read and write
	err = os.WriteFile(savePath, prettyBz, perm)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	fmt.Printf("✅ Generation complete! See %v\n", SAVE_PATH_ASSETLIST)
}
