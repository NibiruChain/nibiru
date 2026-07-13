package testdata

import "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"

var ErrTest = errors.Register("table_testdata", 2, "test")

func (g TableModel) PrimaryKeyFields() []interface{} {
	return []interface{}{g.Id}
}

func (g TableModel) ValidateBasic() error {
	if g.Name == "" {
		return errors.Wrap(ErrTest, "name")
	}
	return nil
}
