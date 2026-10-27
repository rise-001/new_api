package billingexpr

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/expr-lang/expr/ast"
)

// Keep the rule shape aligned with the visual editor's conditional multipliers.
var multiplierFactorPattern = regexp.MustCompile(`(?s)^\((.+) \? ([\d.eE+-]+) : 1\)$`)

type multiplierTraceVisitor struct {
	rules map[int]int // Closing-parenthesis source position -> visual rule index.
}

func newMultiplierTraceVisitor(body string) *multiplierTraceVisitor {
	v := &multiplierTraceVisitor{rules: make(map[int]int)}
	source := []rune(body)
	start, depth, ruleIndex := 0, 0, 0
	var quote rune
	escaped := false
	for i := 0; i <= len(source); i++ {
		if i == len(source) || (quote == 0 && depth == 0 && source[i] == '*') {
			end := i
			for end > start && unicode.IsSpace(source[end-1]) {
				end--
			}
			factor := strings.TrimSpace(string(source[start:end]))
			if multiplierFactorPattern.MatchString(factor) {
				v.rules[end-1] = ruleIndex
				ruleIndex++
			}
			start = i + 1
			continue
		}
		char := source[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if char == '\\' && quote != '`' {
				escaped = true
			} else if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '"', '\'', '`':
			quote = char
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		}
	}
	return v
}

// Visit observes the original condition during the same evaluation that charges
// the request. It returns the boolean unchanged, preserving short-circuiting and
// distinguishing a matched 0x/1x rule from an unmatched rule that returns 1.
func (v *multiplierTraceVisitor) Visit(node *ast.Node) {
	conditional, ok := (*node).(*ast.ConditionalNode)
	if !ok {
		return
	}
	index, ok := v.rules[conditional.Location().From]
	if !ok {
		return
	}
	ast.Patch(&conditional.Cond, &ast.CallNode{
		Callee: &ast.IdentifierNode{Value: "__billing_multiplier"},
		Arguments: []ast.Node{
			&ast.IntegerNode{Value: index},
			conditional.Cond,
		},
	})
}
