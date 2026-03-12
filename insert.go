package sqlhelper

import (
	"context"
	"database/sql"
	"strings"

	"github.com/Masterminds/squirrel"
)

// Insert creates a new InsertExecutor for the specified table, columns, and values.
func (h Helper) Insert(table string, columns []string, values ...[]any) InsertExecutor {
	hh := Helper{alias: "", escapeFunc: h.escapeFunc} // insert does not support alias
	builder := squirrel.Insert(hh.EscapeTable(table)).Columns(hh.EscapeColumns(columns)...)
	for _, vv := range values {
		builder = builder.Values(vv...)
	}
	return InsertExecutor{
		builder: builder,
		helper:  hh,
	}
}

// ModelInsert inserts multiple model instances into the database.
func (h ModelHelper[T, M]) ModelInsert(columns []string, models ...M) InsertExecutor {
	var mapping Mapper
	values := make([][]any, 0, len(models))
	table := ""
	for i, model := range models {
		if i == 0 {
			table = model.TableName()
			mapping = h.MapColumns(model, &columns)
		}
		values = append(values, mapping.MapValues(model, columns))
	}
	return h.Insert(table, columns, values...)
}

// ModelInserts inserts a slice of model instances into the database.
func (h ModelHelper[T, M]) ModelInserts(columns []string, models []T) InsertExecutor {
	ms := make([]M, 0, len(models))
	for i := range models {
		ms = append(ms, M(&models[i]))
	}
	return h.ModelInsert(columns, ms...)
}

// InsertExecutor handles the execution of INSERT queries.
type InsertExecutor struct {
	builder squirrel.InsertBuilder
	helper  Helper
}

// ToSql converts the query to SQL string and arguments.
func (exec InsertExecutor) ToSql() (string, []any, error) {
	return exec.builder.ToSql()
}

// WithOptions applies additional builder options to the query.
func (exec InsertExecutor) WithOptions(opts ...InsertOption) InsertExecutor {
	builder := exec.builder
	for _, opt := range opts {
		builder = opt(builder)
	}
	return InsertExecutor{builder: builder}
}

// Exec executes the insert query and returns the result.
func (exec InsertExecutor) Exec(ctx context.Context, conn Conn) (result sql.Result, err error) {
	statement, args, err := exec.builder.ToSql()
	if err != nil {
		return
	}
	return conn.ExecContext(ctx, statement, args...)
}

// ExecLastInsertId executes the insert query and returns the last insert ID.
func (exec InsertExecutor) ExecLastInsertId(ctx context.Context, conn Conn) (id int64, err error) {
	ret, err := exec.Exec(ctx, conn)
	if err != nil {
		return
	}
	return ret.LastInsertId()
}

// OnDuplicateUpdateValues adds ON DUPLICATE KEY UPDATE clause to the insert query.
func (exec InsertExecutor) OnDuplicateUpdateValues(columns ...string) InsertExecutor {
	return exec.WithOptions(func(builder InsertBuilder) InsertBuilder {
		return builder.Suffix(exec.helper.OnDuplicate(columns...))
	})
}

// OnDuplicate generates the ON DUPLICATE KEY UPDATE SQL clause.
func (h Helper) OnDuplicate(columns ...string) string {
	if len(columns) == 0 {
		return ""
	}
	str := &strings.Builder{}
	str.WriteString("ON DUPLICATE KEY UPDATE ")
	for i, field := range columns {
		if i != 0 {
			str.WriteString(",")
		}
		str.WriteString(h.EscapeColumn(field))
		str.WriteString(" = VALUES(")
		str.WriteString(h.EscapeColumn(field))
		str.WriteString(")")
	}
	return str.String()
}
