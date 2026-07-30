package evmtrader

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

type mockDataError struct {
	msg  string
	data interface{}
}

func (e *mockDataError) Error() string          { return e.msg }
func (e *mockDataError) ErrorData() interface{} { return e.data }

func encodeStringRevert(t *testing.T, reason string) []byte {
	t.Helper()
	selector := crypto.Keccak256([]byte("Error(string)"))[:4]
	stringTy, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	packed, err := abi.Arguments{{Type: stringTy}}.Pack(reason)
	require.NoError(t, err)
	return append(selector, packed...)
}

func TestDecodeRevertErrorData_StandardStringReason(t *testing.T) {
	revertData := encodeStringRevert(t, "market exposure limit reached")
	err := &mockDataError{
		msg:  "execution reverted",
		data: hexutil.Encode(revertData),
	}

	reason, ok := decodeRevertErrorData(err)
	require.True(t, ok)
	require.Equal(t, "market exposure limit reached", reason)
}

func TestDecodeRevertErrorData_StandardStringReasonAsBytes(t *testing.T) {
	revertData := encodeStringRevert(t, "insufficient balance")
	err := &mockDataError{
		msg:  "execution reverted",
		data: revertData,
	}

	reason, ok := decodeRevertErrorData(err)
	require.True(t, ok)
	require.Equal(t, "insufficient balance", reason)
}

func TestDecodeRevertErrorData_NotADataError(t *testing.T) {
	reason, ok := decodeRevertErrorData(errors.New("execution reverted"))
	require.False(t, ok)
	require.Empty(t, reason)
}

func TestDecodeRevertErrorData_UndecodableData(t *testing.T) {
	err := &mockDataError{
		msg:  "execution reverted",
		data: "not-hex-data",
	}

	reason, ok := decodeRevertErrorData(err)
	require.False(t, ok)
	require.Empty(t, reason)
}

func TestDecodeRevertErrorData_TooShort(t *testing.T) {
	err := &mockDataError{
		msg:  "execution reverted",
		data: hexutil.Encode([]byte{0x01, 0x02}),
	}

	reason, ok := decodeRevertErrorData(err)
	require.False(t, ok)
	require.Empty(t, reason)
}
