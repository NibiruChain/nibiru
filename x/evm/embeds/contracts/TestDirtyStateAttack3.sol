// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Uncomment this line to use console.log
// import "hardhat/console.sol";
import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestDirtyStateAttack3 {
    address erc20;
    uint counter = 0;

    constructor(address erc20_) payable {
        erc20 = erc20_;
    }

    function attack(
        address payable sendRecipient,
        string memory bech32Recipient
    ) external {
        counter++;
        try
            TestDirtyStateAttack3(payable(address(this))).transferFunds(
                sendRecipient,
                bech32Recipient
            )
        {} catch
        {
            counter++;
        }
    }

    function transferFunds(
        address payable sendRecipient,
        string memory bech32Recipient
    ) external {
        require(sendRecipient.send(10 ether), "ETH transfer failed"); // 10 NIBI

        (bool success, ) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "sendToBank(address,uint256,string)",
                erc20,
                uint256(10e6), // 10 WNIBI
                bech32Recipient
            )
        );

        require(success, string.concat("Failed to call bankSend"));

        revert();
    }
}
