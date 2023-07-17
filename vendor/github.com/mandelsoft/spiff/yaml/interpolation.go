package yaml

import (
	"encoding/json"
	"strings"
)

func StringToExpression(s string) string {
	str, expr := convertToExpression(s, false)
	if expr == nil {
		return *str
	}
	return "(( " + *expr + " ))"
}

func convertToExpression(s string, unescape bool) (*string, *string) {
	mask := false
	quote := false
	start := false
	ob := 0
	lvl := 0
	expr := ""
	str := ""
	result := ""
	found := false

	for _, c := range s {
		if start {
			if quote {
				// in quotes in expr
				switch c {
				case '"':
					if !mask {
						quote = false
					}
					mask = false
				case '\\':
					mask = !mask
				default:
					mask = false
				}
				expr = expr + string(c)
			} else {
				// in expr outside quotes
				switch c {
				case '(':
					lvl++
				case ')':
					if lvl > 0 {
						lvl--
					} else {
						ob++
						if ob == 2 {
							start = false
							ob = 0
							found = addExpr(&result, &str, &expr, false, unescape) || found
						}
					}
				case '"':
					quote = true
				}
				if start {
					expr = expr + string(c)
				}
				if c != ')' {
					ob = 0
				}
			}
		} else {
			// regular string
			switch c {
			case '(':
				ob++
			default:
				if ob >= 2 {
					start = true
					str = str[:len(str)-2]
					expr = string(c)
					ob = 0
				}
			}
			if !start {
				str = str + string(c)
				if c != '(' {
					ob = 0
				}
			}
		}
	}

	if start {
		str = str + "((" + expr
		expr = ""
	}
	found = addExpr(&result, &str, &expr, true, unescape) || found
	if found {
		return nil, &result
	}
	return &str, nil
}

func addExpr(result, str, expr *string, final, unescape bool) bool {
	if unescape && strings.HasPrefix(*expr, "!") {
		*str += "((" + (*expr)[1:] + ")"
		*expr = ""
	}
	if strings.HasPrefix(*expr, "!") {
		*str += "((" + *expr + ")"
		*expr = ""
	}
	if *expr == "" && (*result == "" || !final) {
		return false
	}
	if *str != "" {
		r, _ := json.Marshal(*str)
		if *result != "" {
			*result += " "
		}
		*result = *result + string(r)
	}
	if *expr != "" {
		*expr = strings.TrimSpace((*expr)[:len(*expr)-1])
	}
	if *result != "" && *expr != "" {
		*result += " "
	}
	*result += *expr
	*expr = ""
	*str = ""
	return true
}
