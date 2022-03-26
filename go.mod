module github.com/MatrixDao/matrix

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.45.1
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/rakyll/statik v0.1.7
	github.com/regen-network/cosmos-proto v0.3.1
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/starport v0.19.5
	github.com/tendermint/tendermint v0.34.14
	github.com/tendermint/tm-db v0.6.4
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	google.golang.org/genproto v0.0.0-20220317150908-0efb43f6373e
	google.golang.org/grpc v1.45.0
	gopkg.in/yaml.v2 v2.4.0
)

// latest grpc doesn't work with with our modified proto compiler, so we need to enforce
// the following version across all dependencies.
replace google.golang.org/grpc => google.golang.org/grpc v1.33.2

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

replace github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76

// Fix upstream GHSA-h395-qcrw-5vmq vulnerability.
// TODO Remove it: https://github.com/cosmos/cosmos-sdk/issues/10409
replace github.com/gin-gonic/gin => github.com/gin-gonic/gin v1.7.0

// x/stablecoin dependencies
replace github.com/MatrixDao/x/stablecoin => ./x/stablecoin

// replace "github.com/MatrixDao/x/stablecoin/starport/cmdrunner" => ./x/stablecoin/starport/cmdrunner
