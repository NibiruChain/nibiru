import dataclasses
from typing import List

import numpy as np
import pandas as pd
import seaborn as sns
from matplotlib import pyplot as plt

from research.pkg import simulation as sim
from research.pkg import stochastic, types


@dataclasses.dataclass
class ProtocolParams:
    exit_fee: float
    entry_fee: float
    mint_fee: float
    burn_fee: float
    frate_to_LA: int
    frate_to_IF: int
    IF_exposure_init: float
    take_profit_chance: float
    take_loss_chance: float
    initial_sc_supply: float


@dataclasses.dataclass
class GammaParameters:
    shape: float
    scale: float


@dataclasses.dataclass
class LeverageAgentParams:
    num_LA_positions_per_period: float
    position_size_gamma_params: GammaParameters
    poisson: float


@dataclasses.dataclass
class StableCoinSeekersParams:
    mint_gamma_params: GammaParameters
    burn_gamma_params: GammaParameters


@dataclasses.dataclass
class StochasticProcessParams:
    s0: float
    mu: float
    sigma: float
    dt: float


@dataclasses.dataclass
class Parameters:
    protocol: ProtocolParams
    LA: LeverageAgentParams
    sseeker: StableCoinSeekersParams
    stochastic: StochasticProcessParams


def get_funding_payment(
    protocol: types.ProtocolState, price: float, bull: bool
) -> float:
    """[summary]

    Args:
        protocol (types.ProtocolState): [description]
        price (float): [description]

    Returns:
        float: [description]
    """

    funding_rate_bps: int
    if bull:
        # LAs pay the IF in bull
        funding_rate_bps = protocol.frate_to_IF
        spec_amt = protocol.LA_amt
    else:
        # IF pays the LA in bear
        funding_rate_bps = protocol.frate_to_LA
        spec_amt = protocol.IF_amt

    funding_payment: float = sim.compute_funding_payment(
        bps=funding_rate_bps,
        spec=types.SpeculativeAssetState(amt=spec_amt, price_usd=price),
    )
    return funding_payment


def get_new_positions(params: LeverageAgentParams, price: float) -> pd.DataFrame:
    # Add the new positions to the protocol
    # Each position size is taken from a gamma distribution defined by the parameters
    # Each leverage is taken from a poisson distribution
    """
    New position:

    collateral_brought    |   leverage    |   entry price
    -------------------------------------------
    10k        |   5           |   4.2
    15k        |   8           |   4.2

    """

    return pd.DataFrame(
        np.vstack(
            [
                np.random.gamma(
                    shape=params.position_size_gamma_params.shape,
                    scale=params.position_size_gamma_params.scale,
                    size=params.num_LA_positions_per_period,
                )
                / price,
                np.random.poisson(params.poisson, params.num_LA_positions_per_period,),
                [price] * params.num_LA_positions_per_period,
            ]
        ).T,
        columns=["collateral_brought", "leverage", "price"],
    )


def create_price_dataframe(
    use_luna_data: bool = True, brownian_parameters: StochasticProcessParams = None
) -> pd.DataFrame:
    """
    Read the data from Luna excel file stored in the repository

    Args:
        use_luna_data (bool): Whether to read the data from the Luna data file or create new brownian process for the 
            price. True by default.
        brownian_parameters (StochasticProcessParams): The parameters to use for the brownian process. Required only if
            use_luna_data is set to False.

    Returns:
        pd.DataFrame: The pandas fiel with the data ready to be used
    """

    if use_luna_data:

        price_dataframe = pd.read_excel("data.xlsx", sheet_name="raw")
        price_dataframe["time"] = pd.to_datetime(price_dataframe.Date)
        price_dataframe["price"] = price_dataframe["Luna"]

    else:
        brownian = stochastic.Brownian()
        price_dataframe = pd.DataFrame(
            zip(
                *[
                    pd.date_range(
                        "2020-01-01", periods=brownian_parameters.n_periods, freq="h"
                    ),
                    brownian.stock_price(
                        brownian_parameters.stochastic.s0,
                        brownian_parameters.stochastic.mu,
                        brownian_parameters.stochastic.sigma,
                        brownian_parameters.n_periods,
                        brownian_parameters.stochastic.dt,
                    ),
                ]
            ),
            columns=["time", "price"],
        )

    return (
        price_dataframe[price_dataframe.price.notnull()][["time", "price"]]
        .sort_values("time")
        .reset_index(drop=True)
    )


def create_scenario(params: Parameters):
    """
    Create a scenario based on the price defined by the parameters

    Args:
        params (Parameters): Simulation parameters

    """

    price_dataframe = create_price_dataframe()

    current_positions: pd.DataFrame = pd.DataFrame(
        columns=["collateral_brought", "leverage", "price"]
    )
    insurance_fund: float = params.protocol.IF_exposure_init
    simulation_result: pd.DataFrame = run_simulation(
        params, price_dataframe, current_positions, insurance_fund
    )

    return simulation_result


def get_stablecoinseeker_fees(params: Parameters, current_sc_supply: float):
    """
    Compute the mint and burn fee to pay to the insurance fund and update the current stablecoin supply.

    Args:
        params (Parameters): The parameters for the scenario
        current_sc_supply (float): The current supply for stablecoins

    Returns:
        tuple: The mint fee, burn fee and the updated current stablecoin supply
    """
    mint = np.random.gamma(
        scale=params.sseeker.mint_gamma_params.scale,
        shape=params.mint_gamma_params.shape,
    )
    burn = (
        np.random.gamma(
            scale=params.sseeker.burn_gamma_params.scale,
            shape=params.burn_gamma_params.shape,
        )
        * current_sc_supply
    )

    current_sc_supply += mint - burn

    return (
        mint * params.protocol.mint_fee,
        burn * params.protocol.burn_fee,
        current_sc_supply,
    )


def run_simulation(
    params: Parameters,
    price_dataframe: pd.DataFrame,
    current_positions: pd.DataFrame,
    insurance_fund: float,
):
    previous_price: float = -1
    current_sc_supply: float = params.protocol.initial_sc_supply

    for i, row in price_dataframe.iterrows():
        price = row["price"]
        new_positions = get_new_positions(params=params.LA, price=price)

        current_positions = pd.concat([current_positions, new_positions])

        # Compute all exits
        liquidations = get_new_liquidations(current_positions, price)
        exits_loss, exits_profit = get_new_exits(
            params.protocol, current_positions, price
        )

        entry_fee, exit_fee = compute_fees(
            params.protocol,
            current_positions,
            new_positions,
            price,
            liquidations,
            exits_loss,
            exits_profit,
        )

        mint_fee, burn_fee, current_sc_supply = get_stablecoinseeker_fees(
            params, current_sc_supply
        )

        # Update the simulation
        insurance_fund += entry_fee + exit_fee + mint_fee + burn_fee

        # remove exits from active positions
        current_positions = current_positions[
            ~(liquidations | exits_loss | exits_profit)
        ].copy()

        LA_amt = (
            current_positions[["collateral_brought", "leverage"]].product(axis=1).sum()
            * price
        )
        protocol = types.ProtocolState(
            LA_amt=LA_amt,
            IF_amt=insurance_fund,
            frate_to_LA=params.protocol.frate_to_LA,
            frate_to_IF=params.protocol.frate_to_IF,
            IA_amt=0,
        )

        # TODO: Are we allowed to take the money from the LAs collaterals?
        # We assume that they pay it from an infinite wallet for now
        bull: bool = price > previous_price
        funding_payment = get_funding_payment(protocol=protocol, price=price, bull=bull)
        if bull:
            insurance_fund += funding_payment
        else:
            insurance_fund -= funding_payment

        updates = {
            "treasury": insurance_fund,
            "entry_fee_income": entry_fee * price,
            "exit_fee_income": exit_fee * price,
            "fees": entry_fee + exit_fee,
            "funding_payments": funding_payment,
            "bull": bull,
            "liquidations": liquidations.sum(),
            "exits": (exits_profit | exits_loss).sum(),
            "LA_exposure": (
                current_positions.collateral_brought * price  # 10_000
            ).sum(),
            "LA_position": (
                current_positions.collateral_brought  # 10_000
                * current_positions.leverage  # 10
                * price
            ).sum(),
        }

        for key, value in updates.items():
            price_dataframe.loc[i, key] = value

        previous_price = price

    return price_dataframe.copy()


def compute_fees(
    params: ProtocolParams,
    current_positions: pd.DataFrame,
    new_positions: pd.DataFrame,
    price: float,
    liquidations: pd.Series,
    exits_loss: pd.Series,
    exits_profit: pd.Series,
) -> tuple:
    """
    Compute the fees collected for entry and exits of positions.

    Args:
        params (ProtocolParams): The parameters of the simulation
        current_positions (pd.DataFrame): Current positions active in the protoocl
        new_positions (pd.DataFrame): New positions for the epoch
        price (float): Current price of the collateral
        liquidations (pd.Series): Series with the new liquidations
        exits_loss (pd.Series): Series with the new exits_loss
        exits_profit (pd.Series): Series with the new exits_profit

    Returns:
        tuple: Entry and exit fees
    """
    entry_fee = (
        new_positions[["collateral_brought", "leverage"]].product(axis=1).sum()
        * params.entry_fee
    ) * price

    exit_fee = (
        current_positions.loc[
            (liquidations | exits_loss | exits_profit),
            ["collateral_brought", "leverage"],
        ]
        .product(axis=1)
        .sum()
        * params.exit_fee
    ) * price

    return entry_fee, exit_fee


def get_new_exits(
    params: ProtocolParams, current_positions: pd.DataFrame, price: float
) -> tuple:
    """
    Compute the exits because of random events and negatiev or positive pnl. 

    Args:
        params (Parameters): The simulation parameters with the take profit and take 
        current_positions (pd.DataFrame): Current positions active in the protoocl
        price (float): Current price of the collateral

    Returns:
        tuple: The exits loss and exits profit series 
    """
    exits_loss = (price - current_positions.price < 0) & (
        np.random.random(len(current_positions)) < params.take_loss_chance
    )

    exits_profit = (price - current_positions.price > 0) & (
        np.random.random(len(current_positions)) < params.take_profit_chance
    )
    return exits_loss, exits_profit


def get_new_liquidations(current_positions: pd.DataFrame, price: float) -> pd.Series:
    """
    Create the liquidations from the current position based on the price

    Args:
        current_positions (pd.DataFrame): Current positions active in the protoocl
        price (float): Current price of the collateral

    Returns:
        pd.Series: Series of bool specifying wether the position got liquidated.
    """
    return (
        current_positions.collateral_brought
        * current_positions.leverage
        * (price - current_positions.price)
        + current_positions.collateral_brought * price
        < 0
    )
