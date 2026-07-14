package db

import (
	"fmt"
	"reflect"
	"strings"
)

type QueryBuilder struct {
	clauses []string
	args    []interface{}
}

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

func (b *QueryBuilder) AddIfNotNil(clause string, value interface{}) {
	if isNil(value) {
		return
	}
	b.args = append(b.args, value)
	b.clauses = append(b.clauses, fmt.Sprintf(clause, len(b.args)))
}

func (b *QueryBuilder) Build(separator string) string {
	return strings.Join(b.clauses, separator)
}

func (b *QueryBuilder) Args() []interface{} {
	return append([]interface{}(nil), b.args...)
}
