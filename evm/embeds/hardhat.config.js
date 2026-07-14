require("@nomicfoundation/hardhat-toolbox");
const path = require("path");

/** @type import('hardhat/config').HardhatUserConfig
 *
 * Find that here: node_modules/hardhat/src/types/config.ts
 * ```
 * export interface HardhatUserConfig {
 *   defaultNetwork?: string;
 *   paths?: ProjectPathsUserConfig;
 *   networks?: NetworksUserConfig;
 *   solidity?: SolidityUserConfig;
 *   mocha?: Mocha.MochaOptions;
 * }
 * ```
 *
 * */
module.exports = {
  solidity: {
    compilers: [
      { version: "0.4.19" },
      {
        version: "0.8.24",
        settings: {
          optimizer: {
            enabled: true,
            runs: 5,
          },
          viaIR: true,
        },
      },
    ],
  },
  resolver: {
    alias: {
      "@nibiruchain/solidity": path.resolve(__dirname), // root of your package
    },
  },
};
