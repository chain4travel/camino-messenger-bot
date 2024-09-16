package cheques

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// TODO@ code was copy-pasted from eth; does it need license header?

func hashStructWithTypeHash(typedData *apitypes.TypedData, dataType string, typeHash []byte) ([]byte, error) {
	if exp, got := len(typedData.Types[dataType]), len(typedData.Message); exp < got {
		return nil, fmt.Errorf("there is extra data provided in the message (%d < %d)", exp, got)
	}

	depth := 1
	buffer := bytes.Buffer{}
	buffer.Write(typeHash)

	for _, field := range typedData.Types[chequeType] {
		encType := field.Type
		encValue := typedData.Message[field.Name]
		if encType[len(encType)-1:] == "]" {
			arrayValue, err := convertDataToSlice(encValue)
			if err != nil {
				return nil, dataMismatchError(encType, encValue)
			}

			arrayBuffer := bytes.Buffer{}
			parsedType := strings.Split(encType, "[")[0]
			for _, item := range arrayValue {
				if typedData.Types[parsedType] != nil {
					mapValue, ok := item.(map[string]interface{})
					if !ok {
						return nil, dataMismatchError(parsedType, item)
					}
					encodedData, err := typedData.EncodeData(parsedType, mapValue, depth+1)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(crypto.Keccak256(encodedData))
				} else {
					bytesValue, err := typedData.EncodePrimitiveValue(parsedType, item, depth)
					if err != nil {
						return nil, err
					}
					arrayBuffer.Write(bytesValue)
				}
			}

			buffer.Write(crypto.Keccak256(arrayBuffer.Bytes()))
		} else if typedData.Types[field.Type] != nil {
			mapValue, ok := encValue.(map[string]interface{})
			if !ok {
				return nil, dataMismatchError(encType, encValue)
			}
			encodedData, err := typedData.EncodeData(field.Type, mapValue, depth+1)
			if err != nil {
				return nil, err
			}
			buffer.Write(crypto.Keccak256(encodedData))
		} else {
			byteValue, err := typedData.EncodePrimitiveValue(encType, encValue, depth)
			if err != nil {
				return nil, err
			}
			buffer.Write(byteValue)
		}
	}
	return crypto.Keccak256(buffer.Bytes()), nil
}

func convertDataToSlice(encValue interface{}) ([]interface{}, error) {
	var outEncValue []interface{}
	rv := reflect.ValueOf(encValue)
	if rv.Kind() == reflect.Slice {
		for i := 0; i < rv.Len(); i++ {
			outEncValue = append(outEncValue, rv.Index(i).Interface())
		}
	} else {
		return outEncValue, fmt.Errorf("provided data '%v' is not slice", encValue)
	}
	return outEncValue, nil
}

func dataMismatchError(encType string, encValue interface{}) error {
	return fmt.Errorf("provided data '%v' doesn't match type '%s'", encValue, encType)
}
