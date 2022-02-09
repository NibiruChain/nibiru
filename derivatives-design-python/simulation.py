#!/usr/bin/env python
import dataclasses

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

def test_delta_market_up():
    amt_osmo: int = 100
    spec_init = SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    spec_final = SpeculativeAssetState(amt=amt_osmo, price_usd=15)
    delta = exposure_delta(spec_init=spec_init, spec_final=spec_final)
    assert delta == 500

def test_delta_market_down():
    amt_osmo: int = 100
    spec_init = SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    spec_final = SpeculativeAssetState(amt=amt_osmo, price_usd=5)
    delta = exposure_delta(spec_init=spec_init, spec_final=spec_final)
    assert delta == -500

def basis_points(x: float) -> float:
    return x * 1e-4

class ProtocolState:
    la: str

FUNDING_RATE: float = 5e-3
# https://docs.perp.fi/getting-started/how-it-works/funding-payments

def foo():
    funding_rate_freq: int = 24
    funding_rate = FUNDING_RATE
    daily_funding = funding_rate * funding_rate_freq

las_revenue = las * daily_funding