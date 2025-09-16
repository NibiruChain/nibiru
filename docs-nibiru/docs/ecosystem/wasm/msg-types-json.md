# Message Types & JSON

Data is transmitted between the host and the Wasm smart contract using JavaScript
Object Notation (JSON). While JSON is intuitive for JavaScript-based
applications, it has its limitations too. JSON has no native binary
type and has inconsistent support for integers larger than 53 bits.

To address these issues, CosmWasm's standard library, `cosmwasm-std`, provides
types optimized for JSON handling. The table below illustrates how standard Rust
types and `cosmwasm_std` types are encoded in JSON.

| Rust type           | [JSON type][json-ref]                             | Example                                                                           | Note                                                                                                                                                                                                                                                                           |
| ------------------- | ----------------------------------------- | --------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| bool                | Boolean (`true`/`false`)                  | `true`                                                                            |                                                                                                                                                                                                                                                                                |
| u32/i32             | Number                                    | `123`                                                                             |                                                                                                                                                                                                                                                                                |
| u64/i64             | Number                                    | `123456`                                                                          | Fully supported in Rust and Go, partial support in other implementations (e.g., JavaScript, `jq`).                                                                                                                                                                             |
| u128/i128           | Number                                    | `340282366920938463463374607431768211455`, `-2766523308300312711084346401884294402` | Full support in Rust only. Previously serialized as a string in serde-json-wasm. For compatibility, switch to `Uint128` / `Int128`. See [Dev Note #4][dev-note-4].                                                                                                              |
| usize/isize         | Number                                    | `123456`                                                                          | Avoid using; different sizes in unit tests (64-bit) and Wasm (32-bit) can cause issues. May trigger float instructions preventing contract upload.                                                                                                                             |
| String              | String                                    | `"foo"`                                                                           |
| &str                | String                                    | `"foo"`                                                                           | Unsupported; message types must be owned (`DeserializeOwned`).                                                                                                                                                                                                                 |
| Option\<T\>         | `null` or JSON type of `T`                | `null`, `{"foo":12}`                                                              |                                                                                                                                                                                                                                                                                |
| Vec\<T\>            | Array of JSON type of `T`                 | `["one", "two", "three"]` (Vec\<String\>), `[true, false]` (Vec\<bool\>)          |
| Vec\<u8\>           | Array of numbers (0-255)                  | `[187, 61, 11, 250]`                                                              | Use discouraged; not as compact as possible. Prefer using `Binary`.                                                                                                                                                                                                           |
| struct MyType { … } | Object                                    | `{"foo":12}`                                                                      |                                                                                                                                                                                                                                                                                |
| [Uint64]/[Int64]    | String with number                        | `"1234321"`, `"-1234321"`                                                         | Supports full uint64/int64 range in all implementations.                                                                                                                                                                                                                       |
| [Uint128]/[Int128]  | String with number                        | `"1234321"`, `"-1234321"`                                                         |                                                                                                                                                                                                                                                                                |
| [Uint256]/[Int256]  | String with number                        | `"1234321"`, `"-1234321"`                                                         |                                                                                                                                                                                                                                                                                |
| [Uint512]/[Int512]  | String with number                        | `"1234321"`, `"-1234321"`                                                         |                                                                                                                                                                                                                                                                                |
| [Decimal]           | String with decimal number                | `"55.6584"`                                                                       |                                                                                                                                                                                                                                                                                |
| [Decimal256]        | String with decimal number                | `"55.6584"`                                                                       |                                                                                                                                                                                                                                                                                |
| [Binary]            | Base64-encoded string                     | `"MTIzCg=="`                                                                      |                                                                                                                                                                                                                                                                                |
| [HexBinary]         | Hexadecimal string                        | `"b5d7d24e428c"`                                                                  |                                                                                                                                                                                                                                                                                |
| [Timestamp]         | String with nanoseconds since epoch       | `"1677687687000000000"`                                                           |                                                                                                                                                                                                                                                                                |

[uint64]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Uint64.html
[uint128]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Uint128.html
[uint256]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Uint256.html
[uint512]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Uint512.html
[int64]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Int64.html
[int128]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Int128.html
[int256]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Int256.html
[int512]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Int512.html
[decimal]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Decimal.html
[decimal256]:
  https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Decimal256.html
[binary]: https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Binary.html
[hexbinary]:
  https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.HexBinary.html
[timestamp]:
  https://docs.rs/cosmwasm-std/1.3.3/cosmwasm_std/struct.Timestamp.html
[dev-note-4]:
  https://medium.com/cosmwasm/dev-note-4-u128-i128-serialization-in-cosmwasm-90cb76784d44

[json-ref]: https://www.json.org/
