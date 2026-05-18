package unitdetect

import "fmt"

func hasAnyKey(data map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := data[key]; ok {
			return true
		}
	}
	return false
}

func asString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func statusData(statusJSON map[string]interface{}) map[string]interface{} {
	if data, ok := statusJSON["data"].(map[string]interface{}); ok {
		return data
	}
	return statusJSON
}
