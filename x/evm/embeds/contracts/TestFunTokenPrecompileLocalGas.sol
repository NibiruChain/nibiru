// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./FunToken.sol";

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
        (bool success,) = FUNTOKEN_PRECOMPILE_ADDRESS.call(
            abi.encodeWithSignature(
                "bankSend(address,uint256,string)",
                erc20,
                amount,
                bech32Recipient
            )
        );
        require(success, "Failed to call bankSend");
    }

    // Calls bankSend of the FunToken Precompile with the gas amount set in parameter.
    // Internal call should fail if the gas provided is insufficient.
    function callBankSendLocalGas(
        uint256 amount,
        string memory bech32Recipient,
        uint256 customGas
    ) public {
        (bool success,) = FUNTOKEN_PRECOMPILE_ADDRESS.call{gas: customGas}(
            abi.encodeWithSignature(
                "bankSend(address,uint256,string)",
                erc20,
                amount,
                bech32Recipient
            )
        );
        require(success, "Failed to call bankSend with custom gas");
    }
}