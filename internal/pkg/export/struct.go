package export

import (
	"reflect"
	"strings"
)

// FromStructs builds a Table from a slice of uniform structs. Column headers
// come from the `export:"Header"` tag when present, otherwise the Go field
// name. Fields tagged `export:"-"` or `json:"-"` are skipped, unexported fields
// are ignored, and numeric fields default to right alignment. Accepts a slice
// of struct or *struct.
func FromStructs(items interface{}) (Table, error) {
	rv := reflect.ValueOf(items)
	if rv.Kind() != reflect.Slice {
		return Table{}, ErrNotSlice
	}
	if rv.Len() == 0 {
		elem := rv.Type().Elem()
		cols := columnsFor(underlyingStructType(elem))
		return Table{Columns: cols}, nil
	}

	first := rv.Index(0)
	fields := exportableFields(first)
	cols := make([]Column, 0, len(fields))
	for _, f := range fields {
		cols = append(cols, Column{Header: f.header, Key: f.key, Align: f.align})
	}

	table := Table{Columns: cols, Rows: make([]Row, 0, rv.Len())}
	for i := 0; i < rv.Len(); i++ {
		row := make(Row, len(fields))
		v := rv.Index(i)
		for _, f := range fields {
			row[f.key] = f.value(v)
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

type fieldSpec struct {
	key    string
	header string
	align  string
	index  int
}

func (f fieldSpec) value(v reflect.Value) interface{} {
	v = deref(v)
	if v.Kind() != reflect.Struct {
		return nil
	}
	fv := v.Field(f.index)
	if !fv.CanInterface() {
		return nil
	}
	return fv.Interface()
}

func exportableFields(v reflect.Value) []fieldSpec {
	t := underlyingStructType(v.Type())
	if t == nil {
		return nil
	}
	var specs []fieldSpec
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := parseExportTag(sf)
		if tag == "-" {
			continue
		}
		key := sf.Name
		header := sf.Name
		if tag != "" {
			header = tag
		}
		specs = append(specs, fieldSpec{
			key:    key,
			header: header,
			align:  alignForType(sf.Type),
			index:  i,
		})
	}
	return specs
}

func columnsFor(t reflect.Type) []Column {
	if t == nil {
		return nil
	}
	var cols []Column
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		tag := parseExportTag(sf)
		if tag == "-" {
			continue
		}
		header := sf.Name
		if tag != "" {
			header = tag
		}
		cols = append(cols, Column{Header: header, Key: sf.Name, Align: alignForType(sf.Type)})
	}
	return cols
}

func parseExportTag(sf reflect.StructField) string {
	if tag, ok := sf.Tag.Lookup("export"); ok {
		return strings.Split(tag, ",")[0]
	}
	if tag, ok := sf.Tag.Lookup("json"); ok {
		if strings.Split(tag, ",")[0] == "-" {
			return "-"
		}
	}
	return ""
}

func alignForType(t reflect.Type) string {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "right"
	default:
		return "left"
	}
}

func underlyingStructType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	return t
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}
