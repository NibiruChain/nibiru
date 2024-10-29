// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract TestNativeSendThenPrecompileSend {
    address erc20;

    constructor(address erc20_) {
        erc20 = erc20_;
    }

    function nativeSendThenPrecompileSend(
        address payable nativeRecipient,
        uint256 nativeAmount,
        string memory precompileRecipient,
        uint256 precompileAmount
    ) public {
        bool isSent = nativeRecipient.send(nativeAmount);
        require(isSent, "Failed to send native token");

        (bool success, ) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "bankSend(address,uint256,string)",
                erc20,
                precompileAmount,
                precompileRecipient
            )
        );

        require(success, string.concat("Failed to call precompile bankSend"));
    }
}
