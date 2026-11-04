package config

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

// TestParseLaneRulesMode pins the accepted "nibiru-bft.lane-rules" values: an
// empty value, from an app.toml written before the key existed, is "auto".
func TestParseLaneRulesMode(t *testing.T) {
	for raw, want := range map[string]LaneRulesMode{
		"":     LaneRulesAuto,
		"auto": LaneRulesAuto,
		"on":   LaneRulesOn,
		"off":  LaneRulesOff,
	} {
		got, err := ParseLaneRulesMode(raw)
		require.NoError(t, err, raw)
		require.Equal(t, want, got, raw)
	}
	for _, raw := range []string{"true", "ON", "enforce"} {
		_, err := ParseLaneRulesMode(raw)
		require.ErrorContains(t, err, FlagLaneRules, raw)
	}
}

// TestNibiruBFTConfigDefaults proves the default app.toml writes
// lane-rules = "auto" and that an invalid value fails validation.
func TestNibiruBFTConfigDefaults(t *testing.T) {
	conf := DefaultConfig()
	require.Equal(t, string(LaneRulesAuto), conf.NibiruBFT.LaneRules)
	require.NoError(t, conf.NibiruBFT.Validate())

	tmpl, err := template.New("app").Parse(DefaultConfigTemplate)
	require.NoError(t, err)
	var rendered bytes.Buffer
	require.NoError(t, tmpl.Execute(&rendered, conf))
	require.Contains(t, rendered.String(), "[nibiru-bft]")
	require.Contains(t, rendered.String(), `lane-rules = "auto"`)

	conf.NibiruBFT.LaneRules = "sometimes"
	require.Error(t, conf.NibiruBFT.Validate())
}
