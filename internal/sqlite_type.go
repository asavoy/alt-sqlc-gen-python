package python

import (
	"log"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"
)

func sqliteType(req *plugin.GenerateRequest, col *plugin.Column) string {
	return sqliteTypeWithConfig(req, col, Config{})
}

func sqliteTypeWithConfig(req *plugin.GenerateRequest, col *plugin.Column, conf Config) string {
	dt := strings.ToLower(sdk.DataType(col.Type))

	switch dt {
	case "int", "integer", "tinyint", "smallint", "mediumint", "bigint", "unsignedbigint", "int2", "int8":
		return "int"
	case "blob":
		return "memoryview"
	case "real", "double", "double precision", "float":
		return "float"
	case "boolean", "bool":
		return "bool"
	case "date":
		return "datetime.date"
	case "datetime", "timestamp", "timestamptz":
		if conf.EmitPydanticModels {
			return "pydantic.AwareDatetime"
		}
		return "datetime.datetime"
	case "any":
		return "Any"
	}

	switch {
	case strings.HasPrefix(dt, "character"),
		strings.HasPrefix(dt, "varchar"),
		strings.HasPrefix(dt, "varyingcharacter"),
		strings.HasPrefix(dt, "nchar"),
		strings.HasPrefix(dt, "nativecharacter"),
		strings.HasPrefix(dt, "nvarchar"),
		dt == "text",
		dt == "clob",
		dt == "json":
		return "str"
	case strings.HasPrefix(dt, "decimal"), dt == "numeric":
		return "decimal.Decimal"

	default:
		log.Printf("unknown SQLite type: %s\n", dt)
		return "Any"
	}
}
