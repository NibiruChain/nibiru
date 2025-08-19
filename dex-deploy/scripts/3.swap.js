const { ethers } = require("hardhat");
const fs = require("fs");
const path = require("path");

// Load deployment addresses from state.json
const statePath = path.join(__dirname, "../state.json");
const state = JSON.parse(fs.readFileSync(statePath, "utf8"));

// Load artifacts
const {
    abi: SWAP_ROUTER_ABI,
    bytecode: SWAP_ROUTER_BYTECODE,
} = require("@uniswap/swap-router-contracts/artifacts/contracts/SwapRouter02.sol/SwapRouter02.json");

async function main() {
    const [signer] = await ethers.getSigners();

    const router = new ethers.Contract(
        state.swapRouter02,
        SWAP_ROUTER_ABI,
        signer
    );

    const WNIBI = "0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729";
    const USDC = "0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F";

    const ERC20_ABI = [
        "function approve(address spender, uint256 amount) external returns (bool)"
    ];
    var amountIn = ethers.parseUnits("1", 3); // swap 1 token
    const WNIBIContract = new ethers.Contract(WNIBI, ERC20_ABI, signer);
    await (await WNIBIContract.approve(router.target, amountIn)).wait();
    console.log("Approved Router to spend WNIBI");

    const params = {
        tokenIn: WNIBI,
        tokenOut: USDC,
        fee: 3000,
        recipient: await signer.getAddress(),
        deadline: Math.floor(Date.now() / 1000) + 60 * 10,
        amountIn: amountIn, // 1 WNIBI
        amountOutMinimum: 0,
        sqrtPriceLimitX96: 0
    };

    const tx = await router.exactInputSingle(params, {
        gasLimit: 500000
    });
    const receipt = await tx.wait();
    console.log("Swap Tx:", receipt.hash);
}

main().catch(console.error);
