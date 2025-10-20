package testutil

import (
	"bytes"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = (*LogRoutingSuite)(nil)

type LogRoutingSuite struct {
	suite.Suite
}

func (s *LogRoutingSuite) SetupSuite() {
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(s)
}

func (s *LogRoutingSuite) Write(p []byte) (int, error) {
	s.T().Helper() // donΓÇÖt attribute to this frame in test output

	// Find first frame outside log/* and this adapter.
	file, line := findCaller(funcNameSkips(
		"log.",     // std log package
		"testing.", // testing harness
		// "yourpkg.(*Suite)", // this adapter's Write frame
	)...)

	// Trim trailing newline so Logf doesnΓÇÖt add an extra blank line.
	msg := string(bytes.TrimRight(p, "\n"))

	if file != "" {
		s.T().Logf("%s:%d: %s", file, line, msg)
	} else {
		s.T().Logf("%s", msg)
	}
	return len(p), nil
}

// ---- helpers ----

func findCaller(skipPrefixes ...string) (string, int) {
	pcs := make([]uintptr, 32)
	// Skip: runtime.Callers, findCaller, Write
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		fr, ok := frames.Next()
		if !ok {
			return "", 0
		}
		fn := fr.Function // e.g. "mypkg.TestFoo" or "log.(*Logger).printf"
		if hasAnyPrefix(fn, skipPrefixes) {
			continue
		}
		// Use base filename to keep logs concise; drop this if you want full path.
		return filepath.Base(fr.File), fr.Line
	}
}

func funcNameSkips(prefixes ...string) []string { return prefixes }

func hasAnyPrefix(s string, prefs []string) bool {
	for _, p := range prefs {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
