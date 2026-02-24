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

	userHelper := sqlhelper.NewModelHelper(func() User { return User{} })

	ctx := context.Background()

	users, total, err := userHelper.ModelPagination(ctx, db, &PageQuery{page: 1, limit: 10})
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Total: %d, Users: %+v\n", total, users)
	}

	user, err := userHelper.ModelSelectWhere("id = ?", 1).One(ctx, db)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Single User: %+v\n", user)
	}
}
