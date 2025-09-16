---
order: 4
---

# Wasm VM and EVM

<!-- # Wasm VM and the EVM -->
<!---->
<!-- ## Pre-requisite Readings -->
<!---->
<!-- - [Wasm VM](./wasm-vm) {prereq} -->

- [Web Assembly (Wasm)](#web-assembly-wasm)
- [Comparing Wasm VM and the EVM](#comparing-wasm-vm-and-the-evm)
- [Wasm VM](#wasm-vm)
- [Advanced: Diving Deeper](#advanced-diving-deeper)

## Web Assembly (Wasm)

Web Assembly (Wasm) was created to make the web faster. That's where the
name "web assembly" comes from. Assembly typically refers to human-readable
languages that are similar to machine code. Nowadays, Wasm isn't actually
used for web development anymore, except for the fact that it can *run* in the
web.

Wasm has a predecessor project, ASM.js, made by Mozilla, where the team tried
to optimize JavaScript to make it execute faster with just-in-time (JIT), or
ahead-of-time (AOT), compilation. ASM.js was designed to serve as a compiler
target for perfomance-critical code and allow browsers to execute certain JS
code faster. In particular, it would make heavy calculations and graphics in
combination with WebGL (Web Graphics Library) faster.

Releasing in March of 2017, Wasm was developed by the same teams behind ASM.js
and (P)NaCl as a joint effort to provide a cross-browser compiler target. These
teams aimed to address the limitations of ASM.js by offering a more efficient
binary format, faster parsing, and better performance. WebAssembly has taken the
ASM.js concept further, offering a more standardized and efficient solution.

Wasm has evolved to offer a small, fast binary form and enables several
complex, high-performance, web-compatible applications we know and love today,
such as Figma, AutoCAD, Google Earth, and CosmWasm.

#### References - Wasm

1. [2017. Why WebAssembly is Faster than ASM.js](https://hacks.mozilla.org/2017/03/why-webassembly-is-faster-than-asm-js/)
2. [2017. Figma. WebAssembly Cut Figma's Load Time by 3x.](https://www.figma.com/blog/webassembly-cut-figmas-load-time-by-3x/)
3. [2023. Ethan Frey. CosmWasm](https://github.com/CosmWasm/cosmwasm)

## Comparing Wasm VM and the EVM

Wasm is generic, whereas the EVM has methods that basically require a
blockchain. The language of the EVM is tied to the blockchain. It's not that
separable from the runtime. The EVM is not just the (virtual) machine; it's the runtime.
This makes the EVM very much like the Java VM (JVM) in that it's for one use case.

This can be good because the EVM is tailored toward its use case. The bad thing
about this is that it cannot inherit tooling directly from the high-level
languages. The EVM had to have get an ecosystem of developers to build tooling
and seek performance optimizations on it for years.

Wasm is different. It's general-purpose, so a lot of different languages
started targeting it and supporting Wasm-specific tooling. One major perk of
Wasm smart contracting is that it didn't require creating a new programming
language. You can compose any tooling implicit in high-level languages with
Wasm binaries simply by modifying its public entry points.

Really proficient developers that work on low-level compilers and interpreters
already actively improve Wasm and build tooling around it. The Web3 community
doesn't have to develop this infrastructure on its own. Wasm allows us to use
top of the line software without recruiting these developers to work on
blockchain projects alongside us. A huge ecosystem of companies can collaborate
on a general-purpose VM layer, which is powerful.

> Note that at the time of the EVM launch, WebAssembly didn't exist. It wasn't
launched and was only in an idea stage.

One could make a **compelling case that the EVM would just be WasmVM instead if
Wasm existed at the time**. The first official release of WebAssembly came in
March 2017, while Ethereum's initial release (Frontier) was in July 2015. Since
WebAssembly wasn't a mature or widely-accepted standard at the time, it would
have been risky for the Ethereum Foundation to base their virtual machine on
it.

For many of the reasons outlined above, chains PolkaDot, Near, and CosmWasm run
on Wasm as well. By Feb 2023, about half of the Cosmos zones (~25) run
the WASM virtual machine and Composable Finance is running a [CosmWasm parachain on Substrate](https://medium.com/composable-finance/how-we-built-a-generalized-cosmwasm-vm-a0ac70fa8219).

## Wasm VM

Wasm is a VM. You'll even hear people refer to "the Wasm VM" in contrast to
"wasm files" (WebAssembly binaries). A virtual machine is essentially a
software-based representation of a computer system that can execute programs as
if they were running on a physical machine. WebAssembly is referred to as a
virtual machine (VM) because it provides an abstract computing environment that
allows code to be executed in a way that's independent of the underlying
hardware and operating system.

Wasm is a bit peculiar in that it doesn't function as traditional VM like the
Java Virtual Machine (JVM) or the .NET runtime. Instead, WebAssembly is closer
to an instruction set architecture (ISA) for a conceptual machine, where
developers write and compile code in various high-level languages (Rust,
TypeScript, C++, Golang, etc.) that can then be **executed on any platform with
a WebAssembly-compatible engine**.

Wasm is a 32 bit processor with some stack calls. It has around 150 operations
and an incredibly simple architecture.

## EVM Tooling Analogs

| Ethereum | Wasm | Description |
| --- | --- | --- |
| Solidity, Vyper | CosmWasm-std + Rust | Smart contract languages |
| Open Zepplin contracts | cw-plus | Smart contract libraries with reusable components. |
| EVM | CosmWasm VM | Virtual machines that execute smart contract bytecode. |
| Web3.js | CosmJS | Client-side JavaScript libraries for interacting with blockchains. |

## Advanced: Diving Deeper

- [Architecture: CosmWasm](./arch)
