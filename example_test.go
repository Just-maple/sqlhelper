package sqlhelper

import (
	"strings"
	"testing"
)

type testUser struct {
	ID    int
	Name  string
	Email string
	Age   int
}

func (u *testUser) TableName() string { return "users" }
func (u *testUser) FieldMapping(dst map[string]any) {
	dst["id"] = &u.ID
	dst["name"] = &u.Name
	dst["email"] = &u.Email
	dst["age"] = &u.Age
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// Query implementations for testing
type testPageQuery struct {
	Page    int
	Limit   int
	SkipCnt bool
}

func (p *testPageQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Limit(uint64(p.Limit)).Offset(uint64((p.Page - 1) * p.Limit))
	}
}
func (p *testPageQuery) Countless() bool { return p.SkipCnt }

type testSortQuery struct {
	Field string
	Desc  bool
}

func (q *testSortQuery) Option(h Helper) SelectBuilderOption {
	order := "ASC"
	if q.Desc {
		order = "DESC"
	}
	return func(b SelectBuilder) SelectBuilder {
		return b.OrderByClause(h.EscapeColumn(q.Field) + " " + order)
	}
}

type testStatusFilterQuery struct {
	Status string
}

func (q *testStatusFilterQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Where("status = ?", q.Status)
	}
}

type testTenantQuery struct {
	TenantID string
}

func (q *testTenantQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Where(h.EscapeColumn("tenant_id")+" = ?", q.TenantID)
	}
}

type testNotDeletedQuery struct{}

func (q *testNotDeletedQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		return b.Where(h.EscapeColumn("deleted_at") + " IS NULL")
	}
}

type testWithOrderQuery struct {
	OrderStatus int
}

func (q *testWithOrderQuery) Option(h Helper) SelectBuilderOption {
	return func(b SelectBuilder) SelectBuilder {
		userIDCol := h.EscapeColumn("id")
		return b.LeftJoin(
			"orders o ON o.user_id = "+userIDCol+" AND o.status = ?",
			q.OrderStatus,
		)
	}
}

// ==================== ModelHelper Tests ====================

func TestExample_ModelHelper_BasicSelect(t *testing.T) {
	tests := []struct {
		name     string
		helper   ModelHelper[testUser, *testUser]
		buildSQL func(ModelHelper[testUser, *testUser]) (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:     "ModelSelect all",
			helper:   NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) { return h.ModelSelect(nil).ToSql() },
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name:   "ModelSelect columns",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelSelect([]string{"id", "name"}).ToSql()
			},
			wantSQL:  "SELECT `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name:   "ModelSelectWhere",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func(h ModelHelper[testUser, *testUser]) (string, []any, error) {
				return h.ModelSelectWhere("id = ?", 1).ToSql()
			},
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id = ?",
			wantArgs: 1,
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

func TestExample_ModelHelper_Where(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "ModelSelectWhere multi",
			buildSQL: func() (string, []any, error) {
				return h.ModelSelectWhere("age > ? AND status = ?", 18, "active").ToSql()
			},
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ? AND status = ?",
			wantArgs: 2,
		},
		{
			name:     "ModelSelectWhere IN",
			buildSQL: func() (string, []any, error) { return h.ModelSelectWhere("id IN (?, ?, ?)", 1, 2, 3).ToSql() },
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id IN (?, ?, ?)",
			wantArgs: 3,
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

func TestExample_ModelHelper_Alias(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} }).Alias("u")

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:     "ModelSelect Alias",
			buildSQL: func() (string, []any, error) { return h.ModelSelect(nil).ToSql() },
			wantSQL:  "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name:     "ModelSelectWhere Alias",
			buildSQL: func() (string, []any, error) { return h.ModelSelectWhere("u.id = ?", 1).ToSql() },
			wantSQL:  "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?",
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

func TestExample_ModelHelper_Options(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })
	hAlias := h.Alias("u")

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "ModelSelect OrderBy Limit",
			buildSQL: func() (string, []any, error) {
				return h.ModelSelect(nil).
					WithOptions(func(b SelectBuilder) SelectBuilder {
						return b.OrderBy("created_at DESC").Limit(10)
					}).ToSql()
			},
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` ORDER BY created_at DESC LIMIT 10",
			wantArgs: 0,
		},
		{
			name: "ModelSelect LeftJoin",
			buildSQL: func() (string, []any, error) {
				return h.ModelSelect(nil).
					WithOptions(func(b SelectBuilder) SelectBuilder {
						return b.LeftJoin("orders o ON o.user_id = users.id")
					}).ToSql()
			},
			wantSQL:  "SELECT `age`, `email`, `id`, `name` FROM `users` LEFT JOIN orders o ON o.user_id = users.id",
			wantArgs: 0,
		},
		{
			name: "ModelSelect MultiJoin",
			buildSQL: func() (string, []any, error) {
				return hAlias.ModelSelect(nil).
					WithOptions(func(b SelectBuilder) SelectBuilder {
						return b.
							Join("orders o ON o.user_id = u.id").
							LeftJoin("profiles p ON p.user_id = u.id")
					}).ToSql()
			},
			wantSQL:  "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u` JOIN orders o ON o.user_id = u.id LEFT JOIN profiles p ON p.user_id = u.id",
			wantArgs: 0,
		},
		{
			name: "ModelSelect Distinct",
			buildSQL: func() (string, []any, error) {
				return h.SelectDistinct("name", "users").ToSql()
			},
			wantSQL:  "SELECT DISTINCT(`name`) FROM `users`",
			wantArgs: 0,
		},
		{
			name: "ModelSelect DistinctWithAlias",
			buildSQL: func() (string, []any, error) {
				return hAlias.SelectDistinct("name", "users").ToSql()
			},
			wantSQL:  "SELECT DISTINCT(`u`.`name`) FROM `users` AS `u`",
			wantArgs: 0,
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

func TestExample_ModelHelper_Insert(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	users := []testUser{
		{Name: "John", Email: "john@test.com"},
		{Name: "Jane", Email: "jane@test.com"},
	}

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "ModelInsert",
			buildSQL: func() (string, []any, error) {
				return h.ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?)",
			wantArgs: 2,
		},
		{
			name: "ModelInserts",
			buildSQL: func() (string, []any, error) {
				return h.ModelInserts([]string{"name", "email"}, users).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?),(?,?)",
			wantArgs: 4,
		},
		{
			name: "ModelInsert OnDuplicate",
			buildSQL: func() (string, []any, error) {
				return h.ModelInsert([]string{"id", "name", "email"}, &testUser{
					ID: 1, Name: "John", Email: "john@test.com",
				}).OnDuplicateUpdateValues("name", "email").ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`id`,`name`,`email`) VALUES (?,?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)",
			wantArgs: 3,
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

func TestExample_ModelHelper_Update(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "ModelUpdate",
			buildSQL: func() (string, []any, error) {
				return h.ModelUpdate(&testUser{ID: 1, Name: "John"}, []string{"name"}).
					Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1",
			wantArgs: 2,
		},
		{
			name: "ModelUpdate Where",
			buildSQL: func() (string, []any, error) {
				return h.ModelUpdate(&testUser{Name: "Updated"}, []string{"name"}).
					Where("age > ?", 18).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE age > ? LIMIT 1",
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

func TestExample_ModelHelper_Columns(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	tests := []struct {
		name      string
		filter    func(string) bool
		wantCount int
		wantSQL   string
	}{
		{
			name:      "Columns all",
			filter:    nil,
			wantCount: 4,
			wantSQL:   "",
		},
		{
			name:      "Columns filtered",
			filter:    func(col string) bool { return col != "email" },
			wantCount: 3,
			wantSQL:   "SELECT `age`, `id`, `name` FROM `users`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := h.Columns(tt.filter)
			if len(cols) != tt.wantCount {
				t.Errorf("Columns len = %d, want %d", len(cols), tt.wantCount)
			}
			if tt.wantSQL != "" {
				sql, _, _ := h.ModelSelect(cols).ToSql()
				testSQL(t, tt.name+" SQL", sql, tt.wantSQL)
			}
		})
	}
}

func TestExample_ModelHelper_MappingModel(t *testing.T) {
	h := NewMappingModelHelper(func(u *testUser) (string, map[string]any) {
		return "users", map[string]any{
			"id":    &u.ID,
			"name":  &u.Name,
			"email": &u.Email,
		}
	})

	tests := []struct {
		name    string
		wantSQL string
	}{
		{
			name:    "MappingModel Select",
			wantSQL: "SELECT `email`, `id`, `name` FROM `users`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, _ := h.ModelSelect(nil).ToSql()
			testSQL(t, tt.name, sql, tt.wantSQL)
		})
	}
}

func TestExample_ModelHelper_Transaction(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "ModelInsert Tx",
			buildSQL: func() (string, []any, error) {
				return h.ModelInsert([]string{"name"}, &testUser{Name: "John"}).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`) VALUES (?)",
			wantArgs: 1,
		},
		{
			name: "ModelUpdate Tx",
			buildSQL: func() (string, []any, error) {
				return h.ModelUpdate(&testUser{ID: 1, Name: "John"}, []string{"name"}).
					Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1",
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

// ==================== Helper Tests ====================

func TestExample_Helper_Select(t *testing.T) {
	tests := []struct {
		name     string
		helper   Helper
		buildSQL func(Helper) (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name:     "Helper Select",
			helper:   Helper{},
			buildSQL: func(h Helper) (string, []any, error) { return h.Select([]string{"id", "name"}, "users").ToSql() },
			wantSQL:  "SELECT `id`, `name` FROM `users`",
			wantArgs: 0,
		},
		{
			name:   "Helper Select Where",
			helper: Helper{},
			buildSQL: func(h Helper) (string, []any, error) {
				return h.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
			},
			wantSQL:  "SELECT `id`, `name` FROM `users` WHERE age > ?",
			wantArgs: 1,
		},
		{
			name:     "Helper Select Alias",
			helper:   Helper{}.Alias("u"),
			buildSQL: func(h Helper) (string, []any, error) { return h.Select([]string{"id", "name"}, "users").ToSql() },
			wantSQL:  "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`",
			wantArgs: 0,
		},
		{
			name: "Helper Select CustomEscape",
			helper: Helper{}.WithEscapeFunc(func(key string, table bool) string {
				return "\"" + key + "\""
			}),
			buildSQL: func(h Helper) (string, []any, error) {
				return h.Select([]string{"id", "name"}, "users").ToSql()
			},
			wantSQL:  "SELECT \"id\", \"name\" FROM \"users\"",
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

func TestExample_Helper_Insert(t *testing.T) {
	h := Helper{}

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "Helper Insert",
			buildSQL: func() (string, []any, error) {
				return h.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?)",
			wantArgs: 2,
		},
		{
			name: "Helper Insert OnDuplicate",
			buildSQL: func() (string, []any, error) {
				return h.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).
					OnDuplicateUpdateValues("name", "email").ToSql()
			},
			wantSQL:  "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)",
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

func TestExample_Helper_Update(t *testing.T) {
	h := Helper{}

	tests := []struct {
		name     string
		buildSQL func() (string, []any, error)
		wantSQL  string
		wantArgs int
	}{
		{
			name: "Helper Update",
			buildSQL: func() (string, []any, error) {
				return h.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
			},
			wantSQL:  "UPDATE `users` SET `name` = ? WHERE id = ?",
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

// ==================== Query Interface Tests ====================

func TestExample_Query(t *testing.T) {
	tests := []struct {
		name     string
		helper   interface{}
		buildSQL func() (string, []any, error)
		wantSQL  string
	}{
		{
			name:   "Pagination",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).
					ModelSelect(nil).SelectExecutor().
					WithQueries(&testPageQuery{Page: 1, Limit: 10}).ToSql()
			},
			wantSQL: "SELECT `age`, `email`, `id`, `name` FROM `users` LIMIT 10 OFFSET 0",
		},
		{
			name:   "Countless Pagination",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).
					ModelSelect(nil).SelectExecutor().
					WithQueries(&testPageQuery{Page: 2, Limit: 20, SkipCnt: true}).ToSql()
			},
			wantSQL: "SELECT `age`, `email`, `id`, `name` FROM `users` LIMIT 20 OFFSET 20",
		},
		{
			name:   "Sort",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).
					ModelSelect(nil).SelectExecutor().
					WithQueries(&testSortQuery{Field: "created_at", Desc: true}).ToSql()
			},
			wantSQL: "SELECT `age`, `email`, `id`, `name` FROM `users` ORDER BY `created_at` DESC",
		},
		{
			name:   "Filter Sort",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).
					ModelSelect(nil).SelectExecutor().
					WithQueries(
						&testStatusFilterQuery{Status: "active"},
						&testSortQuery{Field: "id", Desc: true},
					).ToSql()
			},
			wantSQL: "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE status = ? ORDER BY `id` DESC",
		},
		{
			name:   "Tenant",
			helper: Helper{}.Alias("u"),
			buildSQL: func() (string, []any, error) {
				return Helper{}.Alias("u").
					Select([]string{"id", "name"}, "users").
					WithQueries(&testTenantQuery{TenantID: "tenant_123"}).ToSql()
			},
			wantSQL: "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE `u`.`tenant_id` = ?",
		},
		{
			name:   "SoftDelete",
			helper: NewModelHelper(func() testUser { return testUser{} }),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).
					ModelSelect(nil).SelectExecutor().
					WithQueries(&testNotDeletedQuery{}).ToSql()
			},
			wantSQL: "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE `deleted_at` IS NULL",
		},
		{
			name:   "Join",
			helper: Helper{}.Alias("u"),
			buildSQL: func() (string, []any, error) {
				return Helper{}.Alias("u").
					Select([]string{"id", "name"}, "users").
					WithQueries(&testWithOrderQuery{OrderStatus: 1}).ToSql()
			},
			wantSQL: "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ?",
		},
		{
			name:   "Combined",
			helper: NewModelHelper(func() testUser { return testUser{} }).Alias("u"),
			buildSQL: func() (string, []any, error) {
				return NewModelHelper(func() testUser { return testUser{} }).Alias("u").
					ModelSelect(nil).SelectExecutor().
					WithQueries(
						&testPageQuery{Page: 1, Limit: 10},
						&testTenantQuery{TenantID: "tenant_123"},
						&testStatusFilterQuery{Status: "active"},
						&testSortQuery{Field: "created_at", Desc: true},
						&testWithOrderQuery{OrderStatus: 1},
					).ToSql()
			},
			wantSQL: "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u` LEFT JOIN orders o ON o.user_id = `u`.`id` AND o.status = ? WHERE `u`.`tenant_id` = ? AND status = ? ORDER BY `u`.`created_at` DESC LIMIT 10 OFFSET 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, _, err := tt.buildSQL()
			if err != nil {
				t.Fatalf("BuildSQL error: %v", err)
			}
			testSQL(t, tt.name+" Query", sql, tt.wantSQL)
		})
	}
}
