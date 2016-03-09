//
// Blackfriday Markdown Processor
// Available at http://github.com/russross/blackfriday
//
// Copyright © 2011 Russ Ross <russ@russross.com>.
// Distributed under the Simplified BSD License.
// See README.md for details.
//

//
//
// HTML rendering backend
//
//

package blackfriday

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

type HtmlFlags int

// Html renderer configuration options.
const (
	HtmlFlagsNone           HtmlFlags = 0
	SkipHTML                HtmlFlags = 1 << iota // Skip preformatted HTML blocks
	SkipStyle                                     // Skip embedded <style> elements
	SkipImages                                    // Skip embedded images
	SkipLinks                                     // Skip all links
	Safelink                                      // Only link to trusted protocols
	NofollowLinks                                 // Only link with rel="nofollow"
	NoreferrerLinks                               // Only link with rel="noreferrer"
	HrefTargetBlank                               // Add a blank target
	Toc                                           // Generate a table of contents
	OmitContents                                  // Skip the main contents (for a standalone table of contents)
	CompletePage                                  // Generate a complete HTML page
	UseXHTML                                      // Generate XHTML output instead of HTML
	UseSmartypants                                // Enable smart punctuation substitutions
	SmartypantsFractions                          // Enable smart fractions (with UseSmartypants)
	SmartypantsDashes                             // Enable smart dashes (with UseSmartypants)
	SmartypantsLatexDashes                        // Enable LaTeX-style dashes (with UseSmartypants)
	SmartypantsAngledQuotes                       // Enable angled double quotes (with UseSmartypants) for double quotes rendering
	FootnoteReturnLinks                           // Generate a link at the end of a footnote to return to the source

	TagName               = "[A-Za-z][A-Za-z0-9-]*"
	AttributeName         = "[a-zA-Z_:][a-zA-Z0-9:._-]*"
	UnquotedValue         = "[^\"'=<>`\\x00-\\x20]+"
	SingleQuotedValue     = "'[^']*'"
	DoubleQuotedValue     = "\"[^\"]*\""
	AttributeValue        = "(?:" + UnquotedValue + "|" + SingleQuotedValue + "|" + DoubleQuotedValue + ")"
	AttributeValueSpec    = "(?:" + "\\s*=" + "\\s*" + AttributeValue + ")"
	Attribute             = "(?:" + "\\s+" + AttributeName + AttributeValueSpec + "?)"
	OpenTag               = "<" + TagName + Attribute + "*" + "\\s*/?>"
	CloseTag              = "</" + TagName + "\\s*[>]"
	HTMLComment           = "<!---->|<!--(?:-?[^>-])(?:-?[^-])*-->"
	ProcessingInstruction = "[<][?].*?[?][>]"
	Declaration           = "<![A-Z]+" + "\\s+[^>]*>"
	CDATA                 = "<!\\[CDATA\\[[\\s\\S]*?\\]\\]>"
	HTMLTag               = "(?:" + OpenTag + "|" + CloseTag + "|" + HTMLComment + "|" +
		ProcessingInstruction + "|" + Declaration + "|" + CDATA + ")"
)

var (
	alignments = []string{
		"left",
		"right",
		"center",
	}

	// TODO: improve this regexp to catch all possible entities:
	htmlEntity = regexp.MustCompile(`&[a-z]{2,5};`)
	reHtmlTag  = regexp.MustCompile("(?i)^" + HTMLTag)
)

type HtmlRendererParameters struct {
	// Prepend this text to each relative URL.
	AbsolutePrefix string
	// Add this text to each footnote anchor, to ensure uniqueness.
	FootnoteAnchorPrefix string
	// Show this text inside the <a> tag for a footnote return link, if the
	// HTML_FOOTNOTE_RETURN_LINKS flag is enabled. If blank, the string
	// <sup>[return]</sup> is used.
	FootnoteReturnLinkContents string
	// If set, add this text to the front of each Header ID, to ensure
	// uniqueness.
	HeaderIDPrefix string
	// If set, add this text to the back of each Header ID, to ensure uniqueness.
	HeaderIDSuffix string
}

// Html is a type that implements the Renderer interface for HTML output.
//
// Do not create this directly, instead use the HtmlRenderer function.
type Html struct {
	flags    HtmlFlags
	closeTag string // how to end singleton tags: either " />" or ">"
	title    string // document title
	css      string // optional css file url (used with HTML_COMPLETE_PAGE)

	parameters HtmlRendererParameters

	// table of contents data
	tocMarker    int
	headerCount  int
	currentLevel int
	toc          *bytes.Buffer

	// Track header IDs to prevent ID collision in a single generation.
	headerIDs map[string]int

	smartypants *smartypantsRenderer
	w           HtmlWriter
	ast         *Node
}

const (
	xhtmlClose = " />"
	htmlClose  = ">"
)

// HtmlRenderer creates and configures an Html object, which
// satisfies the Renderer interface.
//
// flags is a set of HtmlFlags ORed together.
// title is the title of the document, and css is a URL for the document's
// stylesheet.
// title and css are only used when HTML_COMPLETE_PAGE is selected.
func HtmlRenderer(flags HtmlFlags, title string, css string) Renderer {
	return HtmlRendererWithParameters(flags, title, css, HtmlRendererParameters{})
}

type HtmlWriter struct {
	output      bytes.Buffer
	captureBuff *bytes.Buffer
	copyBuff    *bytes.Buffer
	dirty       bool
}

func (w *HtmlWriter) Write(p []byte) (n int, err error) {
	w.dirty = true
	if w.copyBuff != nil {
		w.copyBuff.Write(p)
	}
	if w.captureBuff != nil {
		w.captureBuff.Write(p)
		return
	}
	return w.output.Write(p)
}

func (w *HtmlWriter) WriteString(s string) (n int, err error) {
	w.dirty = true
	if w.copyBuff != nil {
		w.copyBuff.WriteString(s)
	}
	if w.captureBuff != nil {
		w.captureBuff.WriteString(s)
		return
	}
	return w.output.WriteString(s)
}

func (w *HtmlWriter) WriteByte(b byte) error {
	w.dirty = true
	if w.copyBuff != nil {
		w.copyBuff.WriteByte(b)
	}
	if w.captureBuff != nil {
		return w.captureBuff.WriteByte(b)
	}
	return w.output.WriteByte(b)
}

// Writes out a newline if the output is not pristine. Used at the beginning of
// every rendering func
func (w *HtmlWriter) Newline() {
	if w.dirty {
		w.WriteByte('\n')
	}
}

func (r *Html) CaptureWrites(processor func()) []byte {
	var output bytes.Buffer
	// preserve old captureBuff state for possible nested captures:
	tmp := r.w.captureBuff
	tmpd := r.w.dirty
	r.w.captureBuff = &output
	r.w.dirty = false
	processor()
	// restore:
	r.w.captureBuff = tmp
	r.w.dirty = tmpd
	return output.Bytes()
}

func (r *Html) CopyWrites(processor func()) []byte {
	var output bytes.Buffer
	r.w.copyBuff = &output
	processor()
	r.w.copyBuff = nil
	return output.Bytes()
}

func (r *Html) Write(b []byte) (int, error) {
	return r.w.Write(b)
}

func (r *Html) GetResult() []byte {
	return r.w.output.Bytes()
}

func HtmlRendererWithParameters(flags HtmlFlags, title string,
	css string, renderParameters HtmlRendererParameters) Renderer {
	// configure the rendering engine
	closeTag := htmlClose
	if flags&UseXHTML != 0 {
		closeTag = xhtmlClose
	}

	if renderParameters.FootnoteReturnLinkContents == "" {
		renderParameters.FootnoteReturnLinkContents = `<sup>[return]</sup>`
	}

	var writer HtmlWriter
	return &Html{
		flags:      flags,
		closeTag:   closeTag,
		title:      title,
		css:        css,
		parameters: renderParameters,

		headerCount:  0,
		currentLevel: 0,
		toc:          new(bytes.Buffer),

		headerIDs: make(map[string]int),

		smartypants: smartypants(flags),
		w:           writer,
	}
}

// Using if statements is a bit faster than a switch statement. As the compiler
// improves, this should be unnecessary this is only worthwhile because
// attrEscape is the single largest CPU user in normal use.
// Also tried using map, but that gave a ~3x slowdown.
func escapeSingleChar(char byte) (string, bool) {
	if char == '"' {
		return "&quot;", true
	}
	if char == '&' {
		return "&amp;", true
	}
	if char == '<' {
		return "&lt;", true
	}
	if char == '>' {
		return "&gt;", true
	}
	return "", false
}

func (r *Html) attrEscape(src []byte) {
	/*
		org := 0
		for i, ch := range src {
			if entity, ok := escapeSingleChar(ch); ok {
				if i > org {
					// copy all the normal characters since the last escape
					r.w.Write(src[org:i])
				}
				org = i + 1
				r.w.WriteString(entity)
			}
		}
		if org < len(src) {
			r.w.Write(src[org:])
		}
	*/
	r.w.Write(attrEscape2(src))
}

func attrEscape2(src []byte) []byte {
	unesc := []byte(html.UnescapeString(string(src)))
	esc1 := []byte(html.EscapeString(string(unesc)))
	esc2 := bytes.Replace(esc1, []byte("&#34;"), []byte("&quot;"), -1)
	return bytes.Replace(esc2, []byte("&#39;"), []byte{'\''}, -1)
}

func (r *Html) entityEscapeWithSkip(src []byte, skipRanges [][]int) {
	end := 0
	for _, rang := range skipRanges {
		r.attrEscape(src[end:rang[0]])
		r.w.Write(src[rang[0]:rang[1]])
		end = rang[1]
	}
	r.attrEscape(src[end:])
}

func (r *Html) GetFlags() HtmlFlags {
	return r.flags
}

func (r *Html) SetAST(ast *Node) {
	r.ast = ast
}

func (r *Html) GetAST() *Node {
	return r.ast
}

func (r *Html) TitleBlock(text []byte) {
	text = bytes.TrimPrefix(text, []byte("% "))
	text = bytes.Replace(text, []byte("\n% "), []byte("\n"), -1)
	r.w.WriteString("<h1 class=\"title\">")
	r.w.Write(text)
	r.w.WriteString("\n</h1>")
}

func (r *Html) BeginHeader(level int, id string) {
	r.w.Newline()

	if id == "" && r.flags&Toc != 0 {
		id = fmt.Sprintf("toc_%d", r.headerCount)
	}

	if id != "" {
		id = r.ensureUniqueHeaderID(id)

		if r.parameters.HeaderIDPrefix != "" {
			id = r.parameters.HeaderIDPrefix + id
		}

		if r.parameters.HeaderIDSuffix != "" {
			id = id + r.parameters.HeaderIDSuffix
		}

		r.w.WriteString(fmt.Sprintf("<h%d id=\"%s\">", level, id))
	} else {
		r.w.WriteString(fmt.Sprintf("<h%d>", level))
	}
}

func (r *Html) EndHeader(level int, id string, header []byte) {
	// are we building a table of contents?
	if r.flags&Toc != 0 {
		r.TocHeaderWithAnchor(header, level, id)
	}

	r.w.WriteString(fmt.Sprintf("</h%d>\n", level))
}

func (r *Html) BlockHtml(text []byte) {
	if r.flags&SkipHTML != 0 {
		return
	}

	r.w.Newline()
	r.w.Write(text)
	r.w.WriteByte('\n')
}

func (r *Html) HRule() {
	r.w.Newline()
	r.w.WriteString("<hr")
	r.w.WriteString(r.closeTag)
	r.w.WriteByte('\n')
}

func (r *Html) BlockCode(text []byte, lang string) {
	r.w.Newline()

	// parse out the language names/classes
	count := 0
	for _, elt := range strings.Fields(lang) {
		if elt[0] == '.' {
			elt = elt[1:]
		}
		if len(elt) == 0 {
			continue
		}
		if count == 0 {
			r.w.WriteString("<pre><code class=\"language-")
		} else {
			r.w.WriteByte(' ')
		}
		r.attrEscape([]byte(elt))
		count++
	}

	if count == 0 {
		r.w.WriteString("<pre><code>")
	} else {
		r.w.WriteString("\">")
	}

	r.attrEscape(text)
	r.w.WriteString("</code></pre>\n")
}

func (r *Html) BlockQuote(text []byte) {
	r.w.Newline()
	r.w.WriteString("<blockquote>\n")
	r.w.Write(text)
	r.w.WriteString("</blockquote>\n")
}

func (r *Html) Table(header []byte, body []byte, columnData []int) {
	r.w.Newline()
	r.w.WriteString("<table>\n<thead>\n")
	r.w.Write(header)
	r.w.WriteString("</thead>\n\n<tbody>\n")
	r.w.Write(body)
	r.w.WriteString("</tbody>\n</table>\n")
}

func (r *Html) TableRow(text []byte) {
	r.w.Newline()
	r.w.WriteString("<tr>\n")
	r.w.Write(text)
	r.w.WriteString("\n</tr>\n")
}

func leadingNewline(out *bytes.Buffer) {
	if out.Len() > 0 {
		out.WriteByte('\n')
	}
}

func (r *Html) TableHeaderCell(out *bytes.Buffer, text []byte, align int) {
	leadingNewline(out)
	switch align {
	case TableAlignmentLeft:
		out.WriteString("<th align=\"left\">")
	case TableAlignmentRight:
		out.WriteString("<th align=\"right\">")
	case TableAlignmentCenter:
		out.WriteString("<th align=\"center\">")
	default:
		out.WriteString("<th>")
	}

	out.Write(text)
	out.WriteString("</th>")
}

func (r *Html) TableCell(out *bytes.Buffer, text []byte, align int) {
	leadingNewline(out)
	switch align {
	case TableAlignmentLeft:
		out.WriteString("<td align=\"left\">")
	case TableAlignmentRight:
		out.WriteString("<td align=\"right\">")
	case TableAlignmentCenter:
		out.WriteString("<td align=\"center\">")
	default:
		out.WriteString("<td>")
	}

	out.Write(text)
	out.WriteString("</td>")
}

func (r *Html) BeginFootnotes() {
	r.w.WriteString("<div class=\"footnotes\">\n")
	r.HRule()
	r.BeginList(ListTypeOrdered)
}

func (r *Html) EndFootnotes() {
	r.EndList(ListTypeOrdered)
	r.w.WriteString("</div>\n")
}

func (r *Html) FootnoteItem(name, text []byte, flags ListType) {
	if flags&ListItemContainsBlock != 0 || flags&ListItemBeginningOfList != 0 {
		r.w.Newline()
	}
	slug := slugify(name)
	r.w.WriteString(`<li id="`)
	r.w.WriteString(`fn:`)
	r.w.WriteString(r.parameters.FootnoteAnchorPrefix)
	r.w.Write(slug)
	r.w.WriteString(`">`)
	r.w.Write(text)
	if r.flags&FootnoteReturnLinks != 0 {
		r.w.WriteString(` <a class="footnote-return" href="#`)
		r.w.WriteString(`fnref:`)
		r.w.WriteString(r.parameters.FootnoteAnchorPrefix)
		r.w.Write(slug)
		r.w.WriteString(`">`)
		r.w.WriteString(r.parameters.FootnoteReturnLinkContents)
		r.w.WriteString(`</a>`)
	}
	r.w.WriteString("</li>\n")
}

func (r *Html) BeginList(flags ListType) {
	r.w.Newline()

	if flags&ListTypeDefinition != 0 {
		r.w.WriteString("<dl>")
	} else if flags&ListTypeOrdered != 0 {
		r.w.WriteString("<ol>")
	} else {
		r.w.WriteString("<ul>")
	}
}

func (r *Html) EndList(flags ListType) {
	if flags&ListTypeDefinition != 0 {
		r.w.WriteString("</dl>\n")
	} else if flags&ListTypeOrdered != 0 {
		r.w.WriteString("</ol>\n")
	} else {
		r.w.WriteString("</ul>\n")
	}
}

func (r *Html) ListItem(text []byte, flags ListType) {
	if (flags&ListItemContainsBlock != 0 && flags&ListTypeDefinition == 0) ||
		flags&ListItemBeginningOfList != 0 {
		r.w.Newline()
	}
	if flags&ListTypeTerm != 0 {
		r.w.WriteString("<dt>")
	} else if flags&ListTypeDefinition != 0 {
		r.w.WriteString("<dd>")
	} else {
		r.w.WriteString("<li>")
	}
	r.w.Write(text)
	if flags&ListTypeTerm != 0 {
		r.w.WriteString("</dt>\n")
	} else if flags&ListTypeDefinition != 0 {
		r.w.WriteString("</dd>\n")
	} else {
		r.w.WriteString("</li>\n")
	}
}

func (r *Html) BeginParagraph() {
	r.w.Newline()
	r.w.WriteString("<p>")
}

func (r *Html) EndParagraph() {
	r.w.WriteString("</p>\n")
}

func (r *Html) AutoLink(link []byte, kind LinkType) {
	skipRanges := htmlEntity.FindAllIndex(link, -1)
	if r.flags&Safelink != 0 && !isSafeLink(link) && kind != LinkTypeEmail {
		// mark it but don't link it if it is not a safe link: no smartypants
		r.w.WriteString("<tt>")
		r.entityEscapeWithSkip(link, skipRanges)
		r.w.WriteString("</tt>")
		return
	}

	r.w.WriteString("<a href=\"")
	if kind == LinkTypeEmail {
		r.w.WriteString("mailto:")
	} else {
		r.maybeWriteAbsolutePrefix(link)
	}

	r.entityEscapeWithSkip(link, skipRanges)

	var relAttrs []string
	if r.flags&NofollowLinks != 0 && !isRelativeLink(link) {
		relAttrs = append(relAttrs, "nofollow")
	}
	if r.flags&NoreferrerLinks != 0 && !isRelativeLink(link) {
		relAttrs = append(relAttrs, "noreferrer")
	}
	if len(relAttrs) > 0 {
		r.w.WriteString(fmt.Sprintf("\" rel=\"%s", strings.Join(relAttrs, " ")))
	}

	// blank target only add to external link
	if r.flags&HrefTargetBlank != 0 && !isRelativeLink(link) {
		r.w.WriteString("\" target=\"_blank")
	}

	r.w.WriteString("\">")

	// Pretty print: if we get an email address as
	// an actual URI, e.g. `mailto:foo@bar.com`, we don't
	// want to print the `mailto:` prefix
	switch {
	case bytes.HasPrefix(link, []byte("mailto://")):
		r.attrEscape(link[len("mailto://"):])
	case bytes.HasPrefix(link, []byte("mailto:")):
		r.attrEscape(link[len("mailto:"):])
	default:
		r.entityEscapeWithSkip(link, skipRanges)
	}

	r.w.WriteString("</a>")
}

func (r *Html) CodeSpan(text []byte) {
	r.w.WriteString("<code>")
	r.attrEscape(text)
	r.w.WriteString("</code>")
}

func (r *Html) DoubleEmphasis(text []byte) {
	r.w.WriteString("<strong>")
	r.w.Write(text)
	r.w.WriteString("</strong>")
}

func (r *Html) Emphasis(text []byte) {
	if len(text) == 0 {
		return
	}
	r.w.WriteString("<em>")
	r.w.Write(text)
	r.w.WriteString("</em>")
}

func (r *Html) maybeWriteAbsolutePrefix(link []byte) {
	if r.parameters.AbsolutePrefix != "" && isRelativeLink(link) && link[0] != '.' {
		r.w.WriteString(r.parameters.AbsolutePrefix)
		if link[0] != '/' {
			r.w.WriteByte('/')
		}
	}
}

func (r *Html) Image(link []byte, title []byte, alt []byte) {
	if r.flags&SkipImages != 0 {
		return
	}

	r.w.WriteString("<img src=\"")
	r.maybeWriteAbsolutePrefix(link)
	r.attrEscape(link)
	r.w.WriteString("\" alt=\"")
	if len(alt) > 0 {
		r.attrEscape(alt)
	}
	if len(title) > 0 {
		r.w.WriteString("\" title=\"")
		r.attrEscape(title)
	}

	r.w.WriteByte('"')
	r.w.WriteString(r.closeTag)
}

func (r *Html) LineBreak() {
	r.w.WriteString("<br")
	r.w.WriteString(r.closeTag)
	r.w.WriteByte('\n')
}

func (r *Html) Link(link []byte, title []byte, content []byte) {
	if r.flags&SkipLinks != 0 {
		// write the link text out but don't link it, just mark it with typewriter font
		r.w.WriteString("<tt>")
		r.attrEscape(content)
		r.w.WriteString("</tt>")
		return
	}

	if r.flags&Safelink != 0 && !isSafeLink(link) {
		// write the link text out but don't link it, just mark it with typewriter font
		r.w.WriteString("<tt>")
		r.attrEscape(content)
		r.w.WriteString("</tt>")
		return
	}

	r.w.WriteString("<a href=\"")
	r.maybeWriteAbsolutePrefix(link)
	r.attrEscape(link)
	if len(title) > 0 {
		r.w.WriteString("\" title=\"")
		r.attrEscape(title)
	}
	var relAttrs []string
	if r.flags&NofollowLinks != 0 && !isRelativeLink(link) {
		relAttrs = append(relAttrs, "nofollow")
	}
	if r.flags&NoreferrerLinks != 0 && !isRelativeLink(link) {
		relAttrs = append(relAttrs, "noreferrer")
	}
	if len(relAttrs) > 0 {
		r.w.WriteString(fmt.Sprintf("\" rel=\"%s", strings.Join(relAttrs, " ")))
	}

	// blank target only add to external link
	if r.flags&HrefTargetBlank != 0 && !isRelativeLink(link) {
		r.w.WriteString("\" target=\"_blank")
	}

	r.w.WriteString("\">")
	r.w.Write(content)
	r.w.WriteString("</a>")
	return
}

func (r *Html) RawHtmlTag(text []byte) {
	if r.flags&SkipHTML != 0 {
		return
	}
	if r.flags&SkipStyle != 0 && isHtmlTag(text, "style") {
		return
	}
	if r.flags&SkipLinks != 0 && isHtmlTag(text, "a") {
		return
	}
	if r.flags&SkipImages != 0 && isHtmlTag(text, "img") {
		return
	}
	r.w.Write(text)
}

func (r *Html) TripleEmphasis(text []byte) {
	r.w.WriteString("<strong><em>")
	r.w.Write(text)
	r.w.WriteString("</em></strong>")
}

func (r *Html) StrikeThrough(text []byte) {
	r.w.WriteString("<del>")
	r.w.Write(text)
	r.w.WriteString("</del>")
}

func (r *Html) FootnoteRef(ref []byte, id int) {
	slug := slugify(ref)
	r.w.WriteString(`<sup class="footnote-ref" id="`)
	r.w.WriteString(`fnref:`)
	r.w.WriteString(r.parameters.FootnoteAnchorPrefix)
	r.w.Write(slug)
	r.w.WriteString(`"><a rel="footnote" href="#`)
	r.w.WriteString(`fn:`)
	r.w.WriteString(r.parameters.FootnoteAnchorPrefix)
	r.w.Write(slug)
	r.w.WriteString(`">`)
	r.w.WriteString(strconv.Itoa(id))
	r.w.WriteString(`</a></sup>`)
}

func (r *Html) Entity(entity []byte) {
	r.w.Write(entity)
}

func (r *Html) NormalText(text []byte) {
	if r.flags&UseSmartypants != 0 {
		r.Smartypants(text)
	} else {
		r.attrEscape(text)
	}
}

func (r *Html) Smartypants2(text []byte) []byte {
	smrt := smartypantsData{false, false}
	var buff bytes.Buffer
	// first do normal entity escaping
	text = attrEscape2(text)
	mark := 0
	for i := 0; i < len(text); i++ {
		if action := r.smartypants[text[i]]; action != nil {
			if i > mark {
				buff.Write(text[mark:i])
			}
			previousChar := byte(0)
			if i > 0 {
				previousChar = text[i-1]
			}
			var tmp bytes.Buffer
			i += action(&tmp, &smrt, previousChar, text[i:])
			buff.Write(tmp.Bytes())
			mark = i + 1
		}
	}
	if mark < len(text) {
		buff.Write(text[mark:])
	}
	return buff.Bytes()
}

func (r *Html) Smartypants(text []byte) {
	smrt := smartypantsData{false, false}

	// first do normal entity escaping
	text = r.CaptureWrites(func() {
		r.attrEscape(text)
	})

	mark := 0
	for i := 0; i < len(text); i++ {
		if action := r.smartypants[text[i]]; action != nil {
			if i > mark {
				r.w.Write(text[mark:i])
			}

			previousChar := byte(0)
			if i > 0 {
				previousChar = text[i-1]
			}
			var tmp bytes.Buffer
			i += action(&tmp, &smrt, previousChar, text[i:])
			r.w.Write(tmp.Bytes())
			mark = i + 1
		}
	}

	if mark < len(text) {
		r.w.Write(text[mark:])
	}
}

func (r *Html) DocumentHeader() {
	if r.flags&CompletePage == 0 {
		return
	}

	ending := ""
	if r.flags&UseXHTML != 0 {
		r.w.WriteString("<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" ")
		r.w.WriteString("\"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\n")
		r.w.WriteString("<html xmlns=\"http://www.w3.org/1999/xhtml\">\n")
		ending = " /"
	} else {
		r.w.WriteString("<!DOCTYPE html>\n")
		r.w.WriteString("<html>\n")
	}
	r.w.WriteString("<head>\n")
	r.w.WriteString("  <title>")
	r.NormalText([]byte(r.title))
	r.w.WriteString("</title>\n")
	r.w.WriteString("  <meta name=\"GENERATOR\" content=\"Blackfriday Markdown Processor v")
	r.w.WriteString(VERSION)
	r.w.WriteString("\"")
	r.w.WriteString(ending)
	r.w.WriteString(">\n")
	r.w.WriteString("  <meta charset=\"utf-8\"")
	r.w.WriteString(ending)
	r.w.WriteString(">\n")
	if r.css != "" {
		r.w.WriteString("  <link rel=\"stylesheet\" type=\"text/css\" href=\"")
		r.attrEscape([]byte(r.css))
		r.w.WriteString("\"")
		r.w.WriteString(ending)
		r.w.WriteString(">\n")
	}
	r.w.WriteString("</head>\n")
	r.w.WriteString("<body>\n")

	r.tocMarker = r.w.output.Len() // XXX
}

func (r *Html) DocumentFooter() {
	// finalize and insert the table of contents
	if r.flags&Toc != 0 {
		r.TocFinalize()

		// now we have to insert the table of contents into the document
		var temp bytes.Buffer

		// start by making a copy of everything after the document header
		temp.Write(r.w.output.Bytes()[r.tocMarker:])

		// now clear the copied material from the main output buffer
		r.w.output.Truncate(r.tocMarker)

		// corner case spacing issue
		if r.flags&CompletePage != 0 {
			r.w.WriteByte('\n')
		}

		// insert the table of contents
		r.w.WriteString("<nav>\n")
		r.w.Write(r.toc.Bytes())
		r.w.WriteString("</nav>\n")

		// corner case spacing issue
		if r.flags&CompletePage == 0 && r.flags&OmitContents == 0 {
			r.w.WriteByte('\n')
		}

		// write out everything that came after it
		if r.flags&OmitContents == 0 {
			r.w.Write(temp.Bytes())
		}
	}

	if r.flags&CompletePage != 0 {
		r.w.WriteString("\n</body>\n")
		r.w.WriteString("</html>\n")
	}

}

func (r *Html) TocHeaderWithAnchor(text []byte, level int, anchor string) {
	for level > r.currentLevel {
		switch {
		case bytes.HasSuffix(r.toc.Bytes(), []byte("</li>\n")):
			// this sublist can nest underneath a header
			size := r.toc.Len()
			r.toc.Truncate(size - len("</li>\n"))

		case r.currentLevel > 0:
			r.toc.WriteString("<li>")
		}
		if r.toc.Len() > 0 {
			r.toc.WriteByte('\n')
		}
		r.toc.WriteString("<ul>\n")
		r.currentLevel++
	}

	for level < r.currentLevel {
		r.toc.WriteString("</ul>")
		if r.currentLevel > 1 {
			r.toc.WriteString("</li>\n")
		}
		r.currentLevel--
	}

	r.toc.WriteString("<li><a href=\"#")
	if anchor != "" {
		r.toc.WriteString(anchor)
	} else {
		r.toc.WriteString("toc_")
		r.toc.WriteString(strconv.Itoa(r.headerCount))
	}
	r.toc.WriteString("\">")
	r.headerCount++

	r.toc.Write(text)

	r.toc.WriteString("</a></li>\n")
}

func (r *Html) TocHeader(text []byte, level int) {
	r.TocHeaderWithAnchor(text, level, "")
}

func (r *Html) TocFinalize() {
	for r.currentLevel > 1 {
		r.toc.WriteString("</ul></li>\n")
		r.currentLevel--
	}

	if r.currentLevel > 0 {
		r.toc.WriteString("</ul>\n")
	}
}

func isHtmlTag(tag []byte, tagname string) bool {
	found, _ := findHtmlTagPos(tag, tagname)
	return found
}

// Look for a character, but ignore it when it's in any kind of quotes, it
// might be JavaScript
func skipUntilCharIgnoreQuotes(html []byte, start int, char byte) int {
	inSingleQuote := false
	inDoubleQuote := false
	inGraveQuote := false
	i := start
	for i < len(html) {
		switch {
		case html[i] == char && !inSingleQuote && !inDoubleQuote && !inGraveQuote:
			return i
		case html[i] == '\'':
			inSingleQuote = !inSingleQuote
		case html[i] == '"':
			inDoubleQuote = !inDoubleQuote
		case html[i] == '`':
			inGraveQuote = !inGraveQuote
		}
		i++
	}
	return start
}

func findHtmlTagPos(tag []byte, tagname string) (bool, int) {
	i := 0
	if i < len(tag) && tag[0] != '<' {
		return false, -1
	}
	i++
	i = skipSpace(tag, i)

	if i < len(tag) && tag[i] == '/' {
		i++
	}

	i = skipSpace(tag, i)
	j := 0
	for ; i < len(tag); i, j = i+1, j+1 {
		if j >= len(tagname) {
			break
		}

		if strings.ToLower(string(tag[i]))[0] != tagname[j] {
			return false, -1
		}
	}

	if i == len(tag) {
		return false, -1
	}

	rightAngle := skipUntilCharIgnoreQuotes(tag, i, '>')
	if rightAngle > i {
		return true, rightAngle
	}

	return false, -1
}

func skipUntilChar(text []byte, start int, char byte) int {
	i := start
	for i < len(text) && text[i] != char {
		i++
	}
	return i
}

func skipSpace(tag []byte, i int) int {
	for i < len(tag) && isspace(tag[i]) {
		i++
	}
	return i
}

func skipChar(data []byte, start int, char byte) int {
	i := start
	for i < len(data) && data[i] == char {
		i++
	}
	return i
}

func isRelativeLink(link []byte) (yes bool) {
	// a tag begin with '#'
	if link[0] == '#' {
		return true
	}

	// link begin with '/' but not '//', the second maybe a protocol relative link
	if len(link) >= 2 && link[0] == '/' && link[1] != '/' {
		return true
	}

	// only the root '/'
	if len(link) == 1 && link[0] == '/' {
		return true
	}

	// current directory : begin with "./"
	if bytes.HasPrefix(link, []byte("./")) {
		return true
	}

	// parent directory : begin with "../"
	if bytes.HasPrefix(link, []byte("../")) {
		return true
	}

	return false
}

func (r *Html) ensureUniqueHeaderID(id string) string {
	for count, found := r.headerIDs[id]; found; count, found = r.headerIDs[id] {
		tmp := fmt.Sprintf("%s-%d", id, count+1)

		if _, tmpFound := r.headerIDs[tmp]; !tmpFound {
			r.headerIDs[id] = count + 1
			id = tmp
		} else {
			id = id + "-1"
		}
	}

	if _, found := r.headerIDs[id]; !found {
		r.headerIDs[id] = 0
	}

	return id
}

func (r *Html) addAbsPrefix(link []byte) []byte {
	if r.parameters.AbsolutePrefix != "" && isRelativeLink(link) && link[0] != '.' {
		newDest := r.parameters.AbsolutePrefix
		if link[0] != '/' {
			newDest += "/"
		}
		newDest += string(link)
		return []byte(newDest)
	}
	return link
}

func appendLinkAttrs(attrs []string, flags HtmlFlags, link []byte) []string {
	if isRelativeLink(link) {
		return attrs
	}
	val := []string{}
	if flags&NofollowLinks != 0 {
		val = append(val, "nofollow")
	}
	if flags&NoreferrerLinks != 0 {
		val = append(val, "noreferrer")
	}
	if flags&HrefTargetBlank != 0 {
		attrs = append(attrs, "target=\"_blank\"")
	}
	if len(val) == 0 {
		return attrs
	}
	attr := fmt.Sprintf("rel=%q", strings.Join(val, " "))
	return append(attrs, attr)
}

func isMailto(link []byte) bool {
	return bytes.HasPrefix(link, []byte("mailto:"))
}

func isSmartypantable(node *Node) bool {
	pt := node.parent.Type
	return pt != Link && pt != CodeBlock && pt != Code
}

func appendLanguageAttr(attrs []string, info []byte) []string {
	infoWords := bytes.Split(info, []byte("\t "))
	if len(infoWords) > 0 && len(infoWords[0]) > 0 {
		attrs = append(attrs, fmt.Sprintf("class=\"language-%s\"", infoWords[0]))
	}
	return attrs
}

func (r *Html) Render(ast *Node) []byte {
	//println("render_Blackfriday")
	//dump(ast)
	var buff bytes.Buffer
	var lastOutput []byte
	disableTags := 0
	out := func(text []byte) {
		if disableTags > 0 {
			buff.Write(reHtmlTag.ReplaceAll(text, []byte{}))
		} else {
			buff.Write(text)
		}
		lastOutput = text
	}
	esc := func(text []byte, preserveEntities bool) []byte {
		return attrEscape2(text)
	}
	esc2 := func(text []byte, preserveEntities bool) []byte {
		e1 := []byte(html.EscapeString(string(text)))
		e2 := bytes.Replace(e1, []byte("&#34;"), []byte("&quot;"), -1)
		return bytes.Replace(e2, []byte("&#39;"), []byte{'\''}, -1)
	}
	cr := func() {
		if len(lastOutput) > 0 {
			out([]byte{'\n'})
		}
	}
	forEachNode(ast, func(node *Node, entering bool) {
		attrs := []string{}
		switch node.Type {
		case Text:
			if r.flags&UseSmartypants != 0 && isSmartypantable(node) {
				out(r.Smartypants2(node.literal))
			} else {
				out(esc(node.literal, false))
			}
			break
		case Softbreak:
			out([]byte("\n"))
			// TODO: make it configurable via out(renderer.softbreak)
		case Hardbreak:
			out(tag("br", nil, true))
			cr()
		case Emph:
			if entering {
				out(tag("em", nil, false))
			} else {
				out(tag("/em", nil, false))
			}
			break
		case Strong:
			if entering {
				out(tag("strong", nil, false))
			} else {
				out(tag("/strong", nil, false))
			}
			break
		case Del:
			if entering {
				out(tag("del", nil, false))
			} else {
				out(tag("/del", nil, false))
			}
		case HtmlSpan:
			//if options.safe {
			//	out("<!-- raw HTML omitted -->")
			//} else {
			out(node.literal)
			//}
		case Link:
			// mark it but don't link it if it is not a safe link: no smartypants
			if r.flags&Safelink != 0 && !isSafeLink(node.destination) && !isMailto(node.destination) {
				if entering {
					out(tag("tt", nil, false))
				} else {
					out(tag("/tt", nil, false))
				}
			} else {
				if entering {
					node.destination = r.addAbsPrefix(node.destination)
					//if (!(options.safe && potentiallyUnsafe(node.destination))) {
					attrs = append(attrs, fmt.Sprintf("href=%q", esc(node.destination, true)))
					//}
					attrs = appendLinkAttrs(attrs, r.flags, node.destination)
					if len(node.title) > 0 {
						attrs = append(attrs, fmt.Sprintf("title=%q", esc(node.title, true)))
					}
					out(tag("a", attrs, false))
				} else {
					out(tag("/a", nil, false))
				}
			}
		case Image:
			if entering {
				node.destination = r.addAbsPrefix(node.destination)
				if disableTags == 0 {
					//if options.safe && potentiallyUnsafe(node.destination) {
					//out(`<img src="" alt="`)
					//} else {
					out([]byte(fmt.Sprintf(`<img src="%s" alt="`, esc(node.destination, true))))
					//}
				}
				disableTags++
			} else {
				disableTags--
				if disableTags == 0 {
					if node.title != nil {
						out([]byte(`" title="`))
						out(esc(node.title, true))
					}
					out([]byte(`" />`))
				}
			}
		case Code:
			out(tag("code", nil, false))
			out(esc2(node.literal, false))
			out(tag("/code", nil, false))
		case Document:
			break
		case Paragraph:
			grandparent := node.parent.parent
			if grandparent != nil && grandparent.Type == List && grandparent.listData.tight {
				break
			}
			if entering {
				if node.prev != nil {
					t := node.prev.Type
					if t == HtmlBlock || t == List || t == Paragraph || t == Header || t == CodeBlock || t == BlockQuote || t == HorizontalRule {
						cr()
					}
				}
				if node.parent.Type == BlockQuote && node.prev == nil {
					cr()
				}
				out(tag("p", attrs, false))
			} else {
				out(tag("/p", attrs, false))
				if !(node.parent.Type == Item && node.next == nil) {
					cr()
				}
			}
			break
		case BlockQuote:
			if entering {
				cr()
				out(tag("blockquote", attrs, false))
			} else {
				out(tag("/blockquote", nil, false))
				cr()
			}
			break
		case HtmlBlock:
			cr()
			out(node.literal)
			cr()
		case Header:
			tagname := fmt.Sprintf("h%d", node.level)
			if entering {
				cr()
				out(tag(tagname, attrs, false))
			} else {
				out(tag("/"+tagname, nil, false))
				if !(node.parent.Type == Item && node.next == nil) {
					cr()
				}
			}
			break
		case HorizontalRule:
			cr()
			out(tag("hr", attrs, r.flags&UseXHTML != 0))
			cr()
			break
		case List:
			tagName := "ul"
			if node.listData.Type == OrderedList {
				tagName = "ol"
			}
			if entering {
				// var start = node.listStart;
				// if (start !== null && start !== 1) {
				//     attrs.push(['start', start.toString()]);
				// }
				cr()
				if node.parent.Type == Item && node.parent.parent.listData.tight {
					cr()
				}
				out(tag(tagName, attrs, false))
				cr()
			} else {
				out(tag("/"+tagName, nil, false))
				//cr()
				//if node.parent.Type != Item {
				//	cr()
				//}
				if node.parent.Type == Item && node.next != nil {
					cr()
				}
				if node.parent.Type == Document || node.parent.Type == BlockQuote {
					cr()
				}
			}
		case Item:
			if entering {
				if node.prev != nil && !node.parent.listData.tight {
					cr()
				}
				out(tag("li", nil, false))
			} else {
				out(tag("/li", nil, false))
				cr()
			}
		case CodeBlock:
			attrs = appendLanguageAttr(attrs, node.info)
			cr()
			out(tag("pre", nil, false))
			out(tag("code", attrs, false))
			out(esc2(node.literal, false))
			out(tag("/code", nil, false))
			out(tag("/pre", nil, false))
			if node.parent.Type != Item {
				cr()
			}
		default:
			panic("Unknown node type " + node.Type.String())
		}
	})
	return buff.Bytes()
}
