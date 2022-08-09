package types

import (
	"strings"

	"gopkg.in/yaml.v2"
)

// String implements fmt.Stringer interface
func (m Pair) String() string {
	out, _ := yaml.Marshal(m)
	return string(out)
}

// Equal implements equal interface
func (m Pair) Equal(pair *Pair) bool {
	return m.Name == pair.Name && m.TobinTax.Equal(pair.TobinTax)
}

// PairList is array of Pair
type PairList []Pair

// String implements fmt.Stringer interface
func (dl PairList) String() (out string) {
	for _, d := range dl {
		out += d.String() + "\n"
	}
	return strings.TrimSpace(out)
}
