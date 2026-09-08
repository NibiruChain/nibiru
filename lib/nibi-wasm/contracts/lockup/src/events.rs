use cosmwasm_std::{Coin, Event};

pub const COINS_LOCKED_EVENT_NAME: &str = "coins_locked";
pub const UNLOCK_INITIATION_EVENT_NAME: &str = "unlock_initiated";
pub const LOCK_FUNDS_WITHDRAWN: &str = "funds_withdrawn";

pub fn event_coins_locked(id: u64, coin: &Coin) -> Event {
    Event::new(COINS_LOCKED_EVENT_NAME)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
}

pub fn event_unlock_initiated(id: u64, coin: &Coin, unlock_block: u64) -> Event {
    Event::new(UNLOCK_INITIATION_EVENT_NAME)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
        .add_attribute("unlock_block", unlock_block.to_string())
}

pub fn event_funds_withdrawn(id: u64, coin: &Coin) -> Event {
    Event::new(LOCK_FUNDS_WITHDRAWN)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
}
