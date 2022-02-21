# import dash
# import dash_core_components as dcc
# import dash_html_components as html

from pkg import scenario
from pkg import plotter

import pandas as pd
import numpy as np
import seaborn as sns
from matplotlib import pyplot as plt

import streamlit as st

if __name__ == "__main__":
    # app = dash.Dash()
    # app.layout = html.Div(children=[dcc.Graph(figure=plotter.plot_insurance_fund()),])

    # app.run_server(debug=True, use_reloader=False)

    st.sidebar.title("Parameters in BP")
    entry_fee = st.sidebar.number_input("Entry fee", 20) * 1e-4
    exit_fee = st.sidebar.number_input("Exit fee", 40) * 1e-4
    frate_to_LA = st.sidebar.number_input(
        "Funding rate paid from insurance fund to leverage agents", 60)
    frate_to_IF = st.sidebar.number_input(
        "Funding rate paid from leverage agents to insurance fund", 120)

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
        entry_fee=entry_fee,
        exit_fee=exit_fee,
        frate_to_LA=frate_to_LA,
        frate_to_IF=frate_to_IF,
        IF_exposure_init=1_000_000,
        take_profit_chance=0.4,
        take_loss_chance=0.3)
    params = scenario.Parameters(LA=la_params, 
                                 protocol=protocol_params, 
                                stochastic=stochastic_process_params)
    result = scenario.create_scenario(params)
    result.to_csv("luna_plot_df.csv")

    st.title("Result of the simulation")
    st.plotly_chart(plotter.plot_insurance_fund(plot_df=result),
                    use_container_width=True)
