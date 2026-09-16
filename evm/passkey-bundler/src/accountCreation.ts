export const DEFAULT_ACCOUNT_CREATION_GAS_LIMIT = 1_500_000n
export const MAX_ACCOUNT_CREATION_GAS_LIMIT = 5_000_000n

export function parseAccountCreationGasLimit(input: unknown): bigint {
  if (input === undefined || input === null) {
    return DEFAULT_ACCOUNT_CREATION_GAS_LIMIT
  }
  if (typeof input !== "string" || !/^0x[0-9a-fA-F]+$/.test(input)) {
    throw new Error("passkey_createAccount gas limit must be a hex quantity")
  }

  const gasLimit = BigInt(input)
  if (gasLimit <= 0n || gasLimit > MAX_ACCOUNT_CREATION_GAS_LIMIT) {
    throw new Error(
      `passkey_createAccount gas limit must be between 1 and ${MAX_ACCOUNT_CREATION_GAS_LIMIT}`,
    )
  }
  return gasLimit
}
