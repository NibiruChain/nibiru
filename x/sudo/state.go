package sudo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/NibiruChain/nibiru/v2/eth"
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
			if _, err := eth.NibiruAddrFromStr(member); err != nil {
				return ErrSudoers("role member addr: " + err.Error())
			}
		}
	}
	return nil
}

// NormalizeRoleMembers parses Bech32 or EVM addresses into unique, sorted
// canonical Nibiru account strings.
func NormalizeRoleMembers(inputs []string) ([]string, error) {
	members := make(map[string]bool)
	for _, input := range inputs {
		addr, err := eth.NibiruAddrFromStr(input)
		if err != nil {
			return nil, err
		}
		members[addr.String()] = true
	}
	result := make([]string, 0, len(members))
	for member := range members {
		result = append(result, member)
	}
	sort.Strings(result)
	return result, nil
}

// NormalizeRoles gives role state a deterministic encoding with canonical members.
func (state *Sudoers) NormalizeRoles() error {
	for i := range state.Roles {
		members, err := NormalizeRoleMembers(state.Roles[i].Members)
		if err != nil {
			return err
		}
		state.Roles[i].Members = members
	}
	sort.Slice(state.Roles, func(i, j int) bool { return state.Roles[i].Role < state.Roles[j].Role })
	return nil
}

type SudoersJson = Sudoers
