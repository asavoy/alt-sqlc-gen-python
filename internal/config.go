package python

type Override struct {
	Column   string `json:"column"`
	PyImport string `json:"py_import"`
	PyType   string `json:"py_type"`
}

type Config struct {
	EmitExactTableNames         bool       `json:"emit_exact_table_names"`
	EmitSyncQuerier             bool       `json:"emit_sync_querier"`
	EmitAsyncQuerier            bool       `json:"emit_async_querier"`
	Package                     string     `json:"package"`
	Out                         string     `json:"out"`
	EmitPydanticModels          bool       `json:"emit_pydantic_models"`
	EmitStrEnum                 bool       `json:"emit_str_enum"`
	QueryParameterLimit         *int32     `json:"query_parameter_limit"`
	InflectionExcludeTableNames []string   `json:"inflection_exclude_table_names"`
	Overrides                   []Override `json:"overrides"`
}
