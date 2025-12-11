/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_RPC_URL?: string
  readonly VITE_CHAIN_ID?: string
  readonly VITE_ENTRYPOINT?: string
  readonly VITE_PASSKEY_FACTORY?: string
  readonly VITE_DEFAULT_FROM?: string
  readonly VITE_SAMPLE_RECIPIENT?: string
  readonly VITE_EVM_INTERFACE?: string
  readonly VITE_COLLATERAL?: string
  readonly VITE_VAULT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
