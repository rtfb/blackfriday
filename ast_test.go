package blackfriday

import (
	"testing"
)

func TestAST(t *testing.T) {
	var tests = []string{
		"# Header 1\n",
		`Document("")
	Header("Header 1")
`,
	}
	var candidate string
	// catch and report panics
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("\npanic while processing [%#v]: %s\n", candidate, err)
		}
	}()
	for i := 0; i+1 < len(tests); i += 2 {
		input := tests[i]
		candidate = input
		expected := tests[i+1]
		ast := NewParser().parse([]byte(input))
		actual := dumpString(ast)
		if actual != expected {
			t.Errorf("\nInput   [%#v]\nExpected[%#v]\nActual  [%#v]",
				candidate, expected, actual)
		}
	}
}
