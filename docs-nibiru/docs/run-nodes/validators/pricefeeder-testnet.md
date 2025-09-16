---
order: 3
---

# Set up a Pricefeeder (Testnet)

Instructions for validators to set up a Testnet pricefeeder. {synopsis}

1. Install the pricefeeder binary

    ```bash
    curl -s https://get.nibiru.fi/pricefeeder@v1.0.2! | bash
    ```

2. Setup the systemd service

    See below for an explanation for the environment variables.

    ```bash
    export CHAIN_ID="nibiru-testnet-1"
    export GRPC_ENDPOINT="localhost:9090"
    export WEBSOCKET_ENDPOINT="ws://localhost:26657/websocket"
    export FEEDER_MNEMONIC="<your feeder mnemonic here>"
    export VALIDATOR_ADDRESS="nibi1valoper..."
    ```

    ```bash
    sudo tee /etc/systemd/system/pricefeeder.service<<EOF
    [Unit]
    Description=Nibiru Pricefeeder
    Requires=network-online.target
    After=network-online.target

    [Service]
    Type=exec
    User=<your user>
    Group=<your group>
    ExecStart=/usr/local/bin/pricefeeder
    Restart=on-failure
    ExecReload=/bin/kill -HUP $MAINPID
    KillSignal=SIGTERM
    PermissionsStartOnly=true
    LimitNOFILE=65535
    Environment=CHAIN_ID='$CHAIN_ID'
    Environment=GRPC_ENDPOINT='$GRPC_ENDPOINT'
    Environment=WEBSOCKET_ENDPOINT='$WEBSOCKET_ENDPOINT'
    Environment=EXCHANGE_SYMBOLS_MAP='$EXCHANGE_SYMBOLS_MAP'
    Environment=FEEDER_MNEMONIC='$FEEDER_MNEMONIC'

    [Install]
    WantedBy=multi-user.target
    EOF
    ```

    ```bash
    sudo systemctl daemon-reload && \
    sudo systemctl enable pricefeeder && \
    sudo systemctl start pricefeeder
    ```

    ```bash
    # view pricefeeder logs
    journalctl -fu pricefeeder
    ```

## Environment Variables

| Name                 | Description                                                                                                   | Example                                                                                                                                                  |
| -------------------- | ------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CHAIN_ID             | the chain id                                                                                                  | nibiru-testnet-1                                                                                                                                             |
| GRPC_ENDPOINT        | the endpoint to fetch query params send txs                                                                   | **localhost:9090** if running locally **lcd.testnet-1.nibiru.fi:443** if running remotely                                                                   |
| WEBSOCKET_ENDPOINT   | the endpoint to fetch streaming updates                                                                       | **ws://localhost:26657/websocket** if running locally **wss://lcd.testnet-1.nibiru.fi/websocket** if running remotely                                        |
| FEEDER_MNEMONIC      | the account that sends price votes                                                                            | guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host |
| VALIDATOR_ADDRESS    | (OPTIONAL) the validator to send price votes for, not needed if the FEEDER_MNEMONIC is the validator mnemonic | nibivaloper1zaavvzxez0elundtn32qnk9lkm8kmcszuwx9jz                                                                                                       |

## How to delegate pricefeeder responsibility

As a validator, if you'd like another account to post prices on your behalf (i.e. you don't want your validator mnemonic sending txs), you can delegate pricefeeder responsibilities to another nibi address.

```bash
nibid tx oracle set-feeder <nibi address> --from validator
```

Now that address can start sending votes on behalf of your validator. Just make sure that the FEEDER_MNEMONIC corresponds to the delegated address and the VALIDATOR_ADDRESS is set.
