[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / SpotExtension

# Interface: SpotExtension

## Table of contents

### Properties

- [spot](SpotExtension.md#spot)

## Properties

### spot

• **spot**: `Readonly`<{ `estimateExitExactAmountIn`: (`poolId`: `number`, `poolSharesIn`: `number`) => `Promise`<`QueryExitExactAmountInResponse`\> ; `estimateExitExactAmountOut`: (`poolId`: `number`) => `Promise`<`QueryExitExactAmountOutResponse`\> ; `estimateJoinExactAmountIn`: (`poolId`: `number`, `tokensIn`: `Coin`[]) => `Promise`<`QueryJoinExactAmountInResponse`\> ; `estimateJoinExactAmountOut`: (`poolId`: `number`) => `Promise`<`QueryJoinExactAmountOutResponse`\> ; `estimateSwapExactAmountIn`: (`poolId`: `number`, `tokenOutDenom`: `string`, `tokenIn?`: `Coin`) => `Promise`<`QuerySwapExactAmountInResponse`\> ; `estimateSwapExactAmountOut`: (`poolId`: `number`, `tokenInDenom`: `string`, `tokenOut?`: `Coin`) => `Promise`<`QuerySwapExactAmountOutResponse`\> ; `numPools`: () => `Promise`<`QueryNumPoolsResponse`\> ; `params`: () => `Promise`<`QueryParamsResponse`\> ; `pool`: (`poolId`: `number`) => `Promise`<`QueryPoolResponse`\> ; `poolNumber`: () => `Promise`<`QueryPoolNumberResponse`\> ; `poolParams`: (`poolId`: `number`) => `Promise`<`QueryPoolParamsResponse`\> ; `pools`: (`pagination?`: [`PageRequest`](PageRequest.md)) => `Promise`<`QueryPoolsResponse`\> ; `spotPrice`: (`poolId`: `number`, `tokenInDenom`: `string`, `tokenOutDenom`: `string`) => `Promise`<`QuerySpotPriceResponse`\> ; `totalLiquidity`: () => `Promise`<`QueryTotalLiquidityResponse`\> ; `totalPoolLiquidity`: (`poolId`: `number`) => `Promise`<`QueryTotalPoolLiquidityResponse`\> ; `totalShares`: (`poolId`: `number`) => `Promise`<`QueryTotalSharesResponse`\> }\>

#### Defined in

[query/spot.ts:58](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L58)
