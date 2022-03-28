package testutil

import (
	"github.com/stretchr/testify/require"
)

func RequireEqualWithMessage(
	t require.TestingT, expected interface{}, actual interface{}, varName string) {

	require.Equalf(t, expected, actual,
		"Expected '%s': %d,\nActual '%s': %d",
		varName, expected, varName, actual)
}
