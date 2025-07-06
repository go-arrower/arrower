package q

import (
	"reflect"
)

/*
   Basic comparisons: Equals, NotEquals, GreaterThan, LessThan, etc.
   Range queries: Between
   Pattern matching: Like
   Set operations: In
   Logical grouping: And, Or with nested groups
   Ordering: OrderBy with ASC/DESC
   Pagination: Limit and Offset

*/

// Operator represents comparison operators
type Operator string // TODO does it have value and must it be exposed? OR jsut use internal strings directly from the WhereQuery functions

const (
	Eq  Operator = "="
	Ne  Operator = "!="
	Gt  Operator = ">"
	Gte Operator = ">="
	Lt  Operator = "<"
	Lte Operator = "<="
)

type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "AND"
	LogicalOr  LogicalOperator = "OR"
)

type Query struct {
	Conditions ConditionGroup
	ordering   *orderBy
}

func (q Query) Where(field string) *WhereQuery {
	return &WhereQuery{query: &q, field: field}
}

func (q Query) Or(cond ...Query) Query {
	qq := &q

	group := ConditionGroup{
		Operator: LogicalOr,
	}
	for _, q := range cond {
		group.Conditions = append(group.Conditions, q.Conditions.Conditions...)
	}

	qq.Conditions.Groups = append(qq.Conditions.Groups, group)

	return *qq
}

func (q *Query) addCondition(field string, op Operator, value any) {
	q.Conditions.Conditions = append(q.Conditions.Conditions, Cond{
		Field:    field,
		Operator: op,
		Value:    value,
	})
}

type ConditionGroup struct {
	Operator   LogicalOperator
	Conditions []Cond
	Groups     []ConditionGroup
}

// Condition represents a single WHERE condition
type Cond struct {
	Field    string
	Operator Operator
	Value    any
	// For BETWEEN operator
	SecondValue any
}

func Where(field string) *WhereQuery {
	return &WhereQuery{query: &Query{Conditions: ConditionGroup{Conditions: make([]Cond, 0)}}, field: field}
}

type WhereQuery struct {
	query *Query
	field string
}

func (f *WhereQuery) Is(value any) Query {
	f.query.addCondition(f.field, "=", value) // TODO string or Operator
	return *f.query
}

// TODO use this in the logical stements
func Field(field string) *FieldQuery {
	return &FieldQuery{}
}

type FieldQuery struct{}

func (f *FieldQuery) Is(value any) FieldQuery { return FieldQuery{} }

type orderBy struct {
	field     string
	direction string
}

func (q Query) OrderBy(field string) *OrderQuery {
	return &OrderQuery{query: &q, field: field}
}

type OrderQuery struct {
	query *Query
	field string
}

func (o *OrderQuery) Ascending() Query {
	o.query.ordering = &orderBy{field: o.field, direction: "ASC"}
	return *o.query
}

func (o *OrderQuery) Descending() Query {
	o.query.ordering = &orderBy{field: o.field, direction: "DESC"}
	return *o.query
}

// F ignores zero values
func F[T any](m T) Query {
	fv := reflect.ValueOf(m)
	ft := fv.Type()

	var conds []Cond

	for i := range fv.NumField() {
		field := fv.Field(i)
		zeroValue := reflect.Zero(field.Type()).Interface()

		if !reflect.DeepEqual(field.Interface(), zeroValue) {
			fVal := field.Interface()

			conds = append(conds, Cond{
				Field:    ft.Field(i).Name,
				Operator: "=",
				Value:    fVal,
			})
		}
	}

	return Query{Conditions: ConditionGroup{Conditions: conds}}
}

// ---------------------------------------------------------------------------------------------------------------------

func ActiveUsers() Query {
	return Query{Conditions: ConditionGroup{Conditions: []Cond{
		{
			Field:    "Active",
			Operator: "=",
			Value:    true,
		},
	}}}
}

type MyUserFilter struct{}
type User struct{}

func (u User) Active() Query {
	return Query{Conditions: ConditionGroup{Conditions: []Cond{
		{
			Field:    "Active",
			Operator: "=",
			Value:    true,
		},
	}}}
}

type UserQuery struct {
	*Query
}

func Users() *UserQuery {
	return &UserQuery{}
}

// Now you can add model-specific helpers
func (q *UserQuery) Active() *UserQuery {
	return q
}

func (q *UserQuery) Adults() *UserQuery {
	return q
}

func (q *UserQuery) WithVerifiedEmail() *UserQuery {
	return q
}

func (q *UserQuery) Find() Query {
	return Query{}
}

/*
------------------------------------------------------------------------------------------------------------------------
*/

// Condition is an interface that filters can implement to influence the
// selected entities that the Repository returns.
type Condition[T any] interface {
	Filter() T
	OrderBy() string
}

func Filter[T any](m T) Condition[T] {
	return filter[T]{model: m, orderBy: ""}
}

func OrderBy[T any](m string) Condition[T] {
	return filter[T]{orderBy: m, model: *new(T)}
}

type filter[T any] struct {
	model   T
	orderBy string
}

func (f filter[T]) Filter() T { //nolint:ireturn // type param of OrderBy is any, but irrelevant
	return f.model
}

func (f filter[T]) OrderBy() string {
	return f.orderBy
}
