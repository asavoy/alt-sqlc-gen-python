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
- Supports `:batchexec` for efficient bulk inserts
- Generates `typing.Protocol` classes for querier testability
- Generates typed error wrapping for database constraint violations

## Usage

```yaml
version: "2"
plugins:
  - name: py
    wasm:
      url: https://github.com/asavoy/alt-sqlc-gen-python/releases/download/v0.2.0/alt-sqlc-gen-python.wasm
      sha256: ae1210f5b91e428b596e6061f4d8b4717ed22c7f5f456c4d8a5f19d59a73558e
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

The query command (`:one`, `:many`, `:exec`, `:execrows`, `:execresult`, `:batchexec`) determines the method signature:

| Command | Sync return type | Async return type |
|---|---|---|
| `:one` | `Model \| None` | `Model \| None` |
| `:many` | `Iterator[Model]` | `AsyncIterator[Model]` |
| `:exec` | `None` | `None` |
| `:execrows` | `int` | `int` |
| `:execresult` | `sqlalchemy.engine.Result` | `sqlalchemy.engine.Result` |
| `:batchexec` | `None` | `None` |

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

### Batch Exec

Command: `:batchexec`

Use `:batchexec` to efficiently insert or modify multiple rows in a single call. The generated method accepts a list of parameter structs and uses SQLAlchemy's `executemany` under the hood.

```sql
-- name: CreateAuthors :batchexec
INSERT INTO authors (name, bio) VALUES ($1, $2);
```

Generated code:

```py
@dataclasses.dataclass()
class CreateAuthorsParams:
    name: str
    bio: str | None


class Querier[T: sqlalchemy.engine.Connection | sqlalchemy.orm.Session]:
    # ...

    def create_authors(self, *, args: list[CreateAuthorsParams]) -> None:
        self._conn.execute(sqlalchemy.text(CREATE_AUTHORS), [{"p1": a.name, "p2": a.bio} for a in args])
```

A Params struct is always generated for `:batchexec` regardless of the `query_parameter_limit` setting.

### Querier Protocols for Testability

Option: `emit_querier_protocol`

Generates `QuerierProtocol` and `AsyncQuerierProtocol` classes using `typing.Protocol`. These mirror every method signature on the concrete querier classes, allowing application code to depend on the protocol and tests to substitute a simple fake.

```yaml
options:
  emit_sync_querier: true
  emit_async_querier: true
  emit_querier_protocol: true
```

Generated code:

```py
class QuerierProtocol(typing.Protocol):
    def get_author(self, *, id: int) -> models.Author | None: ...
    def list_authors(self) -> Iterator[models.Author]: ...
    def create_author(self, *, name: str, bio: str | None) -> None: ...
```

Usage in application code:

```py
# Application code depends on the protocol
def get_author_bio(querier: QuerierProtocol, author_id: int) -> str:
    author = querier.get_author(id=author_id)
    return author.bio if author else "Unknown"

# Tests use a simple fake — fully type-checked
class FakeQuerier:
    def get_author(self, *, id):
        return models.Author(id=1, name="Test", bio="A bio")
    # ... other methods ...
```

Protocols are only emitted when the corresponding `emit_sync_querier` / `emit_async_querier` is also enabled.

### Typed Error Wrapping

Option: `emit_query_errors`

Generates an `errors.py` module with typed exception classes for PostgreSQL constraint violations, and wraps write-query methods (INSERT, UPDATE, DELETE) with try/except to raise these typed errors instead of raw `sqlalchemy.exc.IntegrityError`.

```yaml
options:
  emit_query_errors: true
```

Generated `errors.py` contains:

**Integrity constraint errors** (from `IntegrityError`):

| Exception | PostgreSQL SQLSTATE |
|---|---|
| `UniqueViolationError` | 23505 |
| `ForeignKeyViolationError` | 23503 |
| `CheckViolationError` | 23514 |
| `NotNullViolationError` | 23502 |
| `ExclusionViolationError` | 23P01 |

**Operational errors** (from `OperationalError`):

| Exception | PostgreSQL SQLSTATE |
|---|---|
| `StatementTimeoutError` | 57014 |
| `DeadlockError` | 40P01 |
| `SerializationError` | 40001 |

Unrecognized errors in either category fall back to the base `QueryError`. All exceptions include a `query_name` attribute for identifying which query failed, and a `cause` attribute containing the original SQLAlchemy exception.

All query methods are wrapped with both handlers:

```py
def create_author(self, *, name: str, bio: str | None) -> models.Author | None:
    try:
        row = self._conn.execute(sqlalchemy.text(CREATE_AUTHOR), {"p1": name, "p2": bio}).first()
    except sqlalchemy.exc.IntegrityError as e:
        raise errors._wrap_integrity_error(e, "create_author") from e
    except sqlalchemy.exc.OperationalError as e:
        raise errors._wrap_operational_error(e, "create_author") from e
    if row is None:
        return None
    return models.Author(...)
```

This covers constraint violations on writes and timeouts/deadlocks/serialization failures on any query. The error code extraction uses a portable pattern that works across psycopg2, psycopg3, and asyncpg drivers.

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

## Release process

1. Build the wasm plugin:

   ```sh
   make bin/alt-sqlc-gen-python.wasm
   ```

2. Record the artifact checksum:

   ```sh
   shasum -a 256 bin/alt-sqlc-gen-python.wasm
   ```

3. Run the test suite:

   ```sh
   go test ./...
   ```

4. Update the README usage example with the new release tag and checksum, then commit the README change.

5. Tag the release commit:

   ```sh
   git tag vX.Y.Z
   ```

6. Push the commit and tag:

   ```sh
   git push origin main
   git push origin vX.Y.Z
   ```

7. Create the GitHub release and upload `bin/alt-sqlc-gen-python.wasm`.

8. Verify the published release URL and checksum from the README.
