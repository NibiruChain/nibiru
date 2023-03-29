# `epochs`

Often in the SDK, we would like to run certain code every-so often. The purpose of `epochs` module is to allow other modules to set that they would like to be signaled once every period. So another module can specify it wants to execute code once a week, starting at UTC-time = x. `epochs` creates a generalized epoch interface to other modules so that they can easily be signalled upon such events.

#### Contents 

- [`epochs`](#epochs)
- [Concepts](#concepts)
- [State](#state)
    - [Epoch information type](#epoch-information-type)
- [Events](#events)
  - [BeginBlocker](#beginblocker)
  - [EndBlocker](#endblocker)
- [Keepers](#keepers)
  - [Keeper functions](#keeper-functions)
- [Hooks](#hooks)
  - [Hooks](#hooks-1)
  - [How modules receive hooks](#how-modules-receive-hooks)
- [Queries](#queries)
- [Future Improvements](#future-improvements)
  - [Lack point using this module](#lack-point-using-this-module)
  - [Block-time drifts problem](#block-time-drifts-problem)

# Concepts

The purpose of `epochs` module is to provide generalized epoch interface to other modules so that they can easily implement epochs without keeping own code for epochs.

# State

Epochs module keeps `EpochInfo` objects and modify the information as epochs info changes.
Epochs are initialized as part of genesis initialization, and modified on begin blockers or end blockers.

### Epoch information type

```protobuf
message EpochInfo {
    string identifier = 1;
    google.protobuf.Timestamp start_time = 2 [
        (gogoproto.stdtime) = true,
        (gogoproto.nullable) = false,
        (gogoproto.moretags) = "yaml:\"start_time\""
    ];
    google.protobuf.Duration duration = 3 [
        (gogoproto.nullable) = false,
        (gogoproto.stdduration) = true,
        (gogoproto.jsontag) = "duration,omitempty",
        (gogoproto.moretags) = "yaml:\"duration\""
    ];
    int64 current_epoch = 4;
    google.protobuf.Timestamp current_epoch_start_time = 5 [
        (gogoproto.stdtime) = true,
        (gogoproto.nullable) = false,
        (gogoproto.moretags) = "yaml:\"current_epoch_start_time\""
    ];
    bool epoch_counting_started = 6;
    reserved 7;
    int64 current_epoch_start_height = 8;
}
```

EpochInfo keeps `identifier`, `start_time`,`duration`, `current_epoch`, `current_epoch_start_time`,  `epoch_counting_started`, `current_epoch_start_height`.

1. `identifier` keeps epoch identification string.
2. `start_time` keeps epoch counting start time, if block time passes `start_time`, `epoch_counting_started` is set.
3. `duration` keeps target epoch duration.
4. `current_epoch` keeps current active epoch number.
5. `current_epoch_start_time` keeps the start time of current epoch.
6. `epoch_number` is counted only when `epoch_counting_started` flag is set.
7. `current_epoch_start_height` keeps the start block height of current epoch.

# Events

The `epochs` module emits the following events:

## BeginBlocker

| Type        | Attribute Key | Attribute Value |
| ----------- | ------------- | --------------- |
| epoch_start | epoch_number  | {epoch_number}  |
| epoch_start | start_time    | {start_time}    |

## EndBlocker

| Type        | Attribute Key | Attribute Value |
| ----------- | ------------- | --------------- |
| epoch_end   | epoch_number  | {epoch_number}  |

# Keepers

## Keeper functions

Epochs keeper module provides utility functions to manage epochs.

```go
// Keeper is the interface for epoch module keeper
type Keeper interface {
  // GetEpochInfo returns epoch info by identifier
  GetEpochInfo(ctx sdk.Context, identifier string) types.EpochInfo
  // SetEpochInfo set epoch info
  SetEpochInfo(ctx sdk.Context, epoch types.EpochInfo) 
  // DeleteEpochInfo delete epoch info
  DeleteEpochInfo(ctx sdk.Context, identifier string)
  // IterateEpochInfo iterate through epochs
  IterateEpochInfo(ctx sdk.Context, fn func(index int64, epochInfo types.EpochInfo) (stop bool))
  // Get all epoch infos
  AllEpochInfos(ctx sdk.Context) []types.EpochInfo
}
```

# Hooks

## Hooks
```go
  // the first block whose timestamp is after the duration is counted as the end of the epoch
  AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64)
  // new epoch is next block of epoch end block
  BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64)
```

## How modules receive hooks

On hook receiver function of other modules, they need to filter `epochIdentifier` and only do executions for only specific epochIdentifier.
Filtering epochIdentifier could be in `Params` of other modules so that they can be modified by governance.
Governance can change epoch from `week` to `day` as their need.

# Queries

Epochs module is providing below queries to check the module's state.

```protobuf
service Query {
  // EpochInfos provide running epochInfos
  rpc EpochInfos(QueryEpochsInfoRequest) returns (QueryEpochsInfoResponse) {}
  // CurrentEpoch provide current epoch of specified identifier
  rpc CurrentEpoch(QueryCurrentEpochRequest) returns (QueryCurrentEpochResponse) {}
}
```

# Future Improvements

## Lack point using this module

In current design each epoch should be at least 2 blocks as start block should be different from endblock.
Because of this, each epoch time will be `max(blocks_time x 2, epoch_duration)`.
If epoch_duration is set to `1s`, and `block_time` is `5s`, actual epoch time should be `10s`.
We definitely recommend configure epoch_duration as more than 2x block_time, to use this module correctly.
If you enforce to set it to 1s, it's same as 10s - could make module logic invalid.

TODO for postlaunch: We should see if we can architect things such that the receiver doesn't have to do this filtering, and the epochs module would pre-filter for them.

## Block-time drifts problem

This implementation has block time drift based on block time.
For instance, we have an epoch of 100 units that ends at t=100, if we have a block at t=97 and a block at t=104 and t=110, this epoch ends at t=104.
And new epoch start at t=110. There are time drifts here, for around 1-2 blocks time.
It will slow down epochs.

It's going to slow down epoch by 10-20s per week when epoch duration is 1 week. This should be resolved after launch.