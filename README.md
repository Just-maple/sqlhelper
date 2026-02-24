# SQLHelper

A lightweight SQL helper library for Go, built on top of [squirrel](https://github.com/Masterminds/squirrel). Provides fluent interfaces for database operations with built-in support for model mapping and pagination.

## Features

- **ModelHelper**: Generic-based model operations with full type safety
- **Fluent Query Builder**: Easy-to-use chain API for building SQL queries
- **Query Interface**: Reusable SQL clause injection (pagination, sorting, filtering, multi-tenant, JOINs)
- **Custom Escaping**: Support custom escape functions for different SQL dialects
- **Table Alias**: Automatic column escaping with table alias
- **Transaction Support**: Works with database connections and transactions

## Comparison

| Feature | SQLHelper | GORM | ent | SQLBoiler |
|---------|-----------|------|-----|-----------|
| **Type Safety** | Generics | Reflection | Code Gen | Code Gen |
| **Learning Curve** | Low | Medium | High | High |
| **Bundle Size** | Minimal (~50KB) | Large (~5MB) | Large | Medium |
| **Dependencies** | squirrel only | Many | Many | Few |
| **Migration** | No | Yes | Yes | No |
| **Associations** | No | Yes | Yes | No |
| **Raw SQL Control** | Full | Limited | Limited | Full |
| **Reflection Overhead** | None | High | None | None |

## When to Use SQLHelper

**Recommended for:**
- **Performance-critical applications**: No reflection overhead, direct SQL execution
- **Existing databases**: Working with legacy schemas without migrations
- **Complex queries**: Full control over SQL with type-safe result mapping
- **Microservices**: Minimal dependencies, small binary size
- **Teams familiar with SQL**: Prefer writing SQL over ORM abstractions

**Not recommended for:**
- Projects needing automatic migrations and schema management
- Applications requiring complex associations/relations
- Teams preferring ORM-style data access patterns

## Installation

```bash
go get github.com/Just-maple/sqlhelper
```

## Quick Start

### Define a Model

```go
type User struct {
    ID    int
    Name  string
    Email string
    Age   int
}

func (u *User) TableName() string {
    return "users"
}

func (u *User) FieldMapping(dst map[string]any) {
    dst["id"] = &u.ID
    dst["name"] = &u.Name
    dst["email"] = &u.Email
    dst["age"] = &u.Age
}
```

### Initialize

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/Just-maple/sqlhelper"
)

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/test")
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })
```

## Model Examples

### Basic Select

```go
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

// Select all columns (inferred from FieldMapping)
sql, _ := userHelper.ModelSelect(nil).ToSql()
// SELECT `age`, `email`, `id`, `name` FROM `users`

// Select specific columns
sql, _ = userHelper.ModelSelect([]string{"id", "name"}).ToSql()
// SELECT `id`, `name` FROM `users`

// Execute query
users, err := userHelper.ModelSelect(nil).List(ctx, db)

// Query single row
user, err := userHelper.ModelSelectWhere("id = ?", 1).One(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id = ?
```

### With Where Clause

```go
// Single condition
users, _, err := userHelper.ModelPaginationWhere(ctx, db, &PageQuery{Page: 1, Limit: 10}, "age > ?", 18)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ? LIMIT 10 OFFSET 0

// Multiple conditions
users, _, err := userHelper.ModelSelectWhere("age > ? AND status = ?", 18, "active").List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ? AND status = ?

// IN clause
users, _, err := userHelper.ModelSelectWhere("id IN (?, ?, ?)", 1, 2, 3).List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id IN (?, ?, ?)
```

### With Alias

```go
// Set alias on ModelHelper
userHelper := sqlhelper.NewModelHelper(func() User { return User{} }).Alias("u")

sql, _ := userHelper.ModelSelect(nil).ToSql()
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`

// Alias with WHERE
user, err := userHelper.ModelSelectWhere("u.id = ?", 1).One(ctx, db)
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?
```

### With Options (OrderBy, Limit, Join)

```go
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

// OrderBy and Limit
users, err := userHelper.ModelSelect(nil).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.OrderBy("created_at DESC").Limit(10)
    }).List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` ORDER BY created_at DESC LIMIT 10

// LEFT JOIN with orders (no alias)
users, err := userHelper.ModelSelect(nil).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.LeftJoin("orders o ON o.user_id = users.id")
    }).List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` LEFT JOIN orders o ON o.user_id = users.id

// Multiple JOINs with alias
userHelperAlias := userHelper.Alias("u")
users, err = userHelperAlias.ModelSelect(nil).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.
            Join("orders o ON o.user_id = u.id").
            LeftJoin("profiles p ON p.user_id = u.id")
    }).List(ctx, db)
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`
// JOIN orders o ON o.user_id = u.id
// LEFT JOIN profiles p ON p.user_id = u.id
```

## Query Interface

The `Query` interface enables reusable SQL clause injection through `WithQueries`. For pagination with count control, implement `PaginationQuery`:

```go
// Query for SQL clause injection (WHERE, SORT, JOIN, etc.)
type Query interface {
    Option(helper Helper) SelectBuilderOption
}

// PaginationQuery extends Query with count control for pagination
type PaginationQuery interface {
    Query
    Countless() bool // true=skip count, false=execute count
}
```

**Key Points:**
- `Option` receives `Helper` context for proper column escaping with alias support
- Use `h.EscapeColumn(key)` inside Option to escape columns with alias prefix
- Multiple `Query` instances can be chained via `WithQueries`
- `Query` is used for SQL clause injection (WHERE, SORT, JOIN)
- `PaginationQuery` extends `Query` with `Countless()` for pagination count control:
  - `return false`: Execute concurrent count query (for UI pagination with total count)
  - `return true`: Skip count query (for internal use, infinite scroll, or when total is not needed)

**Common Use Cases:**
- **Pagination**: LIMIT/OFFSET injection
- **Sorting**: ORDER BY with dynamic field and direction
- **Filtering**: WHERE conditions with parameters
- **Multi-tenant**: Automatic tenant_id filtering
- **JOIN injection**: Dynamic table joins based on query context
- **Soft delete**: Automatic `deleted_at IS NULL` filtering

### Pagination Query

```go
type PageQuery struct {
    Page      int
    Limit     int
    Countless bool // Control count query: true=skip, false=execute
}

func (p *PageQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Limit(uint64(p.Limit)).Offset(uint64((p.Page - 1) * p.Limit))
    }
}
func (p *PageQuery) Countless() bool { return p.Countless }

// Usage - developer controls count behavior
// User-facing pagination with total count
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10, Countless: false})
// SELECT `age`, `email`, `id`, `name` FROM `users` LIMIT 10 OFFSET 0

// Internal use or infinite scroll - skip count query for better performance
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10, Countless: true})
// SELECT `age`, `email`, `id`, `name` FROM `users` LIMIT 10 OFFSET 0
```

### Sort Query

```go
type SortQuery struct {
    Field string
    Desc  bool
}

func (q *SortQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    order := "ASC"
    if q.Desc {
        order = "DESC"
    }
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.OrderByClause(h.EscapeColumn(q.Field) + " " + order)
    }
}
// SortQuery implements Query interface

// Usage with WithQueries
users, err := userHelper.ModelSelect(nil).
    WithQueries(&SortQuery{Field: "created_at", Desc: true}).
    List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` ORDER BY `created_at` DESC
```

### Where Filter Query

```go
type StatusFilterQuery struct {
    Status string
}

func (q *StatusFilterQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Where("status = ?", q.Status)
    }
}
// StatusFilterQuery implements Query interface

// Usage - can be combined with other queries
users, err := userHelper.ModelSelect(nil).
    WithQueries(
        &StatusFilterQuery{Status: "active"},
        &SortQuery{Field: "id", Desc: true},
    ).
    List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE status = ? ORDER BY `id` DESC
```

### Multi-tenant Query

```go
type TenantQuery struct {
    TenantID string
}

func (q *TenantQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        // Use h.EscapeColumn for proper alias-aware escaping
        return b.Where(h.EscapeColumn("tenant_id")+" = ?", q.TenantID)
    }
}
// TenantQuery implements Query interface

// Usage with alias
h := sqlhelper.Helper{}.Alias("u")
users, err := h.Select([]string{"id", "name"}, "users").
    WithQueries(&TenantQuery{TenantID: "tenant_123"}).
    QueryRows(ctx, db)
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE `u`.`tenant_id` = ?
```

### Soft Delete Query

```go
type NotDeletedQuery struct{}

func (q *NotDeletedQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Where(h.EscapeColumn("deleted_at") + " IS NULL")
    }
}
// NotDeletedQuery implements Query interface

// Usage - automatically filters soft-deleted records
users, err := userHelper.ModelSelect(nil).
    WithQueries(&NotDeletedQuery{}).
    List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` WHERE `deleted_at` IS NULL
```

### JOIN Injection Query

```go
type WithOrderQuery struct {
    OrderStatus int
}

func (q *WithOrderQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        // Use h.EscapeColumn for proper alias-aware column escaping
        userIDCol := h.EscapeColumn("id")
        return b.LeftJoin(
            "orders o ON o.user_id = "+userIDCol+" AND o.status = ?",
            q.OrderStatus,
        )
    }
}
// WithOrderQuery implements Query interface

// Usage with alias
h := sqlhelper.Helper{}.Alias("u")
users, err := h.Select([]string{"id", "name"}, "users").
    WithQueries(&WithOrderQuery{OrderStatus: 1}).
    QueryRows(ctx, db)
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` 
// LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ?
```

### Combined: Multiple Query Types with Pagination

```go
// Combine pagination, sorting, filtering, and multi-tenant
userHelper := sqlhelper.NewModelHelper(func() User { return User{} }).Alias("u")

users, total, err := userHelper.ModelSelect(nil).
    WithQueries(
        &TenantQuery{TenantID: "tenant_123"},     // Query: WHERE `u`.`tenant_id` = ?
        &StatusFilterQuery{Status: "active"},     // Query: WHERE status = ?
        &SortQuery{Field: "created_at", Desc: true}, // Query: ORDER BY `u`.`created_at` DESC
    ).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.LeftJoin("orders o ON o.user_id = u.id")
    }).
    Pagination(ctx, db, &PageQuery{Page: 1, Limit: 10, Countless: false})

// Generated SQL:
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`
// WHERE `u`.`tenant_id` = ? AND status = ?
// LEFT JOIN orders o ON o.user_id = u.id
// ORDER BY `u`.`created_at` DESC
// LIMIT 10 OFFSET 0
```

### Insert Operations

```go
// Single insert
result, err := userHelper.ModelInsert([]string{"name", "email"}, &User{
    Name: "John", Email: "john@test.com",
}).Exec(ctx, db)
// INSERT INTO `users` (`name`,`email`) VALUES (?,?)

// Multiple inserts
users := []User{
    {Name: "John", Email: "john@test.com"},
    {Name: "Jane", Email: "jane@test.com"},
}
result, err := userHelper.ModelInserts([]string{"name", "email"}, users).Exec(ctx, db)
// INSERT INTO `users` (`name`,`email`) VALUES (?,?),(?,?)

// ON DUPLICATE KEY UPDATE
result, err := userHelper.ModelInsert([]string{"id", "name", "email"}, &User{
    ID: 1, Name: "John", Email: "john@test.com",
}).OnDuplicateUpdateValues("name", "email").Exec(ctx, db)
// INSERT INTO `users` (`id`,`name`,`email`) VALUES (?,?,?) 
// ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)
```

### Update Operations

```go
// Update with WHERE
result, err := userHelper.ModelUpdate(&User{ID: 1, Name: "John"}, []string{"name"}).
    Where("id = ?", 1).
    Exec(ctx, db)
// UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1

// Update with custom conditions
result, err := userHelper.ModelUpdate(&User{Name: "Updated"}, []string{"name"}).
    Where("age > ?", 18).
    Exec(ctx, db)
// UPDATE `users` SET `name` = ? WHERE age > ? LIMIT 1
```

### Columns Filter

```go
// Get all columns
allCols := userHelper.Columns(nil)
// []string{"age", "email", "id", "name"}

// Filter columns
cols := userHelper.Columns(func(col string) bool {
    return col != "email" // exclude email
})
// []string{"age", "id", "name"}

// Use filtered columns in select
users, err := userHelper.ModelSelect(cols).List(ctx, db)
// SELECT `age`, `id`, `name` FROM `users`
```

### MappingModel (Without Model Interface)

```go
// Use MappingModel if you don't want to implement Model interface
type User struct {
    ID    int
    Name  string
    Email string
}

userHelper := sqlhelper.NewMappingModelHelper(func(u *User) (string, map[string]any) {
    return "users", map[string]any{
        "id":    &u.ID,
        "name":  &u.Name,
        "email": &u.Email,
    }
})

// Use same API as ModelHelper
users, err := userHelper.ModelSelect(nil).List(ctx, db)
// SELECT `id`, `name`, `email` FROM `users`
```

### Transaction Support

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// Use transaction as Conn
_, err = userHelper.ModelInsert([]string{"name"}, &User{Name: "John"}).Exec(ctx, tx)
// INSERT INTO `users` (`name`) VALUES (?)

_, err = userHelper.ModelUpdate(&User{ID: 1, Name: "John"}, []string{"name"}).
    Where("id = ?", 1).Exec(ctx, tx)
// UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1

return tx.Commit()
```

## Helper Examples (Raw SQL)

```go
h := sqlhelper.Helper{}

// Basic select
sql, _ := h.Select([]string{"id", "name"}, "users").ToSql()
// SELECT `id`, `name` FROM `users`

// With WHERE
sql, _ = h.Select([]string{"id", "name"}, "users").
    Where("age > ?", 18).
    ToSql()
// SELECT `id`, `name` FROM `users` WHERE age > ?

// With alias
h = sqlhelper.Helper{}.Alias("u")
sql, _ = h.Select([]string{"id", "name"}, "users").ToSql()
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`

// Custom escape (PostgreSQL)
h = sqlhelper.Helper{}.WithEscapeFunc(func(key string, table bool) string {
    return fmt.Sprintf("\"%s\"", key)
})
sql, _ = h.Select([]string{"id", "name"}, "users").ToSql()
// SELECT "id", "name" FROM "users"
```

## API Reference

### Helper

| Method | Description |
|--------|-------------|
| `Select(columns, table)` | Create SELECT query |
| `Insert(table, columns, values...)` | Create INSERT query |
| `Update(table, values)` | Create UPDATE query |
| `Alias(alias)` | Set table alias |
| `EscapeColumn(column)` | Escape column name |
| `EscapeTable(table)` | Escape table name |
| `WithEscapeFunc(fn)` | Set custom escape function |

### SelectExecutor

| Method | Description |
|--------|-------------|
| `Where(pred, args...)` | Add WHERE clause |
| `WithOptions(opts...)` | Apply builder options |
| `WithQueries(queries...)` | Apply Query options |
| `ToSql()` | Get SQL and args |
| `QueryRow(ctx, conn)` | Execute and return single row |
| `QueryRows(ctx, conn)` | Execute and return multiple rows |
| `Count(ctx, conn)` | Get total count |
| `Pagination(ctx, conn, query, alloc)` | Paginated query |

### ModelHelper

| Method | Description |
|--------|-------------|
| `NewModelHelper(alloc)` | Create new ModelHelper |
| `ModelSelect(columns)` | Create SELECT query |
| `ModelSelectWhere(pred, args...)` | Create SELECT with WHERE |
| `ModelPagination(ctx, conn, query)` | Paginated query |
| `ModelPaginationWhere(ctx, conn, query, pred, args...)` | Paginated query with WHERE |
| `ModelInsert(columns, models...)` | Insert records |
| `ModelInserts(columns, models)` | Insert slice of records |
| `ModelUpdate(model, columns)` | Update record |
| `Alias(alias)` | Set table alias |
| `Columns(filter)` | Get column names with optional filter |

### Query Interface

```go
// Query for SQL clause injection
type Query interface {
    Option(helper Helper) SelectBuilderOption
}

// PaginationQuery extends Query with count control
type PaginationQuery interface {
    Query
    Countless() bool
}
```

### Model Interface

```go
type Model interface {
    TableName() string
    FieldMapping(dst map[string]any)
}
```
