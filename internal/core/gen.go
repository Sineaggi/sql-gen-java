package core

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"

	"github.com/Sineaggi/sqlc-gen-java/internal/inflection"
)

var javaIdentPattern = regexp.MustCompile("[^a-zA-Z0-9_]+")

// EmitNullableAnnotations controls whether nullable types are annotated with
// org.jspecify.annotations.Nullable. It is set once per Generate() call from
// the plugin config.
var EmitNullableAnnotations = true

type Constant struct {
	Name  string
	Type  string
	Value string
}

type Enum struct {
	Name      string
	Comment   string
	Constants []Constant
}

type Field struct {
	ID      int
	Name    string
	Type    javaType
	Comment string
}

type Struct struct {
	Table   plugin.Identifier
	Name    string
	Fields  []Field
	Comment string
}

type QueryValue struct {
	Emit   bool
	Name   string
	Struct *Struct
	Typ    javaType
}

func (v QueryValue) EmitStruct() bool {
	return v.Emit
}

func (v QueryValue) IsStruct() bool {
	return v.Struct != nil
}

func (v QueryValue) isEmpty() bool {
	return v.Typ == (javaType{}) && v.Name == "" && v.Struct == nil
}

// Type renders the value's declared type, honoring the value's own
// nullability (used for :many element types, params, and struct fields).
func (v QueryValue) Type() string {
	if v.Typ != (javaType{}) {
		return v.Typ.String()
	}
	if v.Struct != nil {
		return v.Struct.Name
	}
	panic("no type for QueryValue: " + v.Name)
}

// NullableType renders the value's declared type as if it were nullable,
// regardless of the underlying column's nullability. Used for :one return
// types, since "no rows found" makes the result nullable no matter what.
func (v QueryValue) NullableType() string {
	if v.Struct != nil {
		return nullableRef(v.Struct.Name)
	}
	return v.Typ.declare(true)
}

func nullableRef(name string) string {
	if EmitNullableAnnotations {
		return "@Nullable " + name
	}
	return name
}

func isPrimitiveBacked(name string) bool {
	switch name {
	case "Int", "Long", "Short", "Float", "Double", "Boolean":
		return true
	}
	return false
}

func jdbcSet(t javaType, idx int, name string) string {
	if t.IsEnum && t.IsArray {
		return fmt.Sprintf(`stmt.setArray(%d, conn.createArrayOf("%s", %s.stream().map(v -> v.value()).toArray(String[]::new)))`, idx, t.DataType, name)
	}
	if t.IsEnum {
		if t.Engine == "postgresql" {
			return fmt.Sprintf("stmt.setObject(%d, %s.value(), Types.OTHER)", idx, name)
		}
		return fmt.Sprintf("stmt.setString(%d, %s.value())", idx, name)
	}
	if t.IsArray {
		return fmt.Sprintf(`stmt.setArray(%d, conn.createArrayOf("%s", %s.toArray(new %s[0])))`, idx, t.DataType, name, t.boxed())
	}
	if t.IsTime() {
		return fmt.Sprintf("stmt.setObject(%d, %s)", idx, name)
	}
	if t.IsInstant() {
		return fmt.Sprintf("stmt.setTimestamp(%d, Timestamp.from(%s))", idx, name)
	}
	if t.IsUUID() {
		return fmt.Sprintf("stmt.setObject(%d, %s)", idx, name)
	}
	if t.IsBigDecimal() {
		return fmt.Sprintf("stmt.setBigDecimal(%d, %s)", idx, name)
	}
	if t.IsNull && isPrimitiveBacked(t.Name) {
		// setInt/setLong/etc. take a primitive argument, so a null boxed
		// value would NPE on autounboxing. setObject accepts null safely.
		return fmt.Sprintf("stmt.setObject(%d, %s)", idx, name)
	}
	return fmt.Sprintf("stmt.set%s(%d, %s)", t.Name, idx, name)
}

type Params struct {
	Struct  *Struct
	binding []int
}

func (v Params) isEmpty() bool {
	return len(v.Struct.Fields) == 0
}

func (v Params) Args() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	fields := v.Struct.Fields
	for _, f := range fields {
		out = append(out, f.Type.String()+" "+f.Name)
	}
	if len(v.binding) > 0 {
		lookup := map[int]int{}
		for i, v := range v.binding {
			lookup[v] = i
		}
		sort.Slice(out, func(i, j int) bool {
			return lookup[fields[i].ID] < lookup[fields[j].ID]
		})
	}
	if len(out) < 3 {
		return strings.Join(out, ", ")
	}
	return "\n" + indent(strings.Join(out, ",\n"), 6, -1)
}

func (v Params) Bindings() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if len(v.binding) > 0 {
		for i, idx := range v.binding {
			f := v.Struct.Fields[idx-1]
			out = append(out, jdbcSet(f.Type, i+1, f.Name)+";")
		}
	} else {
		for i, f := range v.Struct.Fields {
			out = append(out, jdbcSet(f.Type, i+1, f.Name)+";")
		}
	}
	return indent(strings.Join(out, "\n"), 10, 0)
}

func jdbcGet(t javaType, idx int) string {
	if t.IsEnum && t.IsArray {
		return fmt.Sprintf(`Arrays.stream((String[]) results.getArray(%d).getArray()).map(v -> Objects.requireNonNull(%s.lookup(v))).collect(Collectors.toList())`, idx, t.Name)
	}
	if t.IsEnum {
		return fmt.Sprintf("Objects.requireNonNull(%s.lookup(results.getString(%d)))", t.Name, idx)
	}
	if t.IsArray {
		return fmt.Sprintf(`Arrays.asList((%s[]) results.getArray(%d).getArray())`, t.boxed(), idx)
	}
	if t.IsTime() {
		return fmt.Sprintf(`results.getObject(%d, %s.class)`, idx, t.Name)
	}
	if t.IsInstant() {
		return fmt.Sprintf(`results.getTimestamp(%d).toInstant()`, idx)
	}
	if t.IsUUID() {
		return fmt.Sprintf(`(UUID) results.getObject(%d)`, idx)
	}
	if t.IsBigDecimal() {
		return fmt.Sprintf(`results.getBigDecimal(%d)`, idx)
	}
	return fmt.Sprintf(`results.get%s(%d)`, t.Name, idx)
}

func (v QueryValue) ResultSet() string {
	var out []string
	if v.Struct == nil {
		return jdbcGet(v.Typ, 1)
	}
	for i, f := range v.Struct.Fields {
		out = append(out, jdbcGet(f.Type, i+1))
	}
	ret := indent(strings.Join(out, ",\n"), 4, -1)
	ret = indent("new "+v.Struct.Name+"(\n"+ret+"\n)", 12, 0)
	return ret
}

func indent(s string, n int, firstIndent int) string {
	lines := strings.Split(s, "\n")
	buf := bytes.NewBuffer(nil)
	for i, l := range lines {
		indent := n
		if i == 0 && firstIndent != -1 {
			indent = firstIndent
		}
		if i != 0 {
			buf.WriteRune('\n')
		}
		for i := 0; i < indent; i++ {
			buf.WriteRune(' ')
		}
		buf.WriteString(l)
	}
	return buf.String()
}

// A struct used to generate methods and fields on the Queries struct
type Query struct {
	ClassName    string
	Cmd          string
	Comments     []string
	MethodName   string
	FieldName    string
	ConstantName string
	SQL          string
	SourceName   string
	Ret          QueryValue
	Arg          Params
}

func javaEnumValueName(value string) string {
	id := strings.Replace(value, "-", "_", -1)
	id = strings.Replace(id, ":", "_", -1)
	id = strings.Replace(id, "/", "_", -1)
	id = javaIdentPattern.ReplaceAllString(id, "")
	return strings.ToUpper(id)
}

func BuildEnums(req *plugin.GenerateRequest) []Enum {
	var enums []Enum
	for _, schema := range req.Catalog.Schemas {
		if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
			continue
		}
		for _, enum := range schema.Enums {
			var enumName string
			if schema.Name == req.Catalog.DefaultSchema {
				enumName = enum.Name
			} else {
				enumName = schema.Name + "_" + enum.Name
			}
			e := Enum{
				Name:    dataClassName(enumName, req.Settings),
				Comment: enum.Comment,
			}
			for _, v := range enum.Vals {
				e.Constants = append(e.Constants, Constant{
					Name:  javaEnumValueName(v),
					Value: v,
					Type:  e.Name,
				})
			}
			enums = append(enums, e)
		}
	}
	if len(enums) > 0 {
		sort.Slice(enums, func(i, j int) bool { return enums[i].Name < enums[j].Name })
	}
	return enums
}

func dataClassName(name string, settings *plugin.Settings) string {
	out := ""
	for _, p := range strings.Split(name, "_") {
		out += strings.Title(p)
	}
	return out
}

func memberName(name string, settings *plugin.Settings) string {
	return sdk.LowerTitle(dataClassName(name, settings))
}

func BuildDataClasses(conf Config, req *plugin.GenerateRequest) []Struct {
	var structs []Struct
	for _, schema := range req.Catalog.Schemas {
		if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
			continue
		}
		for _, table := range schema.Tables {
			var tableName string
			if schema.Name == req.Catalog.DefaultSchema {
				tableName = table.Rel.Name
			} else {
				tableName = schema.Name + "_" + table.Rel.Name
			}
			structName := dataClassName(tableName, req.Settings)
			if !conf.EmitExactTableNames {
				structName = inflection.Singular(inflection.SingularParams{
					Name:       structName,
					Exclusions: conf.InflectionExcludeTableNames,
				})
			}
			s := Struct{
				Table:   plugin.Identifier{Schema: schema.Name, Name: table.Rel.Name},
				Name:    structName,
				Comment: table.Comment,
			}
			for _, column := range table.Columns {
				s.Fields = append(s.Fields, Field{
					Name:    memberName(column.Name, req.Settings),
					Type:    makeType(req, column),
					Comment: column.Comment,
				})
			}
			structs = append(structs, s)
		}
	}
	if len(structs) > 0 {
		sort.Slice(structs, func(i, j int) bool { return structs[i].Name < structs[j].Name })
	}
	return structs
}

// javaType describes a single column/param/return-value type. Name is the
// canonical JDBC suffix (e.g. "Int" for getInt/setInt) for the handful of
// primitive-backed types; for everything else (String, UUID, time types,
// enums, BigDecimal, Object) Name is just the Java type name.
type javaType struct {
	Name     string
	IsEnum   bool
	IsArray  bool
	IsNull   bool
	DataType string
	Engine   string
}

// boxed returns the reference-type (boxed) rendering of the type, used for
// generic type arguments (List<T>) and nullable fields.
func (t javaType) boxed() string {
	switch t.Name {
	case "Int":
		return "Integer"
	case "Long":
		return "Long"
	case "Short":
		return "Short"
	case "Float":
		return "Float"
	case "Double":
		return "Double"
	case "Boolean":
		return "Boolean"
	default:
		return t.Name
	}
}

// primitive returns the primitive rendering of the type and true, or ("",
// false) if the type has no primitive form.
func (t javaType) primitive() (string, bool) {
	switch t.Name {
	case "Int":
		return "int", true
	case "Long":
		return "long", true
	case "Short":
		return "short", true
	case "Float":
		return "float", true
	case "Double":
		return "double", true
	case "Boolean":
		return "boolean", true
	default:
		return "", false
	}
}

// declare renders the type for a declaration site (field, param, return
// type) given whether that site is nullable. Arrays are always rendered as
// List<Boxed> and, matching prior behavior, are never annotated nullable.
func (t javaType) declare(nullable bool) string {
	if t.IsArray {
		return fmt.Sprintf("List<%s>", t.boxed())
	}
	if nullable {
		return nullableRef(t.boxed())
	}
	if p, ok := t.primitive(); ok {
		return p
	}
	return t.boxed()
}

func (t javaType) String() string {
	return t.declare(t.IsNull)
}

func (t javaType) IsTime() bool {
	return t.Name == "LocalDate" || t.Name == "LocalDateTime" || t.Name == "LocalTime" || t.Name == "OffsetDateTime"
}

func (t javaType) IsInstant() bool {
	return t.Name == "Instant"
}

func (t javaType) IsUUID() bool {
	return t.Name == "UUID"
}

func (t javaType) IsBigDecimal() bool {
	return t.Name == "BigDecimal"
}

func makeType(req *plugin.GenerateRequest, col *plugin.Column) javaType {
	typ, isEnum := javaInnerType(req, col)
	return javaType{
		Name:     typ,
		IsEnum:   isEnum,
		IsArray:  col.IsArray,
		IsNull:   !col.NotNull,
		DataType: sdk.DataType(col.Type),
		Engine:   req.Settings.Engine,
	}
}

func javaInnerType(req *plugin.GenerateRequest, col *plugin.Column) (string, bool) {
	// TODO: Extend the engine interface to handle types
	switch req.Settings.Engine {
	case "mysql":
		return mysqlType(req, col)
	case "postgresql":
		return postgresType(req, col)
	default:
		return "Object", false
	}
}

type goColumn struct {
	id int
	*plugin.Column
}

func javaColumnsToStruct(req *plugin.GenerateRequest, name string, columns []goColumn, namer func(*plugin.Column, int) string) *Struct {
	gs := Struct{
		Name: name,
	}
	idSeen := map[int]Field{}
	nameSeen := map[string]int{}
	for _, c := range columns {
		if _, ok := idSeen[c.id]; ok {
			continue
		}
		fieldName := memberName(namer(c.Column, c.id), req.Settings)
		if v := nameSeen[c.Name]; v > 0 {
			fieldName = fmt.Sprintf("%s_%d", fieldName, v+1)
		}
		field := Field{
			ID:   c.id,
			Name: fieldName,
			Type: makeType(req, c.Column),
		}
		gs.Fields = append(gs.Fields, field)
		nameSeen[c.Name]++
		idSeen[c.id] = field
	}
	return &gs
}

func javaArgName(name string) string {
	out := ""
	for i, p := range strings.Split(name, "_") {
		if i == 0 {
			out += strings.ToLower(p)
		} else {
			out += strings.Title(p)
		}
	}
	return out
}

func javaParamName(c *plugin.Column, number int) string {
	if c.Name != "" {
		return javaArgName(c.Name)
	}
	return fmt.Sprintf("dollar_%d", number)
}

func javaColumnName(c *plugin.Column, pos int) string {
	if c.Name != "" {
		return c.Name
	}
	return fmt.Sprintf("column_%d", pos+1)
}

var postgresPlaceholderRegexp = regexp.MustCompile(`\B\$\d+\b`)

// HACK: jdbc doesn't support numbered parameters, so we need to transform them to question marks...
// But there's no access to the SQL parser here, so we just do a dumb regexp replace instead. This won't work if
// the literal strings contain matching values, but good enough for a prototype.
func jdbcSQL(s, engine string) (string, []string) {
	if engine != "postgresql" {
		return s, nil
	}
	var args []string
	q := postgresPlaceholderRegexp.ReplaceAllStringFunc(s, func(placeholder string) string {
		args = append(args, placeholder)
		return "?"
	})
	return q, args
}

func parseInts(s []string) ([]int, error) {
	if len(s) == 0 {
		return nil, nil
	}
	var refs []int
	for _, v := range s {
		i, err := strconv.Atoi(strings.TrimPrefix(v, "$"))
		if err != nil {
			return nil, err
		}
		refs = append(refs, i)
	}
	return refs, nil
}

func BuildQueries(req *plugin.GenerateRequest, structs []Struct) ([]Query, error) {
	qs := make([]Query, 0, len(req.Queries))
	for _, query := range req.Queries {
		if query.Name == "" {
			continue
		}
		if query.Cmd == "" {
			continue
		}
		if query.Cmd == metadata.CmdCopyFrom {
			return nil, errors.New("Support for CopyFrom in Java is not implemented")
		}

		ql, args := jdbcSQL(query.Text, req.Settings.Engine)
		refs, err := parseInts(args)
		if err != nil {
			return nil, fmt.Errorf("Invalid parameter reference: %w", err)
		}
		gq := Query{
			Cmd:          query.Cmd,
			ClassName:    strings.Title(query.Name),
			ConstantName: sdk.LowerTitle(query.Name),
			FieldName:    sdk.LowerTitle(query.Name) + "Stmt",
			MethodName:   sdk.LowerTitle(query.Name),
			SourceName:   query.Filename,
			SQL:          ql,
			Comments:     query.Comments,
		}

		var cols []goColumn
		for _, p := range query.Params {
			cols = append(cols, goColumn{
				id:     int(p.Number),
				Column: p.Column,
			})
		}
		params := javaColumnsToStruct(req, gq.ClassName+"Bindings", cols, javaParamName)
		gq.Arg = Params{
			Struct:  params,
			binding: refs,
		}

		if len(query.Columns) == 1 {
			c := query.Columns[0]
			gq.Ret = QueryValue{
				Name: "results",
				Typ:  makeType(req, c),
			}
		} else if len(query.Columns) > 1 {
			var gs *Struct
			var emit bool

			for _, s := range structs {
				if len(s.Fields) != len(query.Columns) {
					continue
				}
				same := true
				for i, f := range s.Fields {
					c := query.Columns[i]
					sameName := f.Name == memberName(javaColumnName(c, i), req.Settings)
					sameType := f.Type == makeType(req, c)
					sameTable := sdk.SameTableName(c.Table, &s.Table, req.Catalog.DefaultSchema)

					if !sameName || !sameType || !sameTable {
						same = false
					}
				}
				if same {
					gs = &s
					break
				}
			}

			if gs == nil {
				var columns []goColumn
				for i, c := range query.Columns {
					columns = append(columns, goColumn{
						id:     i,
						Column: c,
					})
				}
				gs = javaColumnsToStruct(req, gq.ClassName+"Row", columns, javaColumnName)
				emit = true
			}
			gq.Ret = QueryValue{
				Emit:   emit,
				Name:   "results",
				Struct: gs,
			}
		}

		qs = append(qs, gq)
	}
	sort.Slice(qs, func(i, j int) bool { return qs[i].MethodName < qs[j].MethodName })
	return qs, nil
}

type JavaTmplCtx struct {
	Q           string
	Package     string
	Enums       []Enum
	DataClasses []Struct
	Queries     []Query
	Settings    *plugin.Settings
	SqlcVersion string

	// TODO: Race conditions
	SourceName string

	// Set when rendering a single model/enum file.
	CurrentEnum   *Enum
	CurrentStruct *Struct

	EmitNullableAnnotations bool
}

func Offset(v int) int {
	return v + 1
}

func JavaFormat(s string) string {
	// TODO: do more than just skip multiple blank lines, like maybe run google-java-format
	skipNextSpace := false
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		isSpace := len(strings.TrimSpace(l)) == 0
		if !isSpace || !skipNextSpace {
			lines = append(lines, l)
		}
		skipNextSpace = isSpace
	}
	o := strings.Join(lines, "\n")
	o += "\n"
	return o
}
