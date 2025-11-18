// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import { UserOperation } from "../UserOperation.sol";

interface IEntryPoint {
    function balanceOf(address account) external view returns (uint256);

    function handleOps(UserOperation[] calldata ops, address payable beneficiary) external;
}
