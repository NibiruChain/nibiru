# Events

| Event Type     | Attribute Key   | Attribute Value                              | Attribute Type |
|----------------|-----------------|----------------------------------------------|----------------|
| pool_joined    | sender          | sender's address                             | string         |
| pool_joined    | pool_id         | the numeric pool identifier                  | uint64         |
| pool_joined    | tokens_in       | the tokens sent by the user                  | sdk.Coins      |
| pool_joined    | pool_shares_out | the number of LP tokens returned to the user | sdk.Coin       |
| pool_joined    | rem_coins       | the tokens remaining after joining the pool  | sdk.Coins      |
| pool_created   | sender          | sender's address                             | string         |
| pool_created   | pool_id         | pool identifier                              | uint64         |
| pool_exited    | sender          | sender's address                             | string         |
| pool_exited    | pool_id         | pool identifier                              | uint64         |
| pool_exited    | num_shares_in   | number of LP tokens in                       | sdk.Coin       |
| pool_exited    | tokens_out      | tokens returned to the user                  | sdk.Coins      |
| assets_swapped | sender          | sender's address                             | string         |
| assets_swapped | pool_id         | pool identifier                              | uint64         |
| assets_swapped | token_in        | token to swap in                             | sdk.Coin       |
| assets_swapped | token_out       | token returned to user                       | sdk.Coin       |
