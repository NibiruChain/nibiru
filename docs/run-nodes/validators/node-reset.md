---
order: 6
description: Instructions for validators to rebuild in the case of a Testnet chain reset.
---

# Reset a Validator Node (Testnet)

{{ $frontmatter.description }}

Any upcoming resets will be announced in the `#testnet` channel on [Nibiru's Discord server](https://discord.com/invite/HFvbn7Wtud).
To reset your node and rejoin the testnet, please follow the steps below:

## Remove the old chain data and binary

```bash
sudo rm -rf $HOME/.nibid
sudo rm $HOME/go/bin/nibid
```

## Recreate the validator

Follow the same steps from ["Run a Full Node (Testnet)"](../full-nodes/index.md) and ["Become a Validator (Testnet)"](./index.md) again.
