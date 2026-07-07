package core

type Config struct {
	Package                     string   `json:"package"`
	EmitExactTableNames         bool     `json:"emit_exact_table_names"`
	InflectionExcludeTableNames []string `json:"inflection_exclude_table_names"`
	EmitJspecifyAnnotations     *bool    `json:"emit_jspecify_annotations"`
}

// JspecifyAnnotationsEnabled reports whether org.jspecify.annotations.Nullable
// (and the package-level @NullMarked) should be emitted. Defaults to true
// when the option is unset.
func (c Config) JspecifyAnnotationsEnabled() bool {
	if c.EmitJspecifyAnnotations == nil {
		return true
	}
	return *c.EmitJspecifyAnnotations
}
