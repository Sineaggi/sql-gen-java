package core

import (
	"sort"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

type Importer struct {
	Settings    *plugin.Settings
	DataClasses []Struct
	Rows        []*Struct
	Enums       []Enum
	Queries     []Query
}

func (i *Importer) Imports(filename string) [][]string {
	switch filename {
	case "Queries.java":
		return i.interfaceImports()
	case "QueriesImpl.java":
		return i.implImports()
	}
	for _, e := range i.Enums {
		if filename == e.Name+".java" {
			return i.enumImports()
		}
	}
	for idx := range i.DataClasses {
		if filename == i.DataClasses[idx].Name+".java" {
			return i.structImports(&i.DataClasses[idx])
		}
	}
	for idx := range i.Rows {
		if filename == i.Rows[idx].Name+".java" {
			return i.structImports(i.Rows[idx])
		}
	}
	return nil
}

func addTypeImports(std map[string]struct{}, t javaType) {
	switch {
	case t.IsInstant():
		std["java.time.Instant"] = struct{}{}
	case t.Name == "LocalDate":
		std["java.time.LocalDate"] = struct{}{}
	case t.Name == "LocalTime":
		std["java.time.LocalTime"] = struct{}{}
	case t.Name == "LocalDateTime":
		std["java.time.LocalDateTime"] = struct{}{}
	case t.Name == "OffsetDateTime":
		std["java.time.OffsetDateTime"] = struct{}{}
	case t.IsUUID():
		std["java.util.UUID"] = struct{}{}
	case t.IsBigDecimal():
		std["java.math.BigDecimal"] = struct{}{}
	}
}

// enumImports covers the standard-library imports every generated enum
// file needs for its static lookup map.
func (i *Importer) enumImports() [][]string {
	std := map[string]struct{}{
		"java.util.Arrays":            {},
		"java.util.Map":               {},
		"java.util.stream.Collectors": {},
	}
	if EmitNullableAnnotations {
		std["org.jspecify.annotations.Nullable"] = struct{}{}
	}
	return toGroups(std)
}

func (i *Importer) structImports(s *Struct) [][]string {
	std := map[string]struct{}{}
	hasList := false
	hasNullable := false
	for _, f := range s.Fields {
		addTypeImports(std, f.Type)
		if f.Type.IsArray {
			hasList = true
		} else if f.Type.IsNull {
			hasNullable = true
		}
	}
	if hasList {
		std["java.util.List"] = struct{}{}
	}
	if hasNullable && EmitNullableAnnotations {
		std["org.jspecify.annotations.Nullable"] = struct{}{}
	}
	return toGroups(std)
}

func (i *Importer) queryUses(pred func(t javaType) bool) bool {
	for _, q := range i.Queries {
		if !q.Arg.isEmpty() {
			for _, f := range q.Arg.Struct.Fields {
				if pred(f.Type) {
					return true
				}
			}
		}
		if !q.Ret.isEmpty() {
			if q.Ret.Struct != nil {
				for _, f := range q.Ret.Struct.Fields {
					if pred(f.Type) {
						return true
					}
				}
			} else if pred(q.Ret.Typ) {
				return true
			}
		}
	}
	return false
}

func (i *Importer) hasListUsage() bool {
	return i.queryUses(func(t javaType) bool { return t.IsArray }) || i.hasManyUsage()
}

func (i *Importer) hasManyUsage() bool {
	for _, q := range i.Queries {
		if q.Cmd == ":many" {
			return true
		}
	}
	return false
}

func (i *Importer) hasNullableUsage() bool {
	for _, q := range i.Queries {
		if q.Cmd == ":one" {
			return true
		}
	}
	return i.queryUses(func(t javaType) bool { return t.IsNull && !t.IsArray })
}

func (i *Importer) hasEnumUsage() bool {
	return i.queryUses(func(t javaType) bool { return t.IsEnum })
}

func (i *Importer) baseQueryImports() map[string]struct{} {
	std := map[string]struct{}{
		"java.sql.SQLException": {},
	}
	for _, name := range []string{"LocalDate", "LocalTime", "LocalDateTime", "OffsetDateTime"} {
		n := name
		if i.queryUses(func(t javaType) bool { return t.Name == n }) {
			std["java.time."+n] = struct{}{}
		}
	}
	if i.queryUses(func(t javaType) bool { return t.IsInstant() }) {
		std["java.time.Instant"] = struct{}{}
	}
	if i.queryUses(func(t javaType) bool { return t.IsUUID() }) {
		std["java.util.UUID"] = struct{}{}
	}
	if i.queryUses(func(t javaType) bool { return t.IsBigDecimal() }) {
		std["java.math.BigDecimal"] = struct{}{}
	}
	if i.hasListUsage() {
		std["java.util.List"] = struct{}{}
	}
	if i.hasNullableUsage() && EmitNullableAnnotations {
		std["org.jspecify.annotations.Nullable"] = struct{}{}
	}
	return std
}

func (i *Importer) interfaceImports() [][]string {
	return toGroups(i.baseQueryImports())
}

func (i *Importer) implImports() [][]string {
	std := i.baseQueryImports()
	std["java.sql.Connection"] = struct{}{}
	std["java.sql.Statement"] = struct{}{}
	if i.hasListUsage() {
		std["java.util.ArrayList"] = struct{}{}
	}
	if i.hasEnumUsage() {
		std["java.util.Arrays"] = struct{}{}
		std["java.util.Objects"] = struct{}{}
		std["java.util.stream.Collectors"] = struct{}{}
	}
	if i.hasEnumUsage() && i.Settings.Engine == "postgresql" {
		std["java.sql.Types"] = struct{}{}
	}
	if i.queryUses(func(t javaType) bool { return t.IsInstant() }) {
		std["java.sql.Timestamp"] = struct{}{}
	}
	return toGroups(std)
}

func toGroups(std map[string]struct{}) [][]string {
	stds := make([]string, 0, len(std))
	for s := range std {
		stds = append(stds, s)
	}
	sort.Strings(stds)
	return [][]string{stds}
}
