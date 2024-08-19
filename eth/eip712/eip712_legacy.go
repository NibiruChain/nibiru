// Copyright (c) 2023-2024 Nibi, Inc.
package eip712

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect" // #nosec G702 for sensitive import
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/ethereum/go-ethereum/common"
	gethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type FeeDelegationOptions struct {
	FeePayer sdk.AccAddress
}

const (
	typeDefPrefix = "_"
)

// LegacyWrapTxToTypedData is an ultimate method that wraps Amino-encoded Cosmos
// Tx JSON data into an EIP712-compatible TypedData request.
func LegacyWrapTxToTypedData(
	cdc codectypes.AnyUnpacker,
	chainID uint64,
	msg sdk.Msg,
	data []byte,
	feeDelegation *FeeDelegationOptions,
) (apitypes.TypedData, error) {
	txData := make(map[string]interface{})

	if err := json.Unmarshal(data, &txData); err != nil {
		return apitypes.TypedData{}, errorsmod.Wrap(errortypes.ErrJSONUnmarshal, "failed to JSON unmarshal data")
	}

	domain := apitypes.TypedDataDomain{
		Name:              "Cosmos Web3",
		Version:           "1.0.0",
		ChainId:           gethmath.NewHexOrDecimal256(int64(chainID)),
		VerifyingContract: "cosmos",
		Salt:              "0",
	}

	msgTypes, err := extractMsgTypes(cdc, "MsgValue", msg)
	if err != nil {
		return apitypes.TypedData{}, err
	}

	if feeDelegation != nil {
		feeInfo, ok := txData["fee"].(map[string]interface{})
		if !ok {
			return apitypes.TypedData{}, errorsmod.Wrap(errortypes.ErrInvalidType, "cannot parse fee from tx data")
		}

		feeInfo["feePayer"] = feeDelegation.FeePayer.String()

		// also patching msgTypes to include feePayer
		msgTypes["Fee"] = []apitypes.Type{
			{Name: "feePayer", Type: "string"},
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		}
	}

	typedData := apitypes.TypedData{
		Types:       msgTypes,
		PrimaryType: "Tx",
		Domain:      domain,
		Message:     txData,
	}

	return typedData, nil
}

func extractMsgTypes(cdc codectypes.AnyUnpacker, msgTypeName string, msg sdk.Msg) (apitypes.Types, error) {
	rootTypes := apitypes.Types{
		"EIP712Domain": {
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "version",
				Type: "string",
			},
			{
				Name: "chainId",
				Type: "uint256",
			},
			{
				Name: "verifyingContract",
				Type: "string",
			},
			{
				Name: "salt",
				Type: "string",
			},
		},
		"Tx": {
			{Name: "account_number", Type: "string"},
			{Name: "chain_id", Type: "string"},
			{Name: "fee", Type: "Fee"},
			{Name: "memo", Type: "string"},
			{Name: "msgs", Type: "Msg[]"},
			{Name: "sequence", Type: "string"},
			// Note timeout_height was removed because it was not getting filled with the legacyTx
			// {Name: "timeout_height", Type: "string"},
		},
		"Fee": {
			{Name: "amount", Type: "Coin[]"},
			{Name: "gas", Type: "string"},
		},
		"Coin": {
			{Name: "denom", Type: "string"},
			{Name: "amount", Type: "string"},
		},
		"Msg": {
			{Name: "type", Type: "string"},
			{Name: "value", Type: msgTypeName},
		},
		msgTypeName: {},
	}

	if err := walkFields(cdc, rootTypes, msgTypeName, msg); err != nil {
		return nil, err
	}

	return rootTypes, nil
}

func walkFields(cdc codectypes.AnyUnpacker, typeMap apitypes.Types, rootType string, in interface{}) (err error) {
	defer doRecover(&err)

	t := reflect.TypeOf(in)
	v := reflect.ValueOf(in)

	for {
		if t.Kind() == reflect.Ptr ||
			t.Kind() == reflect.Interface {
			t = t.Elem()
			v = v.Elem()

			continue
		}

		break
	}

	return legacyTraverseFields(cdc, typeMap, rootType, typeDefPrefix, t, v)
}

type CosmosAnyWrapper struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// legacyTraverseFields: Recursively inspects the fields of a given
// `reflect.Type` (t) and `reflect.Value`(v) and maps them to an
// Ethereum-compatible type description compliant with EIP-712. For operations
// like EIP-712 signing, complex Go structs need to be translated into a flat
// list of types that can be understood in Ethereum's type system.
func legacyTraverseFields(
	// cdc: A codec capable of unpackaing protobuf
	// `"github.com/cosmos/cosmos-sdk/codec/types".Any` types into Go
	// structs.
	cdc codectypes.AnyUnpacker,
	// typeMap: map storing type descriptions
	typeMap apitypes.Types,
	// rootType: name of the root type processed
	rootType string,
	// prefix: Namespace prefix to avoid name collisions in `typeMap`
	prefix string,
	// t: reflect type of the data to process
	t reflect.Type,
	// v: reflect value of the data to process
	v reflect.Value,
) error {
	// Setup: Check that the number of fields in `typeMap` for the `rootType`
	// or a sanitized version of `prefix` matches the number of fields in
	// type `t`. If they match, the type has already been processed, so we
	// return early.
	numFieldsT := t.NumField()
	if prefix == typeDefPrefix {
		if len(typeMap[rootType]) == numFieldsT {
			return nil
		}
	} else {
		typeDef := sanitizeTypedef(prefix)
		if len(typeMap[typeDef]) == numFieldsT {
			return nil
		}
	}

	// Field Iteration: Iterate over each field of tpye `t`,
	// (1) extracting the type and value of the field,
	// (2) unpacking in the event the field is an `Any`,
	// (3) and skipping empty fields.
	// INFO: If a field is a struct, unpack each field recursively to handle
	// nested data structures.
	for fieldIdx := 0; fieldIdx < numFieldsT; fieldIdx++ {
		var (
			field reflect.Value
			err   error
		)

		if v.IsValid() {
			field = v.Field(fieldIdx)
		}

		fieldType := t.Field(fieldIdx).Type
		fieldName := jsonNameFromTag(t.Field(fieldIdx).Tag)

		if fieldType == typeCosmAny {
			// Unpack field, value as Any
			if fieldType, field, err = UnpackAny(cdc, field); err != nil {
				return err
			}
		}

		// If field is an empty value, do not include in types, since it will not
		// be present in the object
		if field.IsZero() {
			continue
		}

		for {
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()

				if field.IsValid() {
					field = field.Elem()
				}

				continue
			}

			if fieldType.Kind() == reflect.Interface {
				fieldType = reflect.TypeOf(field.Interface())
				continue
			}

			if field.Kind() == reflect.Ptr {
				field = field.Elem()
				continue
			}

			break
		}

		var isCollection bool
		if fieldType.Kind() == reflect.Array || fieldType.Kind() == reflect.Slice {
			if field.Len() == 0 {
				// skip empty collections from type mapping
				continue
			}

			fieldType = fieldType.Elem()
			field = field.Index(0)
			isCollection = true

			if fieldType == typeCosmAny {
				if fieldType, field, err = UnpackAny(cdc, field); err != nil {
					return err
				}
			}
		}

		for {
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()

				if field.IsValid() {
					field = field.Elem()
				}

				continue
			}

			if fieldType.Kind() == reflect.Interface {
				fieldType = reflect.TypeOf(field.Interface())
				continue
			}

			if field.Kind() == reflect.Ptr {
				field = field.Elem()
				continue
			}

			break
		}

		fieldPrefix := fmt.Sprintf("%s.%s", prefix, fieldName)

		ethTyp := TypToEth(fieldType)

		if len(ethTyp) > 0 {
			// Support array of uint64
			if isCollection && fieldType.Kind() != reflect.Slice && fieldType.Kind() != reflect.Array {
				ethTyp += "[]"
			}

			if prefix == typeDefPrefix {
				typeMap[rootType] = append(typeMap[rootType], apitypes.Type{
					Name: fieldName,
					Type: ethTyp,
				})
			} else {
				typeDef := sanitizeTypedef(prefix)
				typeMap[typeDef] = append(typeMap[typeDef], apitypes.Type{
					Name: fieldName,
					Type: ethTyp,
				})
			}

			continue
		}

		if fieldType.Kind() == reflect.Struct {
			var fieldTypedef string

			if isCollection {
				fieldTypedef = sanitizeTypedef(fieldPrefix) + "[]"
			} else {
				fieldTypedef = sanitizeTypedef(fieldPrefix)
			}

			if prefix == typeDefPrefix {
				typeMap[rootType] = append(typeMap[rootType], apitypes.Type{
					Name: fieldName,
					Type: fieldTypedef,
				})
			} else {
				typeDef := sanitizeTypedef(prefix)
				typeMap[typeDef] = append(typeMap[typeDef], apitypes.Type{
					Name: fieldName,
					Type: fieldTypedef,
				})
			}

			if err := legacyTraverseFields(cdc, typeMap, rootType, fieldPrefix, fieldType, field); err != nil {
				return err
			}

			continue
		}
	}

	return nil
}

func jsonNameFromTag(tag reflect.StructTag) string {
	jsonTags := tag.Get("json")
	parts := strings.Split(jsonTags, ",")
	return parts[0]
}

// Unpack the given Any value with Type/Value deconstruction
func UnpackAny(cdc codectypes.AnyUnpacker, field reflect.Value) (reflect.Type, reflect.Value, error) {
	anyData, ok := field.Interface().(*codectypes.Any)
	if !ok {
		return nil, reflect.Value{}, errorsmod.Wrapf(errortypes.ErrPackAny, "%T", field.Interface())
	}

	anyWrapper := &CosmosAnyWrapper{
		Type: anyData.TypeUrl,
	}

	if err := cdc.UnpackAny(anyData, &anyWrapper.Value); err != nil {
		return nil, reflect.Value{}, errorsmod.Wrap(err, "failed to unpack Any in msg struct")
	}

	fieldType := reflect.TypeOf(anyWrapper)
	field = reflect.ValueOf(anyWrapper)

	return fieldType, field, nil
}

var (
	typeEthHash = reflect.TypeOf(common.Hash{})
	typeEthAddr = reflect.TypeOf(common.Address{})
	typeBigInt  = reflect.TypeOf(big.Int{})
	typeCosmInt = reflect.TypeOf(sdkmath.Int{})
	typeCosmDec = reflect.TypeOf(sdkmath.LegacyDec{})
	typeTime    = reflect.TypeOf(time.Time{})
	typeCosmAny = reflect.TypeOf(&codectypes.Any{})
	typeEd25519 = reflect.TypeOf(ed25519.PubKey{})
)

// TypToEth supports only basic types and arrays of basic types.
// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-712.md
func TypToEth(typ reflect.Type) string {
	const str = "string"

	switch typ.Kind() {
	case reflect.String:
		return str
	case reflect.Bool:
		return "bool"
	case reflect.Int:
		return "int64"
	case reflect.Int8:
		return "int8"
	case reflect.Int16:
		return "int16"
	case reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint:
		return "uint64"
	case reflect.Uint8:
		return "uint8"
	case reflect.Uint16:
		return "uint16"
	case reflect.Uint32:
		return "uint32"
	case reflect.Uint64:
		return "uint64"
	case reflect.Slice:
		ethName := TypToEth(typ.Elem())
		if len(ethName) > 0 {
			return ethName + "[]"
		}
	case reflect.Array:
		ethName := TypToEth(typ.Elem())
		if len(ethName) > 0 {
			return ethName + "[]"
		}
	case reflect.Ptr:
		if typ.Elem().ConvertibleTo(typeBigInt) ||
			typ.Elem().ConvertibleTo(typeTime) ||
			typ.Elem().ConvertibleTo(typeEd25519) ||
			typ.Elem().ConvertibleTo(typeCosmDec) ||
			typ.Elem().ConvertibleTo(typeCosmInt) {
			return str
		}
	case reflect.Struct:
		if typ.ConvertibleTo(typeEthHash) ||
			typ.ConvertibleTo(typeEthAddr) ||
			typ.ConvertibleTo(typeBigInt) ||
			typ.ConvertibleTo(typeEd25519) ||
			typ.ConvertibleTo(typeTime) ||
			typ.ConvertibleTo(typeCosmDec) ||
			typ.ConvertibleTo(typeCosmInt) {
			return str
		}
	}

	return ""
}
