import pytest
from pkg import simulation as sim
from pkg import types
from pkg import scenario
import pandas as pd

@pytest.fixture
def params() -> scenario.Parameters:
    la_params = scenario.LeverageAgentParams(
        num_LA_positions_per_period=100, 
        position_size_gamma_params=[3, 10_000], 
        poisson=8)
    stochastic_process_params = scenario.StochasticProcessParams(
        s0=100,
        mu=0.23,
        sigma=0.68,
        dt=1)
    protocol_params = scenario.ProtocolParams(
        entry_fee=20e-4,
        exit_fee=40e-4,
        frate_to_LA=60,
        frate_to_IF=120,
        IF_exposure_init=1_000_000,
        take_profit_chance=0.4,
        take_loss_chance=0.3)
    params = scenario.Parameters(LA=la_params, 
                                 protocol=protocol_params, 
                                 stochastic=stochastic_process_params)
    return params

def test_full_run(params: scenario.Parameters):
    # TODO: refactor, feature: Make 'scenario.create_scenario' more composable.
    # scenario_df: pd.DataFrame = scenario.create_scenario(params)
    # scenario_df.to_csv("luna_plot_csv")
    scenario_df: pd.DataFrame = pd.read_csv("luna_plot_df.csv")
    assert isinstance(scenario_df, pd.DataFrame)
