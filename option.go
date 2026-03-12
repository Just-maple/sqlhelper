package sqlhelper

type (
	ChainBuilder[T any] interface {
		Prefix(string, ...any) T
		Suffix(string, ...any) T
		Where(any, ...any) T
		Limit(uint64) T
		Offset(uint64) T
		FromSelect(SelectBuilder, string) T
		From(string) T
		ToSql() (string, []interface{}, error)
	}

	Options[T ChainBuilder[T]] []func(T) T

	Option[T ChainBuilder[T]] func(T) T
)

func (opt Option[T]) Prefix(str string, args ...any) Option[T] {
	return func(builder T) T { return builder.Prefix(str, args...) }
}

func (opt Option[T]) Suffix(str string, args ...any) Option[T] {
	return func(builder T) T { return builder.Suffix(str, args...) }
}

func (opt Option[T]) Where(pred any, args ...any) Option[T] {
	return func(builder T) T { return builder.Where(pred, args...) }
}

func (opt Option[T]) FromSelect(sel SelectBuilder, alias string) Option[T] {
	return func(builder T) T { return builder.FromSelect(sel, alias) }
}

func (opt Option[T]) From(table string) Option[T] {
	return func(builder T) T { return builder.From(table) }
}

func (opt Option[T]) Limit(limit uint64) Option[T] {
	return func(builder T) T { return builder.Limit(limit) }
}

func (opt Option[T]) Offset(offset uint64) Option[T] {
	return func(builder T) T { return builder.Offset(offset) }
}

func (opt Options[T]) Append(opts ...func(T) T) Options[T] {
	cp := append(make(Options[T], 0, len(opts)+len(opts)), opt...)
	return append(cp, opts...)
}

func (opt Options[T]) Prefix(str string, args ...any) Options[T] {
	return opt.Append(func(builder T) T { return builder.Prefix(str, args...) })
}

func (opt Options[T]) Suffix(str string, args ...any) Options[T] {
	return opt.Append(func(builder T) T { return builder.Suffix(str, args...) })
}

func (opt Options[T]) Where(pred any, args ...any) Options[T] {
	return opt.Append(func(builder T) T { return builder.Where(pred, args...) })
}

func (opt Options[T]) FromSelect(sel SelectBuilder, alias string) Options[T] {
	return opt.Append(func(builder T) T { return builder.FromSelect(sel, alias) })
}

func (opt Options[T]) From(table string) Options[T] {
	return opt.Append(func(builder T) T { return builder.From(table) })
}

func (opt Options[T]) Limit(limit uint64) Options[T] {
	return opt.Append(func(builder T) T { return builder.Limit(limit) })
}

func (opt Options[T]) Offset(offset uint64) Options[T] {
	return opt.Append(func(builder T) T { return builder.Offset(offset) })
}
