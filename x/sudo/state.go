package sudo

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

const RoleWasmDeployer = "wasm_deployer"

// ValidateRole rejects the empty role reserved for root-only checks.
func ValidateRole(role string) error {
	if role == "" || strings.TrimSpace(role) != role {
		return fmt.Errorf("role must be nonempty without surrounding whitespace")
	}
	return nil
}

func (state Sudoers) Validate() error {
	if _, err := sdk.AccAddressFromBech32(state.Root); err != nil {
		return ErrSudoers("root addr: " + err.Error())
	}
	seen := make(map[string]bool)
	for _, entry := range state.Roles {
		if err := ValidateRole(entry.Role); err != nil {
			return err
		}
		if seen[entry.Role] {
			return fmt.Errorf("duplicate role %q", entry.Role)
		}
		seen[entry.Role] = true
		for _, member := range entry.Members {
			if _, err := sdk.AccAddressFromBech32(member); err != nil {
				return ErrSudoers("role member addr: " + err.Error())
			}
		}
	}
	return nil
}

// NormalizeRoles gives role state a deterministic encoding with unique members.
func (state *Sudoers) NormalizeRoles() {
	for i := range state.Roles {
		members := make(map[string]bool)
		for _, member := range state.Roles[i].Members {
			members[member] = true
		}
		state.Roles[i].Members = make([]string, 0, len(members))
		for member := range members {
			state.Roles[i].Members = append(state.Roles[i].Members, member)
		}
		sort.Strings(state.Roles[i].Members)
	}
	sort.Slice(state.Roles, func(i, j int) bool { return state.Roles[i].Role < state.Roles[j].Role })
}

type SudoersJson = Sudoers
