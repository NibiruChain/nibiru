package v1

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
)

// NewLegacyContent creates a new MsgExecLegacyContent from a legacy Content
// interface.
func NewLegacyContent(content v1beta1.Content, authority string) (*MsgExecLegacyContent, error) {
	msg, ok := content.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%T does not implement proto.Message", content)
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, err
	}

	return NewMsgExecLegacyContent(any, authority), nil
}

// LegacyContentFromMessage extracts the legacy Content interface from a
// MsgExecLegacyContent.
func LegacyContentFromMessage(msg *MsgExecLegacyContent) (v1beta1.Content, error) {
	content, ok := msg.Content.GetCachedValue().(v1beta1.Content)
	if !ok {
		return nil, sdkerrors.ErrInvalidType.Wrapf("expected %T, got %T", (*v1beta1.Content)(nil), msg.Content.GetCachedValue())
	}

	return content, nil
}
