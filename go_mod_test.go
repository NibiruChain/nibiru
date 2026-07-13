package nibiru_test

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var bannedUpstream = []string{
	"github.com/cosmos/cosmos-sdk",
	"github.com/cosmos/ibc-go",
}

func isBannedModule(modPath string) (banned string, ok bool) {
	prefix, _, valid := module.SplitPathVersion(modPath)
	if !valid {
		prefix = modPath
	}
	for _, banned := range bannedUpstream {
		if prefix == banned {
			return banned, true
		}
	}
	return "", false
}

func TestGoMod_NoBannedDependencies(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}

	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, req := range f.Require {
		if banned, ok := isBannedModule(req.Mod.Path); ok {
			t.Fatalf("go.mod must not require upstream %s after flattening, found: %s %s", banned, req.Mod.Path, req.Mod.Version)
		}
	}

	for _, repl := range f.Replace {
		if banned, ok := isBannedModule(repl.Old.Path); ok {
			t.Fatalf("go.mod must not replace upstream %s after flattening, found replace for: %s", banned, repl.Old.Path)
		}
	}
}

func TestGoSum_NoBannedDependencies(t *testing.T) {
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
		if banned, ok := isBannedModule(modPath); ok {
			t.Fatalf("go.sum must not reference upstream %s after flattening, found: %s", banned, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedProtoImports_NoUpstreamModules(t *testing.T) {
	for _, root := range []string{"eth", "evm", "x"} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pb.go") {
				return nil
			}

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, importSpec := range file.Imports {
				importPath, err := strconv.Unquote(importSpec.Path.Value)
				if err != nil {
					return err
				}
				for _, banned := range bannedUpstream {
					if importPath == banned || strings.HasPrefix(importPath, banned+"/") {
						return fmt.Errorf("%s imports upstream module %s", path, importPath)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
