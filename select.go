package sqlhelper

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
)

// Alias sets a table alias for the Helper.
func (h Helper) Alias(alias string) Helper {
	return Helper{alias: alias}
}

// SelectDistinct creates a SELECT DISTINCT query for the specified column.
func (h Helper) SelectDistinct(column, table string) SelectExecutor {
	return h.Select([]string{fmt.Sprintf("DISTINCT(%s)", h.EscapeColumn(column))}, table)
}

// Select creates a new SelectExecutor for the specified columns and table.
// Optional builder options can be provided to customize the query.
func (h Helper) Select(columns []string, table string, opts ...SelectOption) SelectExecutor {
	builder := squirrel.Select(h.EscapeColumns(columns)...).From(h.EscapeTable(table))
	for _, opt := range opts {
		builder = opt(builder)
	}
	return SelectExecutor{
		helper:  h,
		builder: builder,
		columns: columns,
	}
}

// SelectExecutor handles the execution of SELECT queries.
type SelectExecutor struct {
	helper  Helper
	builder SelectBuilder
	columns []string
}

// ToSql converts the query to SQL string and arguments.
func (exec SelectExecutor) ToSql() (string, []any, error) {
	return exec.builder.ToSql()
}

// Where adds a WHERE clause to the query.
func (exec SelectExecutor) Where(pred any, args ...any) SelectExecutor {
	return exec.WithOptions(func(builder SelectBuilder) SelectBuilder {
		return builder.Where(pred, args...)
	})
}

// WithOptions applies additional builder options to the query.
func (exec SelectExecutor) WithOptions(opts ...SelectOption) SelectExecutor {
	builder := exec.builder
	for _, opt := range opts {
		builder = opt(builder)
	}
	return SelectExecutor{
		helper:  exec.helper,
		columns: exec.columns,
		builder: builder,
	}
}

func (exec SelectExecutor) Options() Options[SelectBuilder] {
	return exec.helper.SelectOptions()
}

// WithQueries applies Query options (pagination, sorting, filtering) to the query.
// The Query.Option receives the Helper context for proper column escaping with alias support.
func (exec SelectExecutor) WithQueries(queries ...Query) SelectExecutor {
	options := make([]SelectOption, 0, len(queries))
	for _, q := range queries {
		options = append(options, q.Option(exec.helper))
	}
	return exec.WithOptions(options...)
}

// QueryRow executes the query and returns a single row.
func (exec SelectExecutor) QueryRow(ctx context.Context, conn Conn) *sql.Row {
	statement, args := exec.builder.MustSql()
	return conn.QueryRowContext(ctx, statement, args...)
}

// QueryRows executes the query and returns multiple rows.
func (exec SelectExecutor) QueryRows(ctx context.Context, conn Conn) (*sql.Rows, error) {
	statement, args, err := exec.builder.ToSql()
	if err != nil {
		return nil, err
	}
	return conn.QueryContext(ctx, statement, args...)
}

// Scan scans values from a RowScanner into the provided slice.
func (exec SelectExecutor) Scan(rows squirrel.RowScanner, values func(columns []string) []any) (err error) {
	return rows.Scan(values(exec.columns)...)
}

// ScanRows iterates through rows and scans each one using the provided values function.
func (exec SelectExecutor) ScanRows(rows *sql.Rows, values func(columns []string) []any) (err error) {
	for rows.Next() && err == nil {
		err = exec.Scan(rows, values)
	}
	return
}

// ScanModel scans a single row into a Model instance.
func (exec SelectExecutor) ScanModel(rows squirrel.RowScanner, alloc func() Model) (err error) {
	return exec.Scan(rows, ConvertModelMapping(alloc))
}

// ScanModels iterates through rows and scans each into a Model instance.
func (exec SelectExecutor) ScanModels(rows *sql.Rows, alloc func() Model) (err error) {
	return exec.ScanRows(rows, ConvertModelMapping(alloc))
}

// QueryStrings executes the query and returns string values from a single column.
func (exec SelectExecutor) QueryStrings(ctx context.Context, conn Conn) (rets []string, err error) {
	err = exec.QueryRowsScans(ctx, conn, func(columns []string) []any {
		rets = append(rets, "")
		return []any{&rets[len(rets)-1]}
	})
	return
}

// QueryRowScan executes the query and scans a single row using the provided function.
func (exec SelectExecutor) QueryRowScan(ctx context.Context, conn Conn, alloc func(columns []string) []any) (err error) {
	err = exec.Scan(exec.QueryRow(ctx, conn), alloc)
	return
}

// QueryRowScanModel executes the query and scans a single row into a Model.
func (exec SelectExecutor) QueryRowScanModel(ctx context.Context, conn Conn, alloc func() Model) (err error) {
	return exec.QueryRowScan(ctx, conn, ConvertModelMapping(alloc))
}

// QueryRowsScans executes the query and scans all rows using the provided function.
func (exec SelectExecutor) QueryRowsScans(ctx context.Context, conn Conn, alloc func(columns []string) []any) (err error) {
	rows, err := exec.QueryRows(ctx, conn)
	if err != nil {
		return
	}
	defer rows.Close()
	return exec.ScanRows(rows, alloc)
}

// QueryRowsScansModels executes the query and scans all rows into Model instances.
func (exec SelectExecutor) QueryRowsScansModels(ctx context.Context, conn Conn, alloc func() Model) (err error) {
	return exec.QueryRowsScans(ctx, conn, ConvertModelMapping(alloc))
}

// Count returns the total number of rows matching the query.
func (exec SelectExecutor) Count(ctx context.Context, conn Conn) (total int, err error) {
	err = exec.WithOptions(CountOption).QueryRow(ctx, conn).Scan(&total)
	return
}

// Alias sets a table alias for the ModelHelper.
func (h ModelHelper[T, M]) Alias(alias string) ModelHelper[T, M] {
	return ModelHelper[T, M]{
		Helper:    h.Helper.Alias(alias),
		allocFunc: h.allocFunc,
	}
}

// Columns returns the column names from the model, optionally filtered by a function.
func (h ModelHelper[T, M]) Columns(filter func(string) bool) (columns []string) {
	t := h.alloc()
	if h.MapColumns(M(&t), &columns); filter != nil {
		valid := 0
		for _, column := range columns {
			if filter(column) {
				columns[valid] = column
				valid++
			}
		}
		columns = columns[:valid]
	}
	return
}

// ModelSelect creates a new ModelSelectExecutor for the model.
func (h ModelHelper[T, M]) ModelSelect(columns []string, opts ...SelectOption) ModelSelectExecutor[T, M] {
	t := h.alloc()
	model := M(&t)
	h.MapColumns(model, &columns)
	exec := h.Select(columns, model.TableName(), opts...)
	return ModelSelectExecutor[T, M]{
		exec:  exec,
		alloc: h.alloc,
	}
}

// ModelSelectWhere creates a ModelSelectExecutor with an initial WHERE clause.
func (h ModelHelper[T, M]) ModelSelectWhere(pred any, args ...any) ModelSelectExecutor[T, M] {
	return h.ModelSelect(nil).Where(pred, args...)
}

// ModelSelectExecutor is a type-safe executor for model-based SELECT queries.
type ModelSelectExecutor[T any, M ModelPointer[T]] struct {
	exec  SelectExecutor
	alloc func() T
}

// SelectExecutor returns the underlying SelectExecutor.
func (exec ModelSelectExecutor[T, M]) SelectExecutor() SelectExecutor {
	return exec.exec
}

// One returns a single model instance.
func (exec ModelSelectExecutor[T, M]) One(ctx context.Context, conn Conn) (model T, err error) {
	err = exec.exec.QueryRowScanModel(ctx, conn, func() Model {
		model = exec.alloc()
		return M(&model)
	})
	return
}

// List returns all model instances matching the query.
func (exec ModelSelectExecutor[T, M]) List(ctx context.Context, conn Conn) (models []T, err error) {
	err = exec.exec.QueryRowsScansModels(ctx, conn, func() Model {
		models = append(models, exec.alloc())
		return M(&models[len(models)-1])
	})
	return
}

// ToSql converts the query to SQL string and arguments.
func (exec ModelSelectExecutor[T, M]) ToSql() (string, []any, error) {
	return exec.SelectExecutor().ToSql()
}

// Where adds a WHERE clause to the query.
func (exec ModelSelectExecutor[T, M]) Where(pred any, args ...any) ModelSelectExecutor[T, M] {
	return exec.WithOptions(func(builder SelectBuilder) SelectBuilder {
		return builder.Where(pred, args...)
	})
}

// WithOptions applies additional builder options to the query.
func (exec ModelSelectExecutor[T, M]) WithOptions(opts ...SelectOption) ModelSelectExecutor[T, M] {
	return ModelSelectExecutor[T, M]{
		exec:  exec.exec.WithOptions(opts...),
		alloc: exec.alloc,
	}
}

// WithOptions applies additional builder options to the query.
func (exec ModelSelectExecutor[T, M]) WithQueries(queries ...Query) ModelSelectExecutor[T, M] {
	return ModelSelectExecutor[T, M]{
		exec:  exec.exec.WithQueries(queries...),
		alloc: exec.alloc,
	}
}
