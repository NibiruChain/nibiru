import pandas as pd
import numpy as np
from pkg.stochastic import Brownian
import dataclasses
import seaborn as sns
from matplotlib import pyplot as plt


@dataclasses.dataclass
class Parameters:
    # Simulation parameter
    # n_periods: float

    # Protocol parameter
    exit_fee: float
    entry_fee: float
    hourly_funding_rate_payout: float
    initial_insurance_fund: float
    hourly_funding_rate_fee: float

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

    if False:
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

    df = pd.read_excel("data.xlsx", sheet_name="raw")
    df["time"] = pd.to_datetime(df.Date)
    df["price"] = df["Luna"]

    df = (
        df[df.price.notnull()][["time", "price"]]
        .sort_values("time")
        .reset_index(drop=True)
    )

    parameters.n_periods = df.shape[0]

    column_names = ["bet_size", "leverage", "price"]
    current_positions = pd.DataFrame(columns=column_names)

    insurance_fund = parameters.initial_insurance_fund

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
                    )
                    / row["price"],
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
            + current_positions.bet_size * row["price"]
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
        df.loc[i, "fees"] = (entry_fee + exit_fee) * row["price"]
        insurance_fund += (entry_fee + exit_fee) * row["price"]

        # remove exits from active positions
        current_positions = current_positions[
            ~(liquidations | exits_loss | exits_profit)
        ].copy()

        # funding rate calculation
        positive_unrealized_pnl = current_positions.price > row["price"]
        negative_unrealized_pnl = current_positions.price < row["price"]

        # This funding rate is if it's depending on the current insurance fund
        funding_rate_payout = insurance_fund * parameters.hourly_funding_rate_payout
        # funding_rate_income = 2 * funding_rate_payout
        funding_rate_income = (
            current_positions.loc[positive_unrealized_pnl, ["bet_size", "leverage"]]
            .product(axis=1)
            .sum()
            * parameters.hourly_funding_rate_fee
            * row["price"]
        )

        insurance_fund -= funding_rate_payout
        insurance_fund += funding_rate_income

        df.loc[i, "treasury"] = insurance_fund

        df.loc[i, "funding_rate_payout"] = funding_rate_payout
        df.loc[i, "funding_rate_income"] = funding_rate_income
        df.loc[i, "entry_fee_income"] = entry_fee * row["price"]
        df.loc[i, "exit_fee_income"] = exit_fee * row["price"]

        df.loc[i, "liquidations"] = liquidations.sum()
        df.loc[i, "exits"] = (exits_profit | exits_loss).sum()

    return df
