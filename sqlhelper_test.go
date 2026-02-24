package sqlhelper

import (
	"testing"
)

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

func TestHelper_Alias(t *testing.T) {
	h := Helper{}.Alias("u")
	sql, _, _ := h.Select([]string{"id", "name"}, "users").Where("u.id = ?", 1).ToSql()
	want := "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?"
	testSQL(t, "Alias Select Where", sql, want)
}

func TestHelper_CustomEscape(t *testing.T) {
	h := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	})
	sql, _, _ := h.Select([]string{"id", "name"}, "users").ToSql()
	want := "SELECT \"id\", \"name\" FROM \"users\""
	testSQL(t, "CustomEscape", sql, want)
}

func TestHelper_CustomEscapeWithAlias(t *testing.T) {
	h := Helper{}.Alias("u").WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	})
	sql, _, _ := h.Select([]string{"id", "name"}, "users").ToSql()
	want := "SELECT \"u\".\"id\", \"u\".\"name\" FROM \"users\" AS \"u\""
	testSQL(t, "CustomEscape Alias", sql, want)
}

func TestSelectExecutor_Basic(t *testing.T) {
	sql, _, _ := Helper{}.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select", sql, "SELECT `id`, `name` FROM `users`")
}

func TestSelectExecutor_Where(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
	testSQL(t, "Select Where", sql, "SELECT `id`, `name` FROM `users` WHERE age > ?")
	if len(args) != 1 {
		t.Errorf("args len = %d, want 1", len(args))
	}
}

func TestSelectExecutor_Distinct(t *testing.T) {
	sql, _, _ := Helper{}.SelectDistinct("name", "users").ToSql()
	testSQL(t, "SelectDistinct", sql, "SELECT DISTINCT(`name`) FROM `users`")
}

func TestSelectExecutor_WithOptions(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").
		WithOptions(func(b SelectBuilder) SelectBuilder {
			return b.Where("id > ?", 10).OrderBy("id DESC").Limit(10)
		}).ToSql()
	testSQL(t, "WithOptions", sql, "SELECT `id`, `name` FROM `users` WHERE id > ? ORDER BY id DESC LIMIT 10")
	if len(args) != 1 {
		t.Errorf("args len = %d, want 1", len(args))
	}
}

type pageQuery struct{ page, size int }

func (p pageQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Limit(uint64(p.size)).Offset(uint64((p.page - 1) * p.size))
	}
}
func (p pageQuery) Countless() bool { return false }

type orderQuery struct {
	field string
	desc  bool
}

func (q orderQuery) Option(h Helper) SelectBuilderOption {
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

func (q whereQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder { return b.Where(q.cond, q.args...) }
}

func TestSelectExecutor_WithQueries(t *testing.T) {
	tests := []struct {
		name    string
		exec    SelectExecutor
		queries []Query
		want    string
	}{
		{
			name:    "Basic",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{pageQuery{page: 2, size: 20}},
			want:    "SELECT `id`, `name` FROM `users` LIMIT 20 OFFSET 20",
		},
		{
			name:    "Alias",
			exec:    Helper{}.Alias("u").Select([]string{"id", "name"}, "users"),
			queries: []Query{pageQuery{page: 1, size: 10}},
			want:    "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` LIMIT 10 OFFSET 0",
		},
		{
			name:    "Order",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{orderQuery{field: "id", desc: true}},
			want:    "SELECT `id`, `name` FROM `users` ORDER BY id DESC",
		},
		{
			name:    "Where",
			exec:    Helper{}.Select([]string{"id", "name"}, "users"),
			queries: []Query{whereQuery{cond: "status = ?", args: []any{"active"}}},
			want:    "SELECT `id`, `name` FROM `users` WHERE status = ?",
		},
		{
			name: "Combined",
			exec: Helper{}.Alias("u").Select([]string{"id", "name"}, "users"),
			queries: []Query{
				whereQuery{cond: "u.age > ?", args: []any{18}},
				orderQuery{field: "u.id", desc: true},
				pageQuery{page: 1, size: 10},
			},
			want: "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.age > ? ORDER BY u.id DESC LIMIT 10 OFFSET 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, _ := tt.exec.WithQueries(tt.queries...).ToSql()
			testSQL(t, tt.name, sql, tt.want)
		})
	}
}

func TestInsertExecutor_Basic(t *testing.T) {
	sql, args, _ := Helper{}.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).ToSql()
	testSQL(t, "Insert", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?)")
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

func TestInsertExecutor_OnDuplicate(t *testing.T) {
	sql, _, _ := Helper{}.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).
		OnDuplicateUpdateValues("name", "email").ToSql()
	testSQL(t, "OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)")
}

func TestInsertExecutor_CustomEscape(t *testing.T) {
	sql, _, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Insert("users", []string{"name"}, []any{"John"}).ToSql()
	testSQL(t, "Insert CustomEscape", sql, "INSERT INTO \"users\" (\"name\") VALUES (?)")
}

func TestUpdateExecutor_Basic(t *testing.T) {
	sql, args, _ := Helper{}.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update", sql, "UPDATE `users` SET `name` = ? WHERE id = ?")
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

func TestUpdateExecutor_CustomEscape(t *testing.T) {
	sql, _, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update CustomEscape", sql, "UPDATE \"users\" SET \"name\" = ? WHERE id = ?")
}

func TestModelHelper_Select(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect(nil).ToSql()
	testSQL(t, "ModelSelect", sql, "SELECT `age`, `email`, `id`, `name` FROM `users`")
}

func TestModelHelper_SelectColumns(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect([]string{"id", "name"}).ToSql()
	testSQL(t, "ModelSelect columns", sql, "SELECT `id`, `name` FROM `users`")
}

func TestModelHelper_Where(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect(nil).Where("age > ?", 18).ToSql()
	testSQL(t, "ModelSelect Where", sql, "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ?")
	if len(args) != 1 {
		t.Errorf("args len = %d, want 1", len(args))
	}
}

func TestModelHelper_Insert(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).ToSql()
	testSQL(t, "ModelInsert", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?)")
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

func TestModelHelper_InsertOnDuplicate(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).
		OnDuplicateUpdateValues("name").ToSql()
	testSQL(t, "ModelInsert OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)")
}

func TestModelHelper_Update(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelUpdate(&testUser{ID: 1, Name: "John"}, []string{"name"}).Where("id = ?", 1).ToSql()
	testSQL(t, "ModelUpdate", sql, "UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1")
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

func TestModelHelper_Alias(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).Alias("u").ModelSelect(nil).ToSql()
	testSQL(t, "ModelSelect Alias", sql, "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`")
}

func TestModelHelper_Columns(t *testing.T) {
	columns := NewModelHelper(func() testUser { return testUser{} }).Columns(func(col string) bool {
		return col != "email"
	})
	if len(columns) != 3 {
		t.Errorf("Columns len = %d, want 3", len(columns))
	}
}

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
	testSQL(t, "MappingModel Select", sql, "SELECT `age`, `email`, `id`, `name` FROM `users`")
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
