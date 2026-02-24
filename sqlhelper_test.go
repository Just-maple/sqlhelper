package sqlhelper

import (
	"testing"
)

func TestHelper_EscapeColumn(t *testing.T) {
	h := Helper{}

	tests := []struct {
		name     string
		column   string
		expected string
	}{
		{"simple", "id", "`id`"},
		{"with alias", "name", "`name`"},
		{"already escaped", "`name`", "`name`"},
		{"with space", "user name", "user name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.EscapeColumn(tt.column)
			testSQL(t, "EscapeColumn", result, tt.expected)
		})
	}
}

func TestHelper_EscapeTable(t *testing.T) {
	h := Helper{}

	tests := []struct {
		name     string
		table    string
		expected string
	}{
		{"simple", "users", "`users`"},
		{"with alias", "orders", "`orders`"},
		{"already escaped", "`products`", "`products`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.EscapeTable(tt.table)
			testSQL(t, "EscapeTable", result, tt.expected)
		})
	}
}

func TestHelper_Alias(t *testing.T) {
	h := Helper{}.Alias("u")

	testSQL(t, "EscapeTable", h.EscapeTable("users"), "`users` AS `u`")
	testSQL(t, "EscapeColumn", h.EscapeColumn("name"), "`u`.`name`")
}

func TestHelper_Alias_SelectSQL(t *testing.T) {
	sql, args, _ := Helper{}.Alias("u").Select([]string{"id", "name"}, "users").Where("id = ?", 1).ToSql()
	testSQL(t, "Alias Select", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE id = ?")
	testArgsLen(t, "args", args, 1)
}

func TestHelper_WithEscapeFunc(t *testing.T) {
	h := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	})

	testSQL(t, "EscapeColumn", h.EscapeColumn("name"), "\"name\"")
	testSQL(t, "EscapeTable", h.EscapeTable("users"), "\"users\"")
}

func TestHelper_WithEscapeFunc_SelectSQL(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "CustomEscape Select", sql, "SELECT \"id\", \"name\" FROM \"users\"")
	testArgsLen(t, "args", args, 0)
}

func TestHelper_EscapeColumns(t *testing.T) {
	result := Helper{}.EscapeColumns([]string{"id", "name", "email"})
	testSQL(t, "EscapeColumns", result[0]+","+result[1]+","+result[2], "`id`,`name`,`email`")
}

func TestSelectExecutor_ToSql(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Select", sql, "SELECT `id`, `name` FROM `users`")
	testArgsLen(t, "args", args, 0)
}

func TestSelectExecutor_Where(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").Where("id = ?", 1).ToSql()
	testSQL(t, "Select Where", sql, "SELECT `id`, `name` FROM `users` WHERE id = ?")
	testArgsLen(t, "args", args, 1)
}

func TestSelectExecutor_Distinct(t *testing.T) {
	sql, args, _ := Helper{}.SelectDistinct("name", "users").ToSql()
	testSQL(t, "SelectDistinct", sql, "SELECT DISTINCT(`name`) FROM `users`")
	testArgsLen(t, "args", args, 0)
}

func TestSelectExecutor_Where_Alias(t *testing.T) {
	sql, args, _ := Helper{}.Alias("u").Select([]string{"id", "name"}, "users").Where("u.id = ?", 1).ToSql()
	testSQL(t, "Alias Where", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u` WHERE u.id = ?")
	testArgsLen(t, "args", args, 1)
}

func TestInsertExecutor_ToSql(t *testing.T) {
	sql, args, _ := Helper{}.Insert("users", []string{"name", "email"}, []interface{}{"John", "john@example.com"}).ToSql()
	testSQL(t, "Insert", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?)")
	testArgsLen(t, "args", args, 2)
}

func TestInsertExecutor_OnDuplicateUpdateValues(t *testing.T) {
	sql, args, _ := Helper{}.Insert("users", []string{"name", "email"}, []interface{}{"John", "john@example.com"}).
		OnDuplicateUpdateValues("name", "email").ToSql()
	testSQL(t, "Insert OnDuplicate", sql, "INSERT INTO `users` (`name`,`email`) VALUES (?,?) ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)")
	testArgsLen(t, "args", args, 2)
}

func TestInsertExecutor_ToSql_CustomEscape(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Insert("users", []string{"name", "email"}, []interface{}{"John", "john@example.com"}).ToSql()
	testSQL(t, "Insert CustomEscape", sql, "INSERT INTO \"users\" (\"name\",\"email\") VALUES (?,?)")
	testArgsLen(t, "args", args, 2)
}

func TestUpdateExecutor_ToSql(t *testing.T) {
	sql, args, _ := Helper{}.Update("users", map[string]interface{}{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update", sql, "UPDATE `users` SET `name` = ? WHERE id = ?")
	testArgsLen(t, "args", args, 2)
}

func TestUpdateExecutor_WithoutWhere(t *testing.T) {
	sql, args, _ := Helper{}.Update("users", map[string]interface{}{"name": "John"}).ToSql()
	testSQL(t, "Update WithoutWhere", sql, "UPDATE `users` SET `name` = ?")
	testArgsLen(t, "args", args, 1)
}

func TestUpdateExecutor_ToSql_CustomEscape(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "\"" + key + "\""
	}).Update("users", map[string]interface{}{"name": "John"}).Where("id = ?", 1).ToSql()
	testSQL(t, "Update CustomEscape", sql, "UPDATE \"users\" SET \"name\" = ? WHERE id = ?")
	testArgsLen(t, "args", args, 2)
}

func TestHelper_OnDuplicate(t *testing.T) {
	testSQL(t, "OnDuplicate", Helper{}.OnDuplicate("name", "email"), "ON DUPLICATE KEY UPDATE `name` = VALUES(`name`),`email` = VALUES(`email`)")
	testSQL(t, "OnDuplicate empty", Helper{}.OnDuplicate(), "")
}

func TestHelper_Alias_Chained(t *testing.T) {
	sql, args, _ := Helper{}.Alias("u").Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "Alias Chained", sql, "SELECT `u`.`id`, `u`.`name` FROM `users` AS `u`")
	testArgsLen(t, "args", args, 0)
}

func TestHelper_CustomEscape_Chained(t *testing.T) {
	sql, args, _ := Helper{}.WithEscapeFunc(func(key string, table bool) string {
		return "[" + key + "]"
	}).Select([]string{"id", "name"}, "users").ToSql()
	testSQL(t, "CustomEscape Chained", sql, "SELECT [id], [name] FROM [users]")
	testArgsLen(t, "args", args, 0)
}

func TestSelectExecutor_WithOptions(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").
		WithOptions(func(builder SelectBuilder) SelectBuilder {
			return builder.Where("id > ?", 10).OrderBy("id DESC").Limit(10)
		}).ToSql()
	testSQL(t, "Select WithOptions", sql, "SELECT `id`, `name` FROM `users` WHERE id > ? ORDER BY id DESC LIMIT 10")
	testArgsLen(t, "args", args, 1)
}

func TestSelectExecutor_Count(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
	testSQL(t, "Select Count", sql, "SELECT `id`, `name` FROM `users` WHERE age > ?")
	testArgsLen(t, "args", args, 1)
}

func TestSelectExecutor_Pagination(t *testing.T) {
	sql, args, _ := Helper{}.Select([]string{"id", "name"}, "users").Where("age > ?", 18).ToSql()
	testSQL(t, "Select Pagination", sql, "SELECT `id`, `name` FROM `users` WHERE age > ?")
	testArgsLen(t, "args", args, 1)
}
