package cast

import "encoding/json"

func AnyToJSONT[T any](input any) (T, error) {
	var result T
	data, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}
