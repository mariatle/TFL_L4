package parser

import (
	"fmt"
	"strings"
)

/*
[rg] ::= [rg][rg] | [rg]|[rg] | ([rg]) | (? :[rg]) | [rg]* | ([num]) | [a − z]
[num] ::= [1 − 9]
[rg] ::= \[num] - это ссылка на захваченную СТРОКУ
[rg] ::= (?[num]) - ссылка на захваченное выражение
*/

type NodeType int

const (
	NodeUnion NodeType = iota
	NodeConcat
	NodeCreateCaptureGroup
	NodeCreateEscapeCaptureGroup
	NodeStar
	NodeChar
	NodeRefString
	NodeRefExpression
)

type Parser struct {
	tokens        []Token
	captureGroups map[int]*ASTNode
	initialized   map[int]bool
	visited       map[*ASTNode]bool
	pos           int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens:        tokens,
		pos:           0,
		captureGroups: make(map[int]*ASTNode),
		initialized:   make(map[int]bool),
		visited:       make(map[*ASTNode]bool),
	}
}

func (p *Parser) peek() (Token, error) {
	if p.pos >= len(p.tokens) {
		return Token{}, fmt.Errorf("unexpected EOL")
	}
	return p.tokens[p.pos], nil
}

func (p *Parser) next() {
	p.pos++
}

type ASTNode struct {
	Type     NodeType
	Children []*ASTNode
	Value    rune
	RefNum   int
}

func (n *ASTNode) String() string {
	var builder strings.Builder
	n.buildString(&builder, 0)
	return builder.String()
}

func (n *ASTNode) buildString(builder *strings.Builder, level int) {
	indent := strings.Repeat("  ", level)
	switch n.Type {
	case NodeUnion:
		builder.WriteString(fmt.Sprintf("%sUnion\n", indent))
	case NodeConcat:
		builder.WriteString(fmt.Sprintf("%sConcat\n", indent))
	case NodeCreateCaptureGroup:
		builder.WriteString(fmt.Sprintf("%sCreateCaptureGroup (GroupID: %d)\n", indent, n.RefNum))
	case NodeCreateEscapeCaptureGroup:
		builder.WriteString(fmt.Sprintf("%sCreateEscapeCaptureGroup\n", indent))
	case NodeStar:
		builder.WriteString(fmt.Sprintf("%sStar\n", indent))
	case NodeChar:
		builder.WriteString(fmt.Sprintf("%sChar ('%c')\n", indent, n.Value))
	case NodeRefString:
		builder.WriteString(fmt.Sprintf("%sRefString (GroupID: %d)\n", indent, n.RefNum))
	case NodeRefExpression:
		builder.WriteString(fmt.Sprintf("%sRefExpression (GroupID: %d)\n", indent, n.RefNum))
	default:
		builder.WriteString(fmt.Sprintf("%sUnknownNode\n", indent))
	}

	for _, child := range n.Children {
		if n.Type == NodeRefExpression || n.Type == NodeRefString {
			continue
		}
		child.buildString(builder, level+1)
	}
}

type Set map[int]bool

func newSet() Set {
	return make(Set)
}

func (s Set) clone() Set {
	clonedSet := newSet()
	for k, v := range s {
		clonedSet[k] = v
	}
	return clonedSet
}

func intersect(set1, set2 Set) Set {
	res := make(Set)
	for k, v := range set1 {
		if v == set2[k] {
			res[k] = v
		}
	}
	return res
}

func union(set1, set2 Set) Set {
	res := make(Set)
	for k, v := range set1 {
		if v == true {
			res[k] = true
		}
	}
	for k, v := range set2 {
		if v == true {
			res[k] = true
		}
	}
	return res
}

func (p *Parser) checkRefs(node *ASTNode, initSet Set) (Set, error) {
	p.visited[node] = true
	switch node.Type {
	case NodeRefExpression:
		if p.captureGroups[node.RefNum] == nil {
			return nil, fmt.Errorf("группа захвата %d не существует", node.RefNum)
		}
		if len(node.Children) == 0 {
			node.Children = append(node.Children, p.captureGroups[node.RefNum])
		}
		if p.visited[p.captureGroups[node.RefNum]] {
			return newSet(), nil
		}
		return p.checkRefs(p.captureGroups[node.RefNum], initSet)
	case NodeRefString:
		if !initSet[node.RefNum] {
			return nil, fmt.Errorf("строка определяемая группой захвата %d не инициализирована на момент ссылки на строку", node.RefNum)
		}
		node.Children = append(node.Children, p.captureGroups[node.RefNum])
		return newSet(), nil
	case NodeConcat:
		leftSet, err := p.checkRefs(node.Children[0], initSet)
		if err != nil {
			return nil, err
		}
		rightSet, err := p.checkRefs(node.Children[1], leftSet)
		if err != nil {
			return nil, err
		}
		return rightSet, nil
	case NodeCreateEscapeCaptureGroup:
		return p.checkRefs(node.Children[0], initSet)
	case NodeCreateCaptureGroup:
		groupSet, err := p.checkRefs(node.Children[0], initSet)
		if err != nil {
			return nil, err
		}
		currentSet := groupSet.clone()
		currentSet[node.RefNum] = true
		return union(groupSet, currentSet), nil
	case NodeUnion:
		leftSet, err := p.checkRefs(node.Children[0], initSet)
		if err != nil {
			return nil, err
		}
		rightSet, err := p.checkRefs(node.Children[1], initSet)
		if err != nil {
			return nil, err
		}
		return intersect(leftSet, rightSet), nil
	// Т.к. у нас м.б. 0 повторений, то автоматически считаем, что все потомки неинициализированы, поэтому не проверяем
	case NodeStar:
		return newSet(), nil
	case NodeChar:
		return newSet(), nil
	default:
		return nil, fmt.Errorf("неизвестная нода %v", node.Type)
	}
}
func ConvertToAST(tokens []Token) (*ASTNode, error) {
	parser := NewParser(tokens)
	node, err := parser.parseUnion()
	if err != nil {
		return nil, err
	}
	_, err = parser.checkRefs(node, newSet())
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (p *Parser) parseUnion() (*ASTNode, error) {
	first, err := p.parseConcat()
	if err != nil {
		return nil, err
	}
	nextToken, err := p.peek()
	if err != nil {
		return first, nil
	}
	if nextToken.Type == TUnion {
		p.next()
		second, err := p.parseUnion()
		if err != nil {
			return nil, err
		}
		return &ASTNode{
			Type:     NodeUnion,
			Children: []*ASTNode{first, second},
		}, nil
	} else {
		return first, nil
	}
}

func (p *Parser) parseConcat() (*ASTNode, error) {
	first, err := p.parseStar()
	if err != nil {
		return nil, err
	}
	nextToken, err := p.peek()
	if err != nil {
		return first, nil
	}
	if nextToken.Type != TUnion && nextToken.Type != TRightBracket {
		second, err := p.parseConcat()
		if err != nil {
			return nil, err
		}
		return &ASTNode{
			Type:     NodeConcat,
			Children: []*ASTNode{first, second},
		}, nil
	} else {
		return first, nil
	}
}
func (p *Parser) parseStar() (*ASTNode, error) {
	node, err := p.parseBase()
	if err != nil {
		return nil, err
	}
	for val, err := p.peek(); err == nil && val.Type == TStar; val, err = p.peek() {
		p.next()
		node = &ASTNode{
			Type:     NodeStar,
			Children: []*ASTNode{node},
		}
	}
	return node, nil
}

func (p *Parser) parseBase() (*ASTNode, error) {
	token, err := p.peek()
	if err != nil {
		return nil, err
	}
	p.next()
	switch token.Type {
	case TChar:
		return &ASTNode{Type: NodeChar, Value: token.Char}, nil
	case TCreateCaptureGroup:
		groupNum := len(p.captureGroups) + 1
		p.captureGroups[groupNum] = nil
		value, err := p.parseUnion()
		if err != nil {
			return nil, err
		}
		rb, err := p.peek()
		if err != nil {
			return nil, err
		}
		p.captureGroups[groupNum] = value
		if rb.Type != TRightBracket {
			return nil, fmt.Errorf("ожидалась закрывающая скобка на позиции %d", rb.Pos)
		}
		p.next()
		return &ASTNode{Type: NodeCreateCaptureGroup, Children: []*ASTNode{value}, RefNum: groupNum}, nil
	case TEscapeCaptureGroup:
		value, err := p.parseUnion()
		if err != nil {
			return nil, err
		}
		rb, err := p.peek()
		if err != nil {
			return nil, err
		}
		if rb.Type != TRightBracket {
			return nil, fmt.Errorf("ожидалась закрывающая скобка на позиции %d", rb.Pos)
		}
		p.next()
		return &ASTNode{Type: NodeCreateEscapeCaptureGroup, Children: []*ASTNode{value}}, nil
	case TBackRefExpression:
		return &ASTNode{Type: NodeRefExpression, RefNum: token.GroupID}, nil
	case TBackRefString:
		return &ASTNode{Type: NodeRefString, RefNum: token.GroupID}, nil
	default:
		return nil, fmt.Errorf("неожиданный токен %s", token.String())
	}
}
