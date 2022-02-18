#!/usr/bin/env python
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
from plotly import subplots 

def plot_insurance_fund(plot_df: pd.DataFrame) -> go.Figure:

    fig = subplots.make_subplots(
        subplot_titles=["Insurance fund balance"], specs=[[{"secondary_y": True}]]
    )
    fig.add_trace(go.Scatter(x=plot_df.time, y=plot_df.treasury, name="Insurance fund"))
    fig.add_trace(
        go.Scatter(
            x=plot_df.time,
            y=plot_df.price,
            name="Luna price",
            line=dict(color="rgba(0, 0, 0, 0.21)"),
        ),
        secondary_y=True,
    )

    fig.update_layout(legend=dict(yanchor="top", y=0.99, xanchor="left", x=0.01))
    return fig


def plot_la_exposure(plot_df: pd.DataFrame) -> go.Figure:
    fig = subplots.make_subplots(
        subplot_titles=["Leverage Agent Position Exposure"],
        specs=[[{"secondary_y": True}]],
    )
    fig.add_trace(go.Scatter(x=plot_df.time, y=plot_df.LA_exposure, name="Insurance fund"))
    fig.add_trace(
        go.Scatter(
            x=plot_df.time,
            y=plot_df.price,
            name="Luna price",
            line=dict(color="rgba(0, 0, 0, 0.21)"),
        ),
        secondary_y=True,
    )

    fig.update_layout(legend=dict(yanchor="top", y=0.99, xanchor="left", x=0.01))
    return fig
