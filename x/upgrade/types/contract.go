// Package types defines the JSON boundary to the Rust x-upgrade contract.
// Native SDK messages remain in lib/cosmos-sdk/x/upgrade/types.
package types

const (
	ContractCreatorName        = "xupgrade"
	ContractSalt               = "x-upgrade"
	QueryGasLimit       uint64 = 1_000_000
)

type Plan struct {
	Name   string `json:"name"`
	Height int64  `json:"height,string"`
	Info   string `json:"info"`
}
type CompletedUpgrade struct {
	Name   string `json:"name"`
	Height int64  `json:"height,string"`
}
type ModuleVersion struct {
	Name    string `json:"name"`
	Version uint64 `json:"version,string"`
}
type Snapshot struct {
	Plan            *Plan              `json:"plan"`
	Completed       []CompletedUpgrade `json:"completed"`
	ModuleVersions  []ModuleVersion    `json:"module_versions"`
	ProtocolVersion *uint64            `json:"protocol_version,string"`
}
type TimeTarget struct {
	ID         uint64 `json:"id,string"`
	Name       string `json:"name"`
	TargetTime uint64 `json:"target_time,string"`
}
type ScheduleRequest struct {
	Name       string  `json:"name"`
	Height     int64   `json:"height,string"`
	TargetTime *uint64 `json:"target_time,string"`
	Info       string  `json:"info"`
}
type InstantiateMsg struct {
	GovernanceAuthority string   `json:"governance_authority"`
	State               Snapshot `json:"state"`
}
