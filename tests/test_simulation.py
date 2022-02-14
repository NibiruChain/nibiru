import pytest
from pkg import simulation as sim
from pkg import types

def test_exposure_delta_market_up():
    amt_osmo: int = 100
    spec_init = types.SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    spec_final = types.SpeculativeAssetState(amt=amt_osmo, price_usd=15)
    delta = sim.exposure_delta(spec_init=spec_init, spec_final=spec_final)
    assert delta == 500

def test_exposure_delta_market_down():
    amt_osmo: int = 100
    spec_init = types.SpeculativeAssetState(amt=amt_osmo, price_usd=10)
    spec_final = types.SpeculativeAssetState(amt=amt_osmo, price_usd=5)
    delta = sim.exposure_delta(spec_init=spec_init, spec_final=spec_final)
    assert delta == -500

def test_get_leveraged_position():
    c_matrix = 60
    spec_init = types.SpeculativeAssetState(amt=c_matrix, price_usd=10)
    spec_final = types.SpeculativeAssetState(amt=c_matrix, price_usd=12)
    matrix = types.ProtocolState(LA_amt=c_matrix, IA_amt=0, IF_amt=0)

    c_LA = 10

    with pytest.raises(ValueError):
        c_cover = 70
        la_position = sim.get_leveraged_position(
            LA_amt=c_LA, cover_amt=c_cover ,protocol=matrix, 
            spec_init=spec_init, spec_final=spec_final)

    c_cover = 60
    la_position = sim.get_leveraged_position(
        LA_amt=c_LA, cover_amt=c_cover ,protocol=matrix, 
        spec_init=spec_init, spec_final=spec_final)
    assert la_position.value == 20
    assert la_position.leverage_mult == 6

    spec_init = types.SpeculativeAssetState(amt=c_matrix, price_usd=10)
    spec_final = types.SpeculativeAssetState(amt=c_matrix, price_usd=13)
    la_position = sim.get_leveraged_position(
        LA_amt=c_LA, cover_amt=c_cover ,protocol=matrix, 
        spec_init=spec_init, spec_final=spec_final)
    position_value = (
        c_LA * (la_position.leverage_mult * la_position.price_pct_change + 1))

    assert la_position.value - position_value == pytest.approx(0, 0.0001)
