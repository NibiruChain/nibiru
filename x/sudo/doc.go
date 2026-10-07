/*
Package sudo manages a root account and named role memberships.

Root can rotate itself, grant or revoke roles for accounts and contracts,
configure zero-gas actors, and configure the Wasm block-hook registry.
Other addresses receive only permissions whose callers explicitly check a role.
The wasm_deployer role permits guarded Wasm uploads and migrations, subject to
normal Wasm access and contract admin checks.

An empty role passed to CheckPermissions requires root. Membership is resolved
from current state on every call, so revocation and root rotation take effect
immediately. Native governance authorization is handled by each calling module.
*/
package sudo
