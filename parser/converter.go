package parser

import (
	"strconv"
	"unicode"
)

type Converter struct {
	Cfg                map[string][][]string
	NTMap              map[*ASTNode]string
	ConcatCount        int
	UnionCount         int
	StarCount          int
	CaptureCount       int
	EscapeCaptureCount int
	StringRefCount     int
	ExprRefCount       int
}

func NewConverter() *Converter {
	return &Converter{
		Cfg:                make(map[string][][]string),
		NTMap:              make(map[*ASTNode]string),
		ConcatCount:        0,
		UnionCount:         0,
		StarCount:          0,
		CaptureCount:       0,
		EscapeCaptureCount: 0,
		StringRefCount:     0,
		ExprRefCount:       0,
	}
}

func (c *Converter) GetNT(node *ASTNode) string {
	if nt, ok := c.NTMap[node]; ok {
		return nt
	}
	switch node.Type {
	case NodeChar:
		nt := string(unicode.ToUpper(node.Value))
		return nt
	case NodeUnion:
		c.UnionCount++
		count := strconv.Itoa(c.UnionCount)
		c.NTMap[node] = "UN" + count
		return "UN" + count
	case NodeConcat:
		c.ConcatCount++
		count := strconv.Itoa(c.ConcatCount)
		c.NTMap[node] = "CC" + count
		return "CC" + count
	case NodeStar:
		c.StarCount++
		count := strconv.Itoa(c.StarCount)
		c.NTMap[node] = "ST" + count
		return "ST" + count
	case NodeCreateCaptureGroup:
		c.CaptureCount++
		count := strconv.Itoa(c.CaptureCount)
		c.NTMap[node] = "CG" + count
		return "CG" + count
	case NodeCreateEscapeCaptureGroup:
		c.EscapeCaptureCount++
		count := strconv.Itoa(c.EscapeCaptureCount)
		c.NTMap[node] = "EG" + count
		return "EG" + count
	case NodeRefExpression:
		c.ExprRefCount++
		count := strconv.Itoa(c.ExprRefCount)
		c.NTMap[node] = "RE" + count
		return "RE" + count
	case NodeRefString:
		c.StringRefCount++
		count := strconv.Itoa(c.StringRefCount)
		c.NTMap[node] = "RS" + count
		return "RS" + count
	}
	return ""
}

func ConvertToCFG(node *ASTNode) map[string][][]string {
	converter := NewConverter()
	converter.NTMap[node] = "START"
	converter.convert(node)
	return converter.Cfg
}

func (c *Converter) convert(node *ASTNode) {
	switch node.Type {
	case NodeChar:
		nt := c.GetNT(node)
		if _, ok := c.Cfg[nt]; !ok {
			c.Cfg[nt] = append(make([][]string, 0), []string{string(node.Value)})
		}
	case NodeUnion:
		unionNT := c.GetNT(node)
		rules := make([][]string, 0)
		for _, child := range node.Children {
			rules = append(rules, []string{c.GetNT(child)})
			c.convert(child)
		}
		c.Cfg[unionNT] = rules
	case NodeConcat:
		concatNT := c.GetNT(node)
		rule := make([]string, 0)
		for _, child := range node.Children {
			rule = append(rule, c.GetNT(child))
			c.convert(child)
		}
		c.Cfg[concatNT] = [][]string{rule}
	case NodeStar:
		starNT := c.GetNT(node)
		rule := c.GetNT(node.Children[0])
		c.convert(node.Children[0])
		c.Cfg[starNT] = [][]string{{rule, starNT}, {"ε"}}
	case NodeCreateCaptureGroup:
		captureGroupNT := c.GetNT(node)
		rule := c.GetNT(node.Children[0])
		c.convert(node.Children[0])
		c.Cfg[captureGroupNT] = [][]string{{rule}}
	case NodeCreateEscapeCaptureGroup:
		escapeCaptureGroupNT := c.GetNT(node)
		rule := c.GetNT(node.Children[0])
		c.convert(node.Children[0])
		c.Cfg[escapeCaptureGroupNT] = [][]string{{rule}}
	case NodeRefExpression:
		refNT := c.GetNT(node)
		rule := c.GetNT(node.Children[0])
		c.Cfg[refNT] = [][]string{{rule}}
	case NodeRefString:
		refStringNT := c.GetNT(node)
		rule := c.GetNT(node.Children[0])
		c.Cfg[refStringNT] = [][]string{{rule}}
	}
}
