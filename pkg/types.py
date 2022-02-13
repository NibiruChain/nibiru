#!/usr/bin/env python
import dataclasses
from typing import Dict, Sequence

@dataclasses.dataclass
class SpeculativeAssetState:
    amt: int
    price_usd: float

@dataclasses.dataclass
class ProtocolState:
    """[summary]

    Args:
        LA_amt: Collateral provided to Matrix by the leverage agents.
        IA_amt: Collateral provided to Matrix by the insurance agents.
        IF_amt: Collateral provided to Matrix by the insurance fund.

    Attributes:
        LA_amt: Collateral provided to Matrix by the leverage agents.
        IA_amt: Collateral provided to Matrix by the insurance agents.
        IF_amt: Collateral provided to Matrix by the insurance fund.

    Properties:
        total_amt: Total amount of collateral tokens in the protocol.
        pcts (Dict[str, float]): Mapping from "LA", "IA", and "IF" to the 
            percentage of the 'total_amt' collateral.
    
    Methods: 
        update: Updates the protocol state with new values for "LA_amt", 
            "IA_amt", and "IF_amt".
    """

    LA_amt: float
    IA_amt: float
    IF_amt: float
    frate_to_LA: int
    frate_to_IF: int

    def update_amts(self, LA_amt=None, IA_amt=None, IF_amt=None) -> None:
        """Update the asset amounts of the protocol."""
        if LA_amt is not None:
            self.LA_amt = LA_amt
        if IA_amt is not None:
            self.IA_amt = IA_amt
        if IF_amt is not None:
            self.IF_amt = IF_amt
    
    def update_frates(self, frate_to_LA=None, frate_to_IF=None) -> None:
        """Update the funding rates of the protocol."""
        if frate_to_LA is not None:
            self.frate_to_LA = frate_to_LA
        if frate_to_IF is not None:
            self.frate_to_IF = frate_to_IF

    @property
    def total_amt(self) -> float:
        """Total amount of collateral tokens in the protocol."""
        return sum([self.LA_amt, self.IA_amt, self.IF_amt])

    @property
    def pcts(self) -> Dict[str, float]:
        """
        Returns: 
            (Dict[str, float]): Mapping from "LA", "IA", and "IF" to the 
                percentage of the 'total_amt' collateral.
        """
        LA_pct = self.LA_amt / self.total_amt
        IA_pct = self.IA_amt / self.total_amt
        IF_pct =  self.IF_amt / self.total_amt
        return dict(LA=LA_pct, IA=IA_pct, IF=IF_pct)

@dataclasses.dataclass
class LeveragedPosition:
    """A liquidity position held by a leverage agent (LA)
    
    Args:
        c_LA (float): Collateral the LA brings to the protocol.
        price_pct_change (float): Percent change is 
            (price_current - price_initial) / price_current
        c_cover (float): Collateral the agent chooses to cover.
        value (float): Leveraged position value in units of the underlying 
            asset.
    """
    c_LA: float
    price_pct_change: float
    c_cover: float
    value: float
    
    def __post_init__(self):
        if any([c < 0 for c in [self.c_LA, self.c_cover]]):
            raise ValueError("Invalid value passed") # TODO: better error msg

        self.leverage_mult = self.c_cover / self.c_LA
