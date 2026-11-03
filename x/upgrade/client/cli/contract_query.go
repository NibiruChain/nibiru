package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	querytypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/query"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	contracttypes "github.com/NibiruChain/nibiru/v2/x/upgrade/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func printJSON(ctx client.Context, value any) error {
	bz, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return ctx.PrintString(string(bz) + "\n")
}

func contractQuery(cmd *cobra.Command, ctx client.Context, request any) (string, json.RawMessage, error) {
	c := wasmtypes.NewQueryClient(ctx)
	contracts, err := c.ContractsByCreator(cmd.Context(), &wasmtypes.QueryContractsByCreatorRequest{CreatorAddress: authtypes.NewModuleAddress(contracttypes.ContractCreatorName).String(), Pagination: &querytypes.PageRequest{Limit: 1}})
	if err != nil {
		return "", nil, err
	}
	if len(contracts.ContractAddresses) == 0 {
		return "", nil, fmt.Errorf("x-upgrade contract is not initialized")
	}
	addr := contracts.ContractAddresses[0]
	bz, err := json.Marshal(request)
	if err != nil {
		return "", nil, err
	}
	response, err := c.SmartContractState(cmd.Context(), &wasmtypes.QuerySmartContractStateRequest{Address: addr, QueryData: bz})
	if err != nil {
		return "", nil, err
	}
	return addr, json.RawMessage(response.Data), nil
}

func GetAppliedRecordCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "applied-record [upgrade-name]", Short: "Return a completed upgrade's name and height without historical block RPC", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		response, err := upgradetypes.NewQueryClient(ctx).AppliedPlan(cmd.Context(), &upgradetypes.QueryAppliedPlanRequest{Name: args[0]})
		if err != nil {
			return err
		}
		if response.Height == 0 {
			return fmt.Errorf("no upgrade found")
		}
		return printJSON(ctx, contracttypes.CompletedUpgrade{Name: args[0], Height: response.Height})
	}}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetStatusCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "status", Short: "Show pending time targets, concrete plans, and terminal selection failures as JSON", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		// Pin contract and receipt queries to the same committed height.
		receipt, height, err := ctx.QueryWithData("/store/upgrade/key", []byte{0x05})
		if err != nil {
			return err
		}
		ctx = ctx.WithHeight(height)
		addr, pending, err := contractQuery(cmd, ctx, map[string]any{"status": struct{}{}})
		if err != nil {
			return err
		}
		status := "none"
		var target struct {
			ID         uint64  `json:"id,string"`
			Height     *string `json:"height"`
			TargetTime *string `json:"target_time"`
		}
		if string(pending) != "null" {
			if err = json.Unmarshal(pending, &target); err != nil {
				return err
			}
			status = "pending_time"
			if target.Height != nil {
				status = "scheduled"
			}
		}
		result := map[string]any{"contract_address": addr, "status": status, "pending": pending}
		if target.TargetTime != nil {
			nanos, err := strconv.ParseUint(*target.TargetTime, 10, 64)
			if err != nil {
				return err
			}
			result["target_time"] = time.Unix(int64(nanos/1_000_000_000), int64(nanos%1_000_000_000)).UTC().Format(time.RFC3339Nano)
		}
		if len(receipt) > 0 {
			var failure struct {
				RequestID uint64 `json:"request_id,string"`
			}
			if err = json.Unmarshal(receipt, &failure); err != nil {
				return err
			}
			if failure.RequestID == target.ID {
				result["selection_failure"] = json.RawMessage(receipt)
				result["status"] = "failed"
			}
		}
		return printJSON(ctx, result)
	}}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "params", Short: "Show the upgrade contract's governance authority and activation delay as JSON", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		_, data, err := contractQuery(cmd, ctx, map[string]any{"config": struct{}{}})
		if err != nil {
			return err
		}
		return printJSON(ctx, data)
	}}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "history", Short: "List completed upgrades as JSON with height/name cursor pagination", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		ctx, err := client.GetClientQueryContext(cmd)
		if err != nil {
			return err
		}
		limit, _ := cmd.Flags().GetUint32("limit")
		reverse, _ := cmd.Flags().GetBool("reverse")
		height, _ := cmd.Flags().GetInt64("start-after-height")
		name, _ := cmd.Flags().GetString("start-after-name")
		if limit < 1 || limit > 200 {
			return fmt.Errorf("limit must be between 1 and 200")
		}
		var cursor any
		if height != 0 || name != "" {
			if height <= 0 || name == "" {
				return fmt.Errorf("cursor requires both positive start-after-height and start-after-name")
			}
			cursor = map[string]string{"height": strconv.FormatInt(height, 10), "name": name}
		}
		_, data, err := contractQuery(cmd, ctx, map[string]any{"history": map[string]any{"start_after": cursor, "limit": limit, "descending": reverse}})
		if err != nil {
			return err
		}
		return printJSON(ctx, data)
	}}
	cmd.Flags().Uint32("limit", 50, "Maximum records, 1 to 200")
	cmd.Flags().Bool("reverse", false, "List newest upgrades first")
	cmd.Flags().Int64("start-after-height", 0, "Exclusive cursor height")
	cmd.Flags().String("start-after-name", "", "Exclusive cursor name at that height")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
