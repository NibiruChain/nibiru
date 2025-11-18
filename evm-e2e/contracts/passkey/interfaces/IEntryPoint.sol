// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {UserOperation} from "../UserOperation.sol";

interface IEntryPoint {
    function handleOps(UserOperation[] calldata ops, address payable beneficiary) external;
}
