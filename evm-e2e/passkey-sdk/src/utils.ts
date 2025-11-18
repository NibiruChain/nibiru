export function bytes32FromUint(u: Uint8Array): string {
  if (u.length > 32) throw new Error("too long")
  const hex = Buffer.from(u).toString("hex").padStart(64, "0")
  return "0x" + hex
}
