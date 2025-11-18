import { decodeFirst } from "cbor-web"

export interface PasskeyRegistration {
  credentialId: Uint8Array
  qx: Uint8Array
  qy: Uint8Array
  raw: PublicKeyCredential
}

// Registers a WebAuthn passkey and extracts P-256 pubkey coords.
export async function registerPasskey(opts: {
  rpId: string
  rpName?: string
  userId?: ArrayBuffer
  userName?: string
  userDisplayName?: string
}): Promise<PasskeyRegistration> {
  const {
    rpId,
    rpName = "Nibiru Passkey",
    userId = crypto.getRandomValues(new Uint8Array(32)).buffer,
    userName = "nibiru-user",
    userDisplayName = "Nibiru User",
  } = opts

  const publicKey: PublicKeyCredentialCreationOptions = {
    rp: { id: rpId, name: rpName },
    user: {
      id: userId, // ArrayBuffer per WebAuthn spec
      name: userName,
      displayName: userDisplayName,
    },
    challenge: crypto.getRandomValues(new Uint8Array(32)),
    pubKeyCredParams: [{ alg: -7, type: "public-key" }], // ES256 / P-256
    authenticatorSelection: { userVerification: "required" },
  }

  const cred = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential
  if (!cred) throw new Error("passkey registration failed")

  const attObj = decodeFirst((cred.response as AuthenticatorAttestationResponse).attestationObject) as any
  const authData: ArrayBuffer = attObj.authData
  const parsed = parseAuthData(new Uint8Array(authData))
  if (!parsed?.credentialPublicKey) throw new Error("no credentialPublicKey found")

  const { x, y } = extractP256(parsed.credentialPublicKey)
  return {
    credentialId: new Uint8Array(cred.rawId),
    qx: x,
    qy: y,
    raw: cred,
  }
}

function parseAuthData(buf: Uint8Array) {
  // Minimal authData parser: rpIdHash (32) | flags (1) | counter (4) | attestedCredData...
  if (buf.length < 37) return null
  const flags = buf[32]
  const hasAttestedCredData = (flags & 0x40) !== 0
  if (!hasAttestedCredData) return null

  let offset = 37 // skip rpIdHash + flags + counter
  const aaguid = buf.slice(offset, offset + 16)
  offset += 16
  const credIdLen = (buf[offset] << 8) + buf[offset + 1]
  offset += 2
  const credId = buf.slice(offset, offset + credIdLen)
  offset += credIdLen
  const credentialPublicKey = decodeFirst(buf.slice(offset)) as Map<number, any>
  return { aaguid, credId, credentialPublicKey }
}

function extractP256(pubKey: Map<number, any>): { x: Uint8Array; y: Uint8Array } {
  // COSE keys: 1=kty(EC2), -1=crv(1=P-256), -2=x, -3=y
  const crv = pubKey.get(-1)
  if (crv !== 1) throw new Error("unexpected curve (want P-256)")
  const x = pubKey.get(-2)
  const y = pubKey.get(-3)
  if (!(x instanceof Uint8Array) || !(y instanceof Uint8Array)) throw new Error("invalid pubkey coords")
  return { x, y }
}
