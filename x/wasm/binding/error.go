package binding

import (
	"fmt"
)

func ErrorMarshalResponse(resp any) string {
	return fmt.Sprintf("failed to JSON marshal response: %v", resp)
}

var (
	ErrorUnmarshalProtoBz = "failed to unmarshal bytes with proto marshaler"
	ErrorProtoBzToJson    = "failed to marshal proto bytes to JSON"
)
