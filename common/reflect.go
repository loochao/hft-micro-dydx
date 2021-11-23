package common

import (
	"reflect"
	"strings"
)

func DetectDefaultValues(value interface{}, exceptions []string) (bool, []string) {
	if exceptions == nil {
		exceptions = make([]string, 0)
	} else {
		for i := range exceptions {
			exceptions[i] = strings.ToLower(exceptions[i])
		}
	}
	empty := reflect.New(reflect.TypeOf(value))

	t := reflect.TypeOf(value)
	v1 := reflect.ValueOf(value)
	v2 := empty.Elem()
	hasDefaultValue := false
	defaultFields := make([]string, 0)
outerLoop:
	for i := 0; i < v1.NumField(); i++ {
		name := strings.ToLower(t.Field(i).Name)
		for _, e := range exceptions {
			if e == name {
				continue outerLoop
			}
		}
		if v1.Field(i).Interface() == v2.Field(i).Interface() {
			hasDefaultValue = true
			defaultFields = append(defaultFields, t.Field(i).Name)
		}
	}
	return hasDefaultValue, defaultFields
}
