import { useEffect, useMemo, useState } from 'react'
import { encodeAbiParameters, padHex, toHex } from 'viem'
import type { AppConfig, BundlerLogEntry, PasskeyRegistration, ServiceHealth, UnsignedTxRequest } from './types'
import { registerPasskey, signChallengeWithPasskey } from './passkey/webauthn'
import { buildDepositTx } from './lib/sai'
import { makePublicClient } from './lib/passkeyClient'
import { clearPasskey, loadPasskey, savePasskey } from './lib/storage'
import { derivePasskeyAddress } from './lib/passkeyAddress'
import { encodeExecute, PASSKEY_ACCOUNT_ABI } from './lib/passkeyAccount'
import { createPasskeyAccount, fetchBundlerLogs, sendUserOperation } from './lib/bundler'
import { defaultUserOp, getUserOpHash, toRpcUserOperation } from './lib/userop'
import { fromBase64Url } from './lib/base64'
import { getBundlerHealth, getRpcHealth } from './lib/health'
import { Sidebar } from './components/Sidebar'
import { Step } from './components/Step'

const ZERO_ADDRESS = '0x0000000000000000000000000000000000000000'
const envRpc = import.meta.env.VITE_RPC_URL as string | undefined
const envChainId = import.meta.env.VITE_CHAIN_ID as string | undefined
const envFactory = import.meta.env.VITE_PASSKEY_FACTORY as `0x${string}` | undefined
const envEntryPoint = import.meta.env.VITE_ENTRYPOINT as `0x${string}` | undefined
const envDefaultFrom = import.meta.env.VITE_DEFAULT_FROM as `0x${string}` | undefined
const envSampleRecipient = import.meta.env.VITE_SAMPLE_RECIPIENT as `0x${string}` | undefined
const envEvmInterface = import.meta.env.VITE_EVM_INTERFACE as `0x${string}` | undefined
const envCollateral = import.meta.env.VITE_COLLATERAL as `0x${string}` | undefined
const envVault = import.meta.env.VITE_VAULT as string | undefined
const envBundler = import.meta.env.VITE_BUNDLER_URL as string | undefined
const envBundlerSigner = import.meta.env.VITE_BUNDLER_SIGNER as `0x${string}` | undefined
const DEFAULT_BUNDLER_SIGNER = '0xC0f4b45712670cf7865A14816bE9Af9091EDdA1d'

const DEFAULT_CONFIG: AppConfig = {
  rpcUrl: envRpc || 'http://localhost:8545',
  bundlerUrl: envBundler || 'http://127.0.0.1:4337',
  chainId: envChainId ? Number(envChainId) : 9000,
  evmInterfaceAddress: envEvmInterface || ZERO_ADDRESS,
  bundlerSignerAddress: envBundlerSigner || DEFAULT_BUNDLER_SIGNER,
  passkeyFactoryAddress: envFactory || ZERO_ADDRESS,
  entryPointAddress: envEntryPoint || ZERO_ADDRESS,
  vaultAddress: envVault || '',
  collateralAddress: envCollateral || ZERO_ADDRESS,
  sampleRecipient: envSampleRecipient || ZERO_ADDRESS
}

function App() {
  const [config, setConfig] = useState<AppConfig>(DEFAULT_CONFIG)
  const [fromAddress, setFromAddress] = useState<`0x${string}`>(envDefaultFrom || ZERO_ADDRESS)
  const [passkey, setPasskey] = useState<PasskeyRegistration | null>(() => loadPasskey())
  const [status, setStatus] = useState<string>('Idle')
  const [deployStatus, setDeployStatus] = useState<string>('')
  const [balance, setBalance] = useState<string>('0')
  const [bundlerBalance, setBundlerBalance] = useState<string>('0')
  const [bundlerLogs, setBundlerLogs] = useState<BundlerLogEntry[]>([])
  const [isDeployed, setIsDeployed] = useState(false)
  const [serviceHealth, setServiceHealth] = useState<{ rpc: ServiceHealth; bundler: ServiceHealth }>({
    rpc: { status: 'checking' },
    bundler: { status: 'checking' }
  })
  const [opModal, setOpModal] = useState<{ open: boolean; title: string }>({ open: false, title: '' })

  const [depositForm, setDepositForm] = useState({
    depositAmount: '0',
    useErc20Amount: '0',
    wasmMsg: '{"deposit":{}}',
    sendSharesToEvm: true
  })
  const [customTx, setCustomTx] = useState({ to: ZERO_ADDRESS, data: '0x', value: '0' })
  const [sampleSend, setSampleSend] = useState<{ to: `0x${string}`; value: string }>({
    to: (envSampleRecipient || ZERO_ADDRESS) as `0x${string}`,
    value: '10000000000000000'
  })

  const client = useMemo(() => makePublicClient(config), [config])

  // Poll RPC and bundler availability for the state monitor
  useEffect(() => {
    let cancelled = false
    const expectedEntryPoint =
      config.entryPointAddress && config.entryPointAddress !== ZERO_ADDRESS ? config.entryPointAddress : undefined

    const checkHealth = async () => {
      setServiceHealth({ rpc: { status: 'checking' }, bundler: { status: 'checking' } })
      const [rpcStatus, bundlerStatus] = await Promise.all([
        getRpcHealth(config.rpcUrl, config.chainId),
        getBundlerHealth(config.bundlerUrl, expectedEntryPoint)
      ])
      if (!cancelled) {
        setServiceHealth({ rpc: rpcStatus, bundler: bundlerStatus })
      }
    }

    checkHealth()
    const id = setInterval(checkHealth, 8000)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [config.rpcUrl, config.chainId, config.bundlerUrl, config.entryPointAddress])

  // Poll bundler logs (best-effort)
  useEffect(() => {
    let cancelled = false
    if (!config.bundlerUrl) return

    const pollLogs = async () => {
      try {
        const logs = await fetchBundlerLogs({ bundlerUrl: config.bundlerUrl, limit: 50 })
        if (!cancelled) setBundlerLogs(logs)
      } catch {
        if (!cancelled) setBundlerLogs([])
      }
    }

    pollLogs()
    const id = setInterval(pollLogs, 5000)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [config.bundlerUrl])

  // Check deployment status
  useEffect(() => {
    if (!fromAddress || fromAddress === ZERO_ADDRESS) return
    let active = true
    const checkDeployment = async () => {
      try {
        const code = await client.getBytecode({ address: fromAddress })
        if (active) setIsDeployed(!!code)
      } catch {
        // ignore
      }
    }
    checkDeployment()
    const id = setInterval(checkDeployment, 5000)
    return () => {
      active = false
      clearInterval(id)
    }
  }, [fromAddress, client, deployStatus])

  useEffect(() => {
    if (!passkey) {
      const stored = loadPasskey()
      if (stored) {
        setPasskey(stored)
        if (!fromAddress || fromAddress === ZERO_ADDRESS) {
          setFromAddress(derivePasskeyAddress(stored.publicKey))
        }
      }
    } else if (!fromAddress || fromAddress === ZERO_ADDRESS) {
      setFromAddress(derivePasskeyAddress(passkey.publicKey))
    }
  }, [passkey, fromAddress])

  useEffect(() => {
    if (!fromAddress || fromAddress === ZERO_ADDRESS) return
    let active = true
    const fetchBalance = async () => {
      try {
        const bal = await client.getBalance({ address: fromAddress })
        if (active) setBalance(bal.toString())
      } catch {
        // ignore
      }
    }
    fetchBalance()
    const id = setInterval(fetchBalance, 3000)
    return () => {
      active = false
      clearInterval(id)
    }
  }, [client, fromAddress])

  // Bundler signer balance
  useEffect(() => {
    const bundlerAddr = config.bundlerSignerAddress
    if (!bundlerAddr || bundlerAddr === ZERO_ADDRESS) return
    let active = true
    const fetchBundlerBalance = async () => {
      try {
        const bal = await client.getBalance({ address: bundlerAddr })
        if (active) setBundlerBalance(bal.toString())
      } catch {
        // ignore
      }
    }
    fetchBundlerBalance()
    const id = setInterval(fetchBundlerBalance, 5000)
    return () => {
      active = false
      clearInterval(id)
    }
  }, [client, config.bundlerSignerAddress])

  const updateStatus = (message: string) => setStatus(message)
  const openOpModal = (title: string) => setOpModal({ open: true, title })
  const closeOpModal = () => setOpModal({ open: false, title: '' })

  const ensurePasskey = () => {
    if (passkey) return passkey
    const stored = loadPasskey()
    if (stored) {
      setPasskey(stored)
      if (!fromAddress || fromAddress === ZERO_ADDRESS) {
        setFromAddress(derivePasskeyAddress(stored.publicKey))
      }
      return stored
    }
    throw new Error('Create or load a passkey first')
  }

  const parseAmount = (value: string): bigint => {
    if (!value) return 0n
    if (value.startsWith('0x')) {
      return BigInt(value)
    }
    return BigInt(value)
  }

  const buildAndSendUserOp = async (tx: UnsignedTxRequest) => {
    updateStatus('Preparing UserOperation...')
    const pk = ensurePasskey()

    if (!fromAddress || fromAddress === ZERO_ADDRESS) {
      throw new Error('Set the passkey-controlled address (from) before sending')
    }
    if (!config.entryPointAddress || config.entryPointAddress === ZERO_ADDRESS) {
      throw new Error('EntryPoint address is not set')
    }
    if (!config.bundlerUrl) {
      throw new Error('Bundler URL is not set')
    }

    const callData = encodeExecute(tx.to as `0x${string}`, tx.value ?? 0n, (tx.data ?? '0x') as `0x${string}`)

    let nonce = 0n
    try {
      nonce =
        ((await client.readContract({
          address: fromAddress,
          abi: PASSKEY_ACCOUNT_ABI,
          functionName: 'nonce'
        })) as bigint) || 0n
    } catch {
      nonce = 0n
    }

    const gasPrice = await client.getGasPrice()
    const userOp = {
      ...defaultUserOp(fromAddress),
      nonce,
      callData,
      maxFeePerGas: gasPrice,
      maxPriorityFeePerGas: gasPrice
    }

    const chainId = BigInt(config.chainId)
    const userOpHash = getUserOpHash(userOp, config.entryPointAddress, chainId)
    updateStatus('Waiting for passkey signature...')

    const assertion = await signChallengeWithPasskey({
      credentialId: pk.credentialId,
      challengeHex: userOpHash
    })

    const authData = toHex(fromBase64Url(assertion.authenticatorData))
    const clientData = toHex(fromBase64Url(assertion.clientDataJSON))
    const signature = encodeAbiParameters(
      [
        { name: 'authData', type: 'bytes' },
        { name: 'clientDataJSON', type: 'bytes' },
        { name: 'r', type: 'bytes32' },
        { name: 's', type: 'bytes32' }
      ],
      [authData, clientData, assertion.r, assertion.s]
    )

    const rpcUserOp = toRpcUserOperation({ ...userOp, signature: signature as `0x${string}` })
    updateStatus('Sending UserOperation to bundler...')
    const bundlerHash = await sendUserOperation({
      bundlerUrl: config.bundlerUrl,
      userOp: rpcUserOp,
      entryPoint: config.entryPointAddress
    })
    updateStatus(`Submitted userOp: ${bundlerHash}`)
    return bundlerHash
  }

  const handleRegisterPasskey = async (label: string) => {
    try {
      updateStatus('Creating passkey...')
      const reg = await registerPasskey(label)
      setPasskey(reg)
      savePasskey(reg)
      setFromAddress(derivePasskeyAddress(reg.publicKey))
      updateStatus('Passkey registered and stored locally')
    } catch (err: any) {
      console.error(err)
      updateStatus(err?.message ?? 'Passkey creation failed')
    }
  }

  const handleDeploy = async () => {
    try {
      const pk = ensurePasskey()
      if (!config.passkeyFactoryAddress || config.passkeyFactoryAddress === ZERO_ADDRESS) {
        throw new Error('Passkey factory address is not set')
      }
      if (!config.bundlerUrl) {
        throw new Error('Bundler URL is not set')
      }

      setDeployStatus('Requesting bundler to create account...')
      const qx = padHex(pk.publicKey.x, { size: 32 })
      const qy = padHex(pk.publicKey.y, { size: 32 })
      const result = await createPasskeyAccount({
        bundlerUrl: config.bundlerUrl,
        factory: config.passkeyFactoryAddress,
        qx,
        qy
      })
      setFromAddress(result.account)
      setDeployStatus(`Account created at ${result.account} (tx ${result.txHash})`)
      updateStatus(`Account created at ${result.account}`)
      setIsDeployed(true)
    } catch (err: any) {
      console.error(err)
      setDeployStatus(err?.message ?? 'Deploy transaction failed')
    }
  }

  const handleDeposit = async () => {
    try {
      if (!config.vaultAddress || !config.collateralAddress || config.evmInterfaceAddress === ZERO_ADDRESS) {
        throw new Error('Fill vault address, collateral address, and EVM interface address')
      }

      const tx = buildDepositTx(config.evmInterfaceAddress, config.chainId, {
        wasmMsg: depositForm.wasmMsg,
        depositAmount: parseAmount(depositForm.depositAmount),
        useErc20Amount: parseAmount(depositForm.useErc20Amount),
        vaultAddress: config.vaultAddress,
        collateralAddress: config.collateralAddress,
        sendSharesToEvm: depositForm.sendSharesToEvm
      })

      await buildAndSendUserOp(tx)
    } catch (err: any) {
      console.error(err)
      updateStatus(err?.message ?? 'Deposit failed')
    }
  }

  const handleCustomTx = async () => {
    try {
      openOpModal('Custom transaction')
      updateStatus('Preparing UserOperation...')
      await buildAndSendUserOp({
        to: customTx.to as `0x${string}`,
        data: customTx.data as `0x${string}`,
        value: parseAmount(customTx.value)
      })
    } catch (err: any) {
      console.error(err)
      updateStatus(err?.message ?? 'Custom transaction failed')
      if (!opModal.open) openOpModal('Custom transaction failed')
    }
  }

  const handleSampleSend = async () => {
    try {
      openOpModal('Sending NIBI')
      updateStatus('Preparing UserOperation...')
      await buildAndSendUserOp({
        to: sampleSend.to as `0x${string}`,
        data: '0x',
        value: parseAmount(sampleSend.value)
      })
    } catch (err: any) {
      console.error(err)
      updateStatus(err?.message ?? 'Sample send failed')
      if (!opModal.open) openOpModal('Send failed')
    }
  }

  const resetPasskey = () => {
    clearPasskey()
    setPasskey(null)
    updateStatus('Cleared stored passkey')
  }

  return (
    <div className="app-layout">
      <Sidebar
        config={config}
        passkey={passkey}
        accountAddress={fromAddress}
        balance={balance}
        isDeployed={isDeployed}
        serviceHealth={serviceHealth}
        bundlerBalance={bundlerBalance}
        bundlerLogs={bundlerLogs}
      />

      <main className="main-content">
        <div className="header">
          <h1>SAI Passkey Demo</h1>
          <p>Learn how passkeys (WebAuthn) can drive EVM contracts without MetaMask.</p>
        </div>

        <Step
          number={1}
          title="Create Passkey"
          description={
            <>
              <p>
                A passkey is a cryptographic key pair generated by your device (TouchID, FaceID, YubiKey).
                The private key never leaves your device.
              </p>
              <p>
                We will use the public key to compute a deterministic "counterfactual" address for your smart contract account.
              </p>
            </>
          }
          isCompleted={!!passkey}
        >
          <div className="row">
            <button onClick={() => handleRegisterPasskey('sai-passkey')}>Create new passkey</button>
            <button className="secondary" onClick={() => setPasskey(loadPasskey() || null)}>
              Load from localStorage
            </button>
            {passkey && (
              <button className="secondary" onClick={resetPasskey}>
                Clear
              </button>
            )}
          </div>
          {status && <div className="status-message">{status}</div>}
        </Step>

        <Step
          number={2}
          title="Deploy Account"
          description={
            <>
              <p>
                Your account address is already known ({fromAddress.slice(0, 10)}...), but the contract is not yet deployed on-chain.
              </p>
              <p>
                We need to send a transaction to the <strong>PasskeyFactory</strong> to deploy the smart contract wallet that verifies your passkey signatures.
              </p>
            </>
          }
          isActive={!!passkey}
          isCompleted={isDeployed}
        >
          {isDeployed ? (
            <div className="status-message">Account is deployed!</div>
          ) : (
            <>
              <button onClick={handleDeploy} disabled={!passkey}>Deploy Account</button>
              {deployStatus && <div className="status-message">{deployStatus}</div>}
            </>
          )}
        </Step>

        <Step
          number={3}
          title="Funding (Optional)"
          description={
            <>
              <p>
                Bundler fee sponsorship is disabled.
              </p>
              <p>
                If this account needs native tokens for non-gasless flows, fund it directly from a wallet or CLI.
              </p>
            </>
          }
          isActive={isDeployed}
          isCompleted={BigInt(balance) > 0}
        >
          <div style={{ marginTop: '12px', fontSize: '14px', color: '#64748b' }}>
            <p>No bundler funding RPC is available.</p>
            <p>Fund manually if needed:</p>
            <code style={{ background: '#f1f5f9', padding: '4px 8px', borderRadius: '4px', wordBreak: 'break-all' }}>
              {fromAddress}
            </code>
          </div>
        </Step>

        <Step
          number={4}
          title="Interact"
          description={
            <>
              <p>
                Now you can send transactions! The flow is:
              </p>
              <ol style={{ fontSize: '14px', color: '#475569', paddingLeft: '20px' }}>
                <li>Build a UserOperation (UserOp)</li>
                <li>Sign the UserOp hash with your Passkey</li>
                <li>Send the signed UserOp to the Bundler</li>
                <li>Bundler submits it to the EntryPoint contract</li>
              </ol>
            </>
          }
          isActive={isDeployed}
        >
          <div className="grid">
            <div style={{ border: '1px solid #e2e8f0', padding: '16px', borderRadius: '8px' }}>
              <h4>Send NIBI</h4>
              <div className="grid" style={{ marginBottom: '12px' }}>
                <label>
                  Recipient
                  <input
                    value={sampleSend.to}
                    onChange={(e) => setSampleSend({ ...sampleSend, to: e.target.value as `0x${string}` })}
                  />
                </label>
                <label>
                  Amount (wei)
                  <input
                    value={sampleSend.value}
                    onChange={(e) => setSampleSend({ ...sampleSend, value: e.target.value })}
                  />
                </label>
              </div>
              <button onClick={handleSampleSend}>Send</button>
            </div>

            <div style={{ border: '1px solid #e2e8f0', padding: '16px', borderRadius: '8px' }}>
              <h4>Custom Transaction</h4>
              <div className="grid" style={{ marginBottom: '12px' }}>
                <label>
                  To
                  <input value={customTx.to} onChange={(e) => setCustomTx({ ...customTx, to: e.target.value })} />
                </label>
                <label>
                  Data
                  <input
                    value={customTx.data}
                    onChange={(e) => setCustomTx({ ...customTx, data: e.target.value })}
                  />
                </label>
              </div>
              <button onClick={handleCustomTx}>Send</button>
            </div>
          </div>
        </Step>
      </main>

      {opModal.open && (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <div className="modal-card">
            <div className="modal-header">
              <h3>{opModal.title}</h3>
              <button className="icon-button" onClick={closeOpModal} aria-label="Close">
                ✕
              </button>
            </div>
            <div className="modal-body">
              <p className="modal-status">{status}</p>
              <p style={{ color: '#475569', fontSize: '13px' }}>
                Keep this open while the passkey prompt and bundler submission complete.
              </p>
            </div>
            <div className="modal-actions">
              <button className="secondary" onClick={closeOpModal}>
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default App
