BINARY="./nibid"

# validator addr
VALIDATOR_ADDR=$($BINARY keys show validator --address)
echo "Validator address:"
echo "$VALIDATOR_ADDR"


