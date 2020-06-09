package utils

import (
	"bytes"
	"encoding/json"
)

// JSONPrettyPrint предоставляет слайс байт в виде удобочитаемого json объекта
func JSONPrettyPrint(in []byte) []byte {
	var out bytes.Buffer
	err := json.Indent(&out, in, "", "\t")
	if err != nil {
		return in
	}
	return out.Bytes()
}
