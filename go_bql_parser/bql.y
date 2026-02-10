%{
package main

%}

%union {
    str      string
    expr     Expression
    exprs    []Expression
    orderBy  OrderBy
    orderBys []OrderBy
    query    *Query
    whereClause struct {
        field string
        expr  Expression
    }
}

// Token declarations
%token <str> SELECT FROM WHERE GROUP ORDER BY ASC DESC
%token <str> IDENT STRING
%token EQ

// Type declarations for grammar rules
%type <query>       query_statement
%type <exprs>       select_list
%type <expr>        select_expr
%type <str>         from_clause_opt
%type <whereClause> where_clause_opt
%type <exprs>       group_by_clause_opt
%type <orderBys>    order_by_clause_opt
%type <orderBys>    order_by_list
%type <orderBy>     order_by_expr
%type <str>         opt_asc_desc
%type <whereClause> where_expression

%%

query_statement:
    SELECT select_list from_clause_opt where_clause_opt group_by_clause_opt order_by_clause_opt
    {
        $$ = &Query{
            Select:     $2,
            From:       $3,
            Where:      $4.expr,
            WhereField: $4.field,
            GroupBy:    $5,
            OrderBy:    $6,
        }
        yylex.(*BQLLexer).result = $$
    }
;

select_list:
    select_expr
    {
        $$ = []Expression{$1}
    }
|   select_list ',' select_expr
    {
        $$ = append($1, $3)
    }
;

select_expr:
    IDENT
    {
        $$ = Expression{Literal: $1}
    }
|   IDENT '(' select_list ')'
    {
        $$ = Expression{FuncName: $1, FuncArgs: $3}
    }
|   IDENT '(' '*' ')'
    {
        $$ = Expression{FuncName: $1, FuncArgs: []Expression{{Literal: "*"}}}
    }
;

from_clause_opt:
    /* empty */ { $$ = "" }
|   FROM STRING  { $$ = $2 }
;

where_clause_opt:
    /* empty */            { $$.field = ""; $$.expr = Expression{} }
|   WHERE where_expression { $$ = $2 }
;

where_expression:
    IDENT EQ STRING
    {
        $$.field = $1
        $$.expr = Expression{Literal: $3}
    }
;


group_by_clause_opt:
    /* empty */ { $$ = nil }
|   GROUP BY select_list { $$ = $3 }
;

order_by_clause_opt:
    /* empty */      { $$ = nil }
|   ORDER BY order_by_list { $$ = $3 }
;

order_by_list:
    order_by_expr
    {
        $$ = []OrderBy{$1}
    }
|   order_by_list ',' order_by_expr
    {
        $$ = append($1, $3)
    }
;

order_by_expr:
    select_expr opt_asc_desc
    {
        $$ = OrderBy{Expression: $1, Ascending: ($2 != "DESC")}
    }
;

opt_asc_desc:
    /* empty */ { $$ = "ASC" }
|   ASC         { $$ = "ASC" }
|   DESC        { $$ = "DESC" }
;

%%
