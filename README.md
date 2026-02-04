# alt-sqlc-gen-python

## Fork notes

This is a fork of [sqlc-gen-python](https://github.com/sqlc-dev/sqlc-gen-python) v1.3.0 with the following changes:

- Supports `sqlc.embed()`
- Supports overriding Python types for specific database columns
- Supports SQLAlchemy Session/AsyncSession types
- Enforces timezone-aware datetime types using `pydantic.AwareDatetime`
- Generates modern Python syntax:
   - `Type | None` instead of `Optional[Type]`
   - `list[T]` instead of `List[T]`
   - Adds `_conn` type annotations to Querier classes
   - Imports `Iterator` and `AsyncIterator` from `collections.abc` instead of `typing`
   - Assigns unused results to `_` variable
- Handles fields with names that conflict with Python reserved keywords

## Usage

```yaml
version: "2"
plugins:
  - name: py
    wasm:
      url: https://downloads.sqlc.dev/plugin/sqlc-gen-python_1.3.0.wasm
      sha256: fbedae96b5ecae2380a70fb5b925fd4bff58a6cfb1f3140375d098fbab7b3a3c
sql:
  - schema: "schema.sql"
    queries: "query.sql"
    engine: postgresql
    codegen:
      - out: src/authors
        plugin: py
        options:
          package: authors
          emit_sync_querier: true
          emit_async_querier: true
```

### Emit Pydantic Models instead of `dataclasses`

Option: `emit_pydantic_models`

By default, `sqlc-gen-python` will emit `dataclasses` for the models. If you prefer to use [`pydantic`](https://docs.pydantic.dev/latest/) models, you can enable this option.

with `emit_pydantic_models`

```py
from pydantic import BaseModel

class Author(pydantic.BaseModel):
    id: int
    name: str
```

without `emit_pydantic_models`

```py
import dataclasses

@dataclasses.dataclass()
class Author:
    id: int
    name: str
```

### Use `enum.StrEnum` for Enums

Option: `emit_str_enum`

`enum.StrEnum` was introduce in Python 3.11.

`enum.StrEnum` is a subclass of `str` that is also a subclass of `Enum`. This allows for the use of `Enum` values as strings, compared to strings, or compared to other `enum.StrEnum` types.

This is convenient for type checking and validation, as well as for serialization and deserialization.

By default, `sqlc-gen-python` will emit `(str, enum.Enum)` for the enum classes. If you prefer to use `enum.StrEnum`, you can enable this option.

with `emit_str_enum`

```py
class Status(enum.StrEnum):
    """Venues can be either open or closed"""
    OPEN = "op!en"
    CLOSED = "clo@sed"
```

without `emit_str_enum` (current behavior)

```py
class Status(str, enum.Enum):
    """Venues can be either open or closed"""
    OPEN = "op!en"
    CLOSED = "clo@sed"
```

### Type Overrides

Option: `overrides`

You can override the Python type for specific database columns using the `overrides` configuration.

- `column`: The fully-qualified column name in the format `"table_name.column_name"`
- `py_type`: The Python type to use
- `py_import`: The module to import the type from

```yaml
version: "2"
# ...
sql:
  - schema: "schema.sql"
    queries: "query.sql"
    engine: postgresql
    codegen:
      - out: src/authors
        plugin: py
        options:
          package: authors
          overrides:
            - column: "authors.id"
              py_type: "UUID"
              py_import: "uuid"
```
