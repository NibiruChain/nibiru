export interface AppConfig {
  rpcUrl: string
  chainId: number
  evmInterfaceAddress: `0x${string}`
  bundlerUrl: string
  bundlerSignerAddress?: `0x${string}`
  passkeyFactoryAddress?: `0x${string}`
  entryPointAddress?: `0x${string}`
  vaultAddress?: string
  collateralAddress?: `0x${string}`
  sampleRecipient?: `0x${string}`
  passkeyRpcMethod?: string
}

export interface PasskeyPublicKey {
  x: `0x${string}`
  y: `0x${string}`
  alg?: number
}

export interface PasskeyRegistration {
  label: string
  credentialId: string // base64url
  publicKey: PasskeyPublicKey
  attestationObject: string // base64url
  clientDataJSON: string // base64url
}

export interface PasskeyAssertion {
  credentialId: string
  challengeHex: `0x${string}`
  authenticatorData: string // base64url
  clientDataJSON: string // base64url
  signature: string // base64url (DER)
  r: `0x${string}`
  s: `0x${string}`
}

export interface UnsignedTxRequest {
  to: `0x${string}`
  data: `0x${string}`
  value?: bigint
  gas?: bigint
  gasPrice?: bigint
  nonce?: number
  chainId?: number
}

export interface PasskeySignedTxPayload {
  unsignedTx: UnsignedTxRequest & { nonce: number; gas: bigint; gasPrice: bigint; chainId: number }
  passkey: PasskeyRegistration
  assertion: PasskeyAssertion
}

export type ServiceStatus = 'checking' | 'ok' | 'down'

export interface ServiceHealth {
  status: ServiceStatus
  detail?: string
}

export interface BundlerLogEntry {
  ts: number
  level: string
  message: string
}
