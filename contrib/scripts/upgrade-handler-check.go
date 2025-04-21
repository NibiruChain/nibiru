package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/NibiruChain/nibiru/v2/app"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run upgrade-handler-check.go <semver>")
		os.Exit(1)
	}

	input := os.Args[1]
	coreVersion := extractCoreVersion(input)

	found := false
	for _, upgrade := range app.Upgrades {
		if upgrade.UpgradeName == coreVersion {
			found = true
			break
		}
	}

	if found {
		fmt.Printf("Upgrade handler for version %s exists ✅\n", coreVersion)
	} else {
		fmt.Printf("Upgrade handler for version %s does not exist ❌\n", coreVersion)
		os.Exit(1)
	}
}

// extractCoreVersion strips any pre-release or build metadata from a semver string
func extractCoreVersion(version string) string {
	// Trim possible build/pre-release suffix using a simple regex
	re := regexp.MustCompile(`^v\d+\.\d+\.\d+`)
	match := re.FindString(version)
	if match != "" {
		return match
	}
	return strings.SplitN(version, "-", 2)[0] // Fallback just in case
}
