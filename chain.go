package sqlhelper

type Copier[V any] interface{ Copy() V }

type ChainMeta[Carrier Copier[*Carrier], Ptr interface {
	*Carrier
	set(Copier[*Carrier], Builder)
}, Builder any] struct {
	copier  Copier[*Carrier]
	builder Builder
}

func WithChain[Carrier Copier[*Carrier], Ptr interface {
	*Carrier
	set(Copier[*Carrier], Builder)
}, Builder any](cp Ptr, builder Builder, opts ...func(Builder) Builder) Carrier {
	for _, opt := range opts {
		if opt != nil {
			builder = opt(builder)
		}
	}
	cp.set(*cp, builder)
	return *cp
}

func (w ChainMeta[Carrier, Ptr, Builder]) WithOptions(opts ...func(Builder) Builder) (zero Carrier) {
	return WithChain[Carrier, Ptr](w.copier.Copy(), w.builder, opts...)
}

func (w *ChainMeta[Carrier, _, Builder]) set(cp Copier[*Carrier], state Builder) {
	w.copier = cp
	w.builder = state
}
