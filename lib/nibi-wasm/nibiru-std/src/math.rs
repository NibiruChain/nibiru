use crate::errors::MathError;
use std::{
    fmt,
    ops::{Add, Div, Mul, Sub},
    str::FromStr,
};

use cosmwasm_std as cw;

// cosmwasm dec from sdk dec
// TODO: cosmwasm dec from sdk int  -> What's the max value
// Decimal

/// Sign: The sign of a number. "Positive" and "Negative" mean strictly
/// postive and negative (excluding 0), respectively.
#[derive(Debug, Default, Copy, Clone, PartialEq, Eq, Ord, PartialOrd)]
pub enum Sign {
    Positive,
    Negative,
    #[default]
    Zero,
}

/// DecimalExt: Implements a signed version of `cosmwasm_std::Decimal`
/// with extentions for generating protobuf type strings.
#[derive(Copy, Clone, Default, PartialEq, Eq, PartialOrd, Ord, Debug)]
pub struct DecimalExt {
    sign: Sign,
    dec: cw::Decimal,
}

impl DecimalExt {
    pub fn zero() -> Self {
        DecimalExt::default()
    }

    /// Getter for `Sign`, which can be +, -, 0.
    pub fn sign(&self) -> Sign {
        self.sign
    }

    /// Getter for the underlying `cosmwasm_std::Decimal`.
    pub fn abc_cw_dec(&self) -> cw::Decimal {
        self.dec
    }

    pub fn add(&self, other: Self) -> Self {
        if self.sign == other.sign {
            return DecimalExt {
                sign: self.sign,
                dec: self.dec.add(other.dec),
            };
        } else if other.dec.is_zero() {
            return *self;
        }

        let self_dec_gt: bool = self.dec.ge(&other.dec);
        let sign = if self_dec_gt { self.sign } else { other.sign };
        let dec = if self_dec_gt {
            self.dec.sub(other.dec) // if abs(self.dec) > abs(other.dec)
        } else {
            other.dec.sub(self.dec) // if abs(self.dec) < abs(other.dec)
        };
        let sign = if dec.is_zero() { Sign::Zero } else { sign };

        DecimalExt { sign, dec }
    }

    pub fn neg(&self) -> Self {
        match self.sign {
            Sign::Positive => DecimalExt {
                sign: Sign::Negative,
                dec: self.dec,
            },
            Sign::Negative => DecimalExt {
                sign: Sign::Positive,
                dec: self.dec,
            },
            Sign::Zero => *self,
        }
    }

    pub fn sub(&self, other: Self) -> Self {
        self.add(other.neg())
    }

    pub fn mul(&self, other: Self) -> Self {
        let dec = self.dec.mul(other.dec);
        let sign = match (self.sign, other.sign) {
            (Sign::Zero, _) | (_, Sign::Zero) => Sign::Zero,
            (Sign::Positive, Sign::Positive)
            | (Sign::Negative, Sign::Negative) => Sign::Positive,
            (Sign::Positive, Sign::Negative)
            | (Sign::Negative, Sign::Positive) => Sign::Negative,
        };
        DecimalExt { sign, dec }
    }

    pub fn quo(&self, other: Self) -> Result<Self, MathError> {
        let sign = match (self.sign, other.sign) {
            (Sign::Zero, _) => Sign::Zero,
            (_, Sign::Zero) => return Err(MathError::DivisionByZero),
            (Sign::Positive, Sign::Positive)
            | (Sign::Negative, Sign::Negative) => Sign::Positive,
            (Sign::Positive, Sign::Negative)
            | (Sign::Negative, Sign::Positive) => Sign::Negative,
        };
        let dec = self.dec.div(other.dec);
        Ok(DecimalExt { sign, dec })
    }
}

impl From<cw::Decimal> for DecimalExt {
    fn from(cw_dec: cw::Decimal) -> Self {
        if cw_dec.is_zero() {
            return DecimalExt::zero();
        }
        DecimalExt {
            sign: Sign::Positive,
            dec: cw_dec,
        }
    }
}

impl FromStr for DecimalExt {
    type Err = MathError;

    /// Converts the decimal string to a `DecimalExt`
    /// Possible inputs: "-69", "-420.69", "1.23", "1", "0012", "1.123000",
    /// Disallowed: "", ".23"
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        let non_strict_sign = if s.starts_with('-') {
            Sign::Negative
        } else {
            Sign::Positive
        };

        let abs_value = if let Some(s) = s.strip_prefix('-') {
            s // Strip the negative sign for parsing
        } else {
            s
        };

        let cw_dec: cw::Decimal =
            cw::Decimal::from_str(abs_value).map_err(|cw_std_err| {
                MathError::CwDecParseError {
                    dec_str: s.to_string(),
                    err: cw_std_err,
                }
            })?;
        let sign = if cw_dec.is_zero() {
            Sign::Zero
        } else {
            non_strict_sign
        };
        Ok(DecimalExt { sign, dec: cw_dec })
    }
}

impl fmt::Display for DecimalExt {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let prefix = if self.sign == Sign::Negative { "-" } else { "" };
        write!(f, "{}{}", prefix, self.dec)
    }
}

/// SdkDec: Decimal string representing the protobuf string for
/// `"cosmossdk.io/math".LegacyDec`.
/// See https://pkg.go.dev/cosmossdk.io/math@v1.2.0#LegacyDec.
pub struct SdkDec {
    protobuf_repr: String,
}

impl SdkDec {
    pub fn new(dec: &DecimalExt) -> Result<Self, MathError> {
        Ok(Self {
            protobuf_repr: dec.to_sdk_dec_pb_repr()?,
        })
    }

    /// Returns the protobuf representation.
    pub fn pb_repr(&self) -> String {
        self.protobuf_repr.to_string()
    }

    pub fn from_dec(dec: DecimalExt) -> Result<Self, MathError> {
        Self::new(&dec)
    }

    pub fn from_cw_dec(cw_dec: cw::Decimal) -> Result<Self, MathError> {
        Self::new(&DecimalExt::from(cw_dec))
    }
}

impl FromStr for SdkDec {
    type Err = MathError;

    /// Converts the decimal string to an `SdkDec` compatible for use with
    /// protobuf strings corresponding to `"cosmossdk.io/math".LegacyDec`
    /// See https://pkg.go.dev/cosmossdk.io/math@v1.2.0#LegacyDec.
    ///
    /// Possible inputs: "-69", "-420.69", "1.23", "1", "0012", "1.123000",
    /// Disallowed: "", ".23"
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        Self::new(&DecimalExt::from_str(s)?)
    }
}

impl fmt::Display for SdkDec {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let dec =
            DecimalExt::from_sdk_dec(&self.pb_repr()).unwrap_or_else(|err| {
                panic!(
                    "ParseError: could not marshal SdkDec {} to DecimalExt: {}",
                    self.protobuf_repr, err,
                )
            });
        write!(f, "{}", dec)
    }
}

impl DecimalExt {
    pub fn precision_digits() -> usize {
        18
    }

    /// to_sdk_dec_pb_repr: Encodes the `DecimalExt` from the human readable
    /// form to the corresponding SdkDec (`cosmossdk.io/math.LegacyDec`).
    pub fn to_sdk_dec(&self) -> Result<SdkDec, MathError> {
        SdkDec::new(self)
    }

    /// to_sdk_dec_pb_repr: Encodes the `DecimalExt` its SdkDec
    /// (`cosmossdk.io/math.LegacyDec`) protobuf representation.
    pub fn to_sdk_dec_pb_repr(&self) -> Result<String, MathError> {
        if self.dec.is_zero() {
            return Ok("0".repeat(DecimalExt::precision_digits()));
        }

        // Convert Decimal to string
        let abs_str = self.dec.to_string();

        // Handle negative sign
        let neg = self.sign == Sign::Negative;

        // Split into integer and fractional parts
        let parts: Vec<&str> = abs_str.split('.').collect();
        let (int_part, frac_part) = match parts.as_slice() {
            [int_part, frac_part] => (*int_part, *frac_part),
            [int_part] => (*int_part, ""),
            _ => {
                return Err(MathError::SdkDecError(format!(
                    "Invalid decimal format: {}",
                    abs_str
                )))
            }
        };

        // Check for valid number format
        if int_part.is_empty() || (parts.len() == 2 && frac_part.is_empty()) {
            return Err(MathError::SdkDecError(format!(
                "Expected decimal string but got: {}",
                abs_str
            )));
        }

        // ----- Build the `sdk_dec` now that validation is complete. -----
        // Concatenate integer and fractional parts
        let mut sdk_dec = format!("{int_part}{frac_part}");

        // Add trailing zeros to match precision
        let precision_digits = DecimalExt::precision_digits();
        if frac_part.len() > precision_digits {
            return Err(MathError::SdkDecError(format!(
                "Value exceeds max precision digits ({}): {}",
                precision_digits, abs_str
            )));
        }
        for _ in 0..(precision_digits - frac_part.len()) {
            sdk_dec.push('0');
        }

        // Add negative sign if necessary
        if neg {
            sdk_dec.insert(0, '-');
        }

        Ok(sdk_dec)
    }

    pub fn from_sdk_dec(sdk_dec_str: &str) -> Result<DecimalExt, MathError> {
        let precision_digits = DecimalExt::precision_digits();
        if sdk_dec_str.is_empty() {
            return Ok(DecimalExt::zero());
        }

        if sdk_dec_str.contains('.') {
            return Err(MathError::SdkDecError(format!(
                "Expected a decimal string but got '{}'",
                sdk_dec_str
            )));
        }

        // Check if negative and remove the '-' prefix if present
        let (neg, abs_str) =
            if let Some(stripped) = sdk_dec_str.strip_prefix('-') {
                (true, stripped)
            } else {
                (false, sdk_dec_str)
            };

        if abs_str.is_empty() || abs_str.chars().any(|c| !c.is_ascii_digit()) {
            return Err(MathError::SdkDecError(format!(
                "Invalid decimal format: {}",
                sdk_dec_str
            )));
        }

        let input_size = abs_str.len();
        let mut decimal_str = String::new();

        if input_size <= precision_digits {
            // Case 1: Purely decimal number
            decimal_str.push_str("0.");
            decimal_str.push_str(&"0".repeat(precision_digits - input_size));
            decimal_str.push_str(abs_str);
        } else {
            // Case 2: Number has both integer and decimal parts
            let dec_point_place = input_size - precision_digits;
            decimal_str.push_str(&abs_str[..dec_point_place]);
            decimal_str.push('.');
            decimal_str.push_str(&abs_str[dec_point_place..]);
        }

        if neg {
            decimal_str.insert(0, '-');
        }

        DecimalExt::from_str(&decimal_str)
    }
}

#[cfg(test)]
mod test_sign_dec {
    use cosmwasm_std as cw;
    use std::str::FromStr;

    use crate::{
        errors::TestResult,
        math::{DecimalExt, Sign},
    };

    #[test]
    fn default_is_zero() -> TestResult {
        assert_eq!(
            DecimalExt::default(),
            DecimalExt {
                sign: Sign::Zero,
                dec: cw::Decimal::from_str("0")?
            }
        );
        assert_eq!(DecimalExt::default(), DecimalExt::zero());
        assert_eq!(DecimalExt::zero(), cw::Decimal::from_str("0")?.into());
        Ok(())
    }

    #[test]
    fn from_cw() -> TestResult {
        assert_eq!(
            DecimalExt::default(),
            DecimalExt::from(cw::Decimal::from_str("0")?)
        );

        let cw_dec = cw::Decimal::from_str("123.456")?;
        assert_eq!(
            DecimalExt {
                sign: Sign::Positive,
                dec: cw_dec
            },
            DecimalExt::from(cw_dec)
        );

        let num = "123.456";
        assert_eq!(
            DecimalExt {
                sign: Sign::Negative,
                dec: cw::Decimal::from_str(num)?
            },
            DecimalExt::from_str(&format!("-{}", num))?
        );

        Ok(())
    }

    // TODO: How will you handle overflow?
    #[test]
    fn add() -> TestResult {
        let test_cases: &[(&str, &str, &str)] = &[
            ("0", "0", "0"),
            ("0", "420", "420"),
            ("69", "420", "489"),
            ("5", "-3", "2"),
            ("-7", "7", "0"),
            ("-420", "69", "-351"),
            ("-69", "420", "351"),
        ];
        for &(a, b, want_sum_of) in test_cases.iter() {
            let a = DecimalExt::from_str(a)?;
            let b = DecimalExt::from_str(b)?;
            let want_sum_of = DecimalExt::from_str(want_sum_of)?;
            let got_sum_of = a.add(b);
            assert_eq!(want_sum_of, got_sum_of);
        }
        Ok(())
    }

    #[test]
    fn neg() -> TestResult {
        let pos_num = DecimalExt::from_str("69")?;
        let neg_num = DecimalExt::from_str("-69")?;
        let zero_num = DecimalExt::zero();

        assert_eq!(neg_num, pos_num.neg());
        assert_eq!(pos_num, neg_num.neg());
        assert_eq!(zero_num, zero_num.neg());
        Ok(())
    }

    #[test]
    fn mul() -> TestResult {
        let test_cases: &[(&str, &str, &str)] = &[
            ("0", "0", "0"),
            ("0", "420", "0"),
            ("16", "16", "256"),
            ("5", "-3", "-15"),
            ("-7", "7", "-49"),
        ];
        for &(a, b, want_product) in test_cases.iter() {
            let a = DecimalExt::from_str(a)?;
            let b = DecimalExt::from_str(b)?;
            let want_product = DecimalExt::from_str(want_product)?;
            let got_product = a.mul(b);
            assert_eq!(want_product, got_product);
        }
        Ok(())
    }

    #[test]
    fn quo() -> TestResult {
        let test_cases: &[(&str, &str, &str)] = &[
            ("0", "420", "0"),
            ("256", "16", "16"),
            ("-15", "5", "-3"),
            ("-49", "-7", "7"),
        ];
        for &(a, b, want_quo) in test_cases.iter() {
            let a = DecimalExt::from_str(a)?;
            let b = DecimalExt::from_str(b)?;
            let want_quo = DecimalExt::from_str(want_quo)?;
            let got_quo = a.quo(b)?;
            assert_eq!(want_quo, got_quo);
        }
        Ok(())
    }

    #[test]
    fn sdk_dec_int_only() -> TestResult {
        let test_cases: &[(&str, &str)] = &[
            // Zero cases should all be equal
            ("0", &"0".repeat(18)),
            ("000.00", &"0".repeat(18)),
            ("0.00", &"0".repeat(18)),
            ("00000", &"0".repeat(18)),
            // Non-zero cases
            ("10", &format!("10{}", "0".repeat(18))),
            ("-10", &format!("-10{}", "0".repeat(18))),
            ("123", &format!("123{}", "0".repeat(18))),
            ("-123", &format!("-123{}", "0".repeat(18))),
        ];

        for tc in test_cases.iter() {
            let (arg, want_sdk_dec) = tc;
            let want_dec: DecimalExt = DecimalExt::from_str(arg)?;
            let got_sdk_dec: String = want_dec.to_sdk_dec_pb_repr()?;
            assert_eq!(want_sdk_dec.to_owned(), got_sdk_dec);

            let got_dec = DecimalExt::from_sdk_dec(&got_sdk_dec)?;
            assert_eq!(want_dec, got_dec)
        }
        Ok(())
    }

    /// to_sdk_dec test with fractional parts
    #[test]
    fn sdk_dec_fractional() -> TestResult {
        let test_cases: &[(&str, &str)] = &[
            ("0.5", &format!("05{}", "0".repeat(17))),
            ("0.005", &format!("0005{}", "0".repeat(15))),
            ("123.456", &format!("123456{}", "0".repeat(15))),
            ("-123.456", &format!("-123456{}", "0".repeat(15))),
            ("0.00596", &format!("000596{}", "0".repeat(13))),
            ("13.5", &format!("135{}", "0".repeat(17))),
            ("-13.5", &format!("-135{}", "0".repeat(17))),
            ("1574.00005", &format!("157400005{}", "0".repeat(13))),
        ];

        for tc in test_cases.iter() {
            let (arg, want_sdk_dec) = tc;
            let want_dec: DecimalExt = DecimalExt::from_str(arg)?;
            let got_sdk_dec: String = want_dec.to_sdk_dec_pb_repr()?;
            assert_eq!(want_sdk_dec.to_owned(), got_sdk_dec);

            let got_dec = DecimalExt::from_sdk_dec(&got_sdk_dec)?;
            assert_eq!(want_dec, got_dec)
        }
        Ok(())
    }
}
