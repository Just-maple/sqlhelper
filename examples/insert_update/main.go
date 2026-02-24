package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Just-maple/sqlhelper"
)

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

func main() {
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	h := sqlhelper.Helper{}
	ctx := context.Background()

	result, err := h.Insert("users", []string{"name", "email", "age"}, []any{"John", "john@example.com", 25}).Exec(ctx, db)
	if err != nil {
		fmt.Println("Insert Error:", err)
	} else {
		id, _ := result.LastInsertId()
		fmt.Println("Inserted ID:", id)
	}

	result, err = h.Insert("users", []string{"name", "email", "age"}, []any{"John", "john@example.com", 26}).
		OnDuplicateUpdateValues("name", "age").
		Exec(ctx, db)
	if err != nil {
		fmt.Println("Insert OnDuplicate Error:", err)
	} else {
		rows, _ := result.RowsAffected()
		fmt.Println("Rows Affected:", rows)
	}

	rowsAffected, err := h.Update("users", map[string]any{
		"name": "John Updated",
		"age":  26,
	}).Where("id = ?", 1).ExecRowsAffected(ctx, db)
	if err != nil {
		fmt.Println("Update Error:", err)
	} else {
		fmt.Println("Rows Affected:", rowsAffected)
	}

	userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

	result, err = userHelper.ModelInsert([]string{"name", "email", "age"}, &User{Name: "Jane", Email: "jane@example.com", Age: 30}).Exec(ctx, db)
	if err != nil {
		fmt.Println("ModelInsert Error:", err)
	} else {
		id, _ := result.LastInsertId()
		fmt.Println("ModelInserted ID:", id)
	}

	rowsAffected, err = userHelper.ModelUpdate(&User{ID: 1, Name: "Updated", Email: "updated@example.com", Age: 31}, []string{"name", "email", "age"}).ExecRowsAffected(ctx, db)
	if err != nil {
		fmt.Println("ModelUpdate Error:", err)
	} else {
		fmt.Println("ModelUpdate Rows Affected:", rowsAffected)
	}
}
