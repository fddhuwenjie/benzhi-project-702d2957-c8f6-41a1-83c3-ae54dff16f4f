package domain

import "encoding/json"

func Clone(c *DeviationCase) (*DeviationCase, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var out DeviationCase
	if err = json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
