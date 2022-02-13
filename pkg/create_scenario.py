import pandas as pd
import numpy as np
from pkg.stochastic import Brownian
import dataclasses
import seaborn as sns
from matplotlib import pyplot as plt


@dataclasses.dataclass
class Parameters:
    # Simulation parameter
    n_periods: float

    # Protocol parameter
    exit_fee: float
    entry_fee: float
    hourly_funding_rate: float
    initial_insurance_fund: float

    # LA parameters
    n_LAs_position_per_period: float
    size_LAs_position_gamma_parameters: float
    leverages_poisson_parameter: float

    take_profit_chance: float
    take_loss_chance: float

    # Price parameters
    s0: float
    mu: float
    sigma: float

    dt: float


def create_scenario(parameters: Parameters):

    b = Brownian()
    df = pd.DataFrame(
        zip(
            *[
                pd.date_range("2020-01-01", periods=parameters.n_periods, freq="h"),
                b.stock_price(
                    parameters.s0,
                    parameters.mu,
                    parameters.sigma,
                    parameters.n_periods,
                    parameters.dt,
                ),
            ]
        ),
        columns=["time", "price"],
    )

    column_names = ["bet_size", "leverage", "price"]
    current_positions = pd.DataFrame(columns=column_names)

    for i, row in df.iterrows():
        # Add the new positions to the protocol
        # Each position size is taken from a gamma distribution defined by the parameters
        # Each leverage is taken from a poisson distribution
        new_positions = pd.DataFrame(
            np.vstack(
                [
                    np.random.gamma(
                        *parameters.size_LAs_position_gamma_parameters,
                        parameters.n_LAs_position_per_period,
                    ),
                    np.random.poisson(
                        parameters.leverages_poisson_parameter,
                        parameters.n_LAs_position_per_period,
                    ),
                    [row["price"]] * parameters.n_LAs_position_per_period,
                ]
            ).T,
            columns=column_names,
        )
        """
        New position:

        bet_size    |   leverage    |   entry price
        -------------------------------------------
        100k        |   5           |   4.2
        150k        |   8           |   4.2

        """

        current_positions = pd.concat([current_positions, new_positions])

        # Compute all exits
        liquidations = (
            current_positions.bet_size
            * current_positions.leverage
            * (row["price"] - current_positions.price)
            + current_positions.bet_size
            < 0
        )

        exits_loss = (row["price"] - current_positions.price < 0) & (
            np.random.random(len(current_positions)) < parameters.take_loss_chance
        )

        exits_profit = (row["price"] - current_positions.price > 0) & (
            np.random.random(len(current_positions)) < parameters.take_profit_chance
        )

        entry_fee = (
            new_positions[["bet_size", "leverage"]].product(axis=1).sum()
            * parameters.entry_fee
        )

        exit_fee = (
            current_positions.loc[
                (liquidations | exits_loss | exits_profit), ["bet_size", "leverage"]
            ]
            .product(axis=1)
            .sum()
            * parameters.exit_fee
        )

        # remove exits from active positions
        current_positions = current_positions[
            ~(liquidations | exits_loss | exits_profit)
        ].copy()

        # funding rate calculation
        positive_unrealized_pnl = current_positions.price > row["price"]
        negative_unrealized_pnl = current_positions.price < row["price"]

        funding_rate_income = (
            current_positions.loc[positive_unrealized_pnl, ["bet_size", "leverage"]]
            .product(axis=1)
            .sum()
            - current_positions.loc[negative_unrealized_pnl, ["bet_size", "leverage"]]
            .product(axis=1)
            .sum()
        ) * parameters.hourly_funding_rate

        df.loc[i, "fees"] = entry_fee + exit_fee
        df.loc[i, "funding_rate"] = funding_rate_income

        df.loc[i, "liquidations"] = liquidations.sum()
        df.loc[i, "exits"] = (exits_profit | exits_loss).sum()

    df.loc[0, "treasury"] = parameters.initial_insurance_fund
    df["treasury"] = (df.treasury.fillna(0) + df.fees + df.funding_rate).cumsum()
    return df
