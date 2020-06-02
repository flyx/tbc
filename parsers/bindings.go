package parsers

import (
	"fmt"

	"github.com/flyx/tbc/data"

	"github.com/yhirose/go-peg"
)

var bindingsParser *peg.Parser

func init() {
	var err error
	bindingsParser, err = peg.NewParser(`
	ROOT        ← BINDING (',' BINDING)*
	BINDING     ← BOUND ':' (AUTOVAR / TYPEDVAR)
	AUTOVAR     ← IDENTIFIER
	TYPEDVAR    ← '(' IDENTIFIER IDENTIFIER ')'
	IDENTIFIER  ← < [a-zA-Z_][a-zA-Z_0-9]* >
	%whitespace ← [ \t]*
	` + boundSyntax)
	if err != nil {
		panic(err)
	}
	registerBinders(bindingsParser)
	g := bindingsParser.Grammar
	g["IDENTIFIER"].Action = func(v *peg.Values, d peg.Any) (peg.Any, error) {
		return v.Token(), nil
	}
	g["TYPEDVAR"].Action = func(v *peg.Values, d peg.Any) (peg.Any, error) {
		switch v.ToStr(1) {
		case "int":
			return data.Variable{Type: data.IntVar, Name: v.ToStr(0)}, nil
		case "string":
			return data.Variable{Type: data.StringVar, Name: v.ToStr(0)}, nil
		case "bool":
			return data.Variable{Type: data.BoolVar, Name: v.ToStr(0)}, nil
		default:
			return nil, fmt.Errorf("unsupported type: %s", v.ToStr(1))
		}
	}
	g["AUTOVAR"].Action = func(v *peg.Values, d peg.Any) (peg.Any, error) {
		return data.Variable{Type: data.AutoVar, Name: v.ToStr(0)}, nil
	}
	g["BINDING"].Action = func(v *peg.Values, d peg.Any) (peg.Any, error) {
		return data.VariableMapping{Value: v.Vs[0].(data.BoundValue), Variable: v.Vs[1].(data.Variable)}, nil
	}
	g["ROOT"].Action = func(v *peg.Values, d peg.Any) (peg.Any, error) {
		ret := make([]data.VariableMapping, v.Len())
		for i, c := range v.Vs {
			ret[i] = c.(data.VariableMapping)
		}
		return ret, nil
	}
}

// ParseBindings parses the content of a tbc:bindings attribute.
// The Path field in the returned VariableMappings is unset.
func ParseBindings(s string) ([]data.VariableMapping, error) {
	ret, err := bindingsParser.ParseAndGetValue(s, nil)
	if err != nil {
		return nil, err
	}
	return ret.([]data.VariableMapping), nil
}