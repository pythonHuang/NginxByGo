package config

import (
	"unicode"
)

// TokenType Token 类型
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenLeftBrace     // {
	TokenRightBrace    // }
	TokenSemicolon     // ;
	TokenIdent         // 标识符
	TokenString        // 字符串
	TokenVariable      // 变量 $xxx
)

// Token Token 结构
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Column  int
}

// Scanner 词法扫描器
type Scanner struct {
	input  []byte
	pos    int
	line   int
	column int
}

// NewScanner 创建扫描器
func NewScanner(input []byte) *Scanner {
	return &Scanner{
		input:  input,
		line:   1,
		column: 1,
	}
}

// Next 获取下一个 Token
func (s *Scanner) Next() Token {
	s.skipWhitespace()

	if s.pos >= len(s.input) {
		return Token{Type: TokenEOF, Line: s.line, Column: s.column}
	}

	ch := s.input[s.pos]

	switch {
	case ch == '{':
		s.advance()
		return s.makeToken(TokenLeftBrace, "{")
	case ch == '}':
		s.advance()
		return s.makeToken(TokenRightBrace, "}")
	case ch == ';':
		s.advance()
		return s.makeToken(TokenSemicolon, ";")
	case ch == '"':
		return s.scanString()
	case ch == '$':
		return s.scanVariable()
	case unicode.IsLetter(rune(ch)) || ch == '_':
		return s.scanIdent()
	default:
		s.advance()
		return s.makeToken(TokenIdent, string(ch))
	}
}

func (s *Scanner) skipWhitespace() {
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if ch == '\n' {
				s.line++
				s.column = 1
			}
			s.pos++
		} else if ch == '#' {
			s.skipComment()
		} else {
			break
		}
	}
}

func (s *Scanner) skipComment() {
	for s.pos < len(s.input) && s.input[s.pos] != '\n' {
		s.pos++
	}
}

func (s *Scanner) scanString() Token {
	s.pos++ // skip opening quote
	start := s.pos
	
	for s.pos < len(s.input) {
		if s.input[s.pos] == '"' {
			break
		}
		s.pos++
	}
	
	value := string(s.input[start:s.pos])
	s.pos++ // skip closing quote
	
	return s.makeToken(TokenString, value)
}

func (s *Scanner) scanVariable() Token {
	s.pos++ // skip $
	start := s.pos
	
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if !unicode.IsLetter(rune(ch)) && !unicode.IsDigit(rune(ch)) && ch != '_' {
			break
		}
		s.pos++
	}
	
	return s.makeToken(TokenVariable, string(s.input[start:s.pos]))
}

func (s *Scanner) scanIdent() Token {
	start := s.pos
	for s.pos < len(s.input) {
		ch := s.input[s.pos]
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '-' || ch == '.' {
			s.pos++
		} else {
			break
		}
	}
	return s.makeToken(TokenIdent, string(s.input[start:s.pos]))
}

func (s *Scanner) advance() {
	if s.pos < len(s.input) {
		if s.input[s.pos] == '\n' {
			s.line++
			s.column = 1
		} else {
			s.column++
		}
		s.pos++
	}
}

func (s *Scanner) makeToken(t TokenType, value string) Token {
	return Token{
		Type:    t,
		Value:   value,
		Line:    s.line,
		Column:  s.column,
	}
}