package db

// generatedAs describes a generated column. Databases often report the
// expression already wrapped in parentheses, so they are added only when needed.
func generatedAs(expression string) string {
	if !isWrappedInParentheses(expression) {
		expression = "(" + expression + ")"
	}
	return "GENERATED ALWAYS AS " + expression
}

// isWrappedInParentheses reports whether the first "(" closes at the last character.
func isWrappedInParentheses(expression string) bool {
	if len(expression) < 2 || expression[0] != '(' {
		return false
	}
	depth := 0
	for i, r := range expression {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i == len(expression)-1
			}
		}
	}
	return false
}
