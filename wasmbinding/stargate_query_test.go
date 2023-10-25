package wasmbinding_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/NibiruChain/nibiru/wasmbinding"

	"github.com/NibiruChain/nibiru/x/common/set"

	devgas "github.com/NibiruChain/nibiru/x/devgas/v1/types"
	epochs "github.com/NibiruChain/nibiru/x/epochs/types"
	inflation "github.com/NibiruChain/nibiru/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/x/oracle/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
	tokenfactory "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

/*
TestWasmAcceptedStargateQueries: Verifies that the query paths registered in
the Wasm keeper's StargateQuerier are the official method names in the gRPC
query service of each path's respective module.

> ℹ️  "All stargate query paths must be actual GRPC query service methods"

Please see the function doc comment for WasmAcceptedStargateQueries in
stargate_query.go to understand in detail what invariants this test checks
for.

Given only the `PB_MSG.PACKAGE` and the `PB_MSG.NAME` of either the query
request or response, we should know the `QueryRequest::Stargate.path`
deterministically.
*/
func TestWasmAcceptedStargateQueries(t *testing.T) {
	t.Log("stargateQueryPaths: Add nibiru query paths from GRPC service descriptions")
	queryServiceDescriptions := []grpc.ServiceDesc{
		epochs.GrpcQueryServiceDesc(),
		devgas.GrpcQueryServiceDesc(),
		inflation.GrpcQueryServiceDesc(),
		oracle.GrpcQueryServiceDesc(),
		sudotypes.GrpcQueryServiceDesc(),
		tokenfactory.GrpcQueryServiceDesc(),
	}
	stargateQueryPaths := set.New[string]()
	for _, serviceDesc := range queryServiceDescriptions {
		for _, queryMethod := range serviceDesc.Methods {
			stargateQueryPaths.Add(
				fmt.Sprintf("/%v/%v", serviceDesc.ServiceName, queryMethod.MethodName),
			)
		}
	}

	t.Log("stargateQueryPaths: Add cosmos and ibc query paths")
	// The GRPC service descriptions aren't exported as copies from the
	// Cosmos-SDK and remain private vars. Maybe we could ask the maintainers to
	// export them in the future.
	for queryPath := range wasmbinding.WasmAcceptedStargateQueries() {
		stargateQueryPaths.Add(queryPath)
	}

	// It's not required for the response type and the method description of the
	// stargate query's gRPC path to match up exactly as expected. The exception
	// to this convention is when our response type isn't stripped of its
	// "Response" suffix and "Query" prefix is not the same as the method name.
	// This happens when "QueryAAARequest" does not return a "QueryAAAResponse".
	exceptionPaths := set.New[string]("/nibiru.oracle.v1.QueryExchangeRateResponse")

	gotQueryPaths := []string{}
	for queryPath, protobufResponse := range wasmbinding.WasmAcceptedStargateQueries() {
		gotQueryPaths = append(gotQueryPaths, queryPath)

		// Show that the underlying protobuf name and query paths coincide.
		pbQueryResponseTypeUrl := "/" + proto.MessageName(protobufResponse)
		isExceptionPath := exceptionPaths.Has(pbQueryResponseTypeUrl)
		splitResponse := strings.Split(pbQueryResponseTypeUrl, "Response")
		assert.Lenf(t, splitResponse, 2, "typeUrl: %v",
			splitResponse, pbQueryResponseTypeUrl)

		// Get proto message "package" from the response type
		typeUrlMinusSuffix := splitResponse[0]
		typeUrlPartsFromProtoMsg := strings.Split(typeUrlMinusSuffix, ".")
		lenOfParts := len(typeUrlPartsFromProtoMsg)
		assert.GreaterOrEqual(t, lenOfParts, 4, typeUrlPartsFromProtoMsg)
		protoMessagePackage := typeUrlPartsFromProtoMsg[:lenOfParts-1]

		// Get proto message "package" from the query path
		typeUrlPartsFromQueryPath := strings.Split(queryPath, ".")
		assert.GreaterOrEqual(t, len(typeUrlPartsFromQueryPath), 4, typeUrlPartsFromQueryPath)
		queryPathProtoPackage := typeUrlPartsFromQueryPath[:lenOfParts-1]

		// Verify that the packages match
		assert.Equalf(t, queryPathProtoPackage, protoMessagePackage,
			"package names inconsistent:\nfrom query path: %v\nfrom protobuf object: %v",
			queryPath, pbQueryResponseTypeUrl,
		)

		// Verify that the method names match too.
		if isExceptionPath {
			continue
		}
		methodNameFromPb := strings.TrimLeft(typeUrlPartsFromProtoMsg[3], "Query")
		methodNameFromPath := strings.TrimLeft(typeUrlPartsFromQueryPath[3], "Query/")
		assert.Equalf(t, methodNameFromPb, methodNameFromPath,
			"method names inconsistent:\nfrom query path: %v\nfrom protobuf object: %v",
			queryPath, pbQueryResponseTypeUrl,
		)
	}

	t.Log("All stargate query paths must be actual GRPC query service methods")
	assert.ElementsMatch(t, stargateQueryPaths.ToSlice(), gotQueryPaths)
}
