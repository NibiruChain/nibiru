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

        uint256 sentAmount = FUNTOKEN_PRECOMPILE.sendToBank(
            erc20,
            precompileAmount,
            precompileRecipient
        );
        require(
            sentAmount == precompileAmount,
            string.concat(
                "IFunToken.sendToBank succeeded but transferred the wrong amount",
                "sentAmount ",
                Strings.toString(sentAmount),
                "expected ",
                Strings.toString(precompileAmount)
            )
        );
    }
}
