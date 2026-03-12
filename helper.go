package sqlhelper

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
)

// Query defines the interface for SQL clause injection.
// Implement this interface to define custom WHERE/SORT/JOIN logic.
type Query interface {
	// Option returns a SelectOption that applies SQL clauses (e.g., WHERE/SORT/JOIN)
	Option(helper Helper) SelectOption
}

// ModelPointer defines the constraint for model types.
// T must be a pointer to a struct that implements Model.
type ModelPointer[T any] interface {
	*T
	Model
}

// ModelHelper is a generic helper for model-based database operations.
// M is the helper type itself (for method chaining), T is the model struct type.
type ModelHelper[T any, M ModelPointer[T]] struct {
	Helper

	// allocFunc is the function to allocate a new instance of T
	allocFunc func() T
}

// NewModelHelper creates a new ModelHelper with the given allocation function.
// alloc is a function that returns a new instance of type T.
func NewModelHelper[T any, M ModelPointer[T]](alloc func() T) ModelHelper[T, M] {
	return ModelHelper[T, M]{allocFunc: alloc}
}

// Type aliases for squirrel builders
type (
	SelectBuilder = squirrel.SelectBuilder
	UpdateBuilder = squirrel.UpdateBuilder
	InsertBuilder = squirrel.InsertBuilder
)

// Type aliases for builder options
type (
	SelectOption = func(SelectBuilder) SelectBuilder
	UpdateOption = func(UpdateBuilder) UpdateBuilder
	InsertOption = func(InsertBuilder) InsertBuilder
)

// alloc allocates a new instance of T using the allocFunc.
// If allocFunc is nil, returns the zero value of T.
func (h ModelHelper[T, M]) alloc() (t T) {
	if h.allocFunc != nil {
		return h.allocFunc()
	}
	return
}

// EscapeFunc defines a function type for custom escape logic.
// The function receives a key (table/column name) and a bool indicating if it's a table name.
// Returns the escaped string.
type EscapeFunc func(key string, table bool) string

// Helper provides basic SQL building utilities with escaping.
type Helper struct {
	alias      string
	escapeFunc EscapeFunc

	Option struct {
		Select Option[SelectBuilder]
		Update Option[UpdateBuilder]
	}

	Options struct {
		Select Options[SelectBuilder]
		Update Options[UpdateBuilder]
	}
}

// EscapeColumns escapes multiple column names for safe SQL usage.
func (h Helper) EscapeColumns(columns []string) (escaped []string) {
	escaped = make([]string, len(columns))
	for j, column := range columns {
		escaped[j] = h.EscapeColumn(column)
	}
	return escaped
}

// MapColumns creates a Mapper from a Model and populates columns.
func (h Helper) MapColumns(model Model, columns *[]string) Mapper {
	mapping := make(Mapper)
	model.FieldMapping(mapping)
	mapping.MapColumns(columns)
	return mapping
}

// CountOption is a SelectOption that converts a select query to a count query
var CountOption = func(builder SelectBuilder) SelectBuilder {
	return squirrel.Select("COUNT(1)").FromSelect(builder, "t")
}

// EscapeTable escapes a table name for safe SQL usage.
// If an alias is set, returns "table AS alias" format.
func (h Helper) EscapeTable(table string) string {
	r := h.escape(table, true)
	if h.alias != "" {
		return r + " AS " + h.escape(h.alias, true)
	}
	return r
}

// EscapeColumn escapes a column name for safe SQL usage.
// If an alias is set, returns "alias.column" format.
func (h Helper) EscapeColumn(column string) string {
	r := h.escape(column, false)
	if h.alias != "" && !strings.ContainsAny(r, "()") {
		r = h.escape(h.alias, true) + "." + r
	}
	return r
}

// escaped checks if a key already contains escape characters (only for default backtick escaping)
func (h Helper) escaped(key string) bool {
	return strings.ContainsAny(key, "`.() ")
}

// escape performs the actual escaping.
// If custom escapeFunc is set, use it directly (skip escaped check).
// Otherwise, use default backtick escaping with escaped check.
func (h Helper) escape(key string, isTable bool) string {
	if h.escapeFunc != nil {
		return h.escapeFunc(key, isTable)
	}
	if h.escaped(key) {
		return key
	}
	return fmt.Sprintf("`%s`", key)
}

// WithEscapeFunc returns a new Helper with custom escape function.
func (h Helper) WithEscapeFunc(fn EscapeFunc) Helper {
	h.escapeFunc = fn
	return h
}

type ChainableBuilder[Executor Copier[*Executor], Ptr interface {
	*Executor
	set(Copier[*Executor], Builder)
}, Builder ChainBuilder[Builder]] struct {
	ChainMeta[Executor, Ptr, Builder]
	Options Options[Builder]
}

func (w ChainableBuilder[Executor, _, Builder]) ToSql() (sql string, args []interface{}, err error) {
	return w.builder.ToSql()
}

func (w ChainableBuilder[Executor, _, Builder]) Where(pred any, args ...any) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.Where(pred, args...)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) Offset(v uint64) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.Offset(v)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) Limit(v uint64) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.Limit(v)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) Prefix(v string, args ...any) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.Prefix(v, args...)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) Suffix(v string, args ...any) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.Suffix(v, args...)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) From(v string) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.From(v)
	})
}

func (w ChainableBuilder[Executor, _, Builder]) FromSelect(expr SelectBuilder, alias string) Executor {
	return w.WithOptions(func(builder Builder) Builder {
		return builder.FromSelect(expr, alias)
	})
}
