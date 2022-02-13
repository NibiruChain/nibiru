# import dash
# import dash_core_components as dcc
# import dash_html_components as html
from plotly.express.colors import sequential
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots

import pandas as pd
import numpy as np
from pkg.create_scenario import *

import seaborn as sns
from matplotlib import pyplot as plt

import streamlit as st


parameters = Parameters(
    # Simulation parameter
    # n_periods=1000,
    # Protocol parameter
    entry_fee=20e-4,
    exit_fee=40e-4,
    hourly_funding_rate_payout_bp=60,
    hourly_funding_rate_fee_bp=120,
    initial_insurance_fund=1_000_000,
    # LA parameters
    n_LAs_position_per_period=100,
    size_LAs_position_gamma_parameters=[3, 10_000],
    leverages_poisson_parameter=8,
    take_profit_chance=0.4,
    take_loss_chance=0.3,
    # Price parameters
    s0=100,
    mu=0.23,
    sigma=0.68,
    dt=1,
)

import plotly.express as px

result = create_scenario(parameters)


def create_price_blaance_graph():
    fig = make_subplots(specs=[[{"secondary_y": True}]])

    fig.add_trace(go.Scatter(x=result.time, y=result.treasury, name="Insurance fund"))
    fig.add_trace(
        go.Scatter(x=result.time, y=result.price, name="Luna price"), secondary_y=True
    )
    # fig.show()

    return make_subplots(specs=[[{"secondary_y": True}]])


if __name__ == "__main__":
    # app = dash.Dash()
    # app.layout = html.Div(children=[dcc.Graph(figure=create_price_blaance_graph()),])

    # app.run_server(debug=True, use_reloader=False)

    entry_fee = st.sidebar.number_input("entry_fee", 20) * 1e-4
    exit_fee = st.sidebar.number_input("exit_fee", 40) * 1e-4
    hourly_funding_rate_payout_bp = st.sidebar.number_input(
        "hourly_funding_rate_payout_bp", 60
    )
    hourly_funding_rate_fee_bp = st.sidebar.number_input(
        "hourly_funding_rate_fee_bp", 120
    )
    initial_insurance_fund = st.sidebar.number_input(
        "initial_insurance_fund", 1_000_000
    )

    parameters = Parameters(
        # Simulation parameter
        # n_periods=1000,
        # Protocol parameter
        entry_fee=entry_fee,
        exit_fee=exit_fee,
        hourly_funding_rate_payout_bp=hourly_funding_rate_payout_bp,
        hourly_funding_rate_fee_bp=hourly_funding_rate_fee_bp,
        initial_insurance_fund=1_000_000,
        # LA parameters
        n_LAs_position_per_period=100,
        size_LAs_position_gamma_parameters=[3, 10_000],
        leverages_poisson_parameter=8,
        take_profit_chance=0.4,
        take_loss_chance=0.3,
        # Price parameters
        s0=100,
        mu=0.23,
        sigma=0.68,
        dt=1,
    )
    result = create_scenario(parameters)
    st.plotly_chart(create_price_blaance_graph(), use_container_width=True)
