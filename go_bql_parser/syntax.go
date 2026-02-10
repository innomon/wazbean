package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type SyntaxError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type SyntaxResult struct {
	Valid  bool          `json:"valid"`
	Errors []SyntaxError `json:"errors"`
}

var directiveRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+(.*)$`)

var knownDirectives = map[string]bool{
	"open": true, "close": true, "balance": true, "pad": true,
	"event": true, "note": true, "document": true, "custom": true,
	"commodity": true, "price": true, "query": true, "plugin": true,
}

func CheckSyntax(text string) *SyntaxResult {
	result := &SyntaxResult{Valid: true}
	scanner := bufio.NewScanner(strings.NewReader(text))

	lineNum := 0
	inTransaction := false
	txnLine := 0
	postingCount := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		stripped := stripInlineComment(line)
		trimmed := strings.TrimSpace(stripped)

		if trimmed == "" || trimmed[0] == ';' {
			if inTransaction && postingCount < 1 {
				result.addError(txnLine, "transaction has no postings")
			}
			inTransaction = false
			postingCount = 0
			continue
		}

		if strings.HasPrefix(trimmed, "option ") || strings.HasPrefix(trimmed, "include ") || strings.HasPrefix(trimmed, "plugin ") || strings.HasPrefix(trimmed, "poptag") || strings.HasPrefix(trimmed, "pushtag") {
			if inTransaction && postingCount < 1 {
				result.addError(txnLine, "transaction has no postings")
			}
			inTransaction = false
			postingCount = 0
			continue
		}

		if line[0] == ' ' || line[0] == '\t' {
			if !inTransaction {
				result.addError(lineNum, "unexpected indented line outside of a transaction")
				continue
			}
			if postingRe.MatchString(line) {
				postingCount++
			} else if strings.TrimSpace(stripped) != "" {
				result.addError(lineNum, fmt.Sprintf("invalid posting syntax: %s", trimmed))
			}
			continue
		}

		if inTransaction && postingCount < 1 {
			result.addError(txnLine, "transaction has no postings")
		}
		inTransaction = false
		postingCount = 0

		m := directiveRe.FindStringSubmatch(line)
		if m == nil {
			result.addError(lineNum, fmt.Sprintf("unrecognized line: %s", trimmed))
			continue
		}

		rest := strings.TrimSpace(m[2])

		if rest == "" {
			result.addError(lineNum, "missing directive after date")
			continue
		}

		if rest[0] == '*' || rest[0] == '!' {
			inTransaction = true
			txnLine = lineNum

			after := strings.TrimSpace(rest[1:])
			if after == "" {
				result.addError(lineNum, "transaction missing narration")
			} else {
				quotes := quotedStringRe.FindAllString(after, -1)
				if len(quotes) == 0 {
					result.addError(lineNum, "transaction narration must be quoted")
				}
			}
			continue
		}

		fields := strings.Fields(rest)
		directive := fields[0]
		if !knownDirectives[directive] {
			if strings.HasPrefix(directive, "\"") || strings.HasPrefix(directive, "'") {
				result.addError(lineNum, "transaction must have a flag (* or !)")
			} else {
				result.addError(lineNum, fmt.Sprintf("unknown directive: %s", directive))
			}
			continue
		}

		if directive == "open" || directive == "close" || directive == "balance" || directive == "pad" {
			if len(fields) < 2 {
				result.addError(lineNum, fmt.Sprintf("%s directive requires an account", directive))
			}
		}
	}

	if inTransaction && postingCount < 1 {
		result.addError(txnLine, "transaction has no postings")
	}

	return result
}

func (r *SyntaxResult) addError(line int, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, SyntaxError{Line: line, Message: message})
}

func CheckBeancountSyntax(ledgerText string) string {
	result := CheckSyntax(ledgerText)
	jsonResult, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"error": "serialization error: %v"}`, err)
	}
	return string(jsonResult)
}
