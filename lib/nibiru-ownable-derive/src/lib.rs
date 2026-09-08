extern crate proc_macro;
extern crate quote;
extern crate syn;
use proc_macro::TokenStream;
use quote::quote;
use syn::{parse_macro_input, AttributeArgs, DataEnum, DeriveInput};

/// Merges the variants of two enums.
///
/// Adapted from DAO DAO:
/// https://github.com/DA0-DA0/dao-contracts/blob/74bd3881fdd86829e5e8b132b9952dd64f2d0737/packages/dao-macros/src/lib.rs#L9
fn merge_variants(
    metadata: TokenStream,
    left: TokenStream,
    right: TokenStream,
) -> TokenStream {
    use syn::Data::Enum;

    // parse metadata
    let args = parse_macro_input!(metadata as AttributeArgs);
    if let Some(first_arg) = args.first() {
        return syn::Error::new_spanned(first_arg, "macro takes no arguments")
            .to_compile_error()
            .into();
    }

    // parse the left enum
    let mut left: DeriveInput = parse_macro_input!(left);
    let Enum(DataEnum { variants, .. }) = &mut left.data else {
        return syn::Error::new(
            left.ident.span(),
            "only enums can accept variants",
        )
        .to_compile_error()
        .into();
    };

    // parse the right enum
    let right: DeriveInput = parse_macro_input!(right);
    let Enum(DataEnum {
        variants: to_add, ..
    }) = right.data
    else {
        return syn::Error::new(
            left.ident.span(),
            "only enums can provide variants",
        )
        .to_compile_error()
        .into();
    };

    // insert variants from the right to the left
    variants.extend(to_add);

    quote! { #left }.into()
}

/// Append ownership-related execute message variant(s) to an enum.
///
/// For example, apply the `ownable_execute` macro to the following enum:
///
/// ```rust
/// extern crate cosmwasm_schema; // not to be copied
/// extern crate nibiru_ownable;  // not to be copied
/// use cosmwasm_schema::cw_serde;
/// use nibiru_ownable::ownable_execute;
///
/// #[ownable_execute]
/// #[cw_serde]
/// enum ExecuteMsg {
///     Foo {},
///     Bar {},
/// }
/// ```
///
/// Is equivalent to:
///
/// ```rust
/// extern crate cosmwasm_schema; // not to be copied
/// extern crate nibiru_ownable;  // not to be copied
/// use cosmwasm_schema::cw_serde;
/// use nibiru_ownable::Action;
///
/// #[cw_serde]
/// enum ExecuteMsg {
///     UpdateOwnership(Action),
///     Foo {},
///     Bar {},
/// }
///
/// let _msg = ExecuteMsg::Foo{};
/// ```
///
/// Note: `#[ownable_execute]` must be applied _before_ `#[cw_serde]`.
#[proc_macro_attribute]
pub fn ownable_execute(
    metadata: TokenStream,
    input: TokenStream,
) -> TokenStream {
    merge_variants(
        metadata,
        input,
        quote! {
            enum Right {
                /// Update the contract's ownership. The `action` to be provided
                /// can be either to propose transferring ownership to an account,
                /// accept a pending ownership transfer, or renounce the ownership
                /// permanently.
                UpdateOwnership(::nibiru_ownable::Action),
            }
        }
        .into(),
    )
}

/// Append ownership-related query message variant(s) to an enum.
///
/// For example, apply the `ownable_query` macro to the following enum:
///
/// ```rust
/// extern crate cosmwasm_schema; // not to be copied
/// extern crate nibiru_ownable;  // not to be copied
/// use cosmwasm_schema::{cw_serde, QueryResponses};
/// use nibiru_ownable::ownable_query;
///
/// #[ownable_query]
/// #[cw_serde]
/// #[derive(QueryResponses)]
/// enum QueryMsg {
///     #[returns(FooResponse)]
///     Foo {},
///     #[returns(BarResponse)]
///     Bar {},
/// }
///
/// #[cw_serde]
/// pub struct FooResponse {}
/// #[cw_serde]
/// pub struct BarResponse {}
///
/// let _msg = QueryMsg::Foo{};
/// ```
///
/// Is equivalent to:
///
/// ```rust
/// extern crate cosmwasm_schema; // not to be copied
/// extern crate nibiru_ownable;  // not to be copied
/// use cosmwasm_schema::{cw_serde, QueryResponses};
/// use nibiru_ownable::Ownership;
///
/// #[cw_serde]
/// #[derive(QueryResponses)]
/// enum QueryMsg {
///     #[returns(Ownership<String>)]
///     Ownership {},
///     #[returns(FooResponse)]
///     Foo {},
///     #[returns(BarResponse)]
///     Bar {},
/// }
///
/// #[cw_serde]
/// pub struct FooResponse {}
/// #[cw_serde]
/// pub struct BarResponse {}
///
/// let _msg = QueryMsg::Bar{};
/// ```
///
/// Note: `#[ownable_query]` must be applied _before_ `#[cw_serde]`.
#[proc_macro_attribute]
pub fn ownable_query(metadata: TokenStream, input: TokenStream) -> TokenStream {
    merge_variants(
        metadata,
        input,
        quote! {
            enum Right {
                /// Query the contract's ownership information
                #[returns(::nibiru_ownable::Ownership<String>)]
                Ownership {},
            }
        }
        .into(),
    )
}
