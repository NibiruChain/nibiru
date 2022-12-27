package cligen

import (
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestDefaultFromProtoMsg(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
		want Params
	}{
		{
			name: "default from proto message same type",
			msg:  &types.MsgPostPrice{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultFromProtoMsg(tt.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultFromProtoMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParams_Mandatory(t *testing.T) {
	params := Params{
		{Name: "token0", Mandatory: true},
		{Name: "token1", Mandatory: true},
		{Name: "price"},
		{Name: "expiry", Mandatory: true},
	}

	require.Len(t, params.Mandatory(), 3)
}
