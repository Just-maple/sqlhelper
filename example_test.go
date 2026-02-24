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

func (u *testUser) TableName() string {
	return "users"
}

func (u *testUser) FieldMapping(dst map[string]interface{}) {
	dst["id"] = &u.ID
	dst["name"] = &u.Name
	dst["email"] = &u.Email
	dst["age"] = &u.Age
}

func TestExample_ModelHelper_Select(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect(nil).ToSql()
	testSQL(t, "ModelSelect", sql, "SELECT `age`, `email`, `id`, `name` FROM `users`")
}

func TestExample_ModelHelper_Select_WithColumns(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect([]string{"id", "name"}).ToSql()
	testSQL(t, "ModelSelect columns", sql, "SELECT `id`, `name` FROM `users`")
}

func TestExample_ModelHelper_Where(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelect(nil).Where("age > ? AND name = ?", 18, "John").ToSql()
	testSQL(t, "ModelSelect Where", sql, "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE age > ? AND name = ?")
	testArgsLen(t, "args", args, 2)
}

func TestExample_ModelHelper_One(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelSelectWhere("id = ?", 1).ToSql()
	testSQL(t, "ModelSelectWhere", sql, "SELECT `age`, `email`, `id`, `name` FROM `users` WHERE id = ?")
	testArgsLen(t, "args", args, 1)
}

func TestExample_ModelHelper_Insert(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelInsert([]string{"name", "email", "age"}, &testUser{
		Name:  "John",
		Email: "john@example.com",
		Age:   25,
	}).ToSql()
	testSQL(t, "ModelInsert", sql, "INSERT INTO `users` (`name`,`email`,`age`) VALUES (?,?,?)")
	testArgsLen(t, "args", args, 3)
}

func TestExample_ModelHelper_InsertMultiple(t *testing.T) {
	users := []testUser{
		{Name: "John", Email: "john@example.com", Age: 25},
		{Name: "Jane", Email: "jane@example.com", Age: 30},
	}
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelInserts([]string{"name", "email", "age"}, users).ToSql()
	testSQL(t, "ModelInserts", sql, "INSERT INTO `users` (`name`,`email`,`age`) VALUES (?,?,?),(?,?,?)")
	testArgsLen(t, "args", args, 6)
}

func TestExample_ModelHelper_Insert_OnDuplicate(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelInsert([]string{"name", "email", "age"}, &testUser{
		Name:  "John",
		Email: "john@example.com",
		Age:   26,
	}).OnDuplicateUpdateValues("name", "age").ToSql()
	testSQL(t, "ModelInsert OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`,`age`) VALUES (?,?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`age` = VALUES(`age`)")
	testArgsLen(t, "args", args, 3)
}

func TestExample_ModelHelper_Update(t *testing.T) {
	sql, args, _ := NewModelHelper(func() testUser { return testUser{} }).ModelUpdate(&testUser{
		ID:    1,
		Name:  "John Updated",
		Email: "updated@example.com",
		Age:   26,
	}, []string{"name", "email", "age"}).Where("id = ?", 1).ToSql()
	testSQL(t, "ModelUpdate", sql, "UPDATE `users` SET `name` = ?, `email` = ?, `age` = ? WHERE id = ? LIMIT 1")
	testArgsLen(t, "args", args, 4)
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestExample_ModelHelper_Distinct(t *testing.T) {
	sql, _, _ := NewModelHelper(func() testUser { return testUser{} }).SelectDistinct("name", "users").ToSql()
	testSQL(t, "SelectDistinct", sql, "SELECT DISTINCT(`name`) FROM `users`")
}

func TestExample_Helper_Select(t *testing.T) {
	sql, _, _ := Helper{}.Select([]string{"id", "name", "email"}, "users").ToSql()
	testSQL(t, "Select", sql, "SELECT `id`, `name`, `email` FROM `users`")
}

func TestExample_Helper_Select_Where(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").Where("age > ? AND status = ?", 18, "active").ToSql()
	testSQL(t, "Select Where", sql, "SELECT `id`, `name` FROM `users` WHERE age > ? AND status = ?")
	testArgsLen(t, "args", args, 2)
}

func TestExample_Helper_Insert(t *testing.T) {
	sql, args, _ := Helper{}.Insert("users", []string{"name", "email", "age"}, []interface{}{"John", "john@example.com", 25}).ToSql()
	testSQL(t, "Insert", sql, "INSERT INTO `users` (`name`,`email`,`age`) VALUES (?,?,?)")
	testArgsLen(t, "args", args, 3)
}

func TestExample_Helper_Update(t *testing.T) {
	sql, args, _ := Helper{}.Update("users", map[string]interface{}{
		"name":  "John Updated",
		"email": "updated@example.com",
	}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update", sql, "UPDATE `users` SET `name` = ?, `email` = ? WHERE id = ?")
	testArgsLen(t, "args", args, 3)
}

func TestExample_Helper_Alias(t *testing.T) {
	sql, _, _ := Helper{}.Alias("u").Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select Alias", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`")
}

func TestExample_Helper_CustomEscape(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select CustomEscape", sql, "SELECT \"id\", \"name\" FROM \"users\"")
	testArgsLen(t, "args", args, 0)
}

func TestExample_Helper_Insert_CustomEscape(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Insert("users", []string{"name", "email"}, []interface{}{"John", "john@example.com"}).ToSql()
	testSQL(t, "Insert CustomEscape", sql, "INSERT INTO \"users\" (\"name\",\"email\") VALUES (?,?)")
	testArgsLen(t, "args", args, 2)
}

func TestExample_Helper_Update_CustomEscape(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Update("users", map[string]interface{}{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update CustomEscape", sql, "UPDATE \"users\" SET \"name\" = ? WHERE id = ?")
	testArgsLen(t, "args", args, 2)
}

func TestExample_SelectExecutor_Options(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").
		WithOptions(func(builder SelectBuilder) SelectBuilder {
			return builder.Where("age > ?", 18).OrderBy("id DESC").Limit(10)
		}).ToSql()
	testSQL(t, "Select Options", sql, "SELECT `id`, `name` FROM `users` WHERE age > ? ORDER BY id DESC LIMIT 10")
	testArgsLen(t, "args", args, 1)
}

func TestExample_Columns_Filter(t *testing.T) {
	columns := NewModelHelper(func() testUser { return testUser{} }).Columns(func(col string) bool {
		return col != "email"
	})

	if len(columns) != 3 {
		t.Errorf("Columns length mismatch: got %d, want 3", len(columns))
	}
}

func TestExample_Alias_Where(t *testing.T) {
	sql, args, _ := Helper{}.Alias("u").Select([]string{"id", "name"}, "users").Where("u.id = ?", 1).ToSql()
	testSQL(t, "Select Alias Where", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?")
	testArgsLen(t, "args", args, 1)
}

func TestExample_Insert_EmptyValues(t *testing.T) {
	sql, _, _ := Helper{}.Insert("users", []string{"name"}, []interface{}{}).ToSql()
	testSQL(t, "Insert EmptyValues", sql, "INSERT INTO `users` (`name`) VALUES ()")
}

func TestExample_Helper_ChainedEscapeAndAlias(t *testing.T) {
	sql, args, _ := Helper{}.Alias("u").WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select Chained", sql, "SELECT \"u\".\"id\", \"u\".\"name\" FROM \"users\" AS \"u\"")
	testArgsLen(t, "args", args, 0)
}
