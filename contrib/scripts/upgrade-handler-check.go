package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/NibiruChain/nibiru/v2/app"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run upgrade-handler-check.go <semver>")
		os.Exit(1)
	}

	var handlers []string
	for _, upgrade := range app.Upgrades {
		handlers = append(handlers, upgrade.UpgradeName)
	}
	handler, err := upgradeHandlerForVersion(os.Args[1], handlers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Release %s uses upgrade handler %s.\n", os.Args[1], handler)
}

var releaseVersionPattern = regexp.MustCompile(
	`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
		`(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?` +
		`(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`,
)

// upgradeHandlerForVersion assumes patch releases preserve consensus behavior.
// Exact handlers take precedence to support historical patch upgrades.
func upgradeHandlerForVersion(version string, handlers []string) (string, error) {
	parts := releaseVersionPattern.FindStringSubmatch(version)
	if parts == nil {
		return "", fmt.Errorf("invalid release version %q: expected vMAJOR.MINOR.PATCH with optional prerelease and build metadata", version)
	}
	for _, identifier := range strings.Split(parts[4], ".") {
		if len(identifier) > 1 && identifier[0] == '0' && strings.Trim(identifier, "0123456789") == "" {
			return "", fmt.Errorf("invalid release version %q: numeric prerelease identifiers must not have leading zeroes", version)
		}
	}
	core := "v" + strings.Join(parts[1:4], ".")
	for _, handler := range handlers {
		if handler == core {
			return handler, nil
		}
	}
	if parts[3] == "0" {
		return "", fmt.Errorf("release %s requires upgrade handler %s", version, core)
	}
	base := "v" + parts[1] + "." + parts[2] + ".0"
	for _, handler := range handlers {
		if handler == base {
			return handler, nil
		}
	}
	return "", fmt.Errorf("compatible patch %s requires existing upgrade handler %s", version, base)
}
