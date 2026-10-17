use crate::state::Program;
use cosmwasm_std::{Coin, Event};

pub const NEW_PROGRAM_EVENT_NAME: &str = "new_incentives_program";
pub const PROGRAM_FUNDING_EVENT_NAME: &str = "incentives_program_funding";

pub fn new_incentives_program_event(program: &Program) -> Event {
    Event::new(NEW_PROGRAM_EVENT_NAME)
        .add_attribute("id", program.id.to_string())
        .add_attribute("epochs", program.epochs.to_string())
        .add_attribute("epoch_duration", program.epoch_duration.to_string())
        .add_attribute("end_block", program.end_block.to_string())
        .add_attribute("start_block", program.start_block.to_string())
        .add_attribute("lockup_denom", program.lockup_denom.clone())
        .add_attribute(
            "min_lockup_duration_blocks",
            program.min_lockup_duration_blocks.to_string(),
        )
}

pub fn new_program_funding(id: u64, coins: &Vec<Coin>) -> Event {
    Event::new(PROGRAM_FUNDING_EVENT_NAME)
        .add_attribute("id", id.to_string())
        .add_attribute("coins", serde_json_wasm::to_string(coins).unwrap())
}
