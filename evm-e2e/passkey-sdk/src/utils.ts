export function bytes32FromUint(u: Uint8Array): string {
  if (u.length > 32) throw new Error("too long")
  const hex = Array.from(u)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
    .padStart(64, "0")
  return `0x${hex}`
}
