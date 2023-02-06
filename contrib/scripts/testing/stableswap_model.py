"""
This script geenrates random test for swaps on balancer pools.
The objective is to compare the performance of golang uint256 unit used against Python3 integer values.

Python3 int type have no limit in term of size, which means that we can make arbitrary operations on ints and consider them as float.
The curve class comes directly from the curve codebase, and is being used to create the tests, stored in `x/spot/types/misc/stabletests.csv`;.

Theses test are then used to compare the value of python model's DY against the one obtained with our go code. 
These are created for pools with random amount of assets (from 2 to 5), random amplification parameter (from 1 to 4* common.Precision) and for random coins of the pool.

By computing this, we ensure the validity of our SolveConstantProductInvariant function.
"""

from collections import deque
from itertools import permutations
import os

import csv
import random

DECIMALS = 6

N_TESTS = 200

# From https://github.com/curvefi/curve-contract/blob/master/tests/simulation.py
class Curve:

    """
    Python model of Curve pool math.
    """

    def __init__(self, A, D, n, p=None, tokens=None):
        """
        A: Amplification coefficient
        D: Total deposit size
        n: number of currencies
        p: target prices
        """
        self.A = A  # actually A * n ** (n - 1) because it's an invariant
        self.n = n
        self.fee = 0  # 10**7
        if p:
            self.p = p
        else:
            self.p = [10**18] * n
        if isinstance(D, list):
            self.x = D
        else:
            self.x = [D // n * 10**18 // _p for _p in self.p]
        self.tokens = tokens

    def xp(self):
        return [x * p // 10**18 for x, p in zip(self.x, self.p)]

    def D(self):
        """
        D invariant calculation in non-overflowing integer operations
        iteratively

        A * sum(x_i) * n**n + D = A * D * n**n + D**(n+1) / (n**n * prod(x_i))

        Converging solution:
        $D_{j+1} = D_j \frac{An^n \sum x_i + nD_P}{(n+1)D_P+D_j(An^n-1)}$

        with $D_p = \frac{D_j^{n+1}}{n^n \prod x_i}$
        """
        Dprev = 0
        xp = self.xp()
        S = sum(xp)
        D = S
        Ann = self.A * self.n**self.n
        while abs(D - Dprev) > 1:
            D_P = D
            for x in xp:
                D_P = D_P * D // (self.n * x)
            Dprev = D
            D = (Ann * S + D_P * self.n) * D // ((Ann - 1) * D + (self.n + 1) * D_P)

        return D

    def y(self, i, j, x):
        """
        Calculate x[j] if one makes x[i] = x

        Done by solving quadratic equation iteratively.
        x_1**2 + x1 * (sum' - (A*n**n - 1) * D / (A * n**n)) = D ** (n+1)/(n ** (2 * n) * prod' * A)
        x_1**2 + b*x_1 = c

        x_1 = (x_1**2 + c) / (2*x_1 + b)
        """
        D = self.D()
        xx = self.xp()
        xx[i] = x  # x is quantity of underlying asset brought to 1e18 precision
        xx = [xx[k] for k in range(self.n) if k != j]
        Ann = self.A * self.n**self.n
        c = D
        for y in xx:
            c = c * D // (y * self.n)
        c = c * D // (self.n * Ann)
        b = sum(xx) + D // Ann - D
        y_prev = 0
        y = D
        while abs(y - y_prev) > 1:
            y_prev = y
            y = (y**2 + c) // (2 * y + b)
        return y  # the result is in underlying units too

    def y_D(self, i, _D):
        """
        Calculate x[j] if one makes x[i] = x

        Done by solving quadratic equation iteratively.
        x_1**2 + x1 * (sum' - (A*n**n - 1) * D / (A * n**n)) = D ** (n+1)/(n ** (2 * n) * prod' * A)
        x_1**2 + b*x_1 = c

        x_1 = (x_1**2 + c) / (2*x_1 + b - D)
        """
        xx = self.xp()
        xx = [xx[k] for k in range(self.n) if k != i]
        S = sum(xx)
        Ann = self.A * self.n**self.n
        c = _D
        for y in xx:
            c = c * _D // (y * self.n)
        c = c * _D // (self.n * Ann)
        b = S + _D // Ann
        y_prev = 0
        y = _D
        while abs(y - y_prev) > 1:
            y_prev = y
            y = (y**2 + c) // (2 * y + b - _D)
        return y  # the result is in underlying units too

    def dy(self, i, j, dx):
        # dx and dy are in underlying units
        xp = self.xp()
        return xp[j] - self.y(i, j, xp[i] + dx)

    def exchange(self, i, j, dx):
        xp = self.xp()
        x = xp[i] + dx
        y = self.y(i, j, x)
        dy = xp[j] - y
        fee = dy * self.fee // 10**10
        assert dy > 0
        self.x[i] = x * 10**18 // self.p[i]
        self.x[j] = (y + fee) * 10**18 // self.p[j]
        return dy - fee

    def remove_liquidity_imbalance(self, amounts):
        _fee = self.fee * self.n // (4 * (self.n - 1))

        old_balances = self.x
        new_balances = self.x[:]
        D0 = self.D()
        for i in range(self.n):
            new_balances[i] -= amounts[i]
        self.x = new_balances
        D1 = self.D()
        self.x = old_balances
        fees = [0] * self.n
        for i in range(self.n):
            ideal_balance = D1 * old_balances[i] // D0
            difference = abs(ideal_balance - new_balances[i])
            fees[i] = _fee * difference // 10**10
            new_balances[i] -= fees[i]
        self.x = new_balances
        D2 = self.D()
        self.x = old_balances

        token_amount = (D0 - D2) * self.tokens // D0

        return token_amount

    def calc_withdraw_one_coin(self, token_amount, i):
        xp = self.xp()
        if self.fee:
            fee = self.fee - self.fee * xp[i] // sum(xp) + 5 * 10**5
        else:
            fee = 0

        D0 = self.D()
        D1 = D0 - token_amount * D0 // self.tokens
        dy = xp[i] - self.y_D(i, D1)

        return dy - dy * fee // 10**10


def generate_test_cases(n: int):
    """
    Create n test cases and store them in x/spot/types/misc/stable-swap-math.csv

    Args:
        n (int): The number of test to create
    """

    test_cases = []

    for _ in range(n):
        n_coins = random.randint(2, 5)
        exchange_pairs = deque(permutations(range(n_coins), 2))

        # 10% chance of being 1 (constant product if A=1)
        amplification = random.randint(1, 4_000) if random.random() < 0.9 else 1

        exchange_pair = random.choice(exchange_pairs)
        balances = [random.randint(1, 10e16) for i in range(n_coins)]
        balances_save = balances.copy()

        curve_model = Curve(amplification, balances, n_coins)

        send, recv = exchange_pair

        dx = random.randint(1, balances[recv])
        dy = curve_model.exchange(send, recv, dx)

        test_cases.append(
            [
                balances_save,
                amplification,
                send,
                recv,
                dx,
                dy,
            ]
        )

    file_path = os.path.join(
        os.path.dirname(__file__), "../../x/spot/types/misc/stabletests.csv"
    )

    with open(file_path, "w") as f:
        writer = csv.writer(f)
        writer.writerows(test_cases)


if __name__ == "__main__":
    generate_test_cases(N_TESTS)
