package sqlhelper

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"golang.org/x/sync/errgroup"
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
func (h Helper) Select(columns []string, table string, opts ...SelectBuilderOption) SelectExecutor {
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
func (exec SelectExecutor) WithOptions(opts ...SelectBuilderOption) SelectExecutor {
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

// WithQueries applies Query options (pagination, sorting, filtering) to the query.
// The Query.Option receives the Helper context for proper column escaping with alias support.
func (exec SelectExecutor) WithQueries(queries ...Query) SelectExecutor {
	options := make([]SelectBuilderOption, 0, len(queries))
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
	err = exec.WithOptions(countOption).QueryRow(ctx, conn).Scan(&total)
	return
}

// QueryTotals executes the query and optionally gets total count concurrently.
// The alloc function is called for each row to provide values for scanning.
func (exec SelectExecutor) QueryTotals(ctx context.Context, conn Conn, alloc func(columns []string) []any, total *int) (err error) {
	group := errgroup.Group{}
	group.Go(func() (err error) {
		return exec.QueryRowsScans(ctx, conn, alloc)
	})
	if total != nil {
		group.Go(func() (err error) {
			*total, err = exec.WithOptions(func(builder SelectBuilder) SelectBuilder {
				return builder.Limit(10000).Offset(0)
			}).Count(ctx, conn)
			return
		})
	}
	return group.Wait()
}

// QueryTotalsModels executes the query and optionally gets total count concurrently for Models.
func (exec SelectExecutor) QueryTotalsModels(ctx context.Context, conn Conn, alloc func() Model, total *int) (err error) {
	err = exec.QueryTotals(ctx, conn, ConvertModelMapping(alloc), total)
	return
}

// PaginationStrings returns paginated string values and total count.
func (exec SelectExecutor) PaginationStrings(ctx context.Context, conn Conn, query PaginationQuery, opts ...SelectBuilderOption) (rets []string, total int, err error) {
	total, err = exec.Pagination(ctx, conn, query, func(columns []string) []any {
		rets = append(rets, "")
		return []any{&rets[len(rets)-1]}
	}, opts...)
	return
}

// PaginationMaps returns paginated map results and total count.
func (exec SelectExecutor) PaginationMaps(ctx context.Context, conn Conn, query PaginationQuery, opts ...SelectBuilderOption) (rets []map[string]any, total int, err error) {
	total, err = exec.Pagination(ctx, conn, query, func(columns []string) []any {
		ret := make(map[string]any, len(columns))
		rets = append(rets, ret)
		values := make([]any, 0, len(columns))
		for _, column := range columns {
			ret[column] = new(any)
			values = append(values, ret[column])
		}
		return values
	}, opts...)
	return
}

// Pagination executes a paginated query and returns total count.
// The alloc function is called for each row to provide values for scanning.
func (exec SelectExecutor) Pagination(ctx context.Context, conn Conn, query PaginationQuery, alloc func(columns []string) []any, opts ...SelectBuilderOption) (total int, err error) {
	var ptr *int
	count := 0
	if !query.Countless() {
		ptr = &total
	}
	if err = exec.WithQueries(query).WithOptions(opts...).QueryTotals(ctx, conn, func(columns []string) []any {
		count++
		return alloc(columns)
	}, ptr); err != nil {
		return
	}
	if total < count {
		total = count
	}
	return
}

// PaginationModels returns paginated Model instances and total count.
func (exec SelectExecutor) PaginationModels(ctx context.Context, conn Conn, query PaginationQuery, alloc func() Model, opts ...SelectBuilderOption) (total int, err error) {
	return exec.Pagination(ctx, conn, query, ConvertModelMapping(alloc), opts...)
}

// Alias sets a table alias for the ModelHelper.
func (h ModelHelper[M, T]) Alias(alias string) ModelHelper[M, T] {
	return ModelHelper[M, T]{
		Helper:    h.Helper.Alias(alias),
		allocFunc: h.allocFunc,
	}
}

// Columns returns the column names from the model, optionally filtered by a function.
func (h ModelHelper[M, T]) Columns(filter func(string) bool) (columns []string) {
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
func (h ModelHelper[M, T]) ModelSelect(columns []string, opts ...SelectBuilderOption) ModelSelectExecutor[M, T] {
	t := h.alloc()
	model := M(&t)
	h.MapColumns(model, &columns)
	exec := h.Select(columns, model.TableName(), opts...)
	return ModelSelectExecutor[M, T]{
		exec:  exec,
		alloc: h.alloc,
	}
}

// ModelSelectWhere creates a ModelSelectExecutor with an initial WHERE clause.
func (h ModelHelper[M, T]) ModelSelectWhere(pred any, args ...any) ModelSelectExecutor[M, T] {
	return h.ModelSelect(nil).Where(pred, args...)
}

// ModelPagination performs a paginated query on the model.
func (h ModelHelper[M, T]) ModelPagination(ctx context.Context, conn Conn, query PaginationQuery, opts ...SelectBuilderOption) (models []T, total int, err error) {
	return h.ModelSelect(nil).Pagination(ctx, conn, query, opts...)
}

// ModelPaginationWhere performs a paginated query with WHERE clause on the model.
func (h ModelHelper[M, T]) ModelPaginationWhere(ctx context.Context, conn Conn, query PaginationQuery, pred any, args ...any) (models []T, total int, err error) {
	return h.ModelSelectWhere(pred, args...).Pagination(ctx, conn, query)
}

// ModelDistinctPagination performs a paginated DISTINCT query on a single column.
func (h ModelHelper[M, T]) ModelDistinctPagination(ctx context.Context, conn Conn, query PaginationQuery, column string, opts ...SelectBuilderOption) (vals []string, total int, err error) {
	model := h.alloc()
	return h.SelectDistinct(column, M(&model).TableName()).PaginationStrings(ctx, conn, query, opts...)
}

// ModelSelectExecutor is a type-safe executor for model-based SELECT queries.
type ModelSelectExecutor[M modelStruct[T], T any] struct {
	exec  SelectExecutor
	alloc func() T
}

// SelectExecutor returns the underlying SelectExecutor.
func (exec ModelSelectExecutor[M, T]) SelectExecutor() SelectExecutor {
	return exec.exec
}

// One returns a single model instance.
func (exec ModelSelectExecutor[M, T]) One(ctx context.Context, conn Conn) (model T, err error) {
	err = exec.exec.QueryRowScanModel(ctx, conn, func() Model {
		model = exec.alloc()
		return M(&model)
	})
	return
}

// List returns all model instances matching the query.
func (exec ModelSelectExecutor[M, T]) List(ctx context.Context, conn Conn) (models []T, err error) {
	err = exec.exec.QueryRowsScansModels(ctx, conn, func() Model {
		models = append(models, exec.alloc())
		return M(&models[len(models)-1])
	})
	return
}

// ToSql converts the query to SQL string and arguments.
func (exec ModelSelectExecutor[M, T]) ToSql() (string, []any, error) {
	return exec.SelectExecutor().ToSql()
}

// Pagination returns paginated model instances and total count.
func (exec ModelSelectExecutor[M, T]) Pagination(ctx context.Context, conn Conn, query PaginationQuery, opts ...SelectBuilderOption) (models []T, total int, err error) {
	total, err = exec.exec.PaginationModels(ctx, conn, query, func() Model {
		models = append(models, exec.alloc())
		return M(&models[len(models)-1])
	}, opts...)
	return
}

// Where adds a WHERE clause to the query.
func (exec ModelSelectExecutor[M, T]) Where(pred any, args ...any) ModelSelectExecutor[M, T] {
	return exec.WithOptions(func(builder SelectBuilder) SelectBuilder {
		return builder.Where(pred, args...)
	})
}

// WithOptions applies additional builder options to the query.
func (exec ModelSelectExecutor[M, T]) WithOptions(opts ...SelectBuilderOption) ModelSelectExecutor[M, T] {
	return ModelSelectExecutor[M, T]{
		exec:  exec.exec.WithOptions(opts...),
		alloc: exec.alloc,
	}
}
