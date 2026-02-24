package sqlhelper

import (
	"context"
	"database/sql"
	"database/sql/driver"
)

// Model defines the interface for database models.
// Implement this interface to map Go structs to database tables.
type Model interface {
	// TableName returns the name of the database table
	TableName() string
	// FieldMapping populates the dst map with field name to pointer mappings
	FieldMapping(dst map[string]any)
}

// DB is an interface representing a database connection.
// It extends the standard sql.DB functionality with context support.
type DB interface {
	Close() error
	Stats() sql.DBStats
	PingContext(ctx context.Context) error
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Conn
}

// Conn is an interface for database connections that support context-based operations.
type Conn interface {
	QueryContext(ctx context.Context, statement string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, statement string, args ...any) *sql.Row
	ExecContext(ctx context.Context, statement string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, statement string) (*sql.Stmt, error)
}

// Tx is an interface for database transactions.
type Tx interface {
	Conn
	driver.Tx
}
