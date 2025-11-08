package main

import (
	"fmt"
	"log"

	"github.com/NibiruChain/nibiru/sai-trading/tutil"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/gosdk"
)

func main() {
	fmt.Println("This is trivial print statement to show that the build imports are working:")
	fmt.Printf("appconst.DefaultDBBackend: %v\n", appconst.DefaultDBBackend)

	err := tutil.EnsureLocalBlockchain()
	if err != nil {
		log.Fatal(err)
	}

	netInfo := gosdk.NETWORK_INFO_DEFAULT
	grpcUrl := netInfo.GrpcEndpoint
	timeoutSeconds := int64(6)
	grpcConn, err := gosdk.GetGRPCConnection(grpcUrl, true, timeoutSeconds)
	if err != nil {
		log.Fatal(err)
	}

	nibiruSdk, err := gosdk.NewNibiruSdk(netInfo.CmtChainID, grpcConn, netInfo.TmRpcEndpoint)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("nibiruSdk: %#v\n", nibiruSdk)
}
