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

func TestAST2(t *testing.T) {
	var tests = []string{
		"# Header 1\n\n----\n\n> quote",
		"<h1>Header 1</h1>\n\n<hr />\n\n<blockquote>\n<p>quote</p>\n</blockquote>\n",

		"# Header\n\n<div><span><em>plain html</em></span></div>",
		"<h1>Header</h1>\n\n<div><span><em>plain html</em></span></div>\n",
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
		//ast := NewParser().parse([]byte(input))
		renderer := HtmlRenderer(UseXHTML, "", "")
		Markdown([]byte(input), renderer, NoExtensions)
		actual := string(render(renderer.GetAST()))
		if actual != expected {
			t.Errorf("\nInput   [%#v]\nExpected[%#v]\nActual  [%#v]",
				candidate, expected, actual)
		}
	}
}
