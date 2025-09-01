package q

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/georgysavva/scany/v2/dbscan"
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
	q.Conditions.Conditions = append(q.Conditions.Conditions, Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
}

type ConditionGroup struct {
	Operator   LogicalOperator
	Conditions []Condition
	Groups     []ConditionGroup
}

// Condition represents a single WHERE condition
type Condition struct {
	Field    string
	Operator Operator
	Value    any
	// For BETWEEN operator
	SecondValue any
}

func Where(field string) *WhereQuery {
	return &WhereQuery{query: &Query{Conditions: ConditionGroup{Conditions: make([]Condition, 0)}}, field: field}
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

// Filter ignores zero values
func Filter[T any](objFilter T) Query {
	fv := reflect.ValueOf(objFilter)
	ft := fv.Type()

	var conditions []Condition

	for i := range fv.NumField() {
		field := fv.Field(i)
		zeroValue := reflect.Zero(field.Type()).Interface()

		if !reflect.DeepEqual(field.Interface(), zeroValue) {
			fName := fieldName(ft.Field(i))
			fVal := field.Interface()

			conditions = append(conditions, Condition{
				Field:    fName,
				Operator: "=",
				Value:    fVal,
			})
		}
	}

	return Query{Conditions: ConditionGroup{Conditions: conditions}}
}

func fieldName(tField reflect.StructField) string {
	var name string

	if dbTag := tField.Tag.Get("db"); dbTag != "" {
		name = dbTag
		if strings.Contains(name, ".") {
			name = fmt.Sprintf(`"%s"`, dbTag)
		}
	} else {
		name = dbscan.SnakeCaseMapper(tField.Name)
	}

	return name
}
