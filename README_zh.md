# SQLHelper

一个轻量级的 Go SQL 辅助库，基于 [squirrel](https://github.com/Masterminds/squirrel) 构建。提供流畅的数据库操作接口，支持模型映射和分页。

## 特性

- **ModelHelper**: 基于泛型的模型操作，完全类型安全
- **流畅查询构建器**: 易于使用的链式 API 构建 SQL 查询
- **分页支持**: 内置大数据集分页功能
- **自定义转义**: 支持自定义转义函数，适用于不同 SQL 方言
- **事务支持**: 支持数据库连接和事务
- **MappingModel**: 无需实现接口的灵活模型定义

## 与其他框架对比

| 特性 | SQLHelper | GORM | ent | SQLBoiler | XORM |
|------|-----------|------|-----|-----------|------|
| **类型安全** | 泛型 | 反射 | 代码生成 | 代码生成 | 反射 |
| **学习曲线** | 低 | 中 | 高 | 高 | 低 |
| **迁移/自动建表** | 否 | 是 | 是 | 是 | 是 |
| **关联关系** | 否 | 是 | 是 | 是 | 是 |
| **原生 SQL 支持** | 优秀 | 有限 | 有限 | 良好 | 良好 |
| **包体积** | 最小 | 大 | 大 | 大 | 中 |
| **依赖项** | 仅 squirrel | 多 | 多 | 少 | 少 |

## 优势

- **轻量级**: 仅依赖 squirrel，依赖最少
- **类型安全**: 完整的泛型支持，编译时类型检查
- **灵活性**: 按需使用原生 SQL，无 ORM 锁定
- **简洁**: 对 SQL 的抽象最少，易于理解和调试
- **快速**: 无反射开销，直接执行 SQL

## 适用场景

- **性能关键应用**: 需要直接 SQL 控制和类型安全
- **现有数据库**: 使用没有迁移的遗留数据库
- **简单 CRUD**: 不需要复杂关联或迁移
- **混合方案**: 复杂查询使用原生 SQL，结果用模型映射
- **学习/原型开发**: 快速启动，最少样板代码

## 不适用场景

- **GORM/ent**: 需要自动迁移和复杂关联
- **SQLBoiler**: 需要代码生成的最大性能
- **传统 ORM**: 团队熟悉 Django/Rails 风格的 ORM 模式

## 安装

```bash
go get github.com/Just-maple/sqlhelper
```

## 快速开始

### 定义模型

在结构体上实现 `Model` 接口：

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

### 初始化 ModelHelper

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
    "github.com/Just-maple/sqlhelper"
)

db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/test")

// 创建 ModelHelper - 类型自动推断
userHelper := sqlhelper.NewModelHelper(func() User { return User{} })
```

## ModelHelper 用法（推荐）

### 查询操作

```go
// 分页查询 - 返回模型列表和总数
users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{Page: 1, Limit: 10})

// 带 WHERE 条件的分页查询
users, total, err := userHelper.ModelPaginationWhere(ctx, db, &PageQuery{Page: 1, Limit: 10}, "age > ?", 18)

// 查询指定列
exec := userHelper.ModelSelect([]string{"id", "name", "email"})
users, err := exec.List(ctx, db)

// 查询单条记录
user, err := userHelper.ModelSelectWhere("id = ?", 1).One(ctx, db)

// DISTINCT 分页查询单列
emails, total, err := userHelper.ModelDistinctPagination(ctx, db, &PageQuery{Page: 1, Limit: 10}, "email")
```

### 插入操作

```go
// 插入单个模型
result, err := userHelper.ModelInsert(nil, user).Exec(ctx, db)

// 插入多个模型
result, err := userHelper.ModelInsert(nil, user1, user2, user3).Exec(ctx, db)

// 插入并使用 ON DUPLICATE KEY UPDATE
result, err := userHelper.ModelInsert([]string{"id", "name", "email"}, user).
    OnDuplicateUpdateValues("name", "email").
    Exec(ctx, db)
```

### 更新操作

```go
// 按 ID 更新模型 - 需要手动添加 WHERE 条件
result, err := userHelper.ModelUpdate(user, nil).Where("id = ?", user.ID).Exec(ctx, db)

// 自定义 WHERE 更新
result, err := sqlhelper.Helper{}.Update("users", map[string]any{"name": "John"}).Where("age > ?", 18).Exec(ctx, db)
```

## 原生 SQL 辅助用法

### 基本查询

```go
h := sqlhelper.Helper{}

// 简单查询
exec := h.Select([]string{"id", "name", "email"}, "users")
rows, err := exec.QueryRows(ctx, db)

// 带 WHERE 条件
exec = h.Select([]string{"id", "name"}, "users").Where("age > ?", 18)

// DISTINCT 查询
exec = h.SelectDistinct("email", "users")
```

### 插入/更新/删除

```go
// 插入
result, err := h.Insert("users", []string{"name", "email"}, []any{"John", "john@example.com"}).Exec(ctx, db)

// 更新
result, err := h.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).Exec(ctx, db)
```

### 自定义转义函数

```go
// 使用双引号代替反引号（例如 PostgreSQL）
h := sqlhelper.Helper{}
h = h.WithEscapeFunc(func(key string, table bool) string {
    return fmt.Sprintf("\"%s\"", key)
})
```

## 表别名

```go
h := sqlhelper.Helper{}.Alias("u")
// SELECT u.id, u.name FROM users u
exec := h.Select([]string{"id", "name"}, "users")
```

## 分页查询

SQLHelper 使用 `PaginationQuery` 接口进行分页：

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
    return false // true 跳过 count 查询
}
```

## MappingModel（Model 接口的替代方案）

如果不想在结构体上实现 Model 接口，可以使用 MappingModel：

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

## API 参考

### Helper

- `Helper{}.EscapeColumn(column)` - 转义列名
- `Helper{}.EscapeTable(table)` - 转义表名
- `Helper{}.EscapeColumns(columns)` - 转义多个列名
- `Helper{}.Alias(alias)` - 设置表别名
- `Helper{}.WithEscapeFunc(fn)` - 设置自定义转义函数
- `Helper{}.Select(columns, table, opts...)` - 创建 SELECT 查询
- `Helper{}.SelectDistinct(column, table)` - 创建 SELECT DISTINCT 查询
- `Helper{}.Insert(table, columns, values...)` - 创建 INSERT 查询
- `Helper{}.Update(table, values, opts...)` - 创建 UPDATE 查询

### ModelHelper

- `NewModelHelper(alloc func() T)` - 创建新的 ModelHelper
- `ModelPagination(ctx, conn, query)` - 分页查询
- `ModelPaginationWhere(ctx, conn, query, pred, args...)` - 带 WHERE 的分页查询
- `ModelSelect(columns, opts...)` - 创建 SELECT 查询
- `ModelSelectWhere(pred, args...)` - 创建带 WHERE 的 SELECT
- `ModelInsert(columns, models...)` - 插入记录
- `ModelInserts(columns, models)` - 批量插入记录
- `ModelUpdate(model, columns)` - 根据 ID 更新记录
- `ModelDistinctPagination(ctx, conn, query, column)` - DISTINCT 分页查询
- `ModelHelper{}.Alias(alias)` - 为查询设置表别名
- `ModelHelper{}.Columns(filter)` - 获取列名，可选过滤函数
- `ModelHelper{}.Convert(mapper)` - 转换为 MappingModel

### SelectExecutor

- `One(ctx, conn)` - 获取单条记录
- `List(ctx, conn)` - 获取所有记录
- `Where(pred, args...)` - 添加 WHERE 条件
- `WithOptions(opts...)` - 应用构建器选项
- `ToSql()` - 获取 SQL 和参数
- `Count(ctx, conn)` - 获取总数
- `Pagination(ctx, conn, query, alloc)` - 分页查询
- `PaginationModels(ctx, conn, query)` - 分页查询返回模型
- `PaginationStrings(ctx, conn, query)` - 分页查询返回单列字符串
- `PaginationMaps(ctx, conn, query)` - 分页查询返回 map
- `QueryRowScan(ctx, conn, alloc)` - 自定义扫描单行
- `QueryRowsScans(ctx, conn, alloc)` - 自定义扫描多行
- `QueryRowScanModel(ctx, conn, alloc)` - 扫描到 Model
- `QueryRowsScansModels(ctx, conn, alloc)` - 扫描多条到 Model
- `QueryStrings(ctx, conn)` - 获取字符串值
- `QueryTotals(ctx, conn, alloc, total)` - 查询并获取总数

### InsertExecutor

- `Exec(ctx, conn)` - 执行插入
- `ExecLastInsertId(ctx, conn)` - 执行并返回最后插入 ID
- `ToSql()` - 获取 SQL 和参数
- `WithOptions(opts...)` - 应用构建器选项
- `OnDuplicateUpdateValues(columns...)` - 添加 ON DUPLICATE KEY UPDATE

### UpdateExecutor

- `Exec(ctx, conn)` - 执行更新
- `ExecRowsAffected(ctx, conn)` - 执行并返回受影响行数
- `ToSql()` - 获取 SQL 和参数
- `Where(pred, args...)` - 添加 WHERE 条件
- `Limit(limit)` - 设置行数限制
- `WithOptions(opts...)` - 应用构建器选项

### Model 接口

```go
type Model interface {
    TableName() string
    FieldMapping(dst map[string]any)
}
```

### PaginationQuery 接口

```go
type PaginationQuery interface {
    Option(helper Helper) SelectBuilderOption
    Countless() bool
}
```

### 自定义转义函数

```go
type EscapeFunc func(key string, table bool) string
```

- `key`: 要转义的表名或列名
- `table`: true 表示转义表名，false 表示转义列名
- 返回转义后的字符串
