package keeper

//func TestGetLatestCumulativePremiumFraction(t *testing.T) {
//}
//
//func getTestKeeper(t *testing.T) (Keeper, sdk.Context) {
//		storeKey := sdk.NewKVStoreKey(types.StoreKey)
//
//		db := tmdb.NewMemDB()
//		stateStore := store.NewCommitMultiStore(db)
//		stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
//		require.NoError(t, stateStore.LoadLatestVersion())
//
//		keeper:= NewKeeper(
//			codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
//			storeKey,
//		)
//
//		ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, nil)
//
//		return vpoolKeeper, ctx
//	}
//
//}
