package sqlhelper

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
