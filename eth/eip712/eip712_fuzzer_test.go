package eip712_test

import (
	"fmt"
	"strings"

	rand "github.com/cometbft/cometbft/libs/rand"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/NibiruChain/nibiru/eth/eip712"
)

type EIP712FuzzTestParams struct {
	numTestObjects        int
	maxNumFieldsPerObject int
	minStringLength       int
	maxStringLength       int
	randomFloatRange      float64
	maxArrayLength        int
	maxObjectDepth        int
}

const (
	numPrimitiveJSONTypes = 3
	numJSONTypes          = 5
	asciiRangeStart       = 65
	asciiRangeEnd         = 127
	fuzzTestName          = "Flatten"
)

const (
	jsonBoolType   = iota
	jsonStringType = iota
	jsonFloatType  = iota
	jsonArrayType  = iota
	jsonObjectType = iota
)

var params = EIP712FuzzTestParams{
	numTestObjects:        16,
	maxNumFieldsPerObject: 16,
	minStringLength:       16,
	maxStringLength:       48,
	randomFloatRange:      120000000,
	maxArrayLength:        8,
	maxObjectDepth:        4,
}

// TestRandomPayloadFlattening generates many random payloads with different JSON values to ensure
// that Flattening works across all inputs.
// Note that this is a fuzz test, although it doesn't use Go's Fuzz testing suite, since there are
// variable input sizes, types, and fields. While it may be possible to translate a single input into
// a JSON object, it would require difficult parsing, and ultimately approximates our randomized unit
// tests as they are.
func (s *EIP712TestSuite) TestRandomPayloadFlattening() {
	// Re-seed rand generator
	rand.Seed(rand.Int64())

	for i := 0; i < params.numTestObjects; i++ {
		s.Run(fmt.Sprintf("%v%d", fuzzTestName, i), func() {
			payload := s.generateRandomPayload(i)

			flattened, numMessages, err := eip712.FlattenPayloadMessages(payload)

			s.Require().NoError(err)
			s.Require().Equal(numMessages, i)

			s.verifyPayloadAgainstFlattened(payload, flattened)
		})
	}
}

// generateRandomPayload creates a random payload of the desired format, with random sub-objects.
func (s *EIP712TestSuite) generateRandomPayload(numMessages int) gjson.Result {
	payload := s.createRandomJSONObject().Raw
	msgs := make([]gjson.Result, numMessages)

	for i := 0; i < numMessages; i++ {
		msgs[i] = s.createRandomJSONObject()
	}

	payload, err := sjson.Set(payload, msgsFieldName, msgs)
	s.Require().NoError(err)

	return gjson.Parse(payload)
}

// createRandomJSONObject creates a JSON object with random fields.
func (s *EIP712TestSuite) createRandomJSONObject() gjson.Result {
	var err error
	payloadRaw := ""

	numFields := s.createRandomIntInRange(0, params.maxNumFieldsPerObject)
	for i := 0; i < numFields; i++ {
		key := s.createRandomString()

		randField := s.createRandomJSONField(i, 0)
		payloadRaw, err = sjson.Set(payloadRaw, key, randField)
		s.Require().NoError(err)
	}

	return gjson.Parse(payloadRaw)
}

// createRandomJSONField creates a random field with a random JSON type, with the possibility of
// nested fields up to depth objects.
func (s *EIP712TestSuite) createRandomJSONField(t int, depth int) interface{} {
	switch t % numJSONTypes {
	case jsonBoolType:
		return s.createRandomBoolean()
	case jsonStringType:
		return s.createRandomString()
	case jsonFloatType:
		return s.createRandomFloat()
	case jsonArrayType:
		return s.createRandomJSONNestedArray(depth)
	case jsonObjectType:
		return s.createRandomJSONNestedObject(depth)
	default:
		return nil
	}
}

// createRandomJSONNestedArray creates an array of random nested JSON fields.
func (s *EIP712TestSuite) createRandomJSONNestedArray(depth int) []interface{} {
	arr := make([]interface{}, rand.Intn(params.maxArrayLength))
	for i := range arr {
		arr[i] = s.createRandomJSONNestedField(depth)
	}

	return arr
}

// createRandomJSONNestedObject creates a key-value set of objects with random nested JSON fields.
func (s *EIP712TestSuite) createRandomJSONNestedObject(depth int) interface{} {
	numFields := rand.Intn(params.maxNumFieldsPerObject)
	obj := make(map[string]interface{})

	for i := 0; i < numFields; i++ {
		subField := s.createRandomJSONNestedField(depth)

		obj[s.createRandomString()] = subField
	}

	return obj
}

// createRandomJSONNestedField serves as a helper for createRandomJSONField and returns a random
// subfield to populate an array or object type.
func (s *EIP712TestSuite) createRandomJSONNestedField(depth int) interface{} {
	var newFieldType int

	if depth == params.maxObjectDepth {
		newFieldType = rand.Intn(numPrimitiveJSONTypes)
	} else {
		newFieldType = rand.Intn(numJSONTypes)
	}

	return s.createRandomJSONField(newFieldType, depth+1)
}

func (s *EIP712TestSuite) createRandomBoolean() bool {
	return rand.Intn(2) == 0
}

func (s *EIP712TestSuite) createRandomFloat() float64 {
	return (rand.Float64() - 0.5) * params.randomFloatRange
}

func (s *EIP712TestSuite) createRandomString() string {
	bzLen := s.createRandomIntInRange(params.minStringLength, params.maxStringLength)
	bz := make([]byte, bzLen)

	for i := 0; i < bzLen; i++ {
		bz[i] = byte(s.createRandomIntInRange(asciiRangeStart, asciiRangeEnd))
	}

	str := string(bz)

	// Remove control characters, since they will make JSON invalid
	str = strings.ReplaceAll(str, "{", "")
	str = strings.ReplaceAll(str, "}", "")
	str = strings.ReplaceAll(str, "]", "")
	str = strings.ReplaceAll(str, "[", "")

	return str
}

// createRandomIntInRange provides a random integer between [min, max)
func (s *EIP712TestSuite) createRandomIntInRange(min int, max int) int {
	return rand.Intn(max-min) + min
}
