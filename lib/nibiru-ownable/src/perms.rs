//! Contract-local delegated perms and generated execute-message policies.
//!
//! An execute-message enum implements [`PermPolicy`], usually through the
//! `PermPolicy` derive macro. A contract calls [`assert_msg_auth`]
//! before dispatch to enforce that generated policy. The owner may grant or
//! revoke the policy's named perms through [`update_perms`]. Queries can expose
//! both the generated policy and each member's stored perms.

use std::collections::BTreeSet;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Addr, Api, Event, Order, StdError, Storage};
use cw_storage_plus::Map;

use crate::{get_ownership, is_owner, OwnershipError};

const PERM_MEMBERS: Map<(&Addr, &str), ()> = Map::new("perm_members");

/// Authorization requirement generated for one execute message value.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum PermRequirement {
    /// Shared perm enforcement performs no check. The handler may still apply
    /// application-specific authorization and validation.
    Public,
    /// The current owner or a member of any listed perm may proceed. An empty
    /// list means owner-only.
    OwnerOrAny(Vec<&'static str>),
}

/// One discoverable owner-or-perm policy gate.
#[cw_serde]
pub struct PermRule {
    /// Dot-separated policy-message identifier.
    ///
    /// An enum-level `#[perms(namespace = "...")]` declaration prefixes this
    /// value. It does not describe the contract's serialized execute JSON.
    pub exec_msg: String,
    /// Perms that may execute the message in place of the owner.
    ///
    /// An empty list means that only the owner may execute the message.
    pub owner_or_any: Vec<String>,
}

/// Describes the perm policy generated for an execute-message enum.
pub trait PermPolicy {
    /// Returns the authorization requirement for this message value.
    fn perm_requirement(&self) -> PermRequirement;

    /// Returns every owner-or-perm gate defined by the message enum.
    ///
    /// Public messages are omitted because they have no shared perm gate.
    fn perm_rules() -> Vec<PermRule>;

    /// Returns the sorted, deduplicated perm identifiers used by the policy.
    fn perm_ids() -> Vec<&'static str>;
}

/// One owner-controlled delegated membership update.
#[cw_serde]
pub struct PermUpdate {
    /// Whether to grant or revoke membership.
    pub kind: PermUpdateKind,
    /// Perm identifier whose membership will change.
    pub perm: String,
    /// Nibiru Bech32 account address whose membership will change.
    ///
    /// The contract validates this value with [`Api::addr_validate`] before
    /// reading or writing membership state. This accepts both externally owned
    /// accounts and CosmWasm contract addresses.
    pub member: String,
}

/// Operation applied to one delegated perm membership.
#[cw_serde]
pub enum PermUpdateKind {
    /// Add the member to the perm.
    Grant,
    /// Remove the member from the perm.
    Revoke,
}

/// Delegated perms held by one requested member.
#[cw_serde]
pub struct MemberPerms {
    /// Requested member, serialized as a canonical Nibiru Bech32 address.
    pub member: Addr,
    /// Sorted, deduplicated delegated perms held by the member.
    ///
    /// The current owner's result also contains the virtual `owner` perm.
    pub perms: Vec<String>,
}

/// Errors returned by delegated perm storage and authorization helpers.
#[derive(thiserror::Error, Debug, PartialEq)]
pub enum PermError {
    /// A storage or serialization operation failed.
    #[error("{0}")]
    Std(#[from] StdError),

    /// An ownership lookup or check failed.
    #[error("{0}")]
    Ownership(#[from] OwnershipError),

    /// A perm identifier does not match the supported format.
    #[error("Invalid perm identifier: {perm}")]
    InvalidPermId {
        /// Invalid perm identifier supplied by the caller.
        perm: String,
    },

    /// A permission member is not a supported Cosmos address.
    #[error("Invalid permission member address: {member}")]
    InvalidMemberAddress {
        /// Invalid member address supplied by the caller.
        member: String,
    },

    /// An update named a valid perm that the message policy does not define.
    #[error("Contract does not define perm: {perm}")]
    UnknownPerm {
        /// Valid but undefined perm identifier supplied by the caller.
        perm: String,
    },

    /// The caller does not hold the required delegated perm.
    #[error("Caller is not a member of perm: {perm}")]
    NotPermMember {
        /// Perm identifier required by the operation.
        perm: String,
    },

    /// The caller is neither the owner nor a member of an accepted perm.
    #[error("Caller is neither the owner nor a member of any required perm")]
    NotOwnerOrPerm,
}

/// Checks that a perm identifier has the supported form.
///
/// A valid identifier matches `[a-z][a-z0-9_]{0,63}`. The reserved identifier
/// `owner` is invalid because ownership is inherited rather than stored as a
/// delegated membership.
pub fn validate_perm_id(perm: &str) -> Result<(), PermError> {
    let valid = !perm.is_empty()
        && perm.len() <= 64
        && perm
            .chars()
            .next()
            .is_some_and(|ch| ch.is_ascii_lowercase())
        && perm.chars().all(|ch| {
            ch.is_ascii_lowercase() || ch.is_ascii_digit() || ch == '_'
        })
        && perm != "owner";
    if valid {
        Ok(())
    } else {
        Err(PermError::InvalidPermId {
            perm: perm.to_string(),
        })
    }
}

fn validate_member(api: &dyn Api, member: &str) -> Result<Addr, PermError> {
    if member.starts_with("0x") || member.starts_with("0X") {
        return Err(PermError::InvalidMemberAddress {
            member: member.to_string(),
        });
    }
    Ok(api.addr_validate(member)?)
}

/// Returns whether `member` holds the stored delegated `perm`.
///
/// This function validates the perm identifier before reading storage. It does
/// not treat the contract owner as a stored member. Use
/// [`assert_owner_or_perm`] when ownership should satisfy the check.
pub fn has_perm(
    storage: &dyn Storage,
    member: &Addr,
    perm: &str,
) -> Result<bool, PermError> {
    validate_perm_id(perm)?;
    Ok(PERM_MEMBERS.has(storage, (member, perm)))
}

/// Requires `member` to hold the stored delegated `perm`.
///
/// Returns [`PermError::NotPermMember`] when the perm is valid but the member
/// does not hold it.
pub fn assert_perm(
    storage: &dyn Storage,
    member: &Addr,
    perm: &str,
) -> Result<(), PermError> {
    if has_perm(storage, member, perm)? {
        Ok(())
    } else {
        Err(PermError::NotPermMember {
            perm: perm.to_string(),
        })
    }
}

/// Requires the sender to be the owner or hold any listed delegated perm.
///
/// The owner passes without a stored membership. An empty `perms` slice means
/// owner-only and returns [`OwnershipError::NotOwner`] for other senders.
pub fn assert_owner_or_perm(
    storage: &dyn Storage,
    sender: &Addr,
    perms: &[&str],
) -> Result<(), PermError> {
    if is_owner(storage, sender.as_str())? {
        return Ok(());
    }
    if perms.is_empty() {
        return Err(OwnershipError::NotOwner.into());
    }
    for perm in perms {
        if has_perm(storage, sender, perm)? {
            return Ok(());
        }
    }
    Err(PermError::NotOwnerOrPerm)
}

/// Enforces the generated policy for one execute message.
///
/// Public messages pass this shared gate without a check. Their handlers may
/// still perform application-specific authorization. Owner-or-perm messages
/// delegate to [`assert_owner_or_perm`]. Contracts should call this function
/// before dispatching the execute message.
pub fn assert_msg_auth<P: PermPolicy>(
    storage: &dyn Storage,
    sender: &Addr,
    msg: &P,
) -> Result<(), PermError> {
    match msg.perm_requirement() {
        PermRequirement::Public => Ok(()),
        PermRequirement::OwnerOrAny(perms) => {
            assert_owner_or_perm(storage, sender, &perms)
        }
    }
}

/// Applies an owner-authorized membership batch in listed order.
///
/// The function validates every perm identifier and checks it against
/// `P::perm_ids()` before changing storage. Empty batches and duplicate updates
/// are valid. Each input item emits one `perm_update` event. Its `changed`
/// attribute reports whether that item changed storage.
pub fn update_perms<P: PermPolicy>(
    storage: &mut dyn Storage,
    api: &dyn Api,
    sender: &Addr,
    updates: Vec<PermUpdate>,
) -> Result<Vec<Event>, PermError> {
    if !is_owner(storage, sender.as_str())? {
        return Err(OwnershipError::NotOwner.into());
    }

    let known: BTreeSet<_> = P::perm_ids().into_iter().collect();
    let mut validated_updates = Vec::with_capacity(updates.len());
    for update in updates {
        validate_perm_id(&update.perm)?;
        if !known.contains(update.perm.as_str()) {
            return Err(PermError::UnknownPerm {
                perm: update.perm.clone(),
            });
        }
        let member = validate_member(api, &update.member)?;
        validated_updates.push((update, member));
    }

    let mut events = Vec::with_capacity(validated_updates.len());
    for (update, member) in validated_updates {
        let existed = PERM_MEMBERS.has(storage, (&member, update.perm.as_str()));
        let (action, changed) = match update.kind {
            PermUpdateKind::Grant => {
                if !existed {
                    PERM_MEMBERS.save(
                        storage,
                        (&member, update.perm.as_str()),
                        &(),
                    )?;
                }
                ("grant", !existed)
            }
            PermUpdateKind::Revoke => {
                if existed {
                    PERM_MEMBERS
                        .remove(storage, (&member, update.perm.as_str()));
                }
                ("revoke", existed)
            }
        };

        events.push(
            Event::new("perm_update")
                .add_attribute("action", action)
                .add_attribute("perm", update.perm)
                .add_attribute("member", member)
                .add_attribute("changed", changed.to_string()),
        );
    }
    Ok(events)
}

/// Returns stored perms for each requested member.
///
/// Results preserve the input order and repeated members. Each result sorts
/// and deduplicates its `perms`. The current owner also receives the virtual
/// `owner` perm, which is not stored in the delegated membership map.
pub fn perms_for_members(
    storage: &dyn Storage,
    api: &dyn Api,
    members: Vec<String>,
) -> Result<Vec<MemberPerms>, PermError> {
    let owner = get_ownership(storage)?.owner;
    let members: Vec<Addr> = members
        .into_iter()
        .map(|member| validate_member(api, &member))
        .collect::<Result<_, _>>()?;
    members
        .into_iter()
        .map(|member| {
            let mut perms: Vec<String> = PERM_MEMBERS
                .prefix(&member)
                .keys(storage, None, None, Order::Ascending)
                .collect::<Result<_, _>>()?;
            if owner.as_deref() == Some(member.as_str()) {
                perms.push("owner".to_string());
            }
            perms.sort();
            perms.dedup();
            Ok(MemberPerms { member, perms })
        })
        .collect()
}
