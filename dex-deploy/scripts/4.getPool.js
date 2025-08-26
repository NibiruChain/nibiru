// scripts/getPool.js
const { ethers } = require("hardhat");
const fs = require("fs");
const path = require("path");

// Load deployment addresses from state.json
const statePath = path.join(__dirname, "../external/deploy-v3/state.json");
const state = JSON.parse(fs.readFileSync(statePath, "utf8"));

async function main() {
    const [signer] = await ethers.getSigners();
    // 🔧 replace with your deployed factory address
    const FACTORY_ADDRESS = state.v3CoreFactoryAddress;

    // 🔧 replace with your tokens + fee
    const TOKEN0 = "0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729";
    const TOKEN1 = "0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F";
    const FEE = 3000; // 0.3%

    if (!FACTORY_ADDRESS) {
        throw new Error("v3CoreFactoryAddress missing in dex-deploy/state.json");
    }
    if (
        !(ethers.isAddress ? ethers.isAddress(TOKEN0) : ethers.utils.isAddress(TOKEN0)) ||
        !(ethers.isAddress ? ethers.isAddress(TOKEN1) : ethers.utils.isAddress(TOKEN1))
    ) {
        throw new Error("TOKEN0 or TOKEN1 is not a valid Ethereum address");
    }

    // Load artifacts
    const {
        abi: FACTORY_ABI,
        bytecode: FACTORY_BYTECODE
    } = require("@uniswap/v3-core/artifacts/contracts/UniswapV3Factory.sol/UniswapV3Factory.json");

    const provider = ethers.provider;
    const factory = new ethers.Contract(FACTORY_ADDRESS, FACTORY_ABI, provider);

    const poolAddress = await factory.getPool(TOKEN0, TOKEN1, FEE);

    const ZERO_ADDRESS = (ethers.ZeroAddress) ? ethers.ZeroAddress : ((ethers.constants && ethers.constants.AddressZero) || "0x0000000000000000000000000000000000000000");
    if (poolAddress === ZERO_ADDRESS) {
        console.log("❌ Pool does not exist yet");
        return;
    }

    console.log("✅ Pool exists at:", poolAddress);

    // ABI for the pool
    const POOL_ABI = [
        "function slot0() external view returns (uint160 sqrtPriceX96,int24 tick,uint16 observationIndex,uint16 observationCardinality,uint16 observationCardinalityNext,uint8 feeProtocol,bool unlocked)",
        "function liquidity() external view returns (uint128)",
        "function token0() external view returns (address)",
        "function token1() external view returns (address)",
        "function fee() external view returns (uint24)"
    ];

    const poolAddr = poolAddress;
    const pool = new ethers.Contract(poolAddr, POOL_ABI, signer);

    const token0 = await pool.token0();
    const token1 = await pool.token1();
    const fee = await pool.fee();
    const liquidity = await pool.liquidity();
    const slot0 = await pool.slot0();

    console.log("Pool address:", poolAddr);
    console.log("Token0:", token0);
    console.log("Token1:", token1);
    console.log("Fee tier:", fee.toString());
    console.log("Liquidity:", liquidity.toString());
    console.log("slot0:", slot0);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
