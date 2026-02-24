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

func TestExample_ModelHelper(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	sql, _, _ := h.ModelSelect(nil).ToSql()
	testSQL(t, "ModelSelect", sql, "SELECT `age`, `email`, `id`, `name` FROM `users`")

	sql, _, _ = h.ModelSelect([]string{"id", "name"}).ToSql()
	testSQL(t, "ModelSelect columns", sql, "SELECT `id`, `name` FROM `users`")

	sql, args, _ := h.ModelSelect(nil).Where("age > ?", 18).ToSql()
	testSQL(t, "ModelSelect Where", sql, "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ?")
	testArgsLen(t, "ModelSelect Where args", args, 1)

	sql, args, _ = h.ModelSelectWhere("id = ?", 1).ToSql()
	testSQL(t, "ModelSelectWhere", sql, "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id = ?")
	testArgsLen(t, "ModelSelectWhere args", args, 1)

	sql, _, _ = h.Alias("u").ModelSelect(nil).ToSql()
	testSQL(t, "ModelSelect Alias", sql, "SELECT `u`.`age`, `u`.`email`, `u`.`id`, `u`.`name` FROM `users` AS `u`")
}

func TestExample_ModelHelper_Insert(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	sql, args, _ := h.ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).ToSql()
	testSQL(t, "ModelInsert", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?)")
	testArgsLen(t, "ModelInsert args", args, 2)

	users := []testUser{
		{Name: "John", Email: "john@test.com"},
		{Name: "Jane", Email: "jane@test.com"},
	}
	sql, args, _ = h.ModelInserts([]string{"name", "email"}, users).ToSql()
	testSQL(t, "ModelInserts", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?),(?,?)")
	testArgsLen(t, "ModelInserts args", args, 4)

	sql, _, _ = h.ModelInsert([]string{"name", "email"}, &testUser{Name: "John", Email: "john@test.com"}).
		OnDuplicateUpdateValues("name").ToSql()
	testSQL(t, "ModelInsert OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`)")
}

func TestExample_ModelHelper_Update(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	sql, args, _ := h.ModelUpdate(&testUser{ID: 1, Name: "John"}, []string{"name"}).Where("id = ?", 1).ToSql()
	testSQL(t, "ModelUpdate", sql, "UPDATE `users` SET `name` = ? WHERE id = ? LIMIT 1")
	testArgsLen(t, "ModelUpdate args", args, 2)
}

func TestExample_ModelHelper_Columns(t *testing.T) {
	h := NewModelHelper(func() testUser { return testUser{} })

	allCols := h.Columns(nil)
	if len(allCols) != 4 {
		t.Errorf("Columns len = %d, want 4", len(allCols))
	}

	filtered := h.Columns(func(col string) bool { return col != "email" })
	if len(filtered) != 3 {
		t.Errorf("Filtered columns len = %d, want 3", len(filtered))
	}
}

func TestExample_Helper_Select(t *testing.T) {
	h := Helper{}

	sql, _, _ := h.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select", sql, "SELECT `id`, `name` FROM `users`")

	sql, args, _ := h.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
	testSQL(t, "Select Where", sql, "SELECT `id`, `name` FROM `users` WHERE age > ?")
	testArgsLen(t, "Select Where args", args, 1)

	sql, _, _ = h.SelectDistinct("name", "users").ToSql()
	testSQL(t, "SelectDistinct", sql, "SELECT DISTINCT(`name`) FROM `users`")

	sql, args, _ = h.Select([]string{"id", "name"}, "users").
		WithOptions(func(b SelectBuilder) SelectBuilder {
			return b.Where("age > ?", 18).OrderBy("id DESC").Limit(10)
		}).ToSql()
	testSQL(t, "Select WithOptions", sql, "SELECT `id`, `name` FROM `users` WHERE age > ? ORDER BY id DESC LIMIT 10")
	testArgsLen(t, "Select WithOptions args", args, 1)
}

func TestExample_Helper_Alias(t *testing.T) {
	h := Helper{}.Alias("u")

	sql, _, _ := h.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Alias Select", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`")

	sql, args, _ := h.Select([]string{"id", "name"}, "users").Where("u.id = ?", 1).ToSql()
	testSQL(t, "Alias Where", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?")
	testArgsLen(t, "Alias Where args", args, 1)
}

func TestExample_Helper_CustomEscape(t *testing.T) {
	h := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	})

	sql, _, _ := h.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "CustomEscape Select", sql, "SELECT \"id\", \"name\" FROM \"users\"")

	sql, _, _ = h.Insert("users", []string{"name"}, []any{"John"}).ToSql()
	testSQL(t, "CustomEscape Insert", sql, "INSERT INTO \"users\" (\"name\") VALUES (?)")

	sql, _, _ = h.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "CustomEscape Update", sql, "UPDATE \"users\" SET \"name\" = ? WHERE id = ?")

	h2 := Helper{}.Alias("u").WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	})
	sql, _, _ = h2.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "CustomEscape Alias", sql, "SELECT \"u\".\"id\", \"u\".\"name\" FROM \"users\" AS \"u\"")
}

func TestExample_Helper_Insert(t *testing.T) {
	h := Helper{}

	sql, args, _ := h.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).ToSql()
	testSQL(t, "Insert", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?)")
	testArgsLen(t, "Insert args", args, 2)

	sql, _, _ = h.Insert("users", []string{"name", "email"}, []any{"John", "john@test.com"}).
		OnDuplicateUpdateValues("name", "email").ToSql()
	testSQL(t, "Insert OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)")
}

func TestExample_Helper_Update(t *testing.T) {
	h := Helper{}

	sql, args, _ := h.Update("users", map[string]any{"name": "John"}).Where("id = ?", 1).ToSql()
	if !containsAll(sql, "UPDATE", "`users`", "SET", "`name`", "WHERE") {
		t.Errorf("Update SQL mismatch: got %s", sql)
	}
	testArgsLen(t, "Update args", args, 2)
}
