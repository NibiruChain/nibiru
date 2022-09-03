rm -rf /root/.nibid
nibid init oracle --chain-id testing
# add well known key
echo "sand inch devote knee subway basket torch tilt extra test allow way embark dial renew cinnamon away boil document bitter wear badge license subway" | nibid keys add fd --keyring-backend=test --recover

addr=$(nibid keys show fd --keyring-backend=test -a)
val_addr=$(nibid keys show fd  --keyring-backend=test --bech val -a)

nibid add-genesis-account $addr 1000000000000unibi --keyring-backend=test
nibid add-genesis-account fd 10000000unibi --keyring-backend=test
nibid gentx fd 10000000unibi --keyring-backend=test --chain-id testing
nibid collect-gentxs
nibid validate-genesis
sed -i 's/127.0.0.1/0.0.0.0/g' /root/.nibid/config/config.toml
nibid start