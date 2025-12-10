import { decode as cborDecode } from 'cbor-x'
import { p256 } from '@noble/curves/p256'
import type { PasskeyAssertion, PasskeyPublicKey, PasskeyRegistration } from '../types'
import { fromBase64Url, toBase64Url, toHex } from '../lib/base64'

const P256_COSE_X = -2
const P256_COSE_Y = -3

function hexToBytes(hex: string): Uint8Array {
  const normalized = hex.startsWith('0x') ? hex.slice(2) : hex
  const bytes = new Uint8Array(normalized.length / 2)
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(normalized.substr(i * 2, 2), 16)
  }
  return bytes
}

function parseAttestationPublicKey(attestationObjectB64: string): PasskeyPublicKey {
  const attestation = fromBase64Url(attestationObjectB64)
  const decoded = cborDecode(attestation) as { authData: Uint8Array }
  const authData = new Uint8Array(decoded.authData)

  const credIdLen = (authData[53] << 8) + authData[54]
  const credentialIdEnd = 55 + credIdLen
  const coseKeyBytes = authData.slice(credentialIdEnd)
  const coseKey = cborDecode(coseKeyBytes) as Map<number, Uint8Array | number>

  const x = coseKey.get(P256_COSE_X)
  const y = coseKey.get(P256_COSE_Y)

  if (!(x instanceof Uint8Array) || !(y instanceof Uint8Array)) {
    throw new Error('Unable to extract P-256 public key from attestation')
  }

  return {
    x: toHex(new Uint8Array(x)),
    y: toHex(new Uint8Array(y)),
    alg: typeof coseKey.get(3) === 'number' ? (coseKey.get(3) as number) : undefined
  }
}

async function parsePublicKeyFromSpki(
  response: AuthenticatorAttestationResponse
): Promise<PasskeyPublicKey | null> {
  const spki = response.getPublicKey?.()
  if (!spki) return null

  try {
    const key = await crypto.subtle.importKey(
      'spki',
      spki,
      { name: 'ECDSA', namedCurve: 'P-256' },
      true,
      ['verify']
    )
    const jwk = (await crypto.subtle.exportKey('jwk', key)) as JsonWebKey
    if (typeof jwk.x !== 'string' || typeof jwk.y !== 'string') {
      return null
    }
    return {
      x: toHex(fromBase64Url(jwk.x)),
      y: toHex(fromBase64Url(jwk.y)),
      alg: -7 // ES256
    }
  } catch (err) {
    console.error('Failed to parse SPKI public key', err)
    return null
  }
}

export async function registerPasskey(label: string): Promise<PasskeyRegistration> {
  if (!('PublicKeyCredential' in window)) {
    throw new Error('Passkeys/WebAuthn not supported in this browser')
  }

  const challenge = crypto.getRandomValues(new Uint8Array(32))
  const userId = crypto.getRandomValues(new Uint8Array(32))

  const credential = (await navigator.credentials.create({
    publicKey: {
      challenge,
      rp: { name: 'SAI Passkey Demo' },
      user: {
        id: userId,
        name: label || 'passkey-user',
        displayName: label || 'passkey-user'
      },
      pubKeyCredParams: [{ type: 'public-key', alg: -7 }],
      timeout: 60_000,
      authenticatorSelection: {
        residentKey: 'preferred',
        userVerification: 'preferred'
      }
    }
  })) as PublicKeyCredential | null

  if (!credential) {
    throw new Error('Passkey creation was cancelled or failed')
  }

  const response = credential.response as AuthenticatorAttestationResponse
  const attestationObject = toBase64Url(response.attestationObject)
  const clientDataJSON = toBase64Url(response.clientDataJSON)

  let publicKey: PasskeyPublicKey | null = null
  try {
    publicKey = parseAttestationPublicKey(attestationObject)
  } catch (err) {
    console.warn('Failed to parse attestationObject; trying SPKI fallback', err)
    publicKey = await parsePublicKeyFromSpki(response)
  }
  if (!publicKey) {
    throw new Error('Unable to extract P-256 public key from attestation')
  }

  return {
    label,
    credentialId: credential.id,
    publicKey,
    attestationObject,
    clientDataJSON
  }
}

export async function signChallengeWithPasskey(params: {
  credentialId: string
  challengeHex: `0x${string}`
}): Promise<PasskeyAssertion> {
  const { credentialId, challengeHex } = params
  const credentialIdBytes = fromBase64Url(credentialId)
  const credentialIdBuffer = credentialIdBytes.buffer.slice(0) as ArrayBuffer
  const challengeBytes = hexToBytes(challengeHex)
  const challengeBuffer = challengeBytes.buffer.slice(0) as ArrayBuffer

  const allowCredentials = [
    {
      type: 'public-key',
      id: credentialIdBuffer
    }
  ] satisfies PublicKeyCredentialDescriptor[]

  const assertion = (await navigator.credentials.get({
    publicKey: {
      challenge: challengeBuffer,
      timeout: 60_000,
      userVerification: 'preferred',
      allowCredentials
    }
  })) as PublicKeyCredential | null

  if (!assertion) {
    throw new Error('Passkey assertion was cancelled or failed')
  }

  const response = assertion.response as AuthenticatorAssertionResponse
  const authData = new Uint8Array(response.authenticatorData)
  const clientDataJSON = new Uint8Array(response.clientDataJSON)
  const signatureBytes = new Uint8Array(response.signature)

  const sig = p256.Signature.fromDER(signatureBytes)

  return {
    credentialId,
    challengeHex,
    authenticatorData: toBase64Url(authData),
    clientDataJSON: toBase64Url(clientDataJSON),
    signature: toBase64Url(signatureBytes),
    r: `0x${sig.r.toString(16).padStart(64, '0')}`,
    s: `0x${sig.s.toString(16).padStart(64, '0')}`
  }
}
