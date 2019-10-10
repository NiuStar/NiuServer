package STNet

import (
	"reflect"
)

func verificationStruct(type_ reflect.Type, list map[string]interface{}) *string {

	for i := 0; i < type_.NumField(); i++ {
		errorName1 := verificationField(type_.Field(i), list)
		if verificationField(type_.Field(i), list) != nil {
			return errorName1
		}
	}
	return nil
}

func verificationField(field reflect.StructField, list map[string]interface{}) *string {

	fieldTag := field.Tag
	if len(fieldTag.Get("json")) > 0 {
		if fieldTag.Get("required") == "yes" && list[fieldTag.Get("json")] == nil {
			errorName := fieldTag.Get("json")
			return &errorName
		} else if fieldTag.Get("required") == "yes" {

			fieldType := field.Type
			for reflect.Ptr == fieldType.Kind() || reflect.Interface == fieldType.Kind() {
				fieldType = fieldType.Elem()
			}

			if fieldType.Kind() == reflect.Struct {
				errorName := verificationStruct(fieldType, list[fieldTag.Get("json")].(map[string]interface{}))
				if errorName != nil {
					errorName1 := fieldTag.Get("json") + "_" + *errorName
					return &errorName1
				}
			}
		}
	}

	return nil
}
