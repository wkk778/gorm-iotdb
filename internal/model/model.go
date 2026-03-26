// Package model contains internal helpers for mapping GORM schemas to IoTDB concepts.
package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm/schema"
)

// ColumnRole describes how a field participates in an IoTDB table model.
type ColumnRole int

const (
	// ColumnRoleField marks a measurement field.
	ColumnRoleField ColumnRole = iota
	// ColumnRoleTag marks a tag column.
	ColumnRoleTag
	// ColumnRoleTime marks the timestamp column.
	ColumnRoleTime
)

// Column describes a parsed IoTDB column.
type Column struct {
	Field *schema.Field
	Role  ColumnRole
}

// ParseColumns extracts IoTDB column roles from a GORM schema.
func ParseColumns(s *schema.Schema) []Column {
	columns := make([]Column, 0, len(s.Fields))
	for _, field := range s.Fields {
		columns = append(columns, Column{
			Field: field,
			Role:  parseRole(field),
		})
	}
	return columns
}

// TagValueMap extracts tag values from a model instance.
func TagValueMap(s *schema.Schema, value reflect.Value) map[string]any {
	columns := ParseColumns(s)
	tags := make(map[string]any)
	for _, column := range columns {
		if column.Role != ColumnRoleTag {
			continue
		}
		v, zero := column.Field.ValueOf(context.Background(), value)
		if zero {
			continue
		}
		tags[column.Field.DBName] = v
	}
	return tags
}

// FindField resolves a field by GORM name or DB name.
func FindField(s *schema.Schema, name string) (*schema.Field, error) {
	if field := s.LookUpField(name); field != nil {
		return field, nil
	}
	return nil, fmt.Errorf("iotdb: field %q not found on schema %s", name, s.Name)
}

func parseRole(field *schema.Field) ColumnRole {
	tag := strings.ToLower(field.TagSettings["IOTDB"])
	switch {
	case strings.Contains(tag, "time"):
		return ColumnRoleTime
	case strings.Contains(tag, "tag"):
		return ColumnRoleTag
	default:
		if field.DataType == schema.Time {
			return ColumnRoleTime
		}
		return ColumnRoleField
	}
}
