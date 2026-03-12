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
	ChainableBuilder[UpdateExecutor, *UpdateExecutor, UpdateBuilder]
	helper Helper
}

func (exec UpdateExecutor) Copy() *UpdateExecutor { return &exec }

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
	for k := range vv {
		builder = builder.Set(h.EscapeColumn(k), vv[k])
	}
	return WithChain(&UpdateExecutor{helper: h}, builder, opts...)
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
	statement, args, err := exec.ToSql()
	if err != nil {
		return
	}
	if !strings.Contains(strings.ToLower(statement), "where") {
		return nil, fmt.Errorf("update without WHERE clause is not allowed")
	}
	return conn.ExecContext(ctx, statement, args...)
}
