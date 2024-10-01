package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

var correctKey = "7374617465"
var keyHuman = "state"

func main() {
	fmt.Printf("keyHuman: %s\n", keyHuman)
	fmt.Printf("correctKey: %v\n", correctKey)
	fmt.Printf("[]byte(keyHuman): %s\n", []byte(keyHuman))
	fmt.Printf("[]byte(keyHuman): %v\n", []byte(keyHuman))

	keyHumanBz := []byte(keyHuman)
	hexKeyHuman := hex.EncodeToString(keyHumanBz)

	fmt.Printf("hexKeyHuman: %v\n", hexKeyHuman)
	fmt.Printf("hexKeyHuman: %s\n", hexKeyHuman)

	keyBz := []byte(correctKey)
	hexKeyReal := hex.EncodeToString(keyBz)
	fmt.Printf("hexKeyReal: %v\n", hexKeyReal)
	fmt.Printf("hexKeyReal: %s\n", hexKeyReal)

	// {
	//   "data": "eyJjb3VudCI6MCwib3duZXIiOiJuaWJpMXphYXZ2enhlejBlbHVuZHRuMzJxbms5bGttOGttY3N6NDRnN3hsIn0="
	// }

	dataBz := "eyJjb3VudCI6MCwib3duZXIiOiJuaWJpMXphYXZ2enhlejBlbHVuZHRuMzJxbms5bGttOGttY3N6NDRnN3hsIn0="
	decodedBz, err := base64.StdEncoding.DecodeString(dataBz)
	if err != nil {
		panic(err)
	}

	fmt.Printf("decodedBz: %v\n", decodedBz)
	fmt.Printf("decodedBz: %s\n", decodedBz)
}
