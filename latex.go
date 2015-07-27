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
// LaTeX rendering backend
//
//

package blackfriday

import (
	"bytes"
)

// Latex is a type that implements the Renderer interface for LaTeX output.
//
// Do not create this directly, instead use the LatexRenderer function.
type Latex struct {
	w HtmlWriter
}

// LatexRenderer creates and configures a Latex object, which
// satisfies the Renderer interface.
//
// flags is a set of LATEX_* options ORed together (currently no such options
// are defined).
func LatexRenderer(flags int) Renderer {
	var writer HtmlWriter
	return &Latex{
		w: writer,
	}
}

func (r *Latex) captureWrites(processor func()) []byte {
	var buff bytes.Buffer
	r.w.capture = &buff
	processor()
	r.w.capture = nil
	return buff.Bytes()
}

func (r *Latex) Write(b []byte) (int, error) {
	return r.w.Write(b)
}

func (r *Latex) getResult() []byte {
	return r.w.buff.Bytes()
}

func (r *Latex) GetFlags() HtmlFlags {
	return 0
}

// render code chunks using verbatim, or listings if we have a language
func (r *Latex) BlockCode(text []byte, lang string) {
	if lang == "" {
		out.WriteString("\n\\begin{verbatim}\n")
	} else {
		out.WriteString("\n\\begin{lstlisting}[language=")
		out.WriteString(lang)
		out.WriteString("]\n")
	}
	out.Write(text)
	if lang == "" {
		out.WriteString("\n\\end{verbatim}\n")
	} else {
		out.WriteString("\n\\end{lstlisting}\n")
	}
}

func (r *Latex) TitleBlock(text []byte) {

}

func (r *Latex) BlockQuote(text []byte) {
	out.WriteString("\n\\begin{quotation}\n")
	out.Write(text)
	out.WriteString("\n\\end{quotation}\n")
}

func (r *Latex) BlockHtml(text []byte) {
	// a pretty lame thing to do...
	out.WriteString("\n\\begin{verbatim}\n")
	out.Write(text)
	out.WriteString("\n\\end{verbatim}\n")
}

func (r *Latex) BeginHeader(level int, id string) {
	switch level {
	case 1:
		out.WriteString("\n\\section{")
	case 2:
		out.WriteString("\n\\subsection{")
	case 3:
		out.WriteString("\n\\subsubsection{")
	case 4:
		out.WriteString("\n\\paragraph{")
	case 5:
		out.WriteString("\n\\subparagraph{")
	case 6:
		out.WriteString("\n\\textbf{")
	}
}

func (r *Latex) EndHeader(level int, id string, header []byte) {
	out.WriteString("}\n")
}

func (r *Latex) HRule() {
	out.WriteString("\n\\HRule\n")
}

func (r *Latex) BeginList(flags ListType) {
	if flags&ListTypeOrdered != 0 {
		out.WriteString("\n\\begin{enumerate}\n")
	} else {
		out.WriteString("\n\\begin{itemize}\n")
	}
}

func (r *Latex) EndList(flags ListType) {
	if flags&ListTypeOrdered != 0 {
		out.WriteString("\n\\end{enumerate}\n")
	} else {
		out.WriteString("\n\\end{itemize}\n")
	}
}

func (r *Latex) ListItem(text []byte, flags ListType) {
	out.WriteString("\n\\item ")
	out.Write(text)
}

func (r *Latex) BeginParagraph() {
	out.WriteString("\n")
}

func (r *Latex) EndParagraph() {
	out.WriteString("\n")
}

func (r *Latex) Table(header []byte, body []byte, columnData []TableFlags) {
	out.WriteString("\n\\begin{tabular}{")
	for _, elt := range columnData {
		switch elt {
		case TableAlignmentLeft:
			out.WriteByte('l')
		case TableAlignmentRight:
			out.WriteByte('r')
		default:
			out.WriteByte('c')
		}
	}
	out.WriteString("}\n")
	out.Write(header)
	out.WriteString(" \\\\\n\\hline\n")
	out.Write(body)
	out.WriteString("\n\\end{tabular}\n")
}

func (r *Latex) TableRow(text []byte) {
	if out.Len() > 0 {
		out.WriteString(" \\\\\n")
	}
	out.Write(text)
}

func (r *Latex) TableHeaderCell(out *bytes.Buffer, text []byte, align TableFlags) {
	if out.Len() > 0 {
		out.WriteString(" & ")
	}
	out.Write(text)
}

func (r *Latex) TableCell(out *bytes.Buffer, text []byte, align TableFlags) {
	if out.Len() > 0 {
		out.WriteString(" & ")
	}
	out.Write(text)
}

// TODO: this
func (r *Latex) BeginFootnotes() {
}

// TODO: this
func (r *Latex) EndFootnotes() {
}

func (r *Latex) FootnoteItem(name, text []byte, flags ListType) {

}

func (r *Latex) AutoLink(link []byte, kind LinkType) {
	out.WriteString("\\href{")
	if kind == LinkTypeEmail {
		out.WriteString("mailto:")
	}
	out.Write(link)
	out.WriteString("}{")
	out.Write(link)
	out.WriteString("}")
}

func (r *Latex) CodeSpan(text []byte) {
	out.WriteString("\\texttt{")
	escapeSpecialChars(out, text)
	out.WriteString("}")
}

func (r *Latex) DoubleEmphasis(text []byte) {
	out.WriteString("\\textbf{")
	out.Write(text)
	out.WriteString("}")
}

func (r *Latex) Emphasis(text []byte) {
	out.WriteString("\\textit{")
	out.Write(text)
	out.WriteString("}")
}

func (r *Latex) Image(link []byte, title []byte, alt []byte) {
	if bytes.HasPrefix(link, []byte("http://")) || bytes.HasPrefix(link, []byte("https://")) {
		// treat it like a link
		out.WriteString("\\href{")
		out.Write(link)
		out.WriteString("}{")
		out.Write(alt)
		out.WriteString("}")
	} else {
		out.WriteString("\\includegraphics{")
		out.Write(link)
		out.WriteString("}")
	}
}

func (r *Latex) LineBreak() {
	out.WriteString(" \\\\\n")
}

func (r *Latex) Link(link []byte, title []byte, content []byte) {
	out.WriteString("\\href{")
	out.Write(link)
	out.WriteString("}{")
	out.Write(content)
	out.WriteString("}")
}

func (r *Latex) RawHtmlTag(tag []byte) {
}

func (r *Latex) TripleEmphasis(text []byte) {
	out.WriteString("\\textbf{\\textit{")
	out.Write(text)
	out.WriteString("}}")
}

func (r *Latex) StrikeThrough(text []byte) {
	out.WriteString("\\sout{")
	out.Write(text)
	out.WriteString("}")
}

// TODO: this
func (r *Latex) FootnoteRef(ref []byte, id int) {

}

func needsBackslash(c byte) bool {
	for _, r := range []byte("_{}%$&\\~") {
		if c == r {
			return true
		}
	}
	return false
}

func escapeSpecialChars(out *bytes.Buffer, text []byte) {
	for i := 0; i < len(text); i++ {
		// directly copy normal characters
		org := i

		for i < len(text) && !needsBackslash(text[i]) {
			i++
		}
		if i > org {
			out.Write(text[org:i])
		}

		// escape a character
		if i >= len(text) {
			break
		}
		out.WriteByte('\\')
		out.WriteByte(text[i])
	}
}

func (r *Latex) Entity(entity []byte) {
	// TODO: convert this into a unicode character or something
	out.Write(entity)
}

func (r *Latex) NormalText(text []byte) {
	escapeSpecialChars(out, text)
}

// header and footer
func (r *Latex) DocumentHeader() {
	out.WriteString("\\documentclass{article}\n")
	out.WriteString("\n")
	out.WriteString("\\usepackage{graphicx}\n")
	out.WriteString("\\usepackage{listings}\n")
	out.WriteString("\\usepackage[margin=1in]{geometry}\n")
	out.WriteString("\\usepackage[utf8]{inputenc}\n")
	out.WriteString("\\usepackage{verbatim}\n")
	out.WriteString("\\usepackage[normalem]{ulem}\n")
	out.WriteString("\\usepackage{hyperref}\n")
	out.WriteString("\n")
	out.WriteString("\\hypersetup{colorlinks,%\n")
	out.WriteString("  citecolor=black,%\n")
	out.WriteString("  filecolor=black,%\n")
	out.WriteString("  linkcolor=black,%\n")
	out.WriteString("  urlcolor=black,%\n")
	out.WriteString("  pdfstartview=FitH,%\n")
	out.WriteString("  breaklinks=true,%\n")
	out.WriteString("  pdfauthor={Blackfriday Markdown Processor v")
	out.WriteString(Version)
	out.WriteString("}}\n")
	out.WriteString("\n")
	out.WriteString("\\newcommand{\\HRule}{\\rule{\\linewidth}{0.5mm}}\n")
	out.WriteString("\\addtolength{\\parskip}{0.5\\baselineskip}\n")
	out.WriteString("\\parindent=0pt\n")
	out.WriteString("\n")
	out.WriteString("\\begin{document}\n")
}

func (r *Latex) DocumentFooter() {
	out.WriteString("\n\\end{document}\n")
}
