require("@nomicfoundation/hardhat-toolbox");
const path = require("path");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: {
    compilers: [{ version: "0.4.19" }, { version: "0.8.24" }],
  },
  resolver: {
    alias: {
      "@nibiruchain/solidity": path.resolve(__dirname), // root of your package
    },
  },
};
