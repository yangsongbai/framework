/* Copyright © INFINI Ltd. All rights reserved.
 * web: https://infinilabs.com
 * mail: hello#infini.ltd */

package util

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Invoke dynamic execute function via function name and parameters
func Invoke(any interface{}, name string, args ...interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	reflect.ValueOf(any).MethodByName(name).Call(inputs)
}

type Annotation struct {
	Field      string       `json:"field,omitempty"`
	Type       string       `json:"type,omitempty"`
	Tag        string       `json:"tag,omitempty"`
	Annotation []Annotation `json:"annotation,omitempty"`
}

// source should be a struct, target should be a pointer to the struct
func Copy(sourceStruct interface{}, pointToTarget interface{}) (err error) {
	dst := reflect.ValueOf(pointToTarget)
	if dst.Kind() != reflect.Ptr {
		err = errors.New("target is not a pointer")
		return
	}

	element := dst.Elem()
	if element.Kind() != reflect.Struct {
		err = errors.New("target doesn't point to struct")
		return
	}

	srcValue := reflect.ValueOf(sourceStruct)
	srcType := reflect.TypeOf(sourceStruct)
	if srcType.Kind() != reflect.Struct {
		err = errors.New("source is not a struct")
		return
	}

	for i := 0; i < srcType.NumField(); i++ {
		sf := srcType.Field(i)
		sv := srcValue.FieldByName(sf.Name)
		if dv := element.FieldByName(sf.Name); dv.IsValid() && dv.CanSet() {
			dv.Set(sv)
		}
	}
	return
}

func GetTagsByTagName(any interface{}, tagName string) []Annotation {

	t := reflect.TypeOf(any)

	var result []Annotation

	//check if it is as point
	if PrefixStr(t.String(), "*") {
		t = reflect.TypeOf(any).Elem()
	}

	//fmt.Println("")
	//fmt.Println("o: ",any,", ",tagName)
	//fmt.Println("t: ",t,", ",tagName)

	if t.Kind() == reflect.Struct {

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			v := TrimSpaces(field.Tag.Get(tagName))
			if v == "-" {
				continue
			}
			a := Annotation{Field: field.Name, Type: field.Type.Name(), Tag: v}

			//fmt.Println(field.Name)
			//fmt.Println(field.Type)
			//fmt.Println(field.Type.Kind())
			//fmt.Println(field.Tag)
			//fmt.Println(field.Type.Elem())

			if field.Type.Kind() == reflect.Slice || field.Type.Kind() == reflect.Ptr {
				v1 := reflect.New(field.Type.Elem())
				a.Annotation = GetTagsByTagName(v1.Interface(), tagName)
			}

			if field.Type.Kind() == reflect.Struct {
				v1 := reflect.New(field.Type)
				a.Annotation = GetTagsByTagName(v1.Interface(), tagName)
				if field.Anonymous && len(a.Annotation) > 0{
					result = append(result, a.Annotation...)
					continue
				}
			}

			if len(a.Annotation) > 0 || a.Tag != "" {
				result = append(result, a)
			}
		}

	}

	return result
}

// GetFieldValueByTagName return the field value which field was tagged with this tagName, only support string field
func GetFieldValueByTagName(any interface{}, tagName string, tagValue string) string {

	t := reflect.TypeOf(any)
	v := reflect.ValueOf(any)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	if PrefixStr(t.String(), "*") {
		t = reflect.TypeOf(any).Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		switch v.Field(i).Kind() {
		case reflect.Struct:
			//判断是否是嵌套结构
			if v.Field(i).Type().Kind() == reflect.Struct {
				structField := v.Field(i).Type()
				for j := 0; j < structField.NumField(); j++ {
					v := structField.Field(j).Tag.Get(tagName)
					if v != "" {
						if ContainTags(tagValue, v) {
							return reflect.Indirect(reflect.ValueOf(any)).FieldByName(structField.Field(i).Name).String()
						}
					}
				}
				continue
			}
			break
		default:
			v := t.Field(i).Tag.Get(tagName)
			if v != "" {
				if ContainTags(tagValue, v) {
					return reflect.Indirect(reflect.ValueOf(any)).FieldByName(t.Field(i).Name).String()
				}
			}
			break
		}
	}

	//TODO handle property in parent/inner objects
	panic(fmt.Errorf("tag [%v][%v] was not found", tagName, tagValue))
}

func GetTypeName(any interface{}, lowercase bool) string {
	_, t := GetTypeAndPackageName(any, lowercase)
	return t
}

func GetTypeAndPackageName(any interface{}, lowercase bool) (string, string) {
	pkg := reflect.Indirect(reflect.ValueOf(any)).Type().PkgPath()
	name := reflect.Indirect(reflect.ValueOf(any)).Type().Name()
	if lowercase {
		name = strings.ToLower(name)
	}
	return pkg, name
}

func TypeIsArray(any interface{}) bool {
	vt := reflect.TypeOf(any)
	if vt.String() == "[]interface {}" {
		return true
	}
	return false
}

func TypeIsMap(any interface{}) bool {
	vt := reflect.TypeOf(any)
	if vt.String() == "map[string]interface {}" {
		return true
	}
	return false
}

func GetInt64Value(any interface{}) int64 {

	vt := reflect.TypeOf(any)
	if vt.String() == "float64" {
		v := any.(float64)
		var y = int64(v)
		return y
	} else if vt.String() == "float32" {
		v := any.(float32)
		var y = int64(v)
		return y
	} else if vt.String() == "int64" {
		v := any.(int64)
		var y = int64(v)
		return y
	} else if vt.String() == "int32" {
		v := any.(int32)
		var y = int64(v)
		return y
	} else if vt.String() == "uint64" {
		v := any.(uint64)
		var y = int64(v)
		return y
	}else if vt.String() == "uint32" {
		v := any.(uint32)
		var y = int64(v)
		return y
	} else if vt.String() == "uint" {
		v := any.(uint)
		var y = int64(v)
		return y
	} else if vt.String() == "int" {
		v := any.(int)
		var y = int64(v)
		return y
	}
	return -1
}

func ContainTags(tag string, tags string) bool {
	if strings.Contains(tags, ",") {
		arr := strings.Split(tags, ",")
		for _, v := range arr {
			if v == tag {
				return true
			}
		}
	}
	return tag == tags
}

//return field and tags, field name is using key: NAME
func GetFieldAndTags(any interface{},tags[]string) []map[string]string{

	fields:=[]map[string]string{}

	t := reflect.TypeOf(any)
	v := reflect.ValueOf(any)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	if PrefixStr(t.String(), "*") {
		t = reflect.TypeOf(any).Elem()
	}

	for i := 0; i < v.NumField(); i++ {

		field:=map[string]string{}
		field["NAME"]=t.Field(i).Name
		field["TYPE"]=t.Field(i).Type.Name()
		field["KIND"]=v.Field(i).Kind().String()

		if v.Field(i).Kind()==reflect.Slice{
			field["TYPE"]="array"
			field["SUB_TYPE"]=v.Field(i).Type().Elem().String()
		}

		switch v.Field(i).Kind() {
		case reflect.Struct:
			//判断是否是嵌套结构
			if v.Field(i).Type().Kind() == reflect.Struct {
				structField := v.Field(i).Type()
				for j := 0; j < structField.NumField(); j++ {
					for _,tagName:=range tags{
						v := structField.Field(j).Tag.Get(tagName)
						if v!=""{
							field[tagName]=v
						}
					}
				}
				continue
			}
			break
		default:
			for _,tagName:=range tags{
				v := t.Field(i).Tag.Get(tagName)
				if v!=""{
					field[tagName]=v
				}
			}
			break
		}
		fields=append(fields,field)
	}

	return fields
}
