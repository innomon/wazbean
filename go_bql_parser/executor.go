package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Result struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type postingRow struct {
	txn *Transaction
	pst *Posting
}

func Execute(query *Query, ledger *Ledger) (*Result, error) {
	rows := buildRows(ledger)
	rows = applyFrom(rows, query.From)
	rows = applyWhere(rows, query.WhereField, query.Where)

	hasAggregates := containsAggregates(query.Select)

	if len(query.GroupBy) > 0 || hasAggregates {
		return executeGrouped(query, rows)
	}

	result := &Result{
		Columns: columnNames(query.Select),
	}
	for _, r := range rows {
		row, err := projectRow(r, query.Select)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, row)
	}

	applyOrderBy(result, query)
	return result, nil
}

func buildRows(ledger *Ledger) []postingRow {
	var rows []postingRow
	for i := range ledger.Transactions {
		txn := &ledger.Transactions[i]
		for j := range txn.Postings {
			rows = append(rows, postingRow{txn: txn, pst: &txn.Postings[j]})
		}
	}
	return rows
}

func applyFrom(rows []postingRow, from string) []postingRow {
	if from == "" {
		return rows
	}
	seen := make(map[*Transaction]bool)
	for _, r := range rows {
		if strings.HasPrefix(r.pst.Account, from) {
			seen[r.txn] = true
		}
	}
	var filtered []postingRow
	for _, r := range rows {
		if seen[r.txn] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func applyWhere(rows []postingRow, field string, value Expression) []postingRow {
	if field == "" && value.Literal == "" {
		return rows
	}
	var filtered []postingRow
	for _, r := range rows {
		fieldVal := resolveField(r, field)
		if fieldVal == value.Literal {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func resolveField(r postingRow, field string) string {
	switch strings.ToLower(field) {
	case "account":
		return r.pst.Account
	case "date":
		return r.txn.Date
	case "payee":
		return r.txn.Payee
	case "narration":
		return r.txn.Narration
	case "flag":
		return r.txn.Flag
	case "currency":
		return r.pst.Currency
	default:
		return ""
	}
}

func resolveValue(r postingRow, expr Expression) interface{} {
	if expr.Literal != "" {
		return resolveFieldValue(r, expr.Literal)
	}
	return nil
}

func resolveFieldValue(r postingRow, field string) interface{} {
	switch strings.ToLower(field) {
	case "account":
		return r.pst.Account
	case "date":
		return r.txn.Date
	case "payee":
		return r.txn.Payee
	case "narration":
		return r.txn.Narration
	case "flag":
		return r.txn.Flag
	case "currency":
		return r.pst.Currency
	case "amount":
		if r.pst.HasAmount {
			return r.pst.Amount
		}
		return nil
	case "position":
		if r.pst.HasAmount {
			return fmt.Sprintf("%.2f %s", r.pst.Amount, r.pst.Currency)
		}
		return ""
	default:
		return field
	}
}

func projectRow(r postingRow, selectExprs []Expression) ([]interface{}, error) {
	var vals []interface{}
	for _, expr := range selectExprs {
		if expr.FuncName != "" {
			return nil, fmt.Errorf("aggregate function %s() used without GROUP BY", expr.FuncName)
		}
		vals = append(vals, resolveValue(r, expr))
	}
	return vals, nil
}

func columnNames(exprs []Expression) []string {
	var names []string
	for _, e := range exprs {
		if e.FuncName != "" {
			argNames := make([]string, len(e.FuncArgs))
			for i, a := range e.FuncArgs {
				argNames[i] = a.Literal
			}
			names = append(names, strings.ToLower(e.FuncName)+"("+strings.Join(argNames, ", ")+")")
		} else {
			names = append(names, e.Literal)
		}
	}
	return names
}

func containsAggregates(exprs []Expression) bool {
	for _, e := range exprs {
		if e.FuncName != "" {
			return true
		}
	}
	return false
}

func executeGrouped(query *Query, rows []postingRow) (*Result, error) {
	type group struct {
		key  []interface{}
		rows []postingRow
	}

	groups := make(map[string]*group)
	var groupOrder []string

	for _, r := range rows {
		var keyParts []interface{}
		var keyStr string
		for _, g := range query.GroupBy {
			val := resolveValue(r, g)
			keyParts = append(keyParts, val)
			keyStr += fmt.Sprintf("%v|", val)
		}
		if _, ok := groups[keyStr]; !ok {
			groups[keyStr] = &group{key: keyParts}
			groupOrder = append(groupOrder, keyStr)
		}
		groups[keyStr].rows = append(groups[keyStr].rows, r)
	}

	result := &Result{
		Columns: columnNames(query.Select),
	}

	for _, k := range groupOrder {
		g := groups[k]
		var outRow []interface{}
		for _, expr := range query.Select {
			if expr.FuncName != "" {
				val, err := evalAggregate(expr, g.rows)
				if err != nil {
					return nil, err
				}
				outRow = append(outRow, val)
			} else {
				outRow = append(outRow, resolveValue(g.rows[0], expr))
			}
		}
		result.Rows = append(result.Rows, outRow)
	}

	applyOrderBy(result, query)
	return result, nil
}

func evalAggregate(expr Expression, rows []postingRow) (interface{}, error) {
	fn := strings.ToUpper(expr.FuncName)
	switch fn {
	case "COUNT":
		return float64(len(rows)), nil
	case "SUM":
		if len(expr.FuncArgs) != 1 {
			return nil, fmt.Errorf("SUM requires exactly one argument")
		}
		field := expr.FuncArgs[0].Literal
		var total float64
		for _, r := range rows {
			val := resolveFieldValue(r, field)
			if v, ok := val.(float64); ok {
				total += v
			}
		}
		return total, nil
	default:
		return nil, fmt.Errorf("unknown aggregate function: %s", fn)
	}
}

func applyOrderBy(result *Result, query *Query) {
	if len(query.OrderBy) == 0 || len(result.Rows) == 0 {
		return
	}

	colIndex := make(map[string]int)
	for i, c := range result.Columns {
		colIndex[c] = i
	}

	sort.SliceStable(result.Rows, func(i, j int) bool {
		for _, ob := range query.OrderBy {
			colName := ob.Expression.Literal
			if ob.Expression.FuncName != "" {
				argNames := make([]string, len(ob.Expression.FuncArgs))
				for k, a := range ob.Expression.FuncArgs {
					argNames[k] = a.Literal
				}
				colName = strings.ToLower(ob.Expression.FuncName) + "(" + strings.Join(argNames, ", ") + ")"
			}
			idx, ok := colIndex[colName]
			if !ok {
				continue
			}
			vi := result.Rows[i][idx]
			vj := result.Rows[j][idx]
			cmp := compareValues(vi, vj)
			if cmp == 0 {
				continue
			}
			if ob.Ascending {
				return cmp < 0
			}
			return cmp > 0
		}
		return false
	})
}

func compareValues(a, b interface{}) int {
	fa, aIsFloat := toFloat(a)
	fb, bIsFloat := toFloat(b)
	if aIsFloat && bIsFloat {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}
	sa := fmt.Sprintf("%v", a)
	sb := fmt.Sprintf("%v", b)
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}
