<!-- simulation -->

```python
amt_osmo = 100 # qty of osmo = 100
osmo_price = 10 # price of osmo  = 10
osmo_exposure_0 = amt_osmo * osmo_price # inventory exposure to market for Matrix = 1,000
```

What you want to do is transfer the inventory volatility to the agents (leverage and insurance). Imagine a **one to one mapping, which means the entire exposure is covered by leverage**.

#### Analysis of different scenarios (100% down, 50% down, 25% down, 0% , 25% up, 50% up, 100% up)

- Q: The Matrix protocol has how much collateral from LAs and IAs when the market goes down by 50%?
LA = 0 = 0
IA = 0 = 0

- Q: The Matrix protocol has how much collateral from LAs and IAs when the market goes up by 50%?
LA = 100 = 100
IA = 0 = 0

We need to create `LA` and `IA`. We have to say how much collateral we want to take at what price and see what the number from LAs should be and write it. We want to understand the coverage ratio (CR).  
LA = ?? (qty)  
IA = ?? (qty)

Base case - Collateral ratio of 150%.
CR = 150 % implies LA = 50 and IA = 0.
- If `osmo_price` goes up from \$10 to \$15, the protocol gains \$500 in OSMO exposure.
- If `osmo_price` goes down from \$10 to \$5, the protocol loses \$500 in OSMO exposure since `amt_osmo` is 100. LA = 50 implies that the LAs cover `50 * $10 = $500` of exposure, thus the protocol will liquidate the LAs and take their collateral to remain collateralized. Hence, `LA = 0`.

This is problematic. We won't have many people come to the protocol because the change is too drastic in the event of price decreases for the underlying collateral. A tradeoff of huge gains and huge losses makes providing liquidity as an LA similar to an all-in in poker or a coin toss. 

This is where the derivatives come in. Seasoned traders in financial markets will almost never take a fair coin toss ($\pm100\%$), so we can offer a more reasonable tradeoff risk-reward balance like $\pm20\%$. 

How this can happen is that people pay in funding rate. By paying "the shorts" a certain amount in funding payments, you reduce your potential long payoff while still keeping the potential to make a nice profit. Now, you can take larger "bets" (cover more collateral) without having to worry about losing all of your assets.

Let's say that Matrix follows in the footsteps of Perpetual protocol, where the LAs pay the funding payments in a bull market and be paid funding payments in a bear market.

The LAs p&l in a bull market (price up):
```la_bull_pnl = long_leverage_pnl - funding_payments``` 

In a bear market (price down), 
```la_bear_pnl = funding_payments - long_leverage_pnl``` 

In the bear case, the funding payments offset the losses from the fact that the LAs collateral value is decreasing. 

#### Suppose the funding rate is 60 basis points.

A **basis point** is a term used in finance to refer to an incrememnt of 0.01%, e.g. 5 basis points means 0.05%. If an interest rate increases from 5% to 5.25%, that represents an upward move of 25 basis points.
- You could think of "basis points" as an operator that multiplies the number by `1e-4`.

Let the LA OSMO exposure be \$100M with 24 funding payments per day (hourly). Then, 
```python
la_osmo_exposure = 100e6
funding_rate = 60 * 1e-4
daily_epochs = 24
daily_funding = funding_rate * daily_epochs 
# == 0.144

la_osmo_revenue = la_osmo_exposure * daily_funding 
# == 1.2e7
```

Increasing the number of daily epochs and, thus, funding frequency lessens the impact of the collateral volatility on the protocol. This funding rate piece solves the scaling issue that we would've had for handling, say, \$10 billion in the protocol. Without this short hedge, more collateral in the protocol would mean more impact from volatility on liquidations and PnL. 
##### funding rate payment is freq in a bearish regime

for the short —> Insurance fund () + leverage agents pay funding rate
funding_rate_LA = funding_rate = 5E-3  
funding_rate_IF = 10 * 1e-4 # this number has to be higher and higher = 1E-3

- All the fees in the protocol outside of the derivatives platform goes to the Insurance fund to set the funding rate
- we solved this assuming 1:1 mapping CV : LA
- CV will be divided between LA and IA (Incentive pendulum based on fees )

References: 
- Basis Points. Investopedia. https://www.investopedia.com/terms/b/basispoint.asp
- . 

---

# Solution for the Matrix Simulator

Date: 2022-02-10


Derivatives Ex. 1

Start 
- There's 10000 OSMO at a price of $10.
- Mint —> USDM 
- Matrix = 100,000 USD vol
- LA = 10,000 USD —> leverage = 10

The main incentive that an LA has to come to Matrix is that they can take a long  perp position with zero funding rate.


OSMO goes to 15 $
- PROTOCOL PNL : 50,000 USD
- LA : 10,000 -> 100% pnl
- IF =  40,000 —> incentive pendulum —> governance token would vote for the split

IA : over-collateteralize —> yield from the matrix protocol….

Market goes down the LAs would not come to the protocol…..

Derivative protocol : 

IF would take the short side and pay the LAs a funding rate in the down-term 

Every hour there is a funding rate of 100 bps —> IF pays that (Matrix protocol) —> funding rate —> funding rate model —> make the LAs lose 30-40% of their money rather than 100%

OSMO price = 9$

Protocol exposure = 90,000$

10% -> 6 hours 

t_1 = 100,000 * 100 * 1e-4 = 1000
t_2 = 1000
….
t_6 = 1000

LA_losses = 4000 or 40%

#### Extension for clarification

Let:

```python
"""
Args:
    protocol_exposure (float): Collateral exposure of Matrix in USD.
    amt_stables_mintes (float): number of stable coins minted
"""
```

1:1 mapping ⇔ LAs cover all of the protocol volatility

```python
"""
Args:
    leverage_multiplier (float): # ex. 10 
    la_exposure (float): USD exposure brought by LAs
    funding_hourly (float): Hourly funding payment from the Insurance Fund (IF) 
        based on the funding rate.
"""
```

If:
- Total supply = 1 billion
- IF = 8% of the total supply at genesis


Bullish scenario 

1. What is the pnl of the LA and the protocol

Bearish scenario 

2. What is the pnl for the LA, funding rate payments by the IF

3. If the bearish regime lasts what is the time before the IF goes bankrupt (days)k


---


Leverage mult is a linear function parameterized by two variables: $\ell \sim \eta(c_{LA}, c_{cover})$ 
Similarly, la_position_value: $V \sim f(c_{LA}, c_{cover})$, which means I should be able to write $V\sim f(c_{LA}, \eta)$ or $V\sim f(\eta, c_{cover})$.


$$\begin{align}
\ell &= \frac{c_{\text{cov}}}{c_{LA}}  \quad\quad \eta = \frac{c_{\text{cov}} + c_{LA}}{c_{\text{cov}}}\\ 
\psi &= c_{\text{cov}} \cdot (\%\Delta_p) + c_{LA} \\
\psi &= c_{\text{cov}}  \cdot \left(1 - \frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{\text{cov}}\cdot 1  -  c_{\text{cov}} \cdot \left(\frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{\text{cov}} + c_{LA}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) \\
  &= \eta \cdot c_{\text{cov}}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) \\
  &= c_{\text{cov}} \left(\eta - \frac{p_i}{p_f} \right) \\
\end{align}$$

$$\begin{align}
\ell &= \frac{c_{\text{cov}}}{c_{LA}}  \quad\quad \eta = \frac{c_{\text{cov}} + c_{LA}}{c_{\text{cov}}}\\ 
\psi &= c_{\text{cov}}  \cdot \left(1 - \frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{\text{cov}} + c_{LA}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) \\
  &= c_{\text{cov}}\cdot 1  -  c_{\text{cov}} \cdot \left(\frac{p_i}{p_f}\right) + c_{LA} \\
  &= \ell\cdot c_{LA}  -  \ell\cdot c_{LA} \cdot \left(\frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{LA} \left[\ell\left(1 - \frac{p_i}{p_f}\right) + 1\right] \\
  &\boxed{\psi = c_{LA} \left[\ell\cdot \%\Delta_p + 1\right] }\\
\end{align}$$