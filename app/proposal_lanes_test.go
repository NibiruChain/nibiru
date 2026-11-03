package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsOraclePriceWriteMsg pins the JSON shapes accepted as oracle price
// writes: exactly one top-level key, `set_prices` or `set_price`, and nothing
// else in the message.
func TestIsOraclePriceWriteMsg(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  string
		want bool
	}{
		{"set_prices", `{"set_prices":{"prices":[[0,"1.5"]]}}`, true},
		{"set_price", `{"set_price":{"token_id":0,"price_usd":"1.5"}}`, true},
		{"whitespace", " { \"set_prices\" : { } } \n", true},
		{"non-price variant", `{"update_expiration_time":{"expiration_time":1}}`, false},
		{"extra key", `{"set_prices":{},"set_price":{}}`, false},
		{"price key second", `{"create_token":{},"set_prices":{}}`, false},
		{"trailing data", `{"set_prices":{}}{}`, false},
		{"truncated", `{"set_prices":{"prices":[]}`, false},
		{"camel case", `{"setPrices":{}}`, false},
		{"nested", `{"wrapper":{"set_prices":{}}}`, false},
		{"array", `["set_prices"]`, false},
		{"string", `"set_prices"`, false},
		{"empty object", `{}`, false},
		{"empty", ``, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isOraclePriceWriteMsg([]byte(tc.msg)))
		})
	}
}
