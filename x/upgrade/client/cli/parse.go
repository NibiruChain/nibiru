package cli

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/client/cli"
	gov "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
)

func parseArgsToContent(fs *pflag.FlagSet, name string) (gov.Content, error) {
	title, err := fs.GetString(cli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := fs.GetString(cli.FlagDescription) //nolint:staticcheck // we are intentionally using a deprecated flag here.
	if err != nil {
		return nil, err
	}

	height, err := fs.GetInt64(FlagUpgradeHeight)
	if err != nil {
		return nil, err
	}

	info, err := fs.GetString(FlagUpgradeInfo)
	if err != nil {
		return nil, err
	}

	plan := types.Plan{Name: name, Height: height, Info: info}
	if fs.Lookup(FlagUpgradeTime) != nil {
		raw, err := fs.GetString(FlagUpgradeTime)
		if err != nil {
			return nil, err
		}
		if raw != "" {
			t, err := time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				return nil, fmt.Errorf("upgrade-time must be RFC 3339: %w", err)
			}
			plan.Time = t.UTC()
		}
	}
	if err := plan.ValidateScheduleRequest(); err != nil {
		return nil, err
	}
	content := types.NewSoftwareUpgradeProposal(title, description, plan)
	return content, nil
}
