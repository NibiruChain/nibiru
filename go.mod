module github.com/MatrixDao/matrix

go 1.16

require (
	github.com/cosmos/cosmos-proto v1.0.0-alpha7
	github.com/cosmos/cosmos-sdk v0.46.0-alpha3
	github.com/cosmos/cosmos-sdk/api v0.1.0-alpha5
	github.com/cosmos/cosmos-sdk/orm v1.0.0-alpha.10.0.20220223111002-433420091a7a
	github.com/dustin/go-humanize v1.0.1-0.20200219035652-afde56e7acac // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/gogo/protobuf v1.3.3
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.4.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.35.2
	github.com/tendermint/tm-db v0.6.7
	google.golang.org/genproto v0.0.0-20220118154757-00ab72f36ad5
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/keybase/go-keychain => github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4
)
