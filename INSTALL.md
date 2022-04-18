# Install `nibid` binaries          <!-- omit in toc -->

This guide will explain how to install the Nibiru Chain binary, `nibid`, onto your system.

#### Table of Contents
- [1. Update the system](#1-update-the-system)
- [2. Install Golang](#2-install-golang)
- [3. Install build requirements](#3-install-build-requirements)
- [4. Clone the Nibiru Repository](#4-clone-the-nibiru-repository)
- [Upgrade](#upgrade)


## 1. Update the system

On Ubuntu, start by updating your system

```bash
sudo apt update
sudo apt upgrade --yes
```

## 2. Install Golang 

Steps described here: https://go.dev/doc/install

## 3. Install build requirements

Install make and gcc.

```bash
sudo apt install git build-essential ufw curl jq snapd --yes
wget -q -O - https://git.io/vQhTU | bash -s -- --version 1.16
```

After installed, open new terminal to properly load go


## 4. Clone the Nibiru Repository

```sh
cd $HOME
git clone https://github.com/NibiruChain/nibiru
cd nibiru
```

On this fresh clone of the repo, simply run 
```sh
make build 
make install
make localnet
```
and open another terminal.  

---

## Upgrade

The scheduled mainnet upgrade to `nibiru-2` is planned for 

```
cd nibiru
git fetch tags
git checkout v0.0.1
```


 Testnet

One the Nibiru binary has been installed, for further information on joining the testnet, head over to the [testnet repo](https://github.com/NibiruChain/Networks/tree/main/Testnet).

 Mainnet

One the Nibiru binary has been installed, for further information on joining mainnet, head over to the [mainnet repo](https://github.com/NibiruChain/Networks/tree/main/Mainnet).
