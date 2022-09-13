package mock

import (
	"testing"

	sdktestsmocks "github.com/cosmos/cosmos-sdk/tests/mocks"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gomock "github.com/golang/mock/gomock"
)

/*
	AppendCtxWithMockLogger sets the logger for an input context as a mock logger

with 'EXPECT' statements. This enables testing on functions logged to the context.
For example,

```go
// This is a passing test example
import (

	gomock "github.com/golang/mock/gomock"
	sdktestsmocks "github.com/cosmos/cosmos-sdk/tests/mocks"

)

	// assume t is a *testing.T variable.
	ctx, logger := AppendCtxWithMockLogger(t, ctx)
	logger.EXPECT().Debug("debug")
	logger.EXPECT().Info("info")
	logger.EXPECT().Error("error")

	ctx.Logger().Debug("debug")
	ctx.Logger().Info("info")
	ctx.Logger().Error("error")

```
*/
func AppendCtxWithMockLogger(t *testing.T, ctx sdk.Context) (sdk.Context, *sdktestsmocks.MockLogger) {
	ctrl := gomock.NewController(t)
	logger := sdktestsmocks.NewMockLogger(ctrl)
	return ctx.WithLogger(logger), logger
}
