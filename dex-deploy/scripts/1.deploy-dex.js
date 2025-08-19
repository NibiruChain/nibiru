const { spawn } = require("child_process");
const { ethers } = require("hardhat");
const { HDNodeWallet, Mnemonic } = require("ethers");
const fs = require("fs");
const path = require("path");

async function main() {
    const [deployer] = await ethers.getSigners();

    pk = getPrivateKeyFromMnemonic(deployer._accounts.mnemonic, 0);
    if (!pk) {
        throw new Error(
            "Signer does not expose privateKey. Use HardhatConfig accounts or a Wallet instance with a pk."
        );
    }

    paddedPk = pk.privateKey.startsWith("0x")
        ? pk.privateKey
        : "0x" + pk.privateKey;

    console.log("Using private key:", paddedPk);
    const args = [
        "-pk", paddedPk,
        "-j", "http://localhost:8545",
        "-w9", "0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729",
        "-ncl", "WNIBI",
        "-o", "0xC0f4b45712670cf7865A14816bE9Af9091EDdA1d"
    ];

    const child = spawn("yarn", ["start", ...args], {
        cwd: "external/deploy-v3",
        stdio: "inherit",
        env: {
            ...process.env,
            NODE_OPTIONS: "--openssl-legacy-provider",
        },
    });

    child.on("exit", (code) => {
        // Copy state.json after deployment
        const src = path.join("external", "deploy-v3", "state.json");
        const dest = path.join(process.cwd(), "state.json");
        try {
            fs.copyFileSync(src, dest);
            console.log("state.json copied to:", dest);
        } catch (err) {
            console.error("Failed to copy state.json:", err);
        }
        process.exit(code);
    });
}

function getPrivateKeyFromMnemonic(mnemonicPhrase, index = 0) {
    const path = `m/44'/60'/0'/0/0`;

    // First wrap the phrase into a Mnemonic object
    const mnemonic = Mnemonic.fromPhrase(mnemonicPhrase);
    // Create HD wallet root from mnemonic
    const wallet = HDNodeWallet.fromMnemonic(mnemonic, path);
    return {
        privateKey: wallet.privateKey,
    };
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});