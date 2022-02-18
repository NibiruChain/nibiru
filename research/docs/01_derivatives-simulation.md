<!-- simulation -->

# Intro meeting

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


### Assume there's no derivatives protocol in Matrix. 

#### Example.start()

- `amt_osmo` is 10000 and `osmo_price` is \$10.
- When users deposit OSMO (collateral) into Matrix, they mint USDM.
- Matrix has \$100,000 USD vol because `protocol_exposure = amt_osmo * osmo_price`. Here, protocol volatility and protocol exposure are used synonymously.
- Suppose that LAs bring \$10,000 in collateral exposure and choose to cover 100\% of protocol's volatility. This would imply that LAs have 10x leverage, or a `leverage_mult` of 10.


The main incentive that an LA has to come to Matrix is that they can take long leveraged positions. LAs essentially have a long perpetual position with zero funding payment costs.

#### Example.case1(): `osmo_price` goes to 15 $

`osmo_price = 15`

- Matrix has of 10k OSMO (`amt_osmo` ). Since price increased \$5, the protocol exposure has increased by $50k. In other words, the protocol profit is \%50.
- Because the `la_exposure` was $\$10$k at a price of \$10, `la_amt_osmo` is 1000. With a price increase of \$5, $\Delta_{\text{pct\_p}}$ is 50\% (0.5). The LAs' leveraged position value is now
$$ \psi = c_{LA} \left[\ell \cdot \Delta_{\text{pct\_p}} + 1\right] 
  = 1000\left[ 10\cdot 0.5 + 1\right] = 6000 \text{ OSMO}.$$
  Since the LAs started with 1000 OSMO, they have an unrealized profit of 5000 OSMO, or \$25k, implying a profit of 250\% in USD, which is 500\% in OSMO.
  - Notice that this means the LAs receive all of the protocol profits if they exit this leveraged position.
- IF =  40,000 —> incentive pendulum —> governance token would vote for the split

IAs receive yield from the matrix protocol when it's over-collateteralized.

If the market (price of the collateral) goes down, new LAs would not want to come to the protocol.

Derivative protocol : 
- IF would take the short side and pay the LAs a funding rate in the down-term 
- Every hour there is a funding rate of 100 bps —> IF pays that (Matrix protocol) —> funding rate —> funding rate model —> make the LAs lose 30-40% of their money rather than 100%

#### `osmo_price` goes to 9$ over the course of 6 hours.
- Protocol exposure = 90,000$

10% -> 6 hours 
```
from typing import Sequence

amt_osmo = 10e3
osmo_prices: Sequence[float] = np.linspace(10, 9, 6)
matrix_osmo_exposure: Sequence[float] = osmo_prices * amt_osmo
funding_payments = [basis_points(100) * expo for expo in matrix_osmo_exposure]
```

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

Let:
- Total supply = 1 billion
- IF = 8% of the total supply at genesis


Bullish scenario 

1. What is the pnl of the LA and the protocol

Bearish scenario 

2. What is the pnl for the LA, funding rate payments by the IF

3. If the bearish regime lasts what is the time before the IF goes bankrupt (days)k


---

# Derivation of re-parameterized LA position equation

Leverage mult is a linear function (call it $f$) parameterized by two variables:  
$$\ell \sim f(c_{LA}, c_{cover}).$$
Similarly, `la_position_value` is a linear function (call it $\phi$) that can be parmaterized by the same variables:
$$\psi \sim \phi(c_{LA}, c_{cover}).$$
This means we should be able to write 
$$\psi \sim f(c_{LA}, \phi) \quad \text{ or }\quad \psi \sim f(\phi, c_{cover}).$$

Let $\boxed{\ell := \dfrac{c_{\text{cov}}}{c_{LA}} }$, $\boxed{\phi := \dfrac{c_{\text{cov}} + c_{LA}}{c_{\text{cov}}} }$, and $\Delta_{\text{pct\_p}} = \dfrac{p_f - p_i}{p_f} = \left(1 - \dfrac{p_i}{p_f}\right)$.
$$\begin{align}
\psi &= c_{\text{cov}} \cdot \Delta_{\text{pct\_p}} + c_{LA}  \\
  &= c_{\text{cov}}  \cdot \left(1 - \frac{p_i}{p_f}\right) + c_{LA} 
    = c_{\text{cov}}\cdot 1  -  c_{\text{cov}} \cdot \left(\frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{\text{cov}} + c_{LA}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) \\
  &= \phi \cdot c_{\text{cov}}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) \\
  & \therefore \quad \boxed{ \psi = c_{\text{cov}} \left(\phi - \frac{p_i}{p_f} \right) } \\
\end{align}$$

We can similarly derive the leveraged position value, $\psi$ in terms of $\ell$. Starting again from equation 1, 

$$\begin{align}
\psi &= c_{\text{cov}} \cdot \Delta_{\text{pct\_p}} + c_{LA}
    = c_{\text{cov}}  \cdot \left(1 - \frac{p_i}{p_f}\right) + c_{LA} \\
  &= c_{\text{cov}} + c_{LA}  -  c_{\text{cov}} \left(\frac{p_i}{p_f}\right) 
    = c_{\text{cov}}  -  \left[c_{\text{cov}} \left(\frac{p_i}{p_f}\right)\right] + c_{LA} \\
  &= \ell\cdot c_{LA}  -  \ell\cdot c_{LA} \cdot \left(\frac{p_i}{p_f}\right) + c_{LA} 
    = c_{LA} \left[\ell\left(1 - \frac{p_i}{p_f}\right) + 1\right] \\
  &\boxed{\psi = c_{LA} \left[\ell \cdot \Delta_{\text{pct\_p}} + 1\right] }\\
\end{align}$$


---

#  Initial Protocol Simulation Presentation

Date: 2022-02-13

Simulator presentation.

Input parameters: 

1. Choose the implied leverage for the LA
1. Every hour there is a funding payment based on whether that collateral price has increased or decreased. 
    - If price increases (bull), the funding payment goes LA → IF. 
    - If price decreases, the funding payment goes IF → LA.
2. Calculate the P&L of the LA (assuming aggregate agents) and the IF.
3. (Extension) IF should charge higher funding rates based on implied leverage of the LAs.


Next feature - Stablecoin minting and burning

- 3 lockup periods for the stablecoin: 1 hour, 1 day, 1 week 
1. Choose the "amount of stable coin" to be minted at the "Collateral value"
1. Show the different profits based on the duration of how long the stablecoins were in circulation.
