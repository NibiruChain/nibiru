package cligen

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"reflect"
)

type Param struct {
	Name      string
	Type      string
	Mandatory bool
}

type Params []Param

func (p Params) Mandatory() Params {
	mandatoryParams := Params{}
	for _, param := range p {
		if param.Mandatory {
			mandatoryParams = append(mandatoryParams, param)
		}
	}

	return mandatoryParams
}

// DefaultFromProtoMsg TODO
func DefaultFromProtoMsg(message proto.Message) Params {
	protoType := reflect.ValueOf(message).Elem()

	for i := 0; i < protoType.NumField(); i++ {
		field := protoType.Field(i)
		fmt.Println(field.Type().Name())
		fmt.Println(field.Kind().String())
	}

	fmt.Printf("message: %v", protoType)

	return Params{}
}
