# SQLHelper

一个轻量级的 Go SQL 辅助库，基于 [squirrel](https://github.com/Masterminds/squirrel) 构建。提供流畅的数据库操作接口，支持模型映射和分页。

## 特性

- **ModelHelper**: 基于泛型的模型操作，完全类型安全
- **流畅查询构建器**: 易于使用的链式 API 构建 SQL 查询
- **Query 接口**: 可复用 SQL 子句注入（分页、排序、过滤、多租户、连表）
- **自定义转义**: 支持自定义转义函数，适用于不同 SQL 方言
- **表别名**: 使用别名自动转义列名
- **事务支持**: 支持数据库连接和事务

## 对比

| 特性 | SQLHelper | GORM | ent | SQLBoiler |
|------|-----------|------|-----|-----------|
| **类型安全** | 泛型 | 反射 | 代码生成 | 代码生成 |
| **学习曲线** | 低 | 中 | 高 | 高 |
| **包体积** | 最小 (~50KB) | 大 (~5MB) | 大 | 中 |
| **依赖项** | 仅 squirrel | 多 | 多 | 少 |
| **数据库迁移** | 否 | 是 | 是 | 否 |
| **关联关系** | 否 | 是 | 是 | 否 |
| **SQL 控制** | 完全 | 有限 | 有限 | 完全 |
| **反射开销** | 无 | 高 | 无 | 无 |

## 适用场景

**推荐使用：**
- **性能关键应用**: 无反射开销，直接执行 SQL
- **现有数据库**: 使用遗留数据库，无需迁移
- **复杂查询**: 完全控制 SQL，类型安全的结果映射
- **微服务**: 依赖最少，二进制体积小
- **熟悉 SQL 的团队**: 偏好编写 SQL 而非 ORM 抽象

**不推荐使用：**
- 需要自动迁移和 Schema 管理的项目
- 需要复杂关联关系的应用
- 偏好 ORM 风格数据访问的团队

## 安装

```bash
go get github.com/Just-maple/sqlhelper
```

## 快速开始

### 定义模型

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

### 初始化

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/Just-maple/sqlhelper"
)

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/test")
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })
```

## Model 使用示例

### 基本查询

```go
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

// 查询所有列（从 FieldMapping 自动推断）
sql, _ := userHelper.ModelSelect(nil).ToSql()
// SELECT `age`, `email`, `id`, `name` FROM `users`

// 查询指定列
sql, _ = userHelper.ModelSelect([]string{"id", "name"}).ToSql()
// SELECT `id`, `name` FROM `users`

// 执行查询
users, err := userHelper.ModelSelect(nil).List(ctx, db)

// 查询单条记录
user, err := userHelper.ModelSelectWhere("id = ?", 1).One(ctx, db)
```

### Where 条件

```go
// 单个条件
users, _, err := userHelper.ModelPaginationWhere(ctx, db, &PageQuery{Page: 1, Limit: 10}, "age > ?", 18)

// 多个条件
users, _, err := userHelper.ModelSelectWhere("age > ? AND status = ?", 18, "active").List(ctx, db)

// IN 子句
users, _, err := userHelper.ModelSelectWhere("id IN (?, ?, ?)", 1, 2, 3).List(ctx, db)
```

### 使用别名

```go
// 在 ModelHelper 上设置别名
userHelper := sqlhelper.NewModelHelper(func() User { return User{} }).Alias("u")

sql, _ := userHelper.ModelSelect(nil).ToSql()
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`

// 别名 + WHERE
user, err := userHelper.ModelSelectWhere("u.id = ?", 1).One(ctx, db)
```

### 使用 Options (OrderBy, Limit, Join)

```go
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

// OrderBy 和 Limit
users, err := userHelper.ModelSelect(nil).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.OrderBy("created_at DESC").Limit(10)
    }).List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` ORDER BY created_at DESC LIMIT 10

// LEFT JOIN orders 表（无别名）
users, err := userHelper.ModelSelect(nil).
    WithOptions(func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.LeftJoin("orders o ON o.user_id = users.id")
    }).List(ctx, db)
// SELECT `age`, `email`, `id`, `name` FROM `users` LEFT JOIN orders o ON o.user_id = users.id

// 多表 JOIN（使用别名）
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

## Query 接口

`Query` 接口通过 `WithQueries` 实现可复用的 SQL 子句注入。分页场景请使用 `PaginationQuery`：

```go
// Query 用于 SQL 子句注入（WHERE、SORT、JOIN 等）
type Query interface {
    Option(helper Helper) SelectBuilderOption
}

// PaginationQuery 扩展 Query，添加分页 count 控制
type PaginationQuery interface {
    Query
    Countless() bool // true=跳过 count，false=执行 count
}
```

**关键要点：**
- `Option` 接收 `Helper` 上下文，支持带别名的列转义
- 在 Option 内使用 `h.EscapeColumn(key)` 转义带别名前缀的列
- 多个 `Query` 实例可通过 `WithQueries` 链式调用
- `Query` 用于 SQL 子句注入（WHERE、SORT、JOIN）
- `PaginationQuery` 扩展 `Query` 添加 `Countless()` 用于分页 count 控制：
  - `return false`: 执行并发 count 查询（用于前端分页显示总条数）
  - `return true`: 跳过 count 查询（用于内部使用、无限滚动、或不需要总数的场景）

**常见使用场景：**
- **分页**: LIMIT/OFFSET 注入
- **排序**: 动态字段和方向的 ORDER BY
- **过滤**: 带参数的 WHERE 条件
- **多租户**: 自动 tenant_id 过滤
- **连表注入**: 根据查询上下文动态 JOIN
- **软删除**: 自动 `deleted_at IS NULL` 过滤

### 分页查询

```go
type PageQuery struct {
    Page      int
    Limit     int
    Countless bool // 控制 count 查询：true=跳过，false=执行
}

func (p *PageQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Limit(uint64(p.Limit)).Offset(uint64((p.Page - 1) * p.Limit))
    }
}
func (p *PageQuery) Countless() bool { return p.Countless }

// 使用 - 开发者根据场景控制
// 前端分页需要总数
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10, Countless: false})

// 内部使用或无限滚动 - 跳过 count 查询提升性能
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10, Countless: true})
```

### 排序查询

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
// SortQuery 实现 Query 接口

// 与 WithQueries 一起使用
users, err := userHelper.ModelSelect(nil).
    WithQueries(&SortQuery{Field: "created_at", Desc: true}).
    List(ctx, db)
```

### 条件过滤查询

```go
type StatusFilterQuery struct {
    Status string
}

func (q *StatusFilterQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Where("status = ?", q.Status)
    }
}
// StatusFilterQuery 实现 Query 接口

// 使用 - 可与其他 Query 组合
users, err := userHelper.ModelSelect(nil).
    WithQueries(
        &StatusFilterQuery{Status: "active"},
        &SortQuery{Field: "id", Desc: true},
    ).
    List(ctx, db)
```

### 多租户查询

```go
type TenantQuery struct {
    TenantID string
}

func (q *TenantQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        // 使用 h.EscapeColumn 进行正确的别名感知转义
        return b.Where(h.EscapeColumn("tenant_id")+" = ?", q.TenantID)
    }
}
// TenantQuery 实现 Query 接口

// 与别名一起使用
h := sqlhelper.Helper{}.Alias("u")
users, err := h.Select([]string{"id", "name"}, "users").
    WithQueries(&TenantQuery{TenantID: "tenant_123"}).
    QueryRows(ctx, db)
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE `u`.`tenant_id` = ?
```

### 软删除查询

```go
type NotDeletedQuery struct{}

func (q *NotDeletedQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        return b.Where(h.EscapeColumn("deleted_at") + " IS NULL")
    }
}
// NotDeletedQuery 实现 Query 接口

// 使用 - 自动过滤软删除记录
users, err := userHelper.ModelSelect(nil).
    WithQueries(&NotDeletedQuery{}).
    List(ctx, db)
```

### 连表注入查询

```go
type WithOrderQuery struct {
    OrderStatus int
}

func (q *WithOrderQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        // 使用 h.EscapeColumn 进行正确的别名感知列转义
        userIDCol := h.EscapeColumn("id")
        return b.LeftJoin(
            "orders o ON o.user_id = "+userIDCol+" AND o.status = ?",
            q.OrderStatus,
        )
    }
}
// WithOrderQuery 实现 Query 接口

// 与别名一起使用
h := sqlhelper.Helper{}.Alias("u")
users, err := h.Select([]string{"id", "name"}, "users").
    WithQueries(&WithOrderQuery{OrderStatus: 1}).
    QueryRows(ctx, db)
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` 
// LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ?
```

### 连表注入查询

```go
type WithOrderQuery struct {
    OrderStatus int
}

func (q *WithOrderQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
    return func(b sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
        // 使用 h.EscapeColumn 进行正确的别名感知列转义
        userIDCol := h.EscapeColumn("id")
        return b.LeftJoin(
            "orders o ON o.user_id = "+userIDCol+" AND o.status = ?",
            q.OrderStatus,
        )
    }
}
// WithOrderQuery 实现 Query 接口

// 与别名一起使用
h := sqlhelper.Helper{}.Alias("u")
users, err := h.Select([]string{"id", "name"}, "users").
    WithQueries(&WithOrderQuery{OrderStatus: 1}).
    QueryRows(ctx, db)
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` 
// LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ?
```
// LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ?
```

### 综合示例：组合多种 Query 类型与分页

```go
// 组合分页、排序、过滤、多租户
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

// 生成的 SQL:
// SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`
// WHERE `u`.`tenant_id` = ? AND status = ?
// LEFT JOIN orders o ON o.user_id = u.id
// ORDER BY `u`.`created_at` DESC
// LIMIT 10 OFFSET 0
```

### 插入操作

```go
// 单条插入
result, err := userHelper.ModelInsert([]string{"name", "email"}, &User{
    Name: "John", Email: "john@test.com",
}).Exec(ctx, db)

// 批量插入
users := []User{
    {Name: "John", Email: "john@test.com"},
    {Name: "Jane", Email: "jane@test.com"},
}
result, err := userHelper.ModelInserts([]string{"name", "email"}, users).Exec(ctx, db)

// ON DUPLICATE KEY UPDATE
result, err := userHelper.ModelInsert([]string{"id", "name", "email"}, &User{
    ID: 1, Name: "John", Email: "john@test.com",
}).OnDuplicateUpdateValues("name", "email").Exec(ctx, db)
```

### 更新操作

```go
// 带 WHERE 更新
result, err := userHelper.ModelUpdate(&User{ID: 1, Name: "John"}, []string{"name"}).
    Where("id = ?", 1).
    Exec(ctx, db)

// 自定义条件更新
result, err := userHelper.ModelUpdate(&User{Name: "Updated"}, []string{"name"}).
    Where("age > ?", 18).
    Exec(ctx, db)
```

### 列过滤

```go
// 获取所有列
allCols := userHelper.Columns(nil)
// []string{"age", "email", "id", "name"}

// 过滤列
cols := userHelper.Columns(func(col string) bool {
    return col != "email" // 排除 email
})
// []string{"age", "id", "name"}

// 在查询中使用过滤后的列
users, err := userHelper.ModelSelect(cols).List(ctx, db)
```

### MappingModel（无需实现 Model 接口）

```go
// 如果不想实现 Model 接口，使用 MappingModel
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

// 使用与 ModelHelper 相同的 API
users, err := userHelper.ModelSelect(nil).List(ctx, db)
```

### 事务支持

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// 使用事务作为 Conn
_, err = userHelper.ModelInsert([]string{"name"}, &User{Name: "John"}).Exec(ctx, tx)
if err != nil {
    return err
}

_, err = userHelper.ModelUpdate(&User{ID: 1, Name: "John"}, []string{"name"}).
    Where("id = ?", 1).Exec(ctx, tx)
if err != nil {
    return err
}

return tx.Commit()
```

## Helper 使用示例（原生 SQL）

```go
h := sqlhelper.Helper{}

// 基本查询
sql, _ := h.Select([]string{"id", "name"}, "users").ToSql()

// 带 WHERE
sql, _ = h.Select([]string{"id", "name"}, "users").
    Where("age > ?", 18).
    ToSql()

// 带别名
h = sqlhelper.Helper{}.Alias("u")
sql, _ = h.Select([]string{"id", "name"}, "users").ToSql()
// SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`

// 自定义转义（PostgreSQL）
h = sqlhelper.Helper{}.WithEscapeFunc(func(key string, table bool) string {
    return fmt.Sprintf("\"%s\"", key)
})
sql, _ = h.Select([]string{"id", "name"}, "users").ToSql()
// SELECT "id", "name" FROM "users"
```

## API 参考

### Helper

| 方法 | 描述 |
|------|------|
| `Select(columns, table)` | 创建 SELECT 查询 |
| `Insert(table, columns, values...)` | 创建 INSERT 查询 |
| `Update(table, values)` | 创建 UPDATE 查询 |
| `Alias(alias)` | 设置表别名 |
| `EscapeColumn(column)` | 转义列名 |
| `EscapeTable(table)` | 转义表名 |
| `WithEscapeFunc(fn)` | 设置自定义转义函数 |

### SelectExecutor

| 方法 | 描述 |
|------|------|
| `Where(pred, args...)` | 添加 WHERE 条件 |
| `WithOptions(opts...)` | 应用构建器选项 |
| `WithQueries(queries...)` | 应用 Query 选项 |
| `ToSql()` | 获取 SQL 和参数 |
| `QueryRow(ctx, conn)` | 执行并返回单行 |
| `QueryRows(ctx, conn)` | 执行并返回多行 |
| `Count(ctx, conn)` | 获取总数 |
| `Pagination(ctx, conn, query, alloc)` | 分页查询 |

### ModelHelper

| 方法 | 描述 |
|------|------|
| `NewModelHelper(alloc)` | 创建新的 ModelHelper |
| `ModelSelect(columns)` | 创建 SELECT 查询 |
| `ModelSelectWhere(pred, args...)` | 创建带 WHERE 的 SELECT |
| `ModelPagination(ctx, conn, query)` | 分页查询 |
| `ModelPaginationWhere(ctx, conn, query, pred, args...)` | 带 WHERE 的分页查询 |
| `ModelInsert(columns, models...)` | 插入记录 |
| `ModelInserts(columns, models)` | 批量插入记录 |
| `ModelUpdate(model, columns)` | 更新记录 |
| `Alias(alias)` | 设置表别名 |
| `Columns(filter)` | 获取列名，可选过滤函数 |

### Query 接口

```go
// Query 用于 SQL 子句注入
type Query interface {
    Option(helper Helper) SelectBuilderOption
}

// PaginationQuery 扩展 Query 添加 count 控制
type PaginationQuery interface {
    Query
    Countless() bool
}
```

### Model 接口

```go
type Model interface {
    TableName() string
    FieldMapping(dst map[string]any)
}
```
