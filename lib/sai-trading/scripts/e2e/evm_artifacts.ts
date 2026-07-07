import { existsSync, readFileSync } from "fs"
import { join } from "path"
import { ethers } from "ethers"

type RawEvmArtifact = {
  abi: ethers.InterfaceAbi
  bytecode: string | { object?: string }
}

function normalizeBytecode(bytecode: RawEvmArtifact["bytecode"]): string {
  const raw = typeof bytecode === "string" ? bytecode : bytecode.object
  if (!raw) {
    throw new Error("SaiEvm artifact bytecode is missing")
  }
  return raw.startsWith("0x") ? raw : `0x${raw}`
}

export function loadSaiEvmArtifact(scriptDir: string): {
  abi: ethers.InterfaceAbi
  bytecode: string
} {
  const candidates = [
    join(scriptDir, "../../artifacts/SaiEvm.json"),
    join(scriptDir, "../../evm/out/SaiEvm.sol/SaiEvm.json"),
    join(
      scriptDir,
      "../../evm/artifacts/contracts/SaiEvm.sol/SaiEvm.json",
    ),
  ]

  const artifactPath = candidates.find((candidate) => existsSync(candidate))
  if (!artifactPath) {
    throw new Error(
      "SaiEvm artifact not found. Vendor artifacts/SaiEvm.json or run sai-perps just evm-build. Tried: " +
        candidates.join(", "),
    )
  }

  const artifact = JSON.parse(
    readFileSync(artifactPath, "utf-8"),
  ) as RawEvmArtifact
  return { abi: artifact.abi, bytecode: normalizeBytecode(artifact.bytecode) }
}
