package binding

import (
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func ErrorMarshalResponse(resp any) string {
	return fmt.Sprintf("failed to JSON marshal response: %v", resp)
}

var (
	ErrorUnmarshalProtoBz = "failed to unmarshal bytes with proto marshaler"
	ErrorProtoBzToJson    = "failed to marshal proto bytes to JSON"
)

// ProtoMarshaler defines an interface a type must implement to serialize itself
// as a protocol buffer defined message.
// TODO docs
func ProtoToJson(
	protoMarshaler codec.ProtoMarshaler,
	bz []byte,
	cdc codec.Codec,
) (jsonBz []byte, err error) {
	// bytes -> proto
	err = cdc.Unmarshal(bz, protoMarshaler)
	if err != nil {
		errMsgExtension := wasmvmtypes.Unknown{}.Error() + ErrorUnmarshalProtoBz
		return jsonBz, sdkerrors.Wrap(err, errMsgExtension)
	}

	// proto -> json bytes
	jsonBz, err = cdc.MarshalJSON(protoMarshaler)
	if err != nil {
		errMsgExtension := wasmvmtypes.Unknown{}.Error() + ErrorProtoBzToJson
		return jsonBz, sdkerrors.Wrap(err, errMsgExtension)
	}

	protoMarshaler.Reset()

	return jsonBz, nil
}
