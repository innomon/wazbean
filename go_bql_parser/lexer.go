package main

import (
	"strings"
	"text/scanner"
	"fmt"
)

// BQLLexer holds the state of the scanner.
type BQLLexer struct {
	scanner.Scanner
	result *Query
	err    error
}

// NewBQLLexer creates a new lexer for the given BQL query string.
func NewBQLLexer(query string) *BQLLexer {
	var s scanner.Scanner
	s.Init(strings.NewReader(query))
	s.IsIdentRune = func(ch rune, i int) bool {
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch == '_') || (ch == '-') || (i > 0 && ch >= '0' && ch <= '9')
	}
	// Removing ScanChars is the key fix. This allows identifiers to be scanned correctly.
	s.Mode = scanner.ScanIdents | scanner.ScanFloats
	
	return &BQLLexer{Scanner: s}
}

// keywordMap maps BQL keywords to their token types.
var keywordMap = map[string]int{
	"SELECT": SELECT, "FROM": FROM, "WHERE": WHERE,
	"GROUP": GROUP, "ORDER": ORDER, "BY": BY,
	"ASC": ASC, "DESC": DESC,
}

// Lex is the main scanner function.
func (l *BQLLexer) Lex(lval *yySymType) int {
	tok := l.Scan()

	// Handle single-quoted strings manually.
	if tok == '\'' {
		var text strings.Builder
		for l.Peek() != '\'' && l.Peek() != scanner.EOF {
			text.WriteRune(l.Next())
		}
		if l.Peek() == scanner.EOF {
			l.err = fmt.Errorf("unclosed string literal")
			return 0
		}
		l.Next() // Consume the closing quote.
		lval.str = text.String()
		return STRING
	}

	switch tok {
	case scanner.EOF:
		return 0
	case '=':
		return EQ
	}

	if tok == scanner.Ident {
		keyword := strings.ToUpper(l.TokenText())
		if tokType, isKeyword := keywordMap[keyword]; isKeyword {
			return tokType
		}
		lval.str = l.TokenText()
		return IDENT
	}

	return int(tok)
}

// Error is called by the parser on a syntax error.
func (l *BQLLexer) Error(e string) {
	l.err = fmt.Errorf("BQL Parse Error: %s at position %d", e, l.Pos().Offset)
}