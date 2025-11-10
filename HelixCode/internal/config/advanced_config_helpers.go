package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Fixed helper functions for advanced config

func (v *ConfigurationValidator) getNumberValue(value interface{}) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case string:
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

func (v *ConfigurationValidator) deepCopy(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (t *ConfigurationTransformer) getNumberValue(value interface{}) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int8:
		return float64(value), true
	case int16:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case string:
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

func (t *ConfigurationTransformer) convertValueForField(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	sourceType := reflect.TypeOf(value)

	// If types match, return as-is
	if sourceType == targetType {
		return value, nil
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Convert based on target type
	switch targetType.Kind() {
	case reflect.String:
		return fmt.Sprintf("%v", value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, ok := t.getNumberValue(value); ok {
			return t.convertToInt(num, targetType), nil
		}
		return nil, fmt.Errorf("cannot convert %v to %v", value, targetType)
	case reflect.Float32, reflect.Float64:
		if num, ok := t.getNumberValue(value); ok {
			return float64(num), nil
		}
		return nil, fmt.Errorf("cannot convert %v to %v", value, targetType)
	case reflect.Bool:
		if str, ok := value.(string); ok {
			return t.parseBool(str), nil
		}
		if b, ok := value.(bool); ok {
			return b, nil
		}
		return nil, fmt.Errorf("cannot convert %v to bool", value)
	default:
		// For complex types, use JSON marshaling
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}

		target := reflect.New(targetType).Interface()
		return target, json.Unmarshal(data, &target)
	}
}

func (t *ConfigurationTransformer) convertToInt(value float64, targetType reflect.Type) interface{} {
	switch targetType.Kind() {
	case reflect.Int:
		return int(value)
	case reflect.Int8:
		return int8(value)
	case reflect.Int16:
		return int16(value)
	case reflect.Int32:
		return int32(value)
	case reflect.Int64:
		return int64(value)
	default:
		return int(value)
	}
}

func (t *ConfigurationTransformer) parseBool(s string) bool {
	return strings.ToLower(s) == "true" || s == "1" || s == "yes" || s == "on"
}

func (tm *ConfigurationTemplateManager) getNumberValue(value interface{}) (float64, bool) {
	switch value := value.(type) {
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	case int64:
		return float64(value), true
	case float32:
		return float64(value), true
	case float64:
		return value, true
	case string:
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return num, true
		}
	}
	return 0, false
}

func (tm *ConfigurationTemplateManager) deepCopy(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func intPtr(i int) *int             { return &i }
func float64Ptr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool          { return &b }
