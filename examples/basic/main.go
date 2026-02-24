package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Just-maple/sqlhelper"
	_ "github.com/go-sql-driver/mysql"
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

type PageQuery struct {
	page  int
	limit int
}

func (p *PageQuery) Option(h sqlhelper.Helper) sqlhelper.SelectBuilderOption {
	return func(builder sqlhelper.SelectBuilder) sqlhelper.SelectBuilder {
		return builder.Limit(uint64(p.limit)).Offset(uint64((p.page - 1) * p.limit))
	}
}

func (p *PageQuery) Countless() bool {
	return false
}

func main() {
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	h := sqlhelper.Helper{}
	ctx := context.Background()

	sqlStr, args, err := h.Select([]string{"id", "name", "email"}, "users").Where("age > ?", 18).ToSql()
	fmt.Println("Select SQL:", sqlStr, args)

	var user User
	err = h.Select([]string{"id", "name", "email", "age"}, "users").
		Where("id = ?", 1).
		QueryRowScanModel(ctx, db, func() sqlhelper.Model {
			return &User{}
		})
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("User: %+v\n", user)
	}

	query := &PageQuery{page: 1, limit: 10}
	models := make([]User, 0)
	total, err := h.Select(nil, "users").
		PaginationModels(ctx, db, query, func() sqlhelper.Model {
			models = append(models, User{})
			return &models[len(models)-1]
		})
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Total: %d, Users: %+v\n", total, models)
	}
}
