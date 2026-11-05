// SPDX-License-Identifier: MIT
pragma solidity >=0.8.19;

import "./IFunToken.sol";
import "./Wasm.sol";

interface ITestDelegateCallback {
    function onSendToBankCallback(
        address erc20,
        uint256 amount,
        string calldata to
    ) external returns (uint256);
}

/// @notice Test-only router that forwards the original caller identity into
/// mutable Nibiru precompiles using DELEGATECALL, matching SaiEvm's call shape.
contract TestDelegatePrecompile {
    function sendToBank(
        address erc20,
        uint256 amount,
        string calldata to
    ) external returns (uint256) {
        return
            abi.decode(
                _delegate(
                    FUNTOKEN_PRECOMPILE_ADDRESS,
                    abi.encodeCall(IFunToken.sendToBank, (erc20, amount, to))
                ),
                (uint256)
            );
    }

    function sendToEvm(
        string calldata bankDenom,
        uint256 amount,
        string calldata to
    ) external returns (uint256) {
        return
            abi.decode(
                _delegate(
                    FUNTOKEN_PRECOMPILE_ADDRESS,
                    abi.encodeCall(
                        IFunToken.sendToEvm,
                        (bankDenom, amount, to)
                    )
                ),
                (uint256)
            );
    }

    function wasmExecute(
        string calldata contractAddr,
        bytes calldata msgArgs,
        INibiruEvm.BankCoin[] calldata funds
    ) external returns (bytes memory) {
        return
            abi.decode(
                _delegate(
                    WASM_PRECOMPILE_ADDRESS,
                    abi.encodeCall(IWasm.execute, (contractAddr, msgArgs, funds))
                ),
                (bytes)
            );
    }

    function bankMsgSend(
        string calldata to,
        string calldata bankDenom,
        uint256 amount
    ) external returns (bool) {
        return
            abi.decode(
                _delegate(
                    FUNTOKEN_PRECOMPILE_ADDRESS,
                    abi.encodeCall(
                        IFunToken.bankMsgSend,
                        (to, bankDenom, amount)
                    )
                ),
                (bool)
            );
    }

    /// @notice Models EOA -> router -> pool -> router callback -> precompile.
    /// The callback's DELEGATECALL must spend this router's funds, not the
    /// intermediate caller's funds.
    function sendToBankThroughCallback(
        address callbackCaller,
        address erc20,
        uint256 amount,
        string calldata to
    ) external returns (uint256) {
        return
            TestCallbackCaller(callbackCaller).callSendToBank(
                address(this),
                erc20,
                amount,
                to
            );
    }

    function onSendToBankCallback(
        address erc20,
        uint256 amount,
        string calldata to
    ) external returns (uint256) {
        return
            abi.decode(
                _delegate(
                    FUNTOKEN_PRECOMPILE_ADDRESS,
                    abi.encodeCall(IFunToken.sendToBank, (erc20, amount, to))
                ),
                (uint256)
            );
    }

    function _delegate(
        address target,
        bytes memory input
    ) private returns (bytes memory output) {
        (bool ok, bytes memory ret) = target.delegatecall(input);
        if (!ok) {
            assembly {
                revert(add(ret, 32), mload(ret))
            }
        }
        return ret;
    }
}

/// @notice Test stand-in for a pool that calls the router back during a swap.
contract TestCallbackCaller {
    function callSendToBank(
        address callback,
        address erc20,
        uint256 amount,
        string calldata to
    ) external returns (uint256) {
        return
            ITestDelegateCallback(callback).onSendToBankCallback(
                erc20,
                amount,
                to
            );
    }
}

/// @notice Minimal transparent-proxy call shape for delegated-sender tests.
contract TestDelegateProxy {
    address private immutable implementation;

    constructor(address implementation_) {
        implementation = implementation_;
    }

    fallback() external payable {
        address target = implementation;
        assembly {
            calldatacopy(0, 0, calldatasize())
            let result := delegatecall(gas(), target, 0, calldatasize(), 0, 0)
            returndatacopy(0, 0, returndatasize())
            switch result
            case 0 {
                revert(0, returndatasize())
            }
            default {
                return(0, returndatasize())
            }
        }
    }
}
