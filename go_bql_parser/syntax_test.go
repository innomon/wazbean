package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCheckSyntax_ValidLedger(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.beancount")
	if err != nil {
		t.Fatal(err)
	}
	result := CheckSyntax(string(data))
	if !result.Valid {
		t.Errorf("expected valid ledger, got errors: %+v", result.Errors)
	}
}

func TestCheckSyntax_EmptyInput(t *testing.T) {
	result := CheckSyntax("")
	if !result.Valid {
		t.Errorf("expected valid for empty input, got errors: %+v", result.Errors)
	}
}

func TestCheckSyntax_CommentsOnly(t *testing.T) {
	result := CheckSyntax("; just a comment\n; another\n")
	if !result.Valid {
		t.Errorf("expected valid, got errors: %+v", result.Errors)
	}
}

func TestCheckSyntax_TransactionNoPostings(t *testing.T) {
	input := `2024-01-01 * "Payee" "Narration"

2024-01-02 * "Payee2" "Narration2"
  Expenses:Food  10.00 USD
  Assets:Cash   -10.00 USD
`
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Line != 1 {
		t.Errorf("expected error on line 1, got %d", result.Errors[0].Line)
	}
}

func TestCheckSyntax_UnrecognizedLine(t *testing.T) {
	input := "this is garbage\n"
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestCheckSyntax_IndentedLineOutsideTransaction(t *testing.T) {
	input := "  Expenses:Food  10.00 USD\n"
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if result.Errors[0].Message != "unexpected indented line outside of a transaction" {
		t.Errorf("unexpected message: %s", result.Errors[0].Message)
	}
}

func TestCheckSyntax_MissingNarration(t *testing.T) {
	input := `2024-01-01 *
  Expenses:Food  10.00 USD
  Assets:Cash   -10.00 USD
`
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	found := false
	for _, e := range result.Errors {
		if e.Message == "transaction missing narration" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'transaction missing narration' error, got %+v", result.Errors)
	}
}

func TestCheckSyntax_UnquotedNarration(t *testing.T) {
	input := `2024-01-01 * unquoted narration
  Expenses:Food  10.00 USD
  Assets:Cash   -10.00 USD
`
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
}

func TestCheckSyntax_MissingFlag(t *testing.T) {
	input := `2024-01-01 "Payee" "Narration"
  Expenses:Food  10.00 USD
  Assets:Cash   -10.00 USD
`
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
}

func TestCheckSyntax_InvalidPosting(t *testing.T) {
	input := `2024-01-01 * "Payee" "Narration"
  this is not a posting
  Expenses:Food  10.00 USD
`
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
}

func TestCheckSyntax_DirectivesAccepted(t *testing.T) {
	input := `2024-01-01 open Assets:Checking USD
2024-12-31 close Assets:Checking
2024-06-15 balance Assets:Checking 100.00 USD
2024-01-01 pad Assets:Checking Equity:Opening-Balances
option "title" "My Ledger"
`
	result := CheckSyntax(input)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %+v", result.Errors)
	}
}

func TestCheckSyntax_UnknownDirective(t *testing.T) {
	input := "2024-01-01 foobar something\n"
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if result.Errors[0].Message != "unknown directive: foobar" {
		t.Errorf("unexpected message: %s", result.Errors[0].Message)
	}
}

func TestCheckSyntax_OpenMissingAccount(t *testing.T) {
	input := "2024-01-01 open\n"
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
}

func TestCheckBeancountSyntax_JSON(t *testing.T) {
	input := `2024-01-01 * "Payee" "Narration"
  Expenses:Food  10.00 USD
  Assets:Cash   -10.00 USD
`
	jsonStr := CheckBeancountSyntax(input)
	var result SyntaxResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %+v", result.Errors)
	}
}

func TestCheckBeancountSyntax_InvalidJSON(t *testing.T) {
	input := "garbage line\n"
	jsonStr := CheckBeancountSyntax(input)
	var result SyntaxResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors")
	}
}

func TestCheckSyntax_TransactionNoPostingsAtEOF(t *testing.T) {
	input := "2024-01-01 * \"Payee\" \"Narration\""
	result := CheckSyntax(input)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Message != "transaction has no postings" {
		t.Errorf("unexpected message: %s", result.Errors[0].Message)
	}
}
