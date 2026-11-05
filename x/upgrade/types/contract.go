// Package types defines the JSON boundary to the Rust x-upgrade contract.
// Native SDK messages remain in lib/cosmos-sdk/x/upgrade/types.
package types

const (
	ContractCreatorName        = "xupgrade"
	ContractSalt               = "x-upgrade"
	QueryGasLimit       uint64 = 1_000_000
)

// Plan is an executable contract plan. The native protobuf adapter translates
// BlockHeight back to the existing SDK Height field.
type Plan struct {
	Name        string `json:"name"`
	BlockHeight int64  `json:"block_height,string"`
	Info        string `json:"info"`
}

// CompletedUpgrade records an applied handler and its consensus block height;
// the same pair is the exclusive history pagination cursor.
type CompletedUpgrade struct {
	Name        string `json:"name"`
	BlockHeight int64  `json:"block_height,string"`
}

// ModuleVersion is one native module schema version used by RunMigrations.
type ModuleVersion struct {
	Name    string `json:"name"`
	Version uint64 `json:"version,string"`
}

// Snapshot imports and verifies the four legacy state families. Wasm genesis
// export, rather than this logical view, preserves all raw contract records.
type Snapshot struct {
	Plan            *Plan              `json:"plan"`
	Completed       []CompletedUpgrade `json:"completed"`
	ModuleVersions  []ModuleVersion    `json:"module_versions"`
	ProtocolVersion *uint64            `json:"protocol_version,string"`
}

// TimeTarget identifies an unresolved request and its absolute Unix-nanosecond
// target. Selection compares it with consensus block time, not a local clock.
type TimeTarget struct {
	ID                  uint64 `json:"id,string"`
	Name                string `json:"name"`
	TargetTimeUnixNanos uint64 `json:"target_time_unix_nanos,string"`
}

// ScheduleRequest supplies either a positive block height or a future time
// target with zero height. The contract validates and replaces the pending request.
type ScheduleRequest struct {
	Name                string  `json:"name"`
	BlockHeight         int64   `json:"block_height,string"`
	TargetTimeUnixNanos *uint64 `json:"target_time_unix_nanos,string"`
	Info                string  `json:"info"`
}

// InstantiateMsg supplies the native governance account and bootstrap snapshot.
type InstantiateMsg struct {
	GovAuthority string   `json:"gov_authority"`
	State        Snapshot `json:"state"`
}
