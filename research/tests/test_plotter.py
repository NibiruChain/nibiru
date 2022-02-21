import pytest
from pkg import simulation as sim
from pkg import types
from pkg import plotter
import plotly.graph_objects as go
import pandas as pd

@pytest.fixture
def scenario_df() -> pd.DataFrame:
    return pd.read_csv("luna_plot_df.csv")

def test_plot_insurance_fund(scenario_df: pd.DataFrame):
    fig = plotter.plot_insurance_fund(plot_df=scenario_df)
    assert isinstance(fig, go.Figure)
    assert len(fig.data) == 2
    assert isinstance(fig.data[0], go.Scatter)

def test_plot_la_exposure(scenario_df: pd.DataFrame):
    fig = plotter.plot_la_exposure(plot_df=scenario_df)
    assert isinstance(fig, go.Figure)
    assert len(fig.data) == 2
    assert isinstance(fig.data[0], go.Scatter)


