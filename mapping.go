package sqlhelper

import "sort"

// Mapper is a map type that maps column names to values.
// Used for mapping between database columns and struct fields.
type Mapper map[string]any

// MapValuesTo maps model field values to the provided slice based on columns.
func (m Mapper) MapValuesTo(model Model, columns []string, values *[]any) {
	model.FieldMapping(m)
	for _, column := range columns {
		*values = append(*values, m[column])
	}
}

// MapValues returns a slice of values from the model based on columns.
func (m Mapper) MapValues(model Model, columns []string) (values []any) {
	m.MapValuesTo(model, columns, &values)
	return
}

// MapColumns populates the columns slice with all keys from the mapper.
// If columns already has values, it returns without modification.
func (m Mapper) MapColumns(columns *[]string) {
	if len(*columns) > 0 {
		return
	}
	cs := make([]string, 0, len(m))
	for column := range m {
		cs = append(cs, column)
	}
	sort.Strings(cs)
	*columns = cs
}

// ConvertModelMapping creates a function that converts a model allocation function
// to a column-based values function for scanning.
func ConvertModelMapping[M Model](alloc func() (model M)) func(columns []string) []any {
	vv := &struct {
		dst     []any
		mapping Mapper
	}{}
	return func(columns []string) []any {
		if vv.dst == nil {
			vv.dst = make([]any, 0, len(columns))
		}
		if vv.mapping == nil {
			vv.mapping = make(Mapper, len(columns))
		}
		vv.dst = vv.dst[:0]
		vv.mapping.MapValuesTo(alloc(), columns, &vv.dst)
		return vv.dst
	}
}

// MappingModel is a generic model implementation that uses a mapping function
// to define table name and field mappings.
type MappingModel[T any] struct {
	Model   *T
	table   string
	mapping map[string]any
}

// TableName returns the table name for this model.
func (m MappingModel[T]) TableName() string {
	return m.table
}

// FieldMapping populates the mapping with field pointers.
func (m MappingModel[T]) FieldMapping(mapping map[string]any) {
	for k, v := range m.mapping {
		mapping[k] = v
	}
}

// NewMappingModelHelper creates a ModelHelper using a custom allocation function
// that returns both table name and field mappings.
func NewMappingModelHelper[T any](alloc func(*T) (table string, mapping map[string]any)) ModelHelper[*MappingModel[T], MappingModel[T]] {
	return NewModelHelper(NewMappingModel(alloc))
}

// NewMappingModel creates an allocation function for MappingModel.
func NewMappingModel[T any](alloc func(*T) (table string, mapping map[string]any)) func() MappingModel[T] {
	return func() MappingModel[T] {
		var t T
		table, mapping := alloc(&t)
		return MappingModel[T]{
			Model:   &t,
			mapping: mapping,
			table:   table,
		}
	}
}

// Convert creates a MappingModel allocation function from a mapper function.
func (h ModelHelper[M, T]) Convert(mapper func(*T) map[string]any) func() MappingModel[T] {
	return NewMappingModel(func(m *T) (table string, mapping map[string]any) {
		*m = h.alloc()
		return M(m).TableName(), mapper(m)
	})
}
