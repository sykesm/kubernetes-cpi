package cpi

import "encoding/json"

func Remarshal(source interface{}, target interface{}) error {
	encoded, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(encoded, target)
}
