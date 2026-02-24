# SQLHelper

A lightweight SQL helper library for Go, built on top of [squirrel](https://github.com/Masterminds/squirrel). Provides fluent interfaces for database operations with built-in support for model mapping and pagination.

## Features

- **ModelHelper**: Generic-based model operations with full type safety
- **Fluent Query Builder**: Easy-to-use chain API for building SQL queries
- **Pagination Support**: Built-in pagination for large datasets
- **Custom Escaping**: Support custom escape functions for different SQL dialects
- **Transaction Support**: Works with database connections and transactions
- **MappingModel**: Flexible model definition without interface implementation

## Comparison with Other Frameworks

| Feature | SQLHelper | GORM | ent | SQLBoiler | XORM |
|---------|-----------|------|-----|-----------|------|
| **Type Safety** | Generics | Reflection | Code Gen | Code Gen | Reflection |
| **Learning Curve** | Low | Medium | High | High | Low |
| **Migration/Auto Schema** | No | Yes | Yes | Yes | Yes |
| **Associations** | No | Yes | Yes | Yes | Yes |
| **Raw SQL Support** | Excellent | Limited | Limited | Good | Good |
| **Bundle Size** | Minimal | Large | Large | Large | Medium |
| **Dependencies** | squirrel only | Many | Many | Few | Few |

## Advantages

- **Lightweight**: Only depends on squirrel, minimal dependencies
- **Type Safe**: Full generics support for compile-time type checking
- **Flexible**: Work with raw SQL when needed, no ORM lock-in
- **Simple**: Minimal abstraction over SQL, easy to understand and debug
- **Fast**: No reflection overhead, direct SQL execution

## When to Use SQLHelper

- **Performance-critical applications**: When you need direct SQL control with type safety
- **Existing database**: Working with legacy databases without migrations
- **Simple CRUD**: When you don't need complex associations or migrations
- **Hybrid approach**: Use raw SQL for complex queries with model mapping for results
- **Learning/Prototyping**: Quick to start, minimal boilerplate

## When to Use Other Frameworks

- **GORM/ent**: When you need automatic migrations and complex associations
- **SQLBoiler**: When you want maximum performance with code generation
- **Traditional ORMs**: When team is familiar with Django/Rails-style ORM patterns

## Installation

```bash
go get github.com/Just-maple/sqlhelper
```

## Quick Start

### Define a Model

Implement the `Model` interface on your struct:

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

### Initialize ModelHelper

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/Just-maple/sqlhelper"
)

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/test")

// Create ModelHelper - type is automatically inferred
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })
```

## ModelHelper Usage (Recommended)

### Query Operations

```go
// Pagination query - returns models and total count
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10})

// Query with WHERE clause
users, total, err := userHelper.ModelPaginationWhere(ctx, db, &PageQuery{Page: 1, Limit: 10}, "age > ?", 18)

// Select specific columns
exec := userHelper.ModelSelect([]string{"id", "name", "email"})
users, err := exec.List(ctx, db)

// Select single row
user, err := userHelper.ModelSelectWhere("id = ?", 1).One(ctx, db)

// DISTINCT pagination on a column
emails, total, err := userHelper.ModelDistinctPagination(ctx, db, &PageQuery{Page: 1, Limit: 10}, "email")
```

### Insert Operations

```go
// Insert single model
result, err := userHelper.ModelInsert(nil, user).Exec(ctx, db)

// Insert multiple models
result, err := userHelper.ModelInsert(nil, user1, user2, user3).Exec(ctx, db)

// Insert with ON DUPLICATE KEY UPDATE
result, err := userHelper.ModelInsert([]string{"id", "name", "email"}, user).
    OnDuplicateUpdateValues("name", "email").
    Exec(ctx, db)
```

### Update Operations

```go
// Update model by ID (automatically adds WHERE id = ?)
result, err := userHelper.ModelUpdate(user, nil).Exec(ctx, db)

// Update with custom WHERE
result, err := sqlhelper.Helper{}.Update("users", map[string]any{"name": "John"}, sqlhelper.Helper{}.Update("users", map[string]any{"name": "John"}).Where("age > ?", 18)).Exec(ctx, db)
```

## Raw SQL Helper Usage

### Basic Select

```go
h := sqlhelper.Helper{}

// Simple select
exec := h.Select([]string{"id", "name", "email"}, "users")
rows, err := exec.QueryRows(ctx, db)

// Select with WHERE
exec = h.Select([]string{"id", "name"}, "users").Where("age > ?", 18)

// Select distinct
exec = h.SelectDistinct("email", "users")
```

### Insert/Update/Delete

```go
// Insert
result, err := h.Insert("users", []string{"name", "email"}, []any{"John", "john@example.com"}).Exec(ctx, db)

// Update
result, err := h.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).Exec(ctx, db)
```

### Custom Escape Function

```go
// Use double quotes instead of backticks (e.g., PostgreSQL)
h := sqlhelper.Helper{}
h = h.WithEscapeFunc(func(key string, table bool) string {
    return fmt.Sprintf("\"%s\"", key)
})
```

## Table Alias

```go
h := sqlhelper.Helper{}.Alias("u")
// SELECT u.id, u.name FROM users u
exec := h.Select([]string{"id", "name"}, "users")
```

## Pagination Query

SQLHelper uses the `PaginationQuery` interface for pagination:

```go
type PageQuery struct {
    Page  int
    Limit int
}

func (p *PageQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(builder squirrel.SelectBuilder) squirrel.SelectBuilder {
        return builder.Limit(uint64(p.Limit)).Offset(uint64((p.Page - 1) * p.Limit))
    }
}

func (p *PageQuery) Countless() bool {
    return false // true to skip count query
}
```

## MappingModel (Alternative to Model Interface)

If you don't want to implement the Model interface on your struct, use MappingModel:

```go
type User struct {
    ID    int
    Name  string
    Email string
    Age   int
}

userHelper := sqlhelper.NewMappingModelHelper[User](func(u *User) (string, map[string]any) {
    return "users", map[string]any{
        "id":    &u.ID,
        "name":  &u.Name,
        "email": &u.Email,
        "age":   &u.Age,
    }
})
```

## API Reference

### Helper

- `Helper{}.EscapeColumn(column)` - Escape a column name
- `Helper{}.EscapeTable(table)` - Escape a table name
- `Helper{}.EscapeColumns(columns)` - Escape multiple column names
- `Helper{}.Alias(alias)` - Set table alias
- `Helper{}.WithEscapeFunc(fn)` - Set custom escape function
- `Helper{}.Select(columns, table, opts...)` - Create SELECT query
- `Helper{}.SelectDistinct(column, table)` - Create SELECT DISTINCT query
- `Helper{}.Insert(table, columns, values...)` - Create INSERT query
- `Helper{}.Update(table, values, opts...)` - Create UPDATE query

### ModelHelper

- `NewModelHelper(alloc func() T)` - Create new ModelHelper
- `ModelPagination(ctx, conn, query)` - Paginated query
- `ModelPaginationWhere(ctx, conn, query, pred, args...)` - Paginated query with WHERE
- `ModelSelect(columns, opts...)` - Create SELECT query
- `ModelSelectWhere(pred, args...)` - Create SELECT with WHERE
- `ModelInsert(columns, models...)` - Insert records
- `ModelInserts(columns, models)` - Insert slice of records
- `ModelUpdate(model, columns)` - Update record by ID
- `ModelDistinctPagination(ctx, conn, query, column)` - DISTINCT pagination
- `ModelHelper{}.Alias(alias)` - Set table alias for query
- `ModelHelper{}.Columns(filter)` - Get column names with optional filter
- `ModelHelper{}.Convert(mapper)` - Convert to MappingModel

### SelectExecutor

- `One(ctx, conn)` - Get single model
- `List(ctx, conn)` - Get all models
- `Where(pred, args...)` - Add WHERE clause
- `WithOptions(opts...)` - Apply builder options
- `ToSql()` - Get SQL and args
- `Count(ctx, conn)` - Get total count
- `Pagination(ctx, conn, query, alloc)` - Paginated query
- `PaginationModels(ctx, conn, query)` - Paginated models
- `PaginationStrings(ctx, conn, query)` - Paginated single column
- `PaginationMaps(ctx, conn, query)` - Paginated map results
- `QueryRowScan(ctx, conn, alloc)` - Custom scan for single row
- `QueryRowsScans(ctx, conn, alloc)` - Custom scan for multiple rows
- `QueryRowScanModel(ctx, conn, alloc)` - Scan to Model
- `QueryRowsScansModels(ctx, conn, alloc)` - Scan multiple to Models
- `QueryStrings(ctx, conn)` - Get string values
- `QueryTotals(ctx, conn, alloc, total)` - Query with total count

### InsertExecutor

- `Exec(ctx, conn)` - Execute insert
- `ExecLastInsertId(ctx, conn)` - Execute and return last insert ID
- `ToSql()` - Get SQL and args
- `WithOptions(opts...)` - Apply builder options
- `OnDuplicateUpdateValues(columns...)` - Add ON DUPLICATE KEY UPDATE

### UpdateExecutor

- `Exec(ctx, conn)` - Execute update
- `ExecRowsAffected(ctx, conn)` - Execute and return affected rows
- `ToSql()` - Get SQL and args
- `Where(pred, args...)` - Add WHERE clause
- `Limit(limit)` - Set row limit
- `WithOptions(opts...)` - Apply builder options

### Model Interface

```go
type Model interface {
    TableName() string
    FieldMapping(dst map[string]any)
}
```

### PaginationQuery Interface

```go
type PaginationQuery interface {
    Option(helper Helper) SelectBuilderOption
    Countless() bool
}
```

### Custom Escape Function

```go
type EscapeFunc func(key string, table bool) string
```

- `key`: The table or column name to escape
- `table`: true if escaping a table name, false for column name
- Returns the escaped string
