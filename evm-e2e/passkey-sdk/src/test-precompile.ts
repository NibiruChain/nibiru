import { JsonRpcProvider, getBytes, hexlify, sha256, solidityPacked } from "ethers"
import { generateNodePasskey, signUserOpHash } from "./p256-node"

const RPC_URL = process.env.JSON_RPC_ENDPOINT ?? "http://127.0.0.1:8545"

async function main() {
  const provider = new JsonRpcProvider(RPC_URL)
  console.log("Testing RIP-7212 precompile at 0x100 on", RPC_URL)

  // 1. Generate a key pair
  const nodePasskey = generateNodePasskey()
  console.log("Generated key pair")
  console.log("qx:", hexlify(nodePasskey.pubQx))
  console.log("qy:", hexlify(nodePasskey.pubQy))

  // 2. Create a dummy message hash (simulating userOpHash)
  const dummyHash = sha256(new Uint8Array([1, 2, 3]))
  console.log("Dummy hash:", dummyHash)

  // 3. Sign the hash (SDK signs sha256(hash))
  // The contract does: digest = sha256(abi.encodePacked(hash))
  // The SDK does: digest = sha256(hash)
  // These match if 'hash' is bytes32.
  const { r, s } = signUserOpHash(dummyHash, nodePasskey.privKey)
  console.log("Signature r:", r)
  console.log("Signature s:", s)

  // 4. Construct input for precompile
  // input = digest || r || s || qx || qy
  // digest = sha256(hash)
  const digest = sha256(dummyHash)
  console.log("Digest sent to precompile:", digest)

  const input = solidityPacked(
    ["bytes32", "bytes32", "bytes32", "bytes32", "bytes32"],
    [digest, r, s, nodePasskey.pubQx, nodePasskey.pubQy]
  )
  
  console.log("Input length:", getBytes(input).length)
  // console.log("Input hex:", input)

  // 5. Call the precompile
  try {
    const result = await provider.call({
      to: "0x0000000000000000000000000000000000000100",
      data: input
    })
    console.log("Precompile result:", result)
    
    if (result === "0x" || result === "0x00") {
        console.error("FAILURE: Precompile returned empty data. It might not be implemented or input is invalid.")
    } else if (result.endsWith("01")) {
        console.log("SUCCESS: Precompile returned valid signature confirmation.")
    } else {
        console.log("UNKNOWN RESULT:", result)
    }

  } catch (e) {
    console.error("Error calling precompile:", e)
  }
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
