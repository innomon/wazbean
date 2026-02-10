package main

import (
	"encoding/json"
	"os"
	"testing"
)

const testLedger = `
2024-01-15 * "AcmeCo" "Salary deposit"
  Assets:BofA:Checking    3000.00 USD
  Income:Salary:AcmeCo   -3000.00 USD

2024-01-16 * "Whole Foods" "Weekly groceries"
  Expenses:Food:Groceries   87.34 USD
  Assets:BofA:Checking     -87.34 USD

2024-01-20 * "Olive Garden" "Dinner with family"
  Expenses:Food:Restaurant  72.15 USD
  Liabilities:CreditCard:Visa

2024-02-03 * "Trader Joe's" "Groceries"
  Expenses:Food:Groceries  112.60 USD
  Assets:BofA:Checking    -112.60 USD

2024-02-12 * "AcmeCo" "Salary deposit"
  Assets:BofA:Checking    3000.00 USD
  Income:Salary:AcmeCo   -3000.00 USD

2024-02-25 * "Landlord Properties LLC" "February rent"
  Expenses:Rent           1500.00 USD
  Assets:BofA:Checking   -1500.00 USD
`

func TestParseLedger(t *testing.T) {
	ledger, err := ParseLedger(testLedger)
	if err != nil {
		t.Fatalf("ParseLedger failed: %v", err)
	}
	if len(ledger.Transactions) != 6 {
		t.Fatalf("expected 6 transactions, got %d", len(ledger.Transactions))
	}

	txn := ledger.Transactions[0]
	if txn.Date != "2024-01-15" {
		t.Errorf("expected date 2024-01-15, got %s", txn.Date)
	}
	if txn.Payee != "AcmeCo" {
		t.Errorf("expected payee AcmeCo, got %s", txn.Payee)
	}
	if len(txn.Postings) != 2 {
		t.Fatalf("expected 2 postings, got %d", len(txn.Postings))
	}
	if txn.Postings[0].Amount != 3000.00 {
		t.Errorf("expected amount 3000.00, got %f", txn.Postings[0].Amount)
	}

	olive := ledger.Transactions[2]
	if !olive.Postings[0].HasAmount {
		t.Error("expected first posting of Olive Garden to have amount")
	}
	if olive.Postings[1].HasAmount {
		t.Error("expected second posting of Olive Garden to NOT have amount")
	}
}

func TestSelectAllPostings(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, date, narration")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(result.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(result.Columns))
	}
	if result.Columns[0] != "account" || result.Columns[1] != "date" || result.Columns[2] != "narration" {
		t.Errorf("unexpected columns: %v", result.Columns)
	}
	if len(result.Rows) != 12 {
		t.Errorf("expected 12 rows (2 postings * 6 txns), got %d", len(result.Rows))
	}
}

func TestWhereFilter(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, amount WHERE account = 'Expenses:Food:Groceries'")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 grocery rows, got %d", len(result.Rows))
	}
	for _, row := range result.Rows {
		if row[0] != "Expenses:Food:Groceries" {
			t.Errorf("expected account Expenses:Food:Groceries, got %v", row[0])
		}
	}
}

func TestFromFilter(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, amount FROM 'Expenses:Food'")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	for _, row := range result.Rows {
		acct := row[0].(string)
		_ = acct
	}
	if len(result.Rows) < 4 {
		t.Errorf("expected at least 4 rows from food transactions, got %d", len(result.Rows))
	}
}

func TestGroupByWithSum(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, SUM(amount) GROUP BY account")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	found := false
	for _, row := range result.Rows {
		if row[0] == "Expenses:Food:Groceries" {
			found = true
			sum := row[1].(float64)
			expected := 87.34 + 112.60
			if sum < expected-0.01 || sum > expected+0.01 {
				t.Errorf("expected sum ~%.2f, got %.2f", expected, sum)
			}
		}
	}
	if !found {
		t.Error("did not find Expenses:Food:Groceries group")
	}
}

func TestGroupByWithCount(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, COUNT(*) GROUP BY account")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	for _, row := range result.Rows {
		if row[0] == "Assets:BofA:Checking" {
			count := row[1].(float64)
			if count != 5 {
				t.Errorf("expected 5 checking postings, got %.0f", count)
			}
		}
	}
}

func TestOrderBy(t *testing.T) {
	ledger, _ := ParseLedger(testLedger)
	query, _ := Parse("SELECT account, amount WHERE account = 'Expenses:Food:Groceries' ORDER BY amount DESC")
	result, err := Execute(query, ledger)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}
	first := result.Rows[0][1].(float64)
	second := result.Rows[1][1].(float64)
	if first < second {
		t.Errorf("expected descending order: %.2f should be >= %.2f", first, second)
	}
}

func TestExecuteBQLEndToEnd(t *testing.T) {
	jsonStr := ExecuteBQL(
		"SELECT account, amount WHERE account = 'Expenses:Rent'",
		testLedger,
	)
	var result Result
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 rent row, got %d", len(result.Rows))
	}
	if result.Rows[0][0] != "Expenses:Rent" {
		t.Errorf("expected Expenses:Rent, got %v", result.Rows[0][0])
	}
}

func TestExecuteBQLWithSampleFile(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.beancount")
	if err != nil {
		t.Skipf("sample file not found: %v", err)
	}

	jsonStr := ExecuteBQL(
		"SELECT account, SUM(amount) WHERE account = 'Expenses:Food:Groceries' GROUP BY account",
		string(data),
	)
	var result Result
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v\nraw: %s", err, jsonStr)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result.Rows))
	}
	if result.Rows[0][0] != "Expenses:Food:Groceries" {
		t.Errorf("unexpected account: %v", result.Rows[0][0])
	}
	sum := result.Rows[0][1].(float64)
	if sum < 100 {
		t.Errorf("expected total groceries > 100, got %.2f", sum)
	}
}

func TestExecuteBQLParseError(t *testing.T) {
	jsonStr := ExecuteBQL("INVALID QUERY", testLedger)
	if !containsStr(jsonStr, "error") {
		t.Errorf("expected error in result, got: %s", jsonStr)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstr(s, sub))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
