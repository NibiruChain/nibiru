package module_test

import "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"

// AppModuleWithAllExtensions is solely here for the purpose of generating
// mocks to be used in module tests.
type AppModuleWithAllExtensions interface {
	module.AppModule
	module.HasServices
	module.HasGenesis
	module.HasInvariants
	module.HasConsensusVersion
	module.BeginBlockAppModule
	module.EndBlockAppModule
}
