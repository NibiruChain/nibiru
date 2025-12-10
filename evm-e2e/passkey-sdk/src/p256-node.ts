import { p256 } from "@noble/curves/p256"
import { getBytes, hexlify, sha256 } from "ethers"

export interface NodePasskey {
  privKey: Uint8Array
  pubQx: Uint8Array
  pubQy: Uint8Array
}

export function generateNodePasskey(seed?: Uint8Array): NodePasskey {
  const priv = seed ?? p256.utils.randomPrivateKey()
  if (!p256.utils.isValidPrivateKey(priv)) {
    throw new Error("invalid P-256 private key")
  }

  const pub = p256.getPublicKey(priv, false) // uncompressed: 0x04 || x || y
  if (pub.length !== 65 || pub[0] !== 0x04) {
    throw new Error("unexpected public key format")
  }

  const pubQx = pub.slice(1, 33)
  const pubQy = pub.slice(33, 65)
  return {
    privKey: new Uint8Array(priv),
    pubQx: new Uint8Array(pubQx),
    pubQy: new Uint8Array(pubQy),
  }
}

export function signUserOpHash(userOpHash: string, privKey: Uint8Array): { r: string; s: string } {
  if (!p256.utils.isValidPrivateKey(privKey)) {
    throw new Error("invalid private key")
  }
  const digestHex = sha256(userOpHash)
  const digest = getBytes(digestHex)
  if (digest.length !== 32) throw new Error("userOpHash must be 32 bytes")

  const sig = p256.sign(digest, privKey)
  const compact = sig.toCompactRawBytes() // r||s
  if (compact.length !== 64) throw new Error("invalid signature length")
  return {
    r: hexlify(compact.slice(0, 32)),
    s: hexlify(compact.slice(32)),
  }
}
