import { zeroPadValue } from "ethers"

export function bytes32FromUint(u: Uint8Array): string {
  if (u.length > 32) throw new Error("too long")
  return zeroPadValue(u, 32)
}
