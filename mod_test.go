package nibiru_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const bannedIBCGo = "github.com/cosmos/ibc-go"

func isBannedIBCGoModule(modPath string) bool {
	prefix, _, valid := module.SplitPathVersion(modPath)
	if !valid {
		prefix = modPath
	}
	return prefix == bannedIBCGo
}

func TestGoMod_NoUpstreamIBCGo(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, req := range f.Require {
		if isBannedIBCGoModule(req.Mod.Path) {
			t.Fatalf("go.mod must not require upstream ibc-go after flattening, found: %s %s", req.Mod.Path, req.Mod.Version)
		}
	}

	for _, repl := range f.Replace {
		if isBannedIBCGoModule(repl.Old.Path) {
			t.Fatalf("go.mod must not replace upstream ibc-go after flattening, found replace for: %s", repl.Old.Path)
		}
	}
}

func TestGoSum_NoUpstreamIBCGo(t *testing.T) {
	f, err := os.Open("go.sum")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		modPath, _, _ := strings.Cut(line, " ")
		if isBannedIBCGoModule(modPath) {
			t.Fatalf("go.sum must not reference upstream ibc-go after flattening, found: %s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
