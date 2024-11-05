// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./IFunToken.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract TestFunTokenPrecompileLocalGas {
    address erc20;

    constructor(address erc20_) {
        erc20 = erc20_;
    }

    // Calls bankSend of the FunToken Precompile with the default gas.
    // Internal call could use all the gas for the parent call.
    function callBankSend(
        uint256 amount,
        string memory bech32Recipient
    ) public {
        uint256 sentAmount = FUNTOKEN_PRECOMPILE.bankSend(
            erc20,
                amount,
            bech32Recipient
        );
        require(
            sentAmount == amount,
            string.concat(
                "IFunToken.bankSend succeeded but transferred the wrong amount",
                "sentAmount ",
                Strings.toString(sentAmount),
                "expected ",
                Strings.toString(amount)
            )
        );
    }

    // Calls bankSend of the FunToken Precompile with the gas amount set in parameter.
    // Internal call should fail if the gas provided is insufficient.
    function callBankSendLocalGas(
        uint256 amount,
        string memory bech32Recipient,
        uint256 customGas
    ) public {
        uint256 sentAmount = FUNTOKEN_PRECOMPILE.bankSend{gas: customGas}(
            erc20,
            amount,
            bech32Recipient
        );
        require(
            sentAmount == amount,
            string.concat(
                "IFunToken.bankSend succeeded but transferred the wrong amount",
                "sentAmount ",
                Strings.toString(sentAmount),
                "expected ",
                Strings.toString(amount)
            )
        );
    }
}
