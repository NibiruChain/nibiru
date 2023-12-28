# Simulation Tests

This directory contains the simulation tests for the `simapp` module

## Test Cases

### Non-Determinism

```sh
make test-sim-non-determinism
```

This test case checks that the simulation is deterministic. It does so by
running the simulation twice with the same seed and comparing the resulting
state. If the simulation is deterministic, the resulting state should be the
same.

### Full App

```sh
make test-sim-default-genesis-fast
```

This test case runs the simulation with the default genesis file. It checks that
the simulation does not panic and that the resulting state is valid.

### Import/Export

```sh
make test-sim-import-export
```

This test case runs the simulation with the default genesis file. It checks that
the simulation does not panic and that the resulting state is valid. It then
exports the state to a file and imports it back. It checks that the imported
state is the same as the exported state.

### Simulation After Import

```sh
make test-sim-after-import
```

This test case runs the simulation with the default genesis file. It checks that
the simulation does not panic and that the resulting state is valid. It then
exports the state to a file and imports it back. It checks that the imported
state is the same as the exported state. It then runs the simulation again with
the imported state. It checks that the simulation does not panic and that the
resulting state is valid.

## Params

A `params.json` file is included that sets the operation weights for
`CreateValidator` and `EditValidator` to zero. It's a hack to make the
simulation tests pass. The random commission rates sometimes halt the simulation
because the max commission rate is set to 0.25 in the AnteHandler, but sometimes
the random commission rate is higher than that. The random commission is set by
the cosmos-sdk x/staking module simulation operations, so we have no control
over injecting a manual value.
