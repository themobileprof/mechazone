package pii

import (
	"encoding/json"
	"fmt"
	"strings"
)

var forbidden = []string{
	"customer_name", "full_name", "phone", "phone_number",
	"plate", "license_plate", "number_plate",
}

// RejectCloudPII returns an error if a cloud payload includes customer identity fields.
func RejectCloudPII(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return walk(obj, "")
}

func walk(v any, prefix string) error {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := strings.ToLower(k)
			for _, f := range forbidden {
				if key == f {
					return fmt.Errorf("customer identity field %q is not allowed on the cloud ledger", k)
				}
			}
			next := k
			if prefix != "" {
				next = prefix + "." + k
			}
			if err := walk(child, next); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range t {
			if err := walk(child, prefix); err != nil {
				return err
			}
		}
	}
	return nil
}
