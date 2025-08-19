const hre = require("hardhat");
const { ethers } = hre;
const fs = require("fs");
const path = require("path");

// Load deployment addresses from state.json
const statePath = path.join(__dirname, "../state.json");
const state = JSON.parse(fs.readFileSync(statePath, "utf8"));

// Load ABIs from @uniswap/v3-periphery
const npmAbi = require("@uniswap/v3-periphery/artifacts/contracts/NonfungiblePositionManager.sol/NonfungiblePositionManager.json").abi;
const erc20Abi = require("@openzeppelin/contracts/build/contracts/ERC20.json").abi;

async function main() {
    const [deployer] = await ethers.getSigners();

    // Addresses (from your deployment)
    const NPM_ADDRESS = state.nonfungibleTokenPositionManagerAddress; // NonfungiblePositionManager
    const WNIBI = "0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729";
    const USDC = "0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F";
    const FEE = 3000; // 0.3% fee tier

    const npm = new ethers.Contract(NPM_ADDRESS, npmAbi, deployer);

    // -------- Step 1: Create + Initialize Pool --------
    let [token0, token1] = [WNIBI, USDC];
    if (token0.toLowerCase() > token1.toLowerCase()) {
        [token0, token1] = [token1, token0];
    }

    // ---------------------
    // Step 2: Initial Price
    // ---------------------
    const decimals0 = 18; // WNIBI
    const decimals1 = 18;  // USDC

    let reserve0, reserve1;

    if (token0 === WNIBI) {
        // token0 = WNIBI, token1 = USDC → price = 1/100
        reserve0 = 10n ** BigInt(decimals0);       // 1 WNIBI
        reserve1 = (10n ** BigInt(decimals1)) / 100n; // 0.01 USDC
    } else {
        // token0 = USDC, token1 = WNIBI → price = 100
        reserve0 = 10n ** BigInt(decimals1);       // 1 USDC
        reserve1 = (10n ** BigInt(decimals0)) * 100n; // 100 WNIBI
    }

    const sqrtPriceX96 = encodePriceSqrt(reserve1, reserve0).toString();
    console.log("sqrtPriceX96:", sqrtPriceX96);

    const tx = await npm.createAndInitializePoolIfNecessary(
        token0,
        token1,
        FEE,
        sqrtPriceX96
    );
    const receipt = await tx.wait();
    console.log("Pool created/initialized:", receipt.hash);

    // -------- Step 2: Approve tokens for adding liquidity --------
    const wnibi = new ethers.Contract(WNIBI, erc20Abi, deployer);
    const usdc = new ethers.Contract(USDC, erc20Abi, deployer);

    const amountWNIBI = ethers.parseUnits("1000", 18);
    const amountUSDC = ethers.parseUnits("10", 18); //
    await (await wnibi.approve(NPM_ADDRESS, amountWNIBI)).wait();
    await (await usdc.approve(NPM_ADDRESS, amountUSDC)).wait();

    // -------- Step 3: Add Liquidity (mint a position) --------
    const mintParams = {
        token0,
        token1,
        fee: FEE,
        tickLower: -887220, // full range
        tickUpper: 887220,
        amount0Desired: amountWNIBI,
        amount1Desired: amountUSDC,
        amount0Min: 0,
        amount1Min: 0,
        recipient: deployer.address,
        deadline: Math.floor(Date.now() / 1000) + 60 * 10, // 10 min from now
    };

    const mintTx = await npm.mint(mintParams);
    const mintReceipt = await mintTx.wait();
    console.log("Liquidity added, tx:", mintReceipt.hash);
}

function encodePriceSqrt(reserve1, reserve0) {
    // reserve1 = token1 units, reserve0 = token0 units
    const numerator = BigInt(reserve1) << (BigInt(96) * 2n); // * 2^192
    const ratioX192 = numerator / BigInt(reserve0);

    // Integer square root (Babylonian method)
    function sqrt(value) {
        if (value < 0n) throw new Error("negative sqrt");
        if (value < 2n) return value;
        let x0 = value / 2n;
        let x1 = (x0 + value / x0) / 2n;
        while (x1 < x0) {
            x0 = x1;
            x1 = (x0 + value / x0) / 2n;
        }
        return x0;
    }

    return sqrt(ratioX192);
}

main().catch((error) => {
    console.error(error);
    process.exit(1);
});
