extern crate proc_macro;
extern crate quote;
extern crate syn;
use proc_macro::TokenStream;
use quote::quote;
use syn::{
    parse_macro_input, AttributeArgs, DataEnum, DeriveInput, Fields, Lit, Meta,
    NestedMeta, Variant,
};

enum Policy {
    Public,
    OwnerOrAny(Vec<syn::LitStr>),
    Ownership,
}

fn snake_case(name: &str) -> String {
    let mut output = String::new();
    for (index, ch) in name.chars().enumerate() {
        if ch.is_ascii_uppercase() {
            if index > 0 {
                output.push('_');
            }
            output.push(ch.to_ascii_lowercase());
        } else {
            output.push(ch);
        }
    }
    output
}

fn serialized_name(variant: &Variant) -> syn::Result<String> {
    for attr in &variant.attrs {
        if !attr.path.is_ident("serde") {
            continue;
        }
        let Meta::List(list) = attr.parse_meta()? else {
            continue;
        };
        for item in list.nested {
            if let NestedMeta::Meta(Meta::NameValue(value)) = item {
                if value.path.is_ident("rename") {
                    if let Lit::Str(name) = value.lit {
                        return Ok(name.value());
                    }
                }
            }
        }
    }
    Ok(snake_case(&variant.ident.to_string()))
}

fn derives_perm_policy(input: &DeriveInput) -> bool {
    input.attrs.iter().any(|attr| {
        if !attr.path.is_ident("derive") {
            return false;
        }
        let Ok(Meta::List(list)) = attr.parse_meta() else {
            return false;
        };
        list.nested.iter().any(|item| {
            let NestedMeta::Meta(Meta::Path(path)) = item else {
                return false;
            };
            path.segments
                .last()
                .is_some_and(|segment| segment.ident == "PermPolicy")
        })
    })
}

fn valid_namespace(value: &str) -> bool {
    value.len() <= 64
        && value
            .chars()
            .next()
            .is_some_and(|ch| ch.is_ascii_lowercase())
        && value.chars().all(|ch| {
            ch.is_ascii_lowercase() || ch.is_ascii_digit() || ch == '_'
        })
}

fn parse_namespace(input: &DeriveInput) -> syn::Result<Option<syn::LitStr>> {
    let attrs: Vec<_> = input
        .attrs
        .iter()
        .filter(|attr| attr.path.is_ident("perms"))
        .collect();
    if attrs.len() > 1 {
        return Err(syn::Error::new_spanned(
            input,
            "PermPolicy enum accepts at most one #[perms(namespace = \"...\")] declaration",
        ));
    }
    let Some(attr) = attrs.first() else {
        return Ok(None);
    };
    let Meta::List(list) = attr.parse_meta()? else {
        return Err(syn::Error::new_spanned(
            attr,
            "expected #[perms(namespace = \"...\")]",
        ));
    };
    if list.nested.len() != 1 {
        return Err(syn::Error::new_spanned(
            list,
            "expected #[perms(namespace = \"...\")]",
        ));
    }
    let Some(NestedMeta::Meta(Meta::NameValue(value))) = list.nested.first()
    else {
        return Err(syn::Error::new_spanned(
            list,
            "expected #[perms(namespace = \"...\")]",
        ));
    };
    if !value.path.is_ident("namespace") {
        return Err(syn::Error::new_spanned(
            value,
            "expected #[perms(namespace = \"...\")]",
        ));
    }
    let Lit::Str(namespace) = &value.lit else {
        return Err(syn::Error::new_spanned(
            &value.lit,
            "perm namespace must be a string literal",
        ));
    };
    if !valid_namespace(&namespace.value()) {
        return Err(syn::Error::new_spanned(
            namespace,
            "perm namespace must match [a-z][a-z0-9_]{0,63}",
        ));
    }
    Ok(Some(namespace.clone()))
}

fn parse_policy(variant: &Variant) -> syn::Result<Policy> {
    let attrs: Vec<_> = variant
        .attrs
        .iter()
        .filter(|attr| attr.path.is_ident("perms"))
        .collect();
    if attrs.len() != 1 {
        return Err(syn::Error::new_spanned(
            variant,
            "every PermPolicy variant requires exactly one #[perms(...)] declaration",
        ));
    }

    let Meta::List(list) = attrs[0].parse_meta()? else {
        return Err(syn::Error::new_spanned(attrs[0], "expected #[perms(...)]"));
    };
    if list.nested.len() != 1 {
        return Err(syn::Error::new_spanned(
            list,
            "perms declaration must contain exactly one policy",
        ));
    }

    match list.nested.first().expect("checked length") {
        NestedMeta::Meta(Meta::Path(path)) if path.is_ident("public") => {
            Ok(Policy::Public)
        }
        NestedMeta::Meta(Meta::Path(path)) if path.is_ident("ownership") => {
            if variant.ident != "UpdateOwnership"
                || !matches!(&variant.fields, Fields::Unnamed(fields) if fields.unnamed.len() == 1)
            {
                return Err(syn::Error::new_spanned(
                    variant,
                    "#[perms(ownership)] is reserved for UpdateOwnership(Action) generated by #[ownable_execute(perms)]",
                ));
            }
            Ok(Policy::Ownership)
        }
        NestedMeta::Meta(Meta::List(owner))
            if owner.path.is_ident("owner_or_any") =>
        {
            let mut perms = Vec::new();
            for value in &owner.nested {
                let NestedMeta::Lit(Lit::Str(perm)) = value else {
                    return Err(syn::Error::new_spanned(
                        value,
                        "owner_or_any values must be string literals",
                    ));
                };
                let value = perm.value();
                let valid = value.len() <= 64
                    && value
                        .chars()
                        .next()
                        .is_some_and(|ch| ch.is_ascii_lowercase())
                    && value.chars().all(|ch| {
                        ch.is_ascii_lowercase()
                            || ch.is_ascii_digit()
                            || ch == '_'
                    })
                    && value != "owner";
                if !valid {
                    return Err(syn::Error::new_spanned(
                        perm,
                        "perm must match [a-z][a-z0-9_]{0,63} and cannot be `owner`",
                    ));
                }
                perms.push(perm.clone());
            }
            Ok(Policy::OwnerOrAny(perms))
        }
        value => Err(syn::Error::new_spanned(
            value,
            "expected public, owner_or_any(...), or ownership",
        )),
    }
}

fn ignored_pattern(
    enum_name: &syn::Ident,
    variant: &Variant,
) -> proc_macro2::TokenStream {
    let name = &variant.ident;
    match &variant.fields {
        Fields::Unit => quote!(#enum_name::#name),
        Fields::Named(_) => quote!(#enum_name::#name { .. }),
        Fields::Unnamed(_) => quote!(#enum_name::#name ( .. )),
    }
}

/// Implements `nibiru_ownable::PermPolicy` for an execute-message enum.
///
/// Every variant must declare exactly one policy:
///
/// ```text
/// #[perms(public)]
/// #[perms(owner_or_any())]
/// #[perms(owner_or_any("writer", "auditor"))]
/// ```
///
/// An enum may add `#[perms(namespace = "admin")]` after its derive. That namespace prefixes
/// every catalog identifier generated for the enum, such as
/// `admin.intent_close_trade`.
///
/// `public` skips the shared perm gate. `owner_or_any()` with no identifiers is
/// owner-only. A non-empty `owner_or_any(...)` also accepts members of any
/// listed perm. Perm identifiers must match `[a-z][a-z0-9_]{0,63}` and cannot
/// be `owner`.
///
/// The generated implementation provides the requirement for each message, a
/// catalog of gated policy-message identifiers, and the sorted set of referenced
/// perm identifiers. Public variants do not appear in the catalog. Identifiers
/// use an explicit `serde(rename = "...")` when present and otherwise use the
/// snake-case Rust variant name.
///
/// The derive fails to compile if it is applied to a non-enum, a variant omits
/// its policy, a variant has more than one policy, or a namespace is malformed.
#[proc_macro_derive(PermPolicy, attributes(perms))]
pub fn derive_perm_policy(input: TokenStream) -> TokenStream {
    let input = parse_macro_input!(input as DeriveInput);
    let enum_name = &input.ident;
    let namespace = match parse_namespace(&input) {
        Ok(namespace) => namespace,
        Err(err) => return err.to_compile_error().into(),
    };
    let syn::Data::Enum(data) = &input.data else {
        return syn::Error::new_spanned(
            &input,
            "PermPolicy supports enums only",
        )
        .to_compile_error()
        .into();
    };

    let mut requirement_arms = Vec::new();
    let mut catalog_steps = Vec::new();
    let mut id_steps = Vec::new();

    for variant in data.variants.iter() {
        let policy = match parse_policy(variant) {
            Ok(policy) => policy,
            Err(err) => return err.to_compile_error().into(),
        };
        let route = match serialized_name(variant) {
            Ok(route) => route,
            Err(err) => return err.to_compile_error().into(),
        };
        let route = namespace.as_ref().map_or(route.clone(), |namespace| {
            format!("{}.{}", namespace.value(), route)
        });
        let pattern = ignored_pattern(enum_name, variant);

        match policy {
            Policy::Public => {
                requirement_arms.push(quote! {
                    #pattern => ::nibiru_ownable::PermRequirement::Public
                });
            }
            Policy::OwnerOrAny(perms) => {
                requirement_arms.push(quote! {
                    #pattern => ::nibiru_ownable::PermRequirement::OwnerOrAny(
                        vec![#(#perms),*]
                    )
                });
                catalog_steps.push(quote! {
                    rules.push(::nibiru_ownable::PermRule {
                        exec_msg: #route.to_string(),
                        owner_or_any: vec![#(#perms.to_string()),*],
                    });
                });
                if !perms.is_empty() {
                    id_steps.push(quote! {
                        ids.extend([#(#perms),*]);
                    });
                }
            }
            Policy::Ownership => {
                let variant_name = &variant.ident;
                requirement_arms.push(quote! {
                    #enum_name::#variant_name(
                        ::nibiru_ownable::Action::TransferOwnership { .. }
                    ) | #enum_name::#variant_name(
                        ::nibiru_ownable::Action::RenounceOwnership
                    ) => ::nibiru_ownable::PermRequirement::OwnerOrAny(vec![]),
                    #enum_name::#variant_name(
                        ::nibiru_ownable::Action::AcceptOwnership
                    ) => ::nibiru_ownable::PermRequirement::Public
                });
                let transfer_route = format!("{}.transfer_ownership", route);
                let renounce_route = format!("{}.renounce_ownership", route);
                catalog_steps.push(quote! {
                    rules.push(::nibiru_ownable::PermRule {
                        exec_msg: #transfer_route.to_string(),
                        owner_or_any: vec![],
                    });
                    rules.push(::nibiru_ownable::PermRule {
                        exec_msg: #renounce_route.to_string(),
                        owner_or_any: vec![],
                    });
                });
            }
        }
    }

    let (impl_generics, ty_generics, where_clause) =
        input.generics.split_for_impl();
    quote! {
        impl #impl_generics ::nibiru_ownable::PermPolicy
            for #enum_name #ty_generics #where_clause
        {
            fn perm_requirement(&self) -> ::nibiru_ownable::PermRequirement {
                match self {
                    #(#requirement_arms),*
                }
            }

            fn perm_rules() -> Vec<::nibiru_ownable::PermRule> {
                let mut rules = Vec::new();
                #(#catalog_steps)*
                rules
            }

            fn perm_ids() -> Vec<&'static str> {
                let mut ids = Vec::new();
                #(#id_steps)*
                ids.sort_unstable();
                ids.dedup();
                ids
            }
        }
    }
    .into()
}

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
/// Pass `perms` to append these variants:
///
/// ```text
/// UpdateOwnership(nibiru_ownable::Action)
/// UpdatePerms(Vec<nibiru_ownable::PermUpdate>)
/// ```
///
/// Perm mode requires the enum to derive `PermPolicy`. The macro adds an
/// action-aware ownership policy for `UpdateOwnership`: ownership transfer and
/// renunciation are owner-only, while a pending owner may accept ownership.
/// `UpdatePerms` is owner-only.
///
/// This macro defines message variants. It does not enforce the policy or
/// implement their handlers. The contract execute entry point must call
/// `nibiru_ownable::assert_msg_auth` before dispatch, then route the
/// injected variants to `update_ownership` and `update_perms`.
///
/// With no argument, the macro retains its original behavior and appends only
/// `UpdateOwnership`.
///
/// Note: `#[ownable_execute]` must be applied _before_ `#[cw_serde]`.
#[proc_macro_attribute]
pub fn ownable_execute(
    metadata: TokenStream,
    input: TokenStream,
) -> TokenStream {
    let args = parse_macro_input!(metadata as AttributeArgs);
    let perms_enabled = match args.as_slice() {
        [] => false,
        [NestedMeta::Meta(Meta::Path(path))] if path.is_ident("perms") => true,
        [arg] => {
            return syn::Error::new_spanned(
                arg,
                "expected `perms` or no argument",
            )
            .to_compile_error()
            .into()
        }
        _ => {
            return syn::Error::new(
                proc_macro2::Span::call_site(),
                "expected `perms` or no argument",
            )
            .to_compile_error()
            .into()
        }
    };
    let derives_perm_policy = syn::parse::<DeriveInput>(input.clone())
        .map(|input| derives_perm_policy(&input))
        .unwrap_or(false);
    if perms_enabled && !derives_perm_policy {
        return syn::Error::new(
            proc_macro2::Span::call_site(),
            "#[ownable_execute(perms)] requires #[derive(PermPolicy)]",
        )
        .to_compile_error()
        .into();
    }
    let variants = if perms_enabled && derives_perm_policy {
        quote! {
            enum Right {
                /// Update the contract's ownership. The `action` to be provided
                /// can be either to propose transferring ownership to an account,
                /// accept a pending ownership transfer, or renounce the ownership
                /// permanently.
                #[perms(ownership)]
                UpdateOwnership(::nibiru_ownable::Action),

                /// Grant or revoke delegated contract perms.
                #[perms(owner_or_any())]
                UpdatePerms(Vec<::nibiru_ownable::PermUpdate>),
            }
        }
    } else {
        quote! {
            enum Right {
                /// Update the contract's ownership.
                UpdateOwnership(::nibiru_ownable::Action),
            }
        }
    };
    merge_variants(TokenStream::new(), input, variants.into())
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
/// Pass `perms` to append these queries in addition to `Ownership`:
///
/// - `Perms {}` returns `Vec<nibiru_ownable::PermRule>`.
/// - `PermsForMembers { members }` accepts
///   `Vec<nibiru_ownable::UserAddr>` and returns
///   `Vec<nibiru_ownable::MemberPerms>`.
///
/// This macro defines query variants and their response types. The contract
/// query entry point must route `Perms {}` to the policy enum's
/// `perm_rules()` and route `PermsForMembers {}` to
/// `nibiru_ownable::perms_for_members`.
///
/// With no argument, the macro retains its original behavior and appends only
/// `Ownership`.
///
/// Note: `#[ownable_query]` must be applied _before_ `#[cw_serde]`.
#[proc_macro_attribute]
pub fn ownable_query(metadata: TokenStream, input: TokenStream) -> TokenStream {
    let args = parse_macro_input!(metadata as AttributeArgs);
    let perms_enabled = match args.as_slice() {
        [] => false,
        [NestedMeta::Meta(Meta::Path(path))] if path.is_ident("perms") => true,
        [arg] => {
            return syn::Error::new_spanned(
                arg,
                "expected `perms` or no argument",
            )
            .to_compile_error()
            .into()
        }
        _ => {
            return syn::Error::new(
                proc_macro2::Span::call_site(),
                "expected `perms` or no argument",
            )
            .to_compile_error()
            .into()
        }
    };
    let variants = if perms_enabled {
        quote! {
            enum Right {
                /// Query the contract's ownership information
                #[returns(::nibiru_ownable::Ownership<String>)]
                Ownership {},

                /// Query the compiled execute-message perm catalog.
                #[returns(Vec<::nibiru_ownable::PermRule>)]
                Perms {},

                /// Query delegated perms for a supplied set of members.
                #[returns(Vec<::nibiru_ownable::MemberPerms>)]
                PermsForMembers {
                    members: Vec<::nibiru_ownable::UserAddr>,
                },
            }
        }
    } else {
        quote! {
            enum Right {
                /// Query the contract's ownership information
                #[returns(::nibiru_ownable::Ownership<String>)]
                Ownership {},
            }
        }
    };
    merge_variants(TokenStream::new(), input, variants.into())
}
