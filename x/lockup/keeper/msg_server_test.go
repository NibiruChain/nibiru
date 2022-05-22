package keeper

import (
	"context"
	"reflect"
	"testing"

	"github.com/NibiruChain/nibiru/x/lockup/types"
)

// TODO(mercilex): test

func TestNewMsgServerImpl(t *testing.T) {
	type args struct {
		keeper Keeper
	}
	tests := []struct {
		name string
		args args
		want types.MsgServer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMsgServerImpl(tt.args.keeper); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMsgServerImpl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewQueryServerImpl(t *testing.T) {
	type args struct {
		k Keeper
	}
	tests := []struct {
		name string
		args args
		want types.QueryServer
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQueryServerImpl(tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQueryServerImpl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_msgServer_InitiateUnlock(t *testing.T) {
	type fields struct {
		keeper Keeper
	}
	type args struct {
		ctx    context.Context
		unlock *types.MsgInitiateUnlock
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.MsgInitiateUnlockResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := msgServer{
				keeper: tt.fields.keeper,
			}
			got, err := server.InitiateUnlock(tt.args.ctx, tt.args.unlock)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitiateUnlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitiateUnlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_msgServer_LockTokens(t *testing.T) {
	type fields struct {
		keeper Keeper
	}
	type args struct {
		goCtx context.Context
		msg   *types.MsgLockTokens
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.MsgLockTokensResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := msgServer{
				keeper: tt.fields.keeper,
			}
			got, err := server.LockTokens(tt.args.goCtx, tt.args.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("LockTokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LockTokens() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryServer_Lock(t *testing.T) {
	type fields struct {
		k Keeper
	}
	type args struct {
		ctx     context.Context
		request *types.QueryLockRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.QueryLockResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := queryServer{
				k: tt.fields.k,
			}
			got, err := q.Lock(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Lock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryServer_LockedCoins(t *testing.T) {
	type fields struct {
		k Keeper
	}
	type args struct {
		ctx     context.Context
		request *types.QueryLockedCoinsRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.QueryLockedCoinsResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := queryServer{
				k: tt.fields.k,
			}
			got, err := q.LockedCoins(tt.args.ctx, tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("LockedCoins() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LockedCoins() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_queryServer_LocksByAddress(t *testing.T) {
	type fields struct {
		k Keeper
	}
	type args struct {
		ctx     context.Context
		address *types.QueryLocksByAddress
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.QueryLocksByAddressResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := queryServer{
				k: tt.fields.k,
			}
			got, err := q.LocksByAddress(tt.args.ctx, tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("LocksByAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LocksByAddress() got = %v, want %v", got, tt.want)
			}
		})
	}
}
