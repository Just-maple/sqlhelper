package sqlhelper

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
)

// UpdateExecutor handles the execution of UPDATE queries.
type UpdateExecutor struct {
	builder squirrel.UpdateBuilder
	helper  Helper
}

// ModelUpdate updates a single model instance in the database.
func (h ModelHelper[T, M]) ModelUpdate(model M, columns []string, opts ...UpdateOption) UpdateExecutor {
	mapping := h.MapColumns(model, &columns)
	updateMapping := make(map[string]any, len(columns))
	for _, col := range columns {
		updateMapping[col] = mapping[col]
	}
	return h.Update(model.TableName(), updateMapping, opts...).Limit(1)
}

// Update creates a new UpdateExecutor for the specified table and values.
func (h Helper) Update(table string, vv map[string]any, opts ...UpdateOption) UpdateExecutor {
	builder := squirrel.Update(h.EscapeTable(table))
	for _, opt := range opts {
		builder = opt(builder)
	}
	for k, v := range vv {
		builder = builder.Set(h.EscapeColumn(k), v)
	}
	return UpdateExecutor{
		builder: builder,
		helper:  h,
	}
}

// ToSql converts the query to SQL string and arguments.
func (exec UpdateExecutor) ToSql() (string, []any, error) {
	return exec.builder.ToSql()
}

// ExecRowsAffected executes the update query and returns the number of affected rows.
func (exec UpdateExecutor) ExecRowsAffected(ctx context.Context, conn Conn) (rows int64, err error) {
	ret, err := exec.Exec(ctx, conn)
	if err != nil {
		return
	}
	return ret.RowsAffected()
}

// Exec executes the update query and returns the result.
// Returns an error if the query does not contain a WHERE clause.
func (exec UpdateExecutor) Exec(ctx context.Context, conn Conn) (result sql.Result, err error) {
	statement, args, err := exec.builder.ToSql()
	if err != nil {
		return
	}
	if !strings.Contains(strings.ToLower(statement), "where") {
		return nil, fmt.Errorf("update without WHERE clause is not allowed")
	}
	return conn.ExecContext(ctx, statement, args...)
}

// Where adds a WHERE clause to the query.
func (exec UpdateExecutor) Where(pred any, args ...any) UpdateExecutor {
	return exec.WithOptions(func(builder UpdateBuilder) UpdateBuilder {
		return builder.Where(pred, args...)
	})
}

// Limit sets the maximum number of rows to update.
func (exec UpdateExecutor) Limit(limit uint64) UpdateExecutor {
	return exec.WithOptions(func(builder UpdateBuilder) UpdateBuilder {
		return builder.Limit(limit)
	})
}

// WithOptions applies additional builder options to the query.
func (exec UpdateExecutor) WithOptions(opts ...UpdateOption) UpdateExecutor {
	builder := exec.builder
	for _, opt := range opts {
		builder = opt(builder)
	}
	return UpdateExecutor{builder: builder, helper: exec.helper}
}

func (exec UpdateExecutor) Options() Options[UpdateBuilder] {
	return exec.helper.UpdateOptions()
}
