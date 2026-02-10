package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Posting struct {
	Account   string  `json:"account"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	HasAmount bool    `json:"has_amount"`
}

type Transaction struct {
	Date      string    `json:"date"`
	Flag      string    `json:"flag"`
	Payee     string    `json:"payee"`
	Narration string    `json:"narration"`
	Postings  []Posting `json:"postings"`
}

type Ledger struct {
	Transactions []Transaction `json:"transactions"`
}

var txnHeaderRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+(.*)$`)
var quotedStringRe = regexp.MustCompile(`"([^"]*)"`)
var postingRe = regexp.MustCompile(`^[ \t]+([A-Za-z][A-Za-z0-9:\-]*)(?:\s+(-?[0-9]+(?:\.[0-9]*)?)\s+([A-Z]+))?\s*$`)

func ParseLedger(text string) (*Ledger, error) {
	ledger := &Ledger{}
	scanner := bufio.NewScanner(strings.NewReader(text))

	var current *Transaction

	for scanner.Scan() {
		line := scanner.Text()

		line = stripInlineComment(line)

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if current != nil {
				ledger.Transactions = append(ledger.Transactions, *current)
				current = nil
			}
			continue
		}

		if trimmed[0] == ';' {
			continue
		}

		if m := txnHeaderRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				ledger.Transactions = append(ledger.Transactions, *current)
			}

			date := m[1]
			flag := m[2]
			rest := m[3]

			payee, narration := parsePayeeNarration(rest)

			current = &Transaction{
				Date:      date,
				Flag:      flag,
				Payee:     payee,
				Narration: narration,
				Postings:  []Posting{},
			}
			continue
		}

		if current != nil && (line[0] == ' ' || line[0] == '\t') {
			if p := postingRe.FindStringSubmatch(line); p != nil {
				posting := Posting{
					Account: p[1],
				}
				if p[2] != "" {
					amount, err := strconv.ParseFloat(p[2], 64)
					if err != nil {
						return nil, fmt.Errorf("invalid amount %q: %w", p[2], err)
					}
					posting.Amount = amount
					posting.Currency = p[3]
					posting.HasAmount = true
				}
				current.Postings = append(current.Postings, posting)
			}
			continue
		}

		if current != nil {
			ledger.Transactions = append(ledger.Transactions, *current)
			current = nil
		}
	}

	if current != nil {
		ledger.Transactions = append(ledger.Transactions, *current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return ledger, nil
}

func stripInlineComment(line string) string {
	inQuote := false
	for i, ch := range line {
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == ';' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func parsePayeeNarration(rest string) (payee, narration string) {
	matches := quotedStringRe.FindAllStringSubmatch(rest, -1)
	switch len(matches) {
	case 1:
		narration = matches[0][1]
	case 2:
		payee = matches[0][1]
		narration = matches[1][1]
	}
	return
}
