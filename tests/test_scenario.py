import unittest

import pandas as pd
import xmlrunner
from pkg import scenario
from pkg import simulation as sim
from pkg import types


class TestScenario(unittest.TestCase):
    """
    Unit tests for the scenario module.
    """

    def setUp(self):
        self.la_parameters = scenario.LeverageAgentParams(
            num_LA_positions_per_period=100,
            position_size_gamma_params=[3, 10_000],
            poisson=100,
        )

    def test_get_funding_payment(self):
        protocol = types.ProtocolState(
            LA_amt=200, IA_amt=100, IF_amt=400, frate_to_IF=50, frate_to_LA=100
        )

        # Test bear market
        self.assertAlmostEqual(
            scenario.get_funding_payment(protocol=protocol, price=10, bull=True),
            protocol.frate_to_IF * 1e-4 * protocol.LA_amt * 10,
        )

        # Test bull market
        self.assertAlmostEqual(
            scenario.get_funding_payment(protocol=protocol, price=10, bull=False),
            protocol.frate_to_LA * protocol.IF_amt * 10 * 1e-4,
        )

    def test_get_new_positions(self):

        generated_dataframe = scenario.get_new_positions(
            params=self.la_parameters, price=10
        )

        self.assertEqual(
            generated_dataframe.shape[0], self.la_parameters.num_LA_positions_per_period
        )

        self.assertCountEqual(
            generated_dataframe.columns, ["collateral_brought", "leverage", "price"]
        )

    def test_get_new_exits(self):
        current_positions = scenario.get_new_positions(
            params=self.la_parameters, price=10
        )

        scenario_parameters = scenario.ProtocolParams(
            exit_fee=100,
            entry_fee=100,
            frate_to_LA=100,
            frate_to_IF=100,
            IF_exposure_init=100,
            take_profit_chance=1,
            take_loss_chance=1,
        )

        exits_loss, exits_profit = scenario.get_new_exits(
            scenario_parameters, current_positions, 8,
        )

        self.assertCountEqual(exits_loss.tolist(), [True] * len(current_positions))
        self.assertCountEqual(exits_profit.tolist(), [False] * len(current_positions))

        exits_loss, exits_profit = scenario.get_new_exits(
            scenario_parameters, current_positions, 12,
        )

        self.assertCountEqual(exits_loss.tolist(), [False] * len(current_positions))
        self.assertCountEqual(exits_profit.tolist(), [True] * len(current_positions))


if __name__ == "__main__":

    unittest.main(
        testRunner=xmlrunner.XMLTestRunner(output="unit-test-reports"),
        # these make sure that some options that are not applicable
        # remain hidden from the help menu.
        failfast=False,
        buffer=False,
        catchbreak=False,
    )
