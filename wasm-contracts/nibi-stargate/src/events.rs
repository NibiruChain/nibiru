use cosmwasm_std::{Coin, Event};

pub const COINS_LOCKED_EVENT_NAME: &str = "coins_locked";
pub const UNLOCK_INITIATION_EVENT_NAME: &str = "unlock_initiated";
pub const LOCK_FUNDS_WITHDRAWN: &str = "funds_withdrawn";

pub fn new_coins_locked_event(id: u64, coin: &Coin) -> Event {
    Event::new(COINS_LOCKED_EVENT_NAME)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
}

pub fn new_unlock_initiation_event(
    id: u64,
    coin: &Coin,
    unlock_block: u64,
) -> Event {
    Event::new(UNLOCK_INITIATION_EVENT_NAME)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
        .add_attribute("unlock_block", unlock_block.to_string())
}

pub fn new_funds_withdrawn_event(id: u64, coin: &Coin) -> Event {
    Event::new(LOCK_FUNDS_WITHDRAWN)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", coin.to_string())
}
