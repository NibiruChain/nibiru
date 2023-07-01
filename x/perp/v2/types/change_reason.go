package types

import "encoding/json"

type customProtobufType interface {
	Marshal() ([]byte, error)
	MarshalTo(data []byte) (n int, err error)
	Unmarshal(data []byte) error
	Size() int

	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

var _ customProtobufType = (*ChangeReason)(nil)

type ChangeReason string

const (
	ChangeReason_MarketOrder   ChangeReason = "market_order"
	ChangeReason_ClosePosition ChangeReason = "close_position"
	ChangeReason_AddMargin     ChangeReason = "add_margin"
	ChangeReason_RemoveMargin  ChangeReason = "remove_margin"
	ChangeReason_Liquidate     ChangeReason = "liquidate"
)

func (c *ChangeReason) Size() int {
	return len(*c)
}

func (c *ChangeReason) Marshal() ([]byte, error) {
	return []byte(*c), nil
}

func (c *ChangeReason) MarshalTo(data []byte) (n int, err error) {
	return copy(data, *c), nil
}

func (c *ChangeReason) Unmarshal(data []byte) error {
	*c = ChangeReason(data)
	return nil
}

func (c *ChangeReason) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

func (c *ChangeReason) UnmarshalJSON(data []byte) error {
	var s string

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	*c = ChangeReason(s)
	return nil
}
