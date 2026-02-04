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
      url: https://github.com/asavoy/alt-sqlc-gen-python/releases/download/v0.1.0/alt-sqlc-gen-python.wasm
      sha256: TODO
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

### Sync and Async Queriers

Options: `emit_sync_querier`, `emit_async_querier`

These options generate `Querier` and/or `AsyncQuerier` classes that wrap a SQLAlchemy connection and expose a method for each SQL query.

- `Querier` accepts `sqlalchemy.engine.Connection | sqlalchemy.orm.Session`
- `AsyncQuerier` accepts `sqlalchemy.ext.asyncio.AsyncConnection | sqlalchemy.ext.asyncio.AsyncSession`

The query command (`:one`, `:many`, `:exec`, `:execrows`, `:execresult`) determines the method signature:

| Command | Sync return type | Async return type |
|---|---|---|
| `:one` | `Model \| None` | `Model \| None` |
| `:many` | `Iterator[Model]` | `AsyncIterator[Model]` |
| `:exec` | `None` | `None` |
| `:execrows` | `int` | `int` |
| `:execresult` | `sqlalchemy.engine.Result` | `sqlalchemy.engine.Result` |

Example generated code with both options enabled:

```py
class Querier[T: sqlalchemy.engine.Connection | sqlalchemy.orm.Session]:
    _conn: T

    def __init__(self, conn: T):
        self._conn = conn

    def get_user(self, *, id: int) -> models.User | None:
        row = self._conn.execute(sqlalchemy.text(GET_USER), {"p1": id}).first()
        if row is None:
            return None
        return models.User(
            id=cast(int, row[0]),
            name=cast(str, row[1]),
        )

    def list_users(self) -> Iterator[models.User]:
        result = self._conn.execute(sqlalchemy.text(LIST_USERS))
        for row in result:
            yield models.User(
                id=cast(int, row[0]),
                name=cast(str, row[1]),
            )


class AsyncQuerier[T: sqlalchemy.ext.asyncio.AsyncConnection | sqlalchemy.ext.asyncio.AsyncSession]:
    _conn: T

    def __init__(self, conn: T):
        self._conn = conn

    async def get_user(self, *, id: int) -> models.User | None:
        row = (await self._conn.execute(sqlalchemy.text(GET_USER), {"p1": id})).first()
        if row is None:
            return None
        return models.User(
            id=cast(int, row[0]),
            name=cast(str, row[1]),
        )

    async def list_users(self) -> AsyncIterator[models.User]:
        result = await self._conn.stream(sqlalchemy.text(LIST_USERS))
        async for row in result:
            yield models.User(
                id=cast(int, row[0]),
                name=cast(str, row[1]),
            )
```

### Embedded Structs with `sqlc.embed()`

When a query joins multiple tables, you can use `sqlc.embed()` to nest the full model structs in the result rather than flattening all columns.

Given this schema:

```sql
CREATE TABLE authors (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    bio  TEXT
);

CREATE TABLE books (
    id        BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES authors(id),
    title     TEXT NOT NULL,
    isbn      TEXT
);
```

And this query:

```sql
-- name: GetBookWithAuthor :one
SELECT
    sqlc.embed(books),
    sqlc.embed(authors)
FROM books
JOIN authors ON books.author_id = authors.id
WHERE books.id = $1;
```

The plugin generates a row type with nested model fields:

```py
class GetBookWithAuthorRow(pydantic.BaseModel):
    books: models.Book
    authors: models.Author
```

And the querier method constructs each embedded struct from the corresponding columns:

```py
def get_book_with_author(self, *, id: int) -> GetBookWithAuthorRow | None:
    row = self._conn.execute(sqlalchemy.text(GET_BOOK_WITH_AUTHOR), {"p1": id}).first()
    if row is None:
        return None
    return GetBookWithAuthorRow(
        books=models.Book(
            id=cast(int, row[0]),
            author_id=cast(int, row[1]),
            title=cast(str, row[2]),
            isbn=cast(str | None, row[3]),
        ),
        authors=models.Author(
            id=cast(int, row[4]),
            name=cast(str, row[5]),
            bio=cast(str | None, row[6]),
        ),
    )
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
