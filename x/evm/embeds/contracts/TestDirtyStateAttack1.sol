// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Uncomment this line to use console.log
// import "hardhat/console.sol";
import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract TestDirtyStateAttack1 {
    address erc20;

    constructor(address erc20_) {
        erc20 = erc20_;
    }

    function attack(
        address payable sendRecipient,
        string memory bech32Recipient
    ) public {
        bool isSent = sendRecipient.send(10 ether); // 10 NIBI
        require(isSent, "Failed to send ether");

        (bool success, ) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "sendToBank(address,uint256,string)",
                erc20,
                uint256(10e6), // 10 WNIBI
                bech32Recipient
            )
        );

        require(success, string.concat("Failed to call bankSend"));
    }
}
