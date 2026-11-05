//! proto/mod.rs: Protobuf types defined in NibiruChain/nibiru/proto.

mod traits;
mod type_url_cosmos;
mod type_url_nibiru;

pub use traits::*;

pub mod cosmos {
    /// Authentication of accounts and transactions.
    pub mod auth {
        pub mod v1beta1 {
            include!("buf/cosmos.auth.v1beta1.rs");
        }
    }

    pub mod authz {
        pub mod v1beta1 {
            include!("buf/cosmos.authz.v1beta1.rs");
        }
    }

    pub mod bank {
        pub mod v1beta1 {
            include!("buf/cosmos.bank.v1beta1.rs");
        }
    }

    /// Base functionality.
    pub mod base {
        /// Application BlockChain Interface (ABCI).
        ///
        /// Interface that defines the boundary between the replication engine
        /// (the blockchain), and the state machine (the application).
        pub mod abci {
            pub mod v1beta1 {
                include!("buf/cosmos.base.abci.v1beta1.rs");
            }
        }

        /// Key-value pairs.
        pub mod kv {
            pub mod v1beta1 {
                include!("buf/cosmos.base.kv.v1beta1.rs");
            }
        }

        /// Query support.
        pub mod query {
            pub mod v1beta1 {
                include!("buf/cosmos.base.query.v1beta1.rs");
            }
        }

        /// Reflection support.
        pub mod reflection {
            pub mod v1beta1 {
                include!("buf/cosmos.base.reflection.v1beta1.rs");
            }

            pub mod v2alpha1 {
                include!("buf/cosmos.base.reflection.v2alpha1.rs");
            }
        }

        /// Snapshots containing Tendermint state sync info.
        pub mod snapshots {
            pub mod v1beta1 {
                include!("buf/cosmos.base.snapshots.v1beta1.rs");
            }
        }

        /// Data structure that holds the state of the application.
        pub mod store {
            pub mod v1beta1 {
                include!("buf/cosmos.base.store.v1beta1.rs");
            }
        }

        /// Defines base data structures like Coin, DecCoin, IntProto, and
        /// DecProto. These types implement custo method signatures required by
        /// gogoproto.
        pub mod v1beta1 {
            include!("buf/cosmos.base.v1beta1.rs");
        }

        /// For consensus types related to blocks, block headers, and merkle
        /// proofs.
        pub mod tendermint {
            pub mod v1beta1 {
                include!("buf/cosmos.base.tendermint.v1beta1.rs");
            }
        }
    }

    pub mod crisis {
        pub mod v1beta1 {
            include!("buf/cosmos.crisis.v1beta1.rs");
        }
    }

    pub mod crypto {
        pub mod v1beta1 {
            include!("buf/cosmos.crisis.v1beta1.rs");
        }

        pub mod ed25519 {
            include!("buf/cosmos.crypto.ed25519.rs");
        }

        pub mod hd {
            pub mod v1 {
                include!("buf/cosmos.crypto.hd.v1.rs");
            }
        }

        pub mod keyring {
            pub mod v1 {
                include!("buf/cosmos.crypto.keyring.v1.rs");
            }
        }

        pub mod multisig {
            pub mod v1beta1 {
                include!("buf/cosmos.crypto.multisig.v1beta1.rs");
            }
        }
        pub mod secp256r1 {
            include!("buf/cosmos.crypto.secp256r1.rs");
        }
    }

    pub mod genutil {
        pub mod v1beta1 {
            include!("buf/cosmos.genutil.v1beta1.rs");
        }
    }

    /// Types related to decentralized governance of the network.
    pub mod gov {
        pub mod v1 {
            include!("buf/cosmos.gov.v1.rs");
        }
    }

    pub mod group {
        pub mod v1 {
            include!("buf/cosmos.group.v1.rs");
        }
    }

    pub mod mint {
        pub mod v1beta1 {
            include!("buf/cosmos.mint.v1beta1.rs");
        }
    }

    pub mod nft {
        pub mod v1beta1 {
            include!("buf/cosmos.nft.v1beta1.rs");
        }
    }

    pub mod params {
        pub mod v1beta1 {
            include!("buf/cosmos.params.v1beta1.rs");
        }
    }
    pub mod reflection {
        pub mod v1 {
            include!("buf/cosmos.reflection.v1.rs");
        }
    }
    pub mod slashing {
        pub mod v1beta1 {
            include!("buf/cosmos.slashing.v1beta1.rs");
        }
    }
    pub mod staking {
        pub mod v1beta1 {
            include!("buf/cosmos.staking.v1beta1.rs");
        }
    }
    pub mod tx {
        pub mod config {
            pub mod v1 {
                include!("buf/cosmos.tx.config.v1.rs");
            }
        }
        pub mod signing {
            pub mod v1beta1 {
                include!("buf/cosmos.tx.signing.v1beta1.rs");
            }
        }
        pub mod v1beta1 {
            include!("buf/cosmos.tx.v1beta1.rs");
        }
    }

    pub mod upgrade {
        pub mod v1beta1 {
            include!("buf/cosmos.upgrade.v1beta1.rs");
        }
    }

    pub mod vesting {
        pub mod v1beta1 {
            include!("buf/cosmos.vesting.v1beta1.rs");
        }
    }

    // TODO: protobuf mod for cosmos capability
    // TODO: protobuf mod for cosmos consensus
    // TODO: protobuf mod for cosmos crisis
    // TODO: protobuf mod for cosmos crypto
    // TODO: protobuf mod for cosmos distribution
    // TODO: protobuf mod for cosmos evidence
    // TODO: protobuf mod for cosmos feegrant
}

pub mod nibiru {
    pub mod devgas {
        include!("buf/nibiru.devgas.v1.rs");
    }
    pub mod epochs {
        include!("buf/nibiru.epochs.v1.rs");
    }
    pub mod genmsg {
        include!("buf/nibiru.genmsg.v1.rs");
    }
    pub mod inflation {
        include!("buf/nibiru.inflation.v1.rs");
    }
    pub mod oracle {
        include!("buf/nibiru.oracle.v1.rs");
    }
    pub mod perp {
        include!("buf/nibiru.perp.v2.rs");
    }
    pub mod spot {
        include!("buf/nibiru.spot.v1.rs");
    }
    pub mod sudo {
        include!("buf/nibiru.sudo.v1.rs");
    }
    pub mod tokenfactory {
        include!("buf/nibiru.tokenfactory.v1.rs");
    }
}

pub mod tendermint {
    pub mod abci {
        include!("buf/tendermint.abci.rs");
    }
    pub mod crypto {
        include!("buf/tendermint.crypto.rs");
    }
    pub mod p2p {
        include!("buf/tendermint.p2p.rs");
    }
    pub mod types {
        include!("buf/tendermint.types.rs");
    }
    pub mod version {
        include!("buf/tendermint.version.rs");
    }
}

pub mod eth {
    pub mod evm {
        include!("buf/eth.evm.v1.rs");
    }
    pub mod types {
        include!("buf/eth.types.v1.rs");
    }
}

#[cfg(test)]
mod tests {

    use super::{
        cosmos::{self, base::v1beta1::Coin},
        eth,
        nibiru::perp,
    };

    #[test]
    fn nibiru_common_imports() {
        let _ = perp::MsgMarketOrder {
            sender: "sender".to_string(),
            pair: "nibi:usd".to_string(),
            side: perp::Direction::Long.into(),
            quote_asset_amount: "123".into(),
            leverage: "123".into(),
            base_asset_amount_limit: "0".into(),
        };
    }

    #[test]
    fn cosmos_imports() {
        let _ = cosmos::bank::v1beta1::MsgSend {
            from_address: "from".to_string(),
            to_address: "to".to_string(),
            amount: vec![Coin {
                denom: "unibi".to_string(),
                amount: "123".to_string(),
            }],
        };
    }

    #[test]
    fn eth_imports() {
        let _ = eth::evm::MsgEthereumTx {
            data: Some(prost_types::Any {
                type_url: String::from(""),
                value: vec![0, 1, 2],
            }),
            from: "0xfrom".to_string(), // dummy
            hash: "0xhash".to_string(), // dummy
            size: 420f64,
        };
    }
}
