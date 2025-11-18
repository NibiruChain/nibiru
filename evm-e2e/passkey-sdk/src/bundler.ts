import { JsonRpcProvider } from "ethers"
import type { UserOperation } from "./userop"

export async function sendUserOp(opts: {
  bundlerUrl: string
  userOp: UserOperation
  entryPoint: string
}) {
  const provider = new JsonRpcProvider(opts.bundlerUrl)
  return provider.send("eth_sendUserOperation", [opts.userOp, opts.entryPoint])
}
