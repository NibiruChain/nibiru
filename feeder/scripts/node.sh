nibid add-genesis-account nibi1vm2px4f8z0w9urg89xucmrnrdzm49zlwsl5pf0 1000000000000unibi --keyring-backend=te
nibid add-genesis-account fd 10000000unibi --keyring-backend=test
nibid gentx fd 10000000unibi --keyring-backend=test --chain-id oracle-testing
nibid collect-gentxs
nibid validate-genesis