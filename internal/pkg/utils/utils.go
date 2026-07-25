package utils

import "encoding/json"

func DetectNullFields(body []byte) []string {
	var rawMap map[string]*json.RawMessage
	if err := json.Unmarshal(body, &rawMap); err != nil {
		return nil
	}
	var nullFields []string
	for k, v := range rawMap {
		if v == nil || string(*v) == "null" {
			nullFields = append(nullFields, k)
		}
	}
	return nullFields
}
