package blackfriday

import (
	"testing"
)

/*
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
*/

func TestAST2(t *testing.T) {
	var tests = []string{
		"# Header 1\n\n----\n\n> quote",
		"<h1>Header 1</h1>\n\n<hr />\n\n<blockquote>\n<p>quote</p>\n</blockquote>\n",

		"# Header\n\n<div><span><em>plain html</em></span></div>",
		"<h1>Header</h1>\n\n<div><span><em>plain html</em></span></div>\n",

		"* List\n",
		"<ul>\n<li>\n<p>List</p>\n</li>\n</ul>\n",

		"* List\n* Second item",
		"<ul>\n<li>\n<p>List</p>\n</li>\n<li>\n<p>Second item</p>\n</li>\n</ul>\n",

		"B\n-\n",
		"<h2>B</h2>\n",

		"```\nfunc main() {}\n```  ",
		"<pre><code>func main() {}\n</code></pre>\n",

		"``` go\nfunc main() {}\n```  ",
		"<pre><code>func main() {}\n</code></pre>\n",

		"    def foo():\n        pass",
		"<pre><code>def foo():\n    pass\n</code></pre>\n",

		"*foo*",
		"<p><em>foo</em></p>\n",

		"**foo**",
		"<p><strong>foo</strong></p>\n",

		"**foo\nbar**",
		"<p><strong>foo\nbar</strong></p>\n",

		"**What is an algorithm?**\n",
		"<p><strong>What is an algorithm?</strong></p>\n",

		"**foo [bar](/url)**",
		"<p><strong>foo <a href=\"/url\">bar</a></strong></p>\n",

		"*What is A\\* algorithm?*\n",
		"<p><em>What is A* algorithm?</em></p>\n",

		"## *Emphasised* header\n> quote",
		"<h2><em>Emphasised</em> header</h2>\n\n<blockquote>\n<p>quote</p>\n</blockquote>\n",

		// hard line break
		"foo  \nbar",
		"<p>foo<br />\nbar</p>\n",

		// backslash escaping
		"foo\\\nbar",
		"<p>foo<br />\nbar</p>\n",

		"foo\\bar",
		"<p>foo\\bar</p>\n",

		`foo\*bar`,
		"<p>foo*bar</p>\n",

		// backticks
		"foo `moo",
		"<p>foo `moo</p>\n",

		"foo `bar`",
		"<p>foo <code>bar</code></p>\n",

		"some ``  spaced    out   code ``",
		"<p>some <code>spaced out code</code></p>\n",

		// autolink
		"an email <some@one.com>\n",
		"<p>an email <a href=\"mailto:some@one.com\">some@one.com</a></p>\n",

		// XXX: Note there's a difference in behavior between Common Mark and
		// current Blackfriday behavior here: current Blackfriday strips
		// "mailto:" part in the line text, while Common Mark preserves it.
		"an email <mailto:some@one.com>\n",
		"<p>an email <a href=\"mailto:some@one.com\">mailto:some@one.com</a></p>\n",

		"some <http://hyperlink.com>",
		"<p>some <a href=\"http://hyperlink.com\">http://hyperlink.com</a></p>\n",

		// inline html
		"inline <span>html</span>",
		"<p>inline <span>html</span></p>\n",

		"Hello <!-- there -->",
		"<p>Hello <!-- there --></p>\n",

		// entities
		"&lt;&gt;",
		"<p>&lt;&gt;</p>\n",

		"&#35;",
		"<p>#</p>\n",

		"&amp;&quot;&lt;&gt;",
		"<p>&amp;&quot;&lt;&gt;</p>\n",

		// links
		"![foo](/bar/ \"title\")\n",
		"<p><img src=\"/bar/\" alt=\"foo\" title=\"title\" /></p>\n",

		"![foo](/bar/)\n",
		"<p><img src=\"/bar/\" alt=\"foo\" /></p>\n",

		"[link](url)\n",
		"<p><a href=\"url\">link</a></p>\n",
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
		actual := string(render_CommonMark(renderer.GetAST()))
		if actual != expected {
			t.Errorf("\nInput   [%#v]\nExpected[%#v]\nActual  [%#v]",
				candidate, expected, actual)
		}
	}
}
