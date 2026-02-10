package main

type Query struct {
	Select     []Expression `json:"select"`
	From       string       `json:"from,omitempty"`
	Where      Expression   `json:"where"`
	WhereField string       `json:"where_field,omitempty"`
	GroupBy    []Expression `json:"group_by,omitempty"`
	OrderBy    []OrderBy    `json:"order_by,omitempty"`
}

type Expression struct {
	Literal  string       `json:"literal,omitempty"`
	FuncName string       `json:"func_name,omitempty"`
	FuncArgs []Expression `json:"func_args,omitempty"`
}

type OrderBy struct {
	Expression Expression `json:"expression"`
	Ascending  bool       `json:"ascending"`
}