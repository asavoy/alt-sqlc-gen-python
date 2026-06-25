package python

import (
	"fmt"
	"sort"
	"strings"
)

type importSpec struct {
	Module string
	Name   string
	Alias  string
}

func (i importSpec) String() string {
	if i.Alias != "" {
		if i.Name == "" {
			return fmt.Sprintf("import %s as %s", i.Module, i.Alias)
		}
		return fmt.Sprintf("from %s import %s as %s", i.Module, i.Name, i.Alias)
	}
	if i.Name == "" {
		return "import " + i.Module
	}
	return fmt.Sprintf("from %s import %s", i.Module, i.Name)
}

type importer struct {
	Models  []Struct
	Queries []Query
	Enums   []Enum
	C       Config
}

func structUses(name string, s Struct) bool {
	for _, f := range s.Fields {
		fullName := f.Type.Name
		if f.Type.Module != "" {
			fullName = f.Type.Module + "." + f.Type.Name
		}
		if fullName == name {
			return true
		}
	}
	return false
}

func structTypes(s Struct, visit func(pyType)) {
	for _, f := range s.Fields {
		visit(f.Type)
	}
}

func queryValueUses(name string, qv QueryValue) bool {
	if !qv.isEmpty() {
		if qv.IsStruct() {
			if structUses(name, *qv.Struct) {
				return true
			}
		} else {
			// Build full type name from Module.Name
			fullName := qv.Typ.Name
			if qv.Typ.Module != "" {
				fullName = qv.Typ.Module + "." + qv.Typ.Name
			}
			if fullName == name {
				return true
			}
		}
	}
	return false
}

func queryValueTypes(qv QueryValue, visit func(pyType)) {
	if qv.isEmpty() {
		return
	}
	if qv.IsStruct() {
		structTypes(*qv.Struct, visit)
		return
	}
	visit(qv.Typ)
}

func addTypeImport(std, pkg map[string]importSpec, typ pyType) {
	if typ.Module == "" || typ.Module == "models" || typ.Module == "typing" {
		return
	}

	if _, exists := std[typ.Module]; exists {
		return
	}
	pkg[typ.Module] = importSpec{Module: typ.Module}
}

func (i *importer) Imports(fileName string) []string {
	if fileName == "models.py" {
		return i.modelImports()
	}
	return i.queryImports(fileName)
}

func (i *importer) modelImportSpecs() (map[string]importSpec, map[string]importSpec) {
	modelUses := func(name string) bool {
		for _, model := range i.Models {
			if structUses(name, model) {
				return true
			}
		}
		return false
	}

	std := stdImports(modelUses)
	if i.C.EmitPydanticModels {
		std["pydantic"] = importSpec{Module: "pydantic"}
	} else {
		std["dataclasses"] = importSpec{Module: "dataclasses"}
	}
	if len(i.Enums) > 0 {
		std["enum"] = importSpec{Module: "enum"}
	}

	pkg := make(map[string]importSpec)

	for _, model := range i.Models {
		structTypes(model, func(typ pyType) { addTypeImport(std, pkg, typ) })
	}

	return std, pkg
}

func (i *importer) modelImports() []string {
	std, pkg := i.modelImportSpecs()
	importLines := []string{
		buildImportBlock(std),
		"",
		buildImportBlock(pkg),
	}
	return importLines
}

func (i *importer) queryImportSpecs(fileName string) (map[string]importSpec, map[string]importSpec) {
	queryUses := func(name string) bool {
		for _, q := range i.Queries {
			if q.SourceName != fileName {
				continue
			}
			if queryValueUses(name, q.Ret) {
				return true
			}
			for _, arg := range q.Args {
				if queryValueUses(name, arg) {
					return true
				}
			}
		}
		return false
	}

	std := stdImports(queryUses)

	pkg := make(map[string]importSpec)
	pkg["sqlalchemy"] = importSpec{Module: "sqlalchemy"}
	if i.C.EmitSyncQuerier {
		pkg["sqlalchemy.orm"] = importSpec{Module: "sqlalchemy.orm"}
	}
	if i.C.EmitAsyncQuerier {
		pkg["sqlalchemy.ext.asyncio"] = importSpec{Module: "sqlalchemy.ext.asyncio"}
	}
	if i.C.EmitQueryErrors {
		pkg["sqlalchemy.exc"] = importSpec{Module: "sqlalchemy.exc"}
	}

	queryValueModelImports := func(qv QueryValue) {
		if qv.IsStruct() && qv.EmitStruct() {
			if i.C.EmitPydanticModels {
				std["pydantic"] = importSpec{Module: "pydantic"}
			} else {
				std["dataclasses"] = importSpec{Module: "dataclasses"}
			}
		}
	}

	// Always import cast from typing for query files
	std["typing.cast"] = importSpec{Module: "typing", Name: "cast"}

	// Add typing module import when protocols are enabled
	if i.C.EmitQuerierProtocol {
		std["typing"] = importSpec{Module: "typing"}
	}

	for _, q := range i.Queries {
		if q.SourceName != fileName {
			continue
		}
		if q.Cmd == ":many" {
			if i.C.EmitSyncQuerier {
				std["collections.abc.Iterator"] = importSpec{Module: "collections.abc", Name: "Iterator"}
			}
			if i.C.EmitAsyncQuerier {
				std["collections.abc.AsyncIterator"] = importSpec{Module: "collections.abc", Name: "AsyncIterator"}
			}
		}
		queryValueModelImports(q.Ret)
		for _, qv := range q.Args {
			queryValueModelImports(qv)
		}
	}

	for _, q := range i.Queries {
		if q.SourceName != fileName {
			continue
		}

		queryValueTypes(q.Ret, func(typ pyType) { addTypeImport(std, pkg, typ) })

		for _, arg := range q.Args {
			queryValueTypes(arg, func(typ pyType) { addTypeImport(std, pkg, typ) })
		}
	}

	return std, pkg
}

func (i *importer) queryImports(fileName string) []string {
	std, pkg := i.queryImportSpecs(fileName)

	modelImportStr := fmt.Sprintf("from %s import models", i.C.Package)

	importLines := []string{
		buildImportBlock(std),
		"",
		buildImportBlock(pkg),
		"",
		modelImportStr,
	}
	return importLines
}

type importFromSpec struct {
	Module string
	Names  []string
	Alias  string
}

func buildImportBlock2(pkgs map[string]importSpec) []importFromSpec {
	pkgImports := make([]importFromSpec, 0)
	fromImports := make(map[string][]string)
	for _, is := range pkgs {
		if is.Name == "" || is.Alias != "" {
			pkgImports = append(pkgImports, importFromSpec{
				Module: is.Module,
				Names:  []string{is.Name},
				Alias:  is.Alias,
			})
		} else {
			names, ok := fromImports[is.Module]
			if !ok {
				names = make([]string, 0, 1)
			}
			names = append(names, is.Name)
			fromImports[is.Module] = names
		}
	}
	for modName, names := range fromImports {
		sort.Strings(names)
		pkgImports = append(pkgImports, importFromSpec{
			Module: modName,
			Names:  names,
		})
	}
	sort.Slice(pkgImports, func(i, j int) bool {
		return pkgImports[i].Module < pkgImports[j].Module || pkgImports[i].Names[0] < pkgImports[j].Names[0]
	})
	return pkgImports
}

func buildImportBlock(pkgs map[string]importSpec) string {
	pkgImports := make([]importSpec, 0)
	fromImports := make(map[string][]string)
	for _, is := range pkgs {
		if is.Name == "" || is.Alias != "" {
			pkgImports = append(pkgImports, is)
		} else {
			names, ok := fromImports[is.Module]
			if !ok {
				names = make([]string, 0, 1)
			}
			names = append(names, is.Name)
			fromImports[is.Module] = names
		}
	}

	importStrings := make([]string, 0, len(pkgImports)+len(fromImports))
	for _, is := range pkgImports {
		importStrings = append(importStrings, is.String())
	}
	for modName, names := range fromImports {
		sort.Strings(names)
		nameString := strings.Join(names, ", ")
		importStrings = append(importStrings, fmt.Sprintf("from %s import %s", modName, nameString))
	}
	sort.Strings(importStrings)
	return strings.Join(importStrings, "\n")
}

func stdImports(uses func(name string) bool) map[string]importSpec {
	std := make(map[string]importSpec)
	if uses("decimal.Decimal") {
		std["decimal"] = importSpec{Module: "decimal"}
	}
	if uses("datetime.date") || uses("datetime.time") || uses("datetime.datetime") || uses("datetime.timedelta") {
		std["datetime"] = importSpec{Module: "datetime"}
	}
	if uses("uuid.UUID") {
		std["uuid"] = importSpec{Module: "uuid"}
	}
	if uses("typing.Any") {
		std["typing.Any"] = importSpec{Module: "typing", Name: "Any"}
	}
	return std
}
