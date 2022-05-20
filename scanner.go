package bibtex

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
)

var parseField bool

// scanner is a lexical scanner
type scanner struct {
	r   *bufio.Reader
	pos tokenPos
}

// newScanner returns a new instance of scanner.
func newScanner(r io.Reader) *scanner {
	return &scanner{r: bufio.NewReader(r), pos: tokenPos{Char: 0, Lines: []int{}}}
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.eof is returned).
func (s *scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	if ch == '\n' {
		s.pos.Lines = append(s.pos.Lines, s.pos.Char)
		s.pos.Char = 0
	} else {
		s.pos.Char++
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *scanner) unread() {
	_ = s.r.UnreadRune()
	if s.pos.Char == 0 {
		s.pos.Char = s.pos.Lines[len(s.pos.Lines)-1]
		s.pos.Lines = s.pos.Lines[:len(s.pos.Lines)-1]
	} else {
		s.pos.Char--
	}
}

// Scan returns the next token and literal value.
func (s *scanner) Scan() (tok token, lit string) {
	ch := s.read()
	if isWhitespace(ch) {
		s.ignoreWhitespace()
		ch = s.read()
	}
	if isAlphanum(ch) {
		s.unread()
		return s.scanIdent()
	}
	switch ch {
	case eof:
		return 0, ""
	case '@':
		return tATSIGN, string(ch)
	case ':':
		return tCOLON, string(ch)
	case ',':
		parseField = false // reset parseField if reached end of field.
		return tCOMMA, string(ch)
	case '=':
		parseField = true // set parseField if = sign outside quoted or ident.
		return tEQUAL, string(ch)
	case '"':
		return s.scanQuoted()
	case '{':
		if parseField {
			return s.scanBraced()
		}
		return tLBRACE, string(ch)
	case '}':
		if parseField { // reset parseField if reached end of entry.
			parseField = false
		}
		return tRBRACE, string(ch)
	case '#':
		return tPOUND, string(ch)
	case ' ':
		s.ignoreWhitespace()
	}
	return tILLEGAL, string(ch)
}

// scanIdent categorises a string to one of three categories.
func (s *scanner) scanIdent() (tok token, lit string) {
	switch ch := s.read(); ch {
	case '"':
		return s.scanQuoted()
	case '{':
		return s.scanBraced()
	default:
		s.unread() // Not open quote/brace.
		return s.scanBare()
	}
}

func (s *scanner) scanBare() (token, string) {
	var buf bytes.Buffer
	var trailingWhitespace int
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isAlphanum(ch) && !isBareSymbol(ch) && !isWhitespace(ch) {
			s.unread()
			break
		} else {
			if isWhitespace(ch) {
				trailingWhitespace += 1
			} else {
				trailingWhitespace = 0
			}
			_, _ = buf.WriteRune(ch)
		}
	}
	buf.Truncate(buf.Len() - trailingWhitespace)
	str := buf.String()
	if strings.ToLower(str) == "comment" {
		return tCOMMENT, str
	} else if strings.ToLower(str) == "preamble" {
		return tPREAMBLE, str
	} else if strings.ToLower(str) == "string" {
		return tSTRING, str
	} else if _, err := strconv.Atoi(str); err == nil && parseField { // Special case for numeric
		return tIDENT, str
	}
	return tBAREIDENT, str
}

// scanBraced parses a braced string, like {this}.
func (s *scanner) scanBraced() (token, string) {
	var buf bytes.Buffer
	brace := 1
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '\\' {
			_, _ = buf.WriteRune(ch)
		} else if ch == '{' {
			_, _ = buf.WriteRune(ch)
			brace++
		} else if ch == '}' {
			brace--
			if brace == 0 { // Balances open brace.
				return tIDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else if isWhitespace(ch) {
			_, _ = buf.WriteRune(ch)
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return tILLEGAL, buf.String()
}

// scanQuoted parses a quoted string, like "this".
func (s *scanner) scanQuoted() (token, string) {
	var buf bytes.Buffer
	brace := 0
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '{' {
			brace++
		} else if ch == '}' {
			brace--
		} else if ch == '"' {
			if brace == 0 { // Matches open quote, unescaped
				return tIDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return tILLEGAL, buf.String()
}

// ignoreWhitespace consumes the current rune and all contiguous whitespace.
func (s *scanner) ignoreWhitespace() {
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		}
	}
}
