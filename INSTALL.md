# Install matrixd


## Update the system
This guide will explain how to install the osmosisd binary onto your system.

On Ubuntu, start by updating your system

```bash
sudo apt update
sudo apt upgrade --yes
```

## Install build requirements

Install make and gcc.

```bash
sudo apt install git build-essential ufw curl jq snapd --yes
wget -q -O - https://git.io/vQhTU | bash -s -- --version 1.16
```

After installed, open new terminal to properly load go


## Clone the Matrix Repository

```
cd $HOME
git clone https://github.com/MatrixDAO/matrix
cd matrix
git checkout v0.0.1
make install
```

## Other recommended steps

- Increase number of open files limit
- Set your firewall rules

## Upgrade

The scheduled mainnet upgrade to `matrix-2` is planned for 

```
cd matrix
git fetch tags
git checkout v0.0.1
make install
```

 Testnet

One the Matrix binary has been installed, for further information on joining the testnet, head over to the [testnet repo](https://github.com/MatrixDao/Networks/tree/main/Testnet).

 Mainnet

One the Matrix binary has been installed, for further information on joining mainnet, head over to the [mainnet repo](https://github.com/MatrixDao/Networks/tree/main/Mainnet).
