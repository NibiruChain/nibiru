package contrib

import (
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

var replacesThatMustMatchRoot = []string{
	"github.com/ethereum/go-ethereum",
	"github.com/gogo/protobuf",
	"github.com/linxGnu/grocksdb",
	"github.com/syndtr/goleveldb",
	"golang.org/x/exp",
}

func TestReplaceMatchesRoot(t *testing.T) {
	rootMod := readGoMod(t, "../go.mod")
	saiTradingMod := readGoMod(t, "../sai-trading/go.mod")

	for _, modulePath := range replacesThatMustMatchRoot {
		t.Run(modulePath, func(t *testing.T) {
			rootReplace := findReplace(t, rootMod, "../go.mod", modulePath)
			saiTradingReplace := findReplace(t, saiTradingMod, "../sai-trading/go.mod", modulePath)

			if rootReplace.New.Path != saiTradingReplace.New.Path ||
				rootReplace.New.Version != saiTradingReplace.New.Version {
				t.Fatalf(
					"replace directive for %q must match root go.mod\nroot:        %s => %s %s\nsai-trading: %s => %s %s",
					modulePath,
					rootReplace.Old.Path,
					rootReplace.New.Path,
					rootReplace.New.Version,
					saiTradingReplace.Old.Path,
					saiTradingReplace.New.Path,
					saiTradingReplace.New.Version,
				)
			}
		})
	}
}

func readGoMod(t *testing.T, path string) *modfile.File {
	t.Helper()

	goModBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	goMod, err := modfile.Parse(path, goModBytes, nil)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	return goMod
}

func findReplace(t *testing.T, goMod *modfile.File, path, modulePath string) *modfile.Replace {
	t.Helper()

	for _, replace := range goMod.Replace {
		if replace.Old.Path == modulePath {
			return replace
		}
	}

	t.Fatalf("missing replace directive for %q in %s", modulePath, path)
	return nil
}
