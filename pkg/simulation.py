#!/usr/bin/env python
import dataclasses
from multiprocessing.sharedctypes import Value
import numpy as np
import pandas as pd
from pkg import types
from typing import Dict, List, Sequence

def exposure_in_spec(spec: types.SpeculativeAssetState) -> float:
    exposure_in_spec = spec.amt * spec.price_usd
    return exposure_in_spec

def exposure_delta(spec_init: types.SpeculativeAssetState, 
                   spec_final: types.SpeculativeAssetState) -> float:
    exposure_init, exposure_final = [
        exposure_in_spec(spec=s) for s in [spec_init, spec_final]]
    return exposure_final - exposure_init

def basis_points(x: float) -> float:
    return x * 1e-4

def get_leveraged_position(LA_amt: float, 
                           cover_amt: float,
                           protocol: types.ProtocolState, 
                           spec_init: types.SpeculativeAssetState, 
                           spec_final: types.SpeculativeAssetState
                           ) -> types.LeveragedPosition:
    """[summary]

    Args:
        LA_amt (float): Amount of collateral the agent brings to the protocol.
        cover_amt (float): Amount of collateral the agent chooses to cover.
        protocol (types.ProtocolState): Initial Matrix protocol state.
        spec_init (types.SpeculativeAssetState): Collateral initial state.
        spec_final (types.SpeculativeAssetState): Collateral final state.

    Returns:
        types.LeveragedPosition: Representation for the corresponding leverage agent
            liquidity position. 
    """
            
    price_pct_change = (1 - spec_init.price_usd / spec_final.price_usd)
    if cover_amt > protocol.total_amt:
        raise ValueError("An LA cannot cover more collateral than what's "
                         "available in Matrix.\n" 
                         f"'cover_amt': {cover_amt}\n"
                         f"'protcol.total_amt': {protocol.total_amt}")
    position_value_in_spec = LA_amt + protocol.total_amt * price_pct_change
    return types.LeveragedPosition(
        c_LA=LA_amt, price_pct_change=price_pct_change, 
        c_cover=cover_amt, value=position_value_in_spec)

def example_0():
    total_spec_supply = 1e9 # 1 billion

def example_1():
    amt_osmo = 10e3
    spec_init = types.SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    matrix_exposure = exposure_in_spec(spec=spec_init)
    assert matrix_exposure == 1e5

    la_exposure = 1e4
    la_amt = la_exposure / spec_init.price_usd
    assert la_amt == 1000
    leverage_mult = matrix_exposure / la_exposure
    assert leverage_mult == 10

    # Price of OSMO increases 5 bucks
    spec_1 = types.SpeculativeAssetState(amt=amt_osmo, price_usd=15)
    protocol_pnl = exposure_delta(spec_init=spec_init, spec_final=spec_1)
    assert protocol_pnl == 5e4

def compute_funding_payment(bps: int, 
                            spec: types.SpeculativeAssetState) -> float:
    funding_rate: float = basis_points(bps)
    exposure: float = exposure_in_spec(spec=spec)
    funding_payment = funding_rate * exposure
    return funding_payment


class Simulator:

    price_current: float
    ts_current: float
    payment_cycles_passed: int
    simulation_log: List[dict]

    def __init__(self):
        self.simulation_log = []
    
    def log_simulation(self):
        self.simulation_log.append(dict(
            ts=self.ts_current, 
            price=self.price_current,
            LA_amt=self.protocol.LA_amt,
            IF_amt=self.protocol.IF_amt,
            IA_amt=self.protocol.IA_amt,
            total_amt=self.protocol.total_amt,
            cycles_passed=self.payment_cycles_passed
        ))

    def get_funding_payment(self, 
                            protocol: types.ProtocolState, 
                            price: float) -> float:
        """[summary]

        Args:
            protocol (types.ProtocolState): [description]
            price (float): [description]

        Returns:
            float: [description]
        """
            
        funding_rate_bps: int
        if self.market_state == "bull": 
            # LAs pay the IF in bull
            funding_rate_bps = protocol.frate_to_IF 
            spec_amt = protocol.LA_amt
        else: 
            # IF pays the LA in bear 
            funding_rate_bps = protocol.frate_to_LA 
            spec_amt = protocol.IF_amt
        
        funding_payment: float = compute_funding_payment(
            bps=funding_rate_bps, 
            spec=types.SpeculativeAssetState(amt=spec_amt, price_usd=price))
        return funding_payment
    
    @property
    def market_state(self) -> str:
        price_delta = self.price_current - self.price_at_previous_funding_payment
        if price_delta >= 0:
            market: str = "bull"
        else:
            market: str = "bear"
        return market

    def distribute_funding_payment(self,
                                   protocol: types.ProtocolState,  
                                   payment: float, 
                                   ) -> types.ProtocolState:
        """Updates the protocol state based on the given funding payment.
        - Funding payments go IF → LA  in a bear market.
        - Funding payments go LA → IF  in a bull market.

        Args: TODO: 
            protocol (types.ProtocolState)
            payment (float): Funding payment.
            to (str): Receiver of the funding payment. Either "LA" or "IF".

        Returns: 
            (types.ProtocolState)
        """

        if self.market_state == "bull":
            # LAs pays the Insurance Fund
            IF_amt = protocol.IF_amt + payment
            LA_amt = protocol.LA_amt - payment
        else: # self.market_state == "bear" 
            # Insurance fund pays the LAs
            LA_amt = protocol.LA_amt + payment
            IF_amt = protocol.IF_amt - payment

        protocol.update_amts(LA_amt=LA_amt, IF_amt=IF_amt)
        return protocol

    def funding_payment_simulation(self, 
                                   protocol: types.ProtocolState, 
                                   prices: Sequence[float],
                                   timestamps: Sequence[pd.Timestamp],
                                   daily_payments: int = 24,
                                ) -> pd.DataFrame:
        """[summary]

        Args:
            protocol (types.ProtocolState)
            prices (Sequence[float]): Time series of prices.
            timestamps (Sequence[pd.Timestamp]): Timestamps corresponding to the 
                price series, 'prices'.
            funding_rate_bps (int): Funding rate in terms of basis points. Defaults 
                to 60, which corresponds to a funding rate 60e-4 (== 0.6%).
            daily_payments (int): Frequency of funding payments per day. Defaults 
                to 24 (hourly payments).

        Returns: 
            (pd.DataFrame): Simulation log as a dataframe.

        # https://docs.perp.fi/getting-started/how-it-works/funding-payments
        """
        if not isinstance(prices, np.ndarray):
            prices: np.ndarray = np.asarray(prices)

        self.simulation_log = []
        time_index: int = 0
        max_time_index: int = len(timestamps) - 1
        self.price_at_previous_funding_payment: float = prices[time_index]
        self.payment_cycles_passed: int = 0
        self.log_simulation()

        cycle_duration_in_seconds = 3600 * 24 / daily_payments

        done: bool = (protocol.IF_amt > 0) and (protocol.LA_amt >= 0) 
        while not done:
            self.price_current = prices[time_index] 
            self.ts_current = timestamps[time_index]
            time_delta: pd.Timedelta = self.ts_current - timestamps[0]
            time_delta_in_cycles = (
                time_delta.total_seconds() / cycle_duration_in_seconds)

            payment_cycle_has_passed: bool = (
                time_delta_in_cycles - payment_cycles_passed >= 1)
            if payment_cycle_has_passed:
                funding_payment: float = self.get_funding_payment(
                    protocol=protocol, price=self.current_price)
                
                if protocol.IF_amt - funding_payment < 0:
                    self.log_simulation()
                    done = True; break

                protocol: types.ProtocolState = self.distribute_funding_payment(
                    protocol=protocol, price=self.price_current, 
                    payment=funding_payment, 
                    to="fpayments")
                payment_cycles_passed += 1
                self.price_at_previous_funding_payment = self.price_current
                self.log_simulation()
            
            time_index += 1
            done = (not((protocol.IF_amt > 0) and (protocol.LA_amt >= 0)) 
                    or (time_index + 1 > max_time_index))
        
        return pd.DataFrame(self.simulation_log)