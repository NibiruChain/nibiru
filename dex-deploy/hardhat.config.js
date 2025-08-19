require("@nomicfoundation/hardhat-toolbox");
require("dotenv").config(); // load .env

const mnemonic = process.env.MNEMONIC || "";

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    compilers: [
      { version: "0.7.6" }, // For Uniswap V3
      { version: "0.8.20" } // If you have newer contracts
    ]
  },
  networks: {
    localEVM: {
      url: "http://127.0.0.1:8545",
      accounts: {
        mnemonic,
      }
    }
  }
};
