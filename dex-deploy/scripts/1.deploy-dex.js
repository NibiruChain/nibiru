const { spawn } = require("child_process");
const { ethers } = require("hardhat");
const { HDNodeWallet, Mnemonic } = require("ethers");
const fs = require("fs");
const path = require("path");

async function main() {
    // Path to the file you want to delete
    const filePath = path.join(__dirname, "../external/deploy-v3/state.json");

    // Delete file if it exists
    try {
        if (fs.existsSync(filePath)) {
            fs.unlinkSync(filePath);
            console.log("Deleted old state.json");
        } else {
            console.log("No previous state.json found, skipping deletion");
        }
    } catch (err) {
        console.error("Error deleting state.json:", err);
    }
    const [deployer] = await ethers.getSigners();

    pk = getPrivateKeyFromMnemonic(deployer._accounts.mnemonic, 0);
    if (!pk) {
        throw new Error(
            "Signer does not expose privateKey. Use HardhatConfig accounts or a Wallet instance with a pk."
        );
    }

    const paddedPk = pk.privateKey.startsWith("0x")
        ? pk.privateKey
        : "0x" + pk.privateKey;

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