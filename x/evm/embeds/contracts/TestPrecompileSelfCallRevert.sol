// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.24;

import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract TestPrecompileSelfCallRevert {
    address erc20;
    uint counter = 0;

    constructor(address erc20_) payable {
        erc20 = erc20_;
    }

    function selfCallTransferFunds(
        address payable nativeRecipient,
        uint256 nativeAmount,
        string memory precompileRecipient,
        uint256 precompileAmount
    ) external {
        counter++;
        try
            TestPrecompileSelfCallRevert(payable(address(this))).transferFunds(
                nativeRecipient,
                nativeAmount,
                precompileRecipient,
                precompileAmount
            )
        {} catch // [1]
        {
            counter++;
        }
    }

    function transferFunds(
        address payable nativeRecipient,
        uint256 nativeAmount,
        string memory precompileRecipient,
        uint256 precompileAmount
    ) external {
        require(nativeRecipient.send(nativeAmount), "ETH transfer failed"); // wei

        uint256 sentAmount = FUNTOKEN_PRECOMPILE.sendToBank(
            erc20,
            precompileAmount, // micro-WNIBI
            precompileRecipient
        );

        require(
            sentAmount == precompileAmount,
            string.concat(
                "IFunToken.sendToBank succeeded but transferred the wrong amount",
                "sentAmount ",
                Strings.toString(nativeAmount),
                "expected ",
                Strings.toString(precompileAmount)
            )
        );

        revert(); // [4]
    }
}
