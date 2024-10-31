// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

// Uncomment this line to use console.log
// import "hardhat/console.sol";
import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract TestERC20TransferThenPrecompileSend {
    address erc20;

    constructor(address erc20_) {
        erc20 = erc20_;
    }

    function erc20TransferThenPrecompileSend(
        address payable transferRecipient,
        uint256 transferAmount,
        string memory precompileRecipient,
        uint256 precompileAmount
    ) public {
        require(
            ERC20(erc20).transfer(transferRecipient, transferAmount),
            "ERC-20 transfer failed"
        );

        (bool success, ) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "bankSend(address,uint256,string)",
                erc20,
                uint256(precompileAmount),
                    precompileRecipient
            )
        );

        require(success, string.concat("Failed to call bankSend"));
    }
}
