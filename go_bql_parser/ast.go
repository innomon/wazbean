package main

// Query represents a full BQL query.
type Query struct {
	Select      []Expression `json:"select"`
	From        string       `json:"from,omitempty"`
	Where       Expression   `json:"where"`
	GroupBy     []Expression `json:"group_by,omitempty"`
	OrderBy     []OrderBy    `json:"order_by,omitempty"`
}

// Expression represents a value or computation.
type Expression struct {
	Literal string `json:"literal,omitempty"`
	// This will be expanded to handle binary operators, function calls, etc.
}

// OrderBy represents a single 'ORDER BY' clause.
type OrderBy struct {
	Expression Expression `json:"expression"`
	Ascending  bool       `json:"ascending"`
}