extern crate core;

use cosmwasm_std::Coin;

mod contract;
mod error;
mod events;
mod msgs;
mod state;
#[cfg(test)]
mod testing;

pub fn add_coins(coins: &mut Vec<Coin>, coin: Coin) {
    if let Some(existing_coin) = coins.iter_mut().find(|c| c.denom == coin.denom)
    {
        existing_coin.amount += coin.amount;
    } else {
        coins.push(coin);
    }
}

#[cfg(test)]
mod tests {
    use cosmwasm_std::{Coin, Uint128};

    use crate::add_coins;

    #[test]
    fn add_coins_existing_denom() {
        let mut coins = vec![
            Coin {
                denom: "usd".to_string(),
                amount: 50u128.into(),
            },
            Coin {
                denom: "eur".to_string(),
                amount: 60u128.into(),
            },
        ];
        add_coins(
            &mut coins,
            Coin {
                denom: "usd".to_string(),
                amount: 25u128.into(),
            },
        );
        assert_eq!(coins[0].amount, Uint128::from(75u128)); // 50 + 25
    }

    #[test]
    fn add_coins_new_denom() {
        let mut coins = vec![
            Coin {
                denom: "usd".to_string(),
                amount: Uint128::from(50u128),
            },
            Coin {
                denom: "eur".to_string(),
                amount: Uint128::from(60u128),
            },
        ];
        add_coins(
            &mut coins,
            Coin {
                denom: "yen".to_string(),
                amount: Uint128::from(100u128),
            },
        );
        assert_eq!(coins.len(), 3);
        assert_eq!(coins[2].denom, "yen");
        assert_eq!(coins[2].amount, Uint128::from(100u128));
    }
}
