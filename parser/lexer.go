package parser

import (
	"fmt"
	"unicode"
)

const maxGroups = 9

type TokenType int

const (
	TUnion TokenType = iota
	TRightBracket
	TEscapeCaptureGroup
	TCreateCaptureGroup
	TStar
	TChar
	TBackRefString
	TBackRefExpression
)

type Token struct {
	Type    TokenType
	Pos     int
	Char    rune
	GroupID int
}

func (t Token) String() string {
	switch t.Type {
	case TUnion:
		return fmt.Sprintf("TUnion(pos=%d)", t.Pos)
	case TRightBracket:
		return fmt.Sprintf("TRightBracket(pos=%d)", t.Pos)
	case TEscapeCaptureGroup:
		return fmt.Sprintf("TEscapeCaptureGroup(pos=%d)", t.Pos)
	case TCreateCaptureGroup:
		return fmt.Sprintf("TCreateCaptureGroup(pos=%d, group=%d)", t.Pos, t.GroupID)
	case TStar:
		return fmt.Sprintf("TStar(pos=%d)", t.Pos)
	case TChar:
		return fmt.Sprintf("TChar(pos=%d, char=%q)", t.Pos, t.Char)
	case TBackRefString:
		return fmt.Sprintf("TBackRefString(pos=%d, group=%d)", t.Pos, t.GroupID)
	case TBackRefExpression:
		return fmt.Sprintf("TBackRefExpression(pos=%d, group=%d)", t.Pos, t.GroupID)
	default:
		return fmt.Sprintf("UnknownToken(pos=%d)", t.Pos)
	}
}

func Tokenize(regex string) ([]Token, error) {
	var (
		tokens         []Token
		pos            = 0
		length         = len(regex)
		groupCount     = 0
		bracketBalance = 0
	)

	for pos < length {
		ch := rune(regex[pos])

		switch {
		case unicode.IsLetter(ch):
			tokens = append(tokens, Token{
				Type: TChar,
				Pos:  pos,
				Char: ch,
			})
			pos++

		case ch == '|':
			tokens = append(tokens, Token{
				Type: TUnion,
				Pos:  pos,
			})
			pos++

		case ch == '*':
			tokens = append(tokens, Token{
				Type: TStar,
				Pos:  pos,
			})
			pos++

		case ch == ')':
			if bracketBalance == 0 {
				return nil, fmt.Errorf("Лишняя закрывающая скобка на позиции %d", pos)
			}
			bracketBalance--

			tokens = append(tokens, Token{
				Type: TRightBracket,
				Pos:  pos,
			})
			pos++

		case ch == '(':
			pos++

			if pos >= length {
				return nil, fmt.Errorf("Обнаружен символ '(' в конце строки (позиция %d)", pos-1)
			}

			nextCh := rune(regex[pos])

			if nextCh != '?' {
				bracketBalance++
				groupCount++
				if groupCount > maxGroups {
					return nil, fmt.Errorf("Слишком много групп захвата (%d), лимит = %d", groupCount, maxGroups)
				}
				tokens = append(tokens, Token{
					Type:    TCreateCaptureGroup,
					Pos:     pos - 1,
					GroupID: groupCount,
				})
				continue
			}

			pos++

			if pos >= length {
				return nil, fmt.Errorf("После '(?' не хватает символов (позиция %d)", pos-1)
			}

			currentCh := rune(regex[pos])

			switch {
			case unicode.IsDigit(currentCh):
				num := int(currentCh - '0')
				if num == 0 {
					return nil, fmt.Errorf("Номер группы не может быть 0 (позиция %d)", pos)
				}

				tokens = append(tokens, Token{
					Type:    TBackRefExpression,
					Pos:     pos - 2,
					GroupID: num,
				})
				pos++

				if pos >= length || rune(regex[pos]) != ')' {
					if pos < length {
						return nil, fmt.Errorf("Ожидался символ ')', но найден %q (позиция %d)", rune(regex[pos]), pos)
					}
					return nil, fmt.Errorf("Ожидался символ ')', но строка закончилась (позиция %d)", pos)
				}

				pos++

			case currentCh == ':':
				bracketBalance++

				tokens = append(tokens, Token{
					Type: TEscapeCaptureGroup,
					Pos:  pos - 2,
				})
				pos++

			default:
				return nil, fmt.Errorf("Неизвестная конструкция '(?%c' на позиции %d", currentCh, pos)
			}

		case ch == '\\':
			pos++

			if pos >= length {
				return nil, fmt.Errorf("Обнаружен '\\' в конце строки (позиция %d)", pos-1)
			}

			refCh := rune(regex[pos])
			if !unicode.IsDigit(refCh) {
				return nil, fmt.Errorf("После '\\' ожидается цифра для ссылки на группу, но найден %q (позиция %d)", refCh, pos)
			}

			num := int(refCh - '0')
			if num == 0 {
				return nil, fmt.Errorf("Номер группы не может быть 0 (позиция %d)", pos)
			}

			tokens = append(tokens, Token{
				Type:    TBackRefString,
				Pos:     pos - 1,
				GroupID: num,
			})
			pos++

		default:
			return nil, fmt.Errorf("Неожиданный символ %q на позиции %d", ch, pos)
		}
	}

	if bracketBalance != 0 {
		return nil, fmt.Errorf("Несоответствие открывающих и закрывающих скобок, осталось %d незакрытых", bracketBalance)
	}

	return tokens, nil
}
