// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Uncomment this line to use console.log
// import "hardhat/console.sol";
import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestDirtyStateAttack2 {
    address erc20;

    constructor(address erc20_) {
        erc20 = erc20_;
    }

    function attack(
        address payable sendRecipient,
        string memory bech32Recipient
    ) public {
        require(
            ERC20(erc20).transfer(sendRecipient, 1e6), // 1 WNIBI
            "ERC-20 transfer failed"
        );

        (bool success, ) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "sendToBank(address,uint256,string)",
                erc20,
                uint256(9e6), // 9 WNIBI
                bech32Recipient
            )
        );

        require(success, string.concat("Failed to call sendToBank"));
    }
}
