#!/usr/bin/env python
import dataclasses
from typing import Dict

@dataclasses.dataclass
class SpeculativeAssetState:
    amt: int
    price_usd: float

def exposure_in_spec(spec: SpeculativeAssetState) -> float:
    exposure_in_spec = spec.amt * spec.price_usd
    return exposure_in_spec

def exposure_delta(spec_init: SpeculativeAssetState, 
                   spec_final: SpeculativeAssetState) -> float:
    exposure_init, exposure_final = [
        exposure_in_spec(spec=s) for s in [spec_init, spec_final]]
    return exposure_final - exposure_init

def basis_points(x: float) -> float:
    return x * 1e-4

@dataclasses.dataclass
class ProtocolState:

    LA_amt: float
    IA_amt: float
    IF_amt: float

    def update(self, LA_amt, IA_amt, IF_amt) -> None:
        """Update the protocol state."""
        self.LA_amt = LA_amt
        self.IA_amt = IA_amt
        self.IF_amt = IF_amt

    @property
    def total_amt(self) -> float:
        """Total amount of collateral tokens in the protocol."""
        return sum([self.LA_amt, self.IA_amt, self.IF_amt])

    @property
    def pcts(self) -> Dict[str, float]:
        LA_pct = self.LA_amt / self.total_amt
        IA_pct = self.IA_amt / self.total_amt
        IF_pct =  self.IF_amt / self.total_amt
        return dict(LA=LA_pct, IA=IA_pct, IF=IF_pct)

def example_0():
    total_spec_supply = 1e9 # 1 billion

@dataclasses.dataclass
class LeveragedPosition:
    """A liquidity position held by a leverage agent (LA)
    
    Args:
        c_LA (float): Collateral the LA brings to the protocol.
        price_pct_change (float): Percent change is 
            (price_current - price_initial) / price_current
        c_matrix (float): Collateral the agent chooses to cover.
        value (float): Leveraged position value in units of the underlying 
            asset.
    """
    c_LA: float
    price_pct_change: float
    c_matrix: float
    value: float
    
    def __post_init__(self):
        if any([c < 0 for c in [self.c_LA, self.c_matrix]]):
            raise ValueError("Invalid value passed") # TODO: better error msg

        self.leverage_mult = self.c_matrix / self.c_LA


def leveraged_position_value(LA_amt: float, 
                             cover_amt: float,
                             protocol: ProtocolState, 
                             spec_init: SpeculativeAssetState, 
                             spec_final: SpeculativeAssetState
                             ) -> LeveragedPosition:
    """[summary]

    Args:
        LA_amt (float): Amount of collateral the agent brings to the protocol.
        cover_amt (float): Amount of collateral the agent chooses to cover.
        protocol (ProtocolState): Initial Matrix protocol state.
        spec_init (SpeculativeAssetState): Collateral initial state.
        spec_final (SpeculativeAssetState): Collateral final state.

    Returns:
        LeveragedPosition: [description]
    """
            
    price_pct_change = (1 - spec_init.price_usd / spec_final.price_usd)
    if cover_amt > protocol.total_amt:
        raise ValueError("An LA cannot cover more collateral than what's "
                         "available in Matrix.\n" 
                         f"'cover_amt': {cover_amt}\n"
                         f"'protcol.total_amt': {protocol.total_amt}")
    position_value_in_spec = LA_amt + protocol.total_amt * price_pct_change
    return LeveragedPosition(
        c_LA=LA_amt, price_pct_change=price_pct_change, 
        c_matrix=cover_amt, value=position_value_in_spec)

def example_1():
    amt_osmo = 10e3
    spec_init = SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    matrix_exposure = exposure_in_spec(spec=spec_init)
    assert matrix_exposure == 1e5

    la_exposure = 1e4
    la_amt = la_exposure / spec_init.price_usd
    assert la_amt == 1000
    leverage_mult = matrix_exposure / la_exposure
    assert leverage_mult == 10

    # Price of OSMO increases 5 bucks
    spec_1 = SpeculativeAssetState(amt=amt_osmo, price_usd=15)
    protocol_pnl = exposure_delta(spec_init=spec_init, spec_final=spec_1)
    assert protocol_pnl == 5e4

def foo():
    funding_rate_freq: int = 24
    funding_rate = FUNDING_RATE
    daily_funding = funding_rate * funding_rate_freq

if __name__ == "__main__":
    example_1()

    # las_revenue = las * daily_funding

    FUNDING_RATE: float = 5e-3
    # https://docs.perp.fi/getting-started/how-it-works/funding-payments

