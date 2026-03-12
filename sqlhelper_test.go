package sqlhelper

import (
	"testing"
)

// ==================== Helper Tests ====================

func TestHelper_EscapeColumn(t *testing.T) {
	h := Helper{}
	tests := []struct {
		name   string
		column string
		want   string
	}{
		{"simple", "id", "`id`"},
		{"escaped", "`name`", "`name`"},
		{"with_space", "user name", "user name"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.EscapeColumn(tt.column); got != tt.want {
				t.Errorf("EscapeColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelper_EscapeTable(t *testing.T) {
	h := Helper{}
	tests := []struct {
		name  string
		table string
		want  string
	}{
		{"simple", "users", "`users`"},
		{"escaped", "`products`", "`products`"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.EscapeTable(tt.table); got != tt.want {
				t.Errorf("EscapeTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelper(t *testing.T) {
	tests := []struct {
		name     string
		helper   Helper
		buildSQL func(Helper) (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:   "Alias",
			helper: Helper{}.Alias("u"),
			buildSQL: func(h Helper) (string, []any, error) {
				return h.Select([]string{"id", "name"}, "users",
					h.SelectOptions().Limit(2)...,
				).Where("u.id = ?", 1).ToSql()
			},
			wantSQL:  "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ? LIMIT 2",
			wantArgs: 1,
		},
		{
			name: "CustomEscape",
			helper: Helper{}.WithEscapeFunc(func(key string, table bool) string {
				return "\"" + key + "\""
			}),
			buildSQL: func(h Helper) (string, []any, error) {
				return h.Select([]string{"id", "name"}, "users").ToSql()
			},
			wantSQL:  "SELECT \"id\", \"name\" FROM \"users\"",
			wantArgs: 0,
		},
		{
			name: "CustomEscapeWithAlias",
			helper: Helper{}.Alias("u").WithEscapeFunc(func(key string, table bool) string {
				return "\"" + key + "\""
			}),
			buildSQL: func(h Helper) (string, []any, error) {
				return h.Select([]string{"id", "name"}, "users").ToSql()
			},
			wantSQL:  "SELECT \"u\".\"id\", \"u\".\"name\" FROM \"users\" AS \"u\"",
			wantArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.buildSQL(tt.helper)
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name, sql, tt.wantSQL)
			testArgsLen(t, tt.name+" args", args, tt.wantArgs)
		})
	}
}

// ==================== SelectExecutor Tests ====================

func TestSelectExecutor(t *testing.T) {
	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:     "Basic",
			buildSQL: func() (string, []any, error) { return Helper{}.Select([]string{"id", "name"}, "users").ToSql() },
			wantSQL:  "SELECT `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name: "Alias",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Alias("u").Select([]string{"id", "name"}, "users").ToSql()
			},
			wantSQL:  "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name: "AliasSelectDistinct",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Alias("u").Select([]string{"DISTINCT(user)"}, "users").ToSql()
			},
			wantSQL:  "SELECT DISTINCT(user) FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name: "AliasSelectDistinct2",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Alias("u").Select([]string{"DISTINCT(`u`.`user`)"}, "users").ToSql()
			},
			wantSQL:  "SELECT DISTINCT(`u`.`user`) FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name: "Where",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
			},
			wantSQL:  "SELECT `id`, `name` FROM `users` WHERE age > ?",
			wantArgs: 1,
		},
		{
			name:     "Distinct",
			buildSQL: func() (string, []any, error) { return Helper{}.SelectDistinct("name", "users").ToSql() },
			wantSQL:  "SELECT DISTINCT(`name`) FROM `users`",
			wantArgs: 0,
		},
		{
			name:     "DistinctWithAlias",
			buildSQL: func() (string, []any, error) { return Helper{}.Alias("u").SelectDistinct("name", "users").ToSql() },
			wantSQL:  "SELECT DISTINCT(`u`.`name`) FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name: "WithOptions",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Select([]string{"id", "name"}, "users").
					WithOptions(func(b SelectBuilder) SelectBuilder {
						return b.Where("id > ?", 10).OrderBy("id DESC").Limit(10)
					}).ToSql()
			},
			wantSQL:  "SELECT `id`, `name` FROM `users` WHERE id > ? ORDER BY id DESC LIMIT 10",
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.buildSQL()
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name, sql, tt.wantSQL)
			testArgsLen(t, tt.name+" args", args, tt.wantArgs)
		})
	}
}

type pageQuery struct{ page, size int }

func (p pageQuery) Option(h Helper) SelectOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Limit(uint64(p.size)).Offset(uint64((p.page - 1) * p.size))
	}
}
func (p pageQuery) Countless() bool { return false }

type orderQuery struct {
	field string
	desc  bool
}

func (q orderQuery) Option(h Helper) SelectOption {
	o := q.field
	if q.desc {
		o += " DESC"
	}
	return func(b SelectBuilder) SelectBuilder { return b.OrderByClause(o) }
}

type whereQuery struct {
	cond string
	args []any
}

func (q whereQuery) Option(h Helper) SelectOption {
	return func(b SelectBuilder) SelectBuilder { return b.Where(q.cond, q.args...) }
}

func TestSelectExecutor_WithQueries(t *testing.T) {
	tests := []struct {
		name    string
		exec    SelectExecutor
		queries []Query
		wantSQL string
	}{
		{
			name:    "Basic",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{pageQuery{page: 2, size: 20}},
			wantSQL: "SELECT `id`, `name` FROM `users` LIMIT 20 OFFSET 20",
		},
		{
			name:    "Alias",
			exec:    Helper{}.Alias("u").Select([]string{"id", "name"}, "users"),
			queries: []Query{pageQuery{page: 1, size: 10}},
			wantSQL: "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` LIMIT 10 OFFSET 0",
		},
		{
			name:    "Order",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{orderQuery{field: "id", desc: true}},
			wantSQL: "SELECT `id`, `name` FROM `users` ORDER BY id DESC",
		},
		{
			name:    "Where",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{whereQuery{cond: "status = ?", args: []any{"active"}}},
			wantSQL: "SELECT `id`, `name` FROM `users` WHERE status = ?",
		},
		{
			name: "Combined",
			exec: Helper{}.Alias("u").Select([]string{"id", "name"}, "users"),
			queries: []Query{
				whereQuery{cond: "u.age > ?", args: []any{18}},
				orderQuery{field: "u.id", desc: true},
				pageQuery{page: 1, size: 10},
			},
			wantSQL: "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.age > ? ORDER BY u.id DESC LIMIT 10 OFFSET 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, _ := tt.exec.WithQueries(tt.queries...).ToSql()
			testSQL(t, tt.name, sql, tt.wantSQL)
		})
	}
}

// ==================== InsertExecutor Tests ====================

func TestInsertExecutor(t *testing.T) {
	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "Basic",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?)",
			wantArgs: 2,
		},
		{
			name: "OnDuplicate",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).
					OnDuplicateUpdateValues("name", "email").ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)",
			wantArgs: 2,
		},
		{
			name: "CustomEscape",
			buildSQL: func() (string, []any, error) {
				return Helper{}.WithEscapeFunc(func(key string, table bool) string {
					return "\"" + key + "\""
				}).Insert("users", []string{"name"}, []any{"John"}).ToSql()
			},
			wantSQL:  "INSERT INTO \"users\" (\"name\") VALUES (?)",
			wantArgs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.buildSQL()
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name, sql, tt.wantSQL)
			testArgsLen(t, tt.name+" args", args, tt.wantArgs)
		})
	}
}

// ==================== UpdateExecutor Tests ====================

func TestUpdateExecutor(t *testing.T) {
	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "Basic",
			buildSQL: func() (string, []any, error) {
				return Helper{}.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE id = ?",
			wantArgs: 2,
		},
		{
			name: "CustomEscape",
			buildSQL: func() (string, []any, error) {
				return Helper{}.WithEscapeFunc(func(key string, table bool) string {
					return "\"" + key + "\""
				}).Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE \"users\" SET \"name\" = ? WHERE id = ?",
			wantArgs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.buildSQL()
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name, sql, tt.wantSQL)
			testArgsLen(t, tt.name+" args", args, tt.wantArgs)
		})
	}
}

// ==================== ModelHelper Tests ====================

func TestModelHelper(t *testing.T) {
	tests := []struct {
		name     string
		helper   ModelHelper[testUser, *testUser]
		buildSQL func(ModelHelper[testUser, *testUser]) (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:     "Select",
			helper:   NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) { return h.ModelSelect(nil).ToSql() },
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name:   "SelectColumns",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelSelect([]string{"id", "name"}).ToSql()
			},
			wantSQL:  "SELECT `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name:   "Where",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelSelect(nil).Where("age > ?", 18).ToSql()
			},
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ?",
			wantArgs: 1,
		},
		{
			name:   "Insert",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?)",
			wantArgs: 2,
		},
		{
			name:   "InsertOnDuplicate",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).
					OnDuplicateUpdateValues("name").ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)",
			wantArgs: 2,
		},
		{
			name:   "Update",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelUpdate(&testUser{ID: 1, Name: "John"}, []string{"name"}).Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1",
			wantArgs: 2,
		},
		{
			name:     "Alias",
			helper:   NewModelHelper(func() testUser { return testUser{} }).Alias("u"),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) { return h.ModelSelect(nil).ToSql() },
			wantSQL:  "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`",
			wantArgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := tt.buildSQL(tt.helper)
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name, sql, tt.wantSQL)
			testArgsLen(t, tt.name+" args", args, tt.wantArgs)
		})
	}
}

func TestModelHelper_Columns(t *testing.T) {
	tests := []struct {
		name      string
		filter    func(string) bool
		wantCount int
	}{
		{
			name:      "All",
			filter:    nil,
			wantCount: 4,
		},
		{
			name:      "Filtered",
			filter:    func(col string) bool { return col != "email" },
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewModelHelper(func() testUser { return testUser{} })
			columns := h.Columns(tt.filter)
			if len(columns) != tt.wantCount {
				t.Errorf("Columns len = %d, want %d", len(columns), tt.wantCount)
			}
		})
	}
}

// ==================== MappingModel Tests ====================

func TestMappingModel(t *testing.T) {
	h := NewMappingModelHelper(func(u *testUser) (string, map[string]any) {
		return "users", map[string]any{
			"id":    &u.ID,
			"name":  &u.Name,
			"email": &u.Email,
			"age":   &u.Age,
		}
	})

	sql, _, _ := h.ModelSelect(nil).ToSql()
	testSQL(t, "Select", sql, "SELECT `age`, `email`, `id`, `name` FROM `users`")
}

func TestModelHelper_Convert(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })
	convertFn := h.Convert(func(u *testUser) map[string]any {
		return map[string]any{
			"id":   &u.ID,
			"name": &u.Name,
		}
	})
	mm := convertFn()
	if mm.TableName() != "users" {
		t.Errorf("TableName = %s, want users", mm.TableName())
	}
}

// ==================== Mapper Tests ====================

func TestMapper_MapColumns(t *testing.T) {
	m := Mapper{"id": 1, "name": "John", "email": "john@test.com"}
	var columns []string
	m.MapColumns(&columns)
	if len(columns) != 3 {
		t.Errorf("MapColumns len = %d, want 3", len(columns))
	}
}

func TestMapper_MapValues(t *testing.T) {
	u := &testUser{ID: 1, Name: "John", Email: "john@test.com", Age: 25}
	mapping := make(Mapper)
	u.FieldMapping(mapping)
	values := mapping.MapValues(u, []string{"id", "name"})
	if len(values) != 2 {
		t.Errorf("MapValues len = %d, want 2", len(values))
	}
}

func TestConvertModelMapping(t *testing.T) {
	alloc := func() Model { return &testUser{} }
	fn := ConvertModelMapping(alloc)
	values := fn([]string{"id", "name"})
	if len(values) != 2 {
		t.Errorf("ConvertModelMapping len = %d, want 2", len(values))
	}
}
