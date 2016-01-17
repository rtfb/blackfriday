package blackfriday

import (
	"bytes"
	"fmt"
	"strings"
)

func tag(name string, attrs []string, selfClosing bool) []byte {
	result := "<" + name
	if attrs != nil && len(attrs) > 0 {
		result += " " + strings.Join(attrs, " ")
	}
	if selfClosing {
		result += " /"
	}
	return []byte(result + ">")
}

func render(ast *Node) []byte {
	var buff bytes.Buffer
	var lastOutput []byte
	out := func(text []byte) {
		buff.Write(text)
		lastOutput = text
	}
	// XXX: this out("\n") is only for compatibility with existing Blackfriday
	// tests. Not necessary otherwise and should probably be eliminated when
	// the time comes
	compatibilityNewline := func() {
		out([]byte("\n"))
	}
	esc := func(text []byte, preserveEntities bool) []byte {
		// XXX: impl
		return text
	}
	cr := func() {
		if len(lastOutput) > 0 && !bytes.Equal(lastOutput, []byte("\n")) {
			buff.WriteString("\n")
			lastOutput = []byte("\n")
		}
	}
	forEachNode(ast, func(node *Node, entering bool) {
		attrs := []string{}
		switch node.Type {
		case Text:
			out(esc(node.literal, false))
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
		case HtmlSpan:
			//if options.safe {
			//	out("<!-- raw HTML omitted -->")
			//} else {
			out(node.literal)
			//}
		case Link:
			if entering {
				//if (!(options.safe && potentiallyUnsafe(node.destination))) {
				attrs = append(attrs, fmt.Sprintf("href=%q", esc(node.destination, true)))
				//}
				if node.title != nil {
					attrs = append(attrs, fmt.Sprintf("title=%q", esc(node.title, true)))
				}
				out(tag("a", attrs, false))
			} else {
				out(tag("/a", nil, false))
			}
		case Code:
			out(tag("code", nil, false))
			out(esc(node.literal, false))
			out(tag("/code", nil, false))
		case Document:
			break
		case Paragraph:
			/*
			   grandparent = node.parent.parent;
			   if (grandparent !== null &&
			       grandparent.type === 'List') {
			       if (grandparent.listTight) {
			           break;
			       }
			   }
			*/
			if entering {
				cr()
				out(tag("p", attrs, false))
			} else {
				out(tag("/p", attrs, false))
				cr()
			}
			break
		case BlockQuote:
			if entering {
				cr()
				compatibilityNewline()
				out(tag("blockquote", attrs, false))
				cr()
			} else {
				cr()
				out(tag("/blockquote", nil, false))
				cr()
			}
			break
		case HtmlBlock:
			compatibilityNewline()
			out(node.literal)
			cr()
		case Header:
			tagname := fmt.Sprintf("h%d", node.level)
			if entering {
				cr()
				out(tag(tagname, attrs, false))
			} else {
				out(tag("/"+tagname, nil, false))
				cr()
			}
			break
		case HorizontalRule:
			cr()
			compatibilityNewline()
			out(tag("hr", attrs, true))
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
				out(tag(tagName, attrs, false))
				cr()
			} else {
				cr()
				out(tag("/"+tagName, nil, false))
				cr()
			}
		case Item:
			if entering {
				out(tag("li", nil, false))
			} else {
				out(tag("/li", nil, false))
				cr()
			}
		case CodeBlock:
			// TODO:
			// info_words = node.info ? node.info.split(/\s+/) : [];
			// if (info_words.length > 0 && info_words[0].length > 0) {
			//     attrs.push(['class', 'language-' + esc(info_words[0], true)]);
			// }
			cr()
			out(tag("pre", nil, false))
			out(tag("code", nil, false))
			out(esc(node.literal, false))
			out(tag("/code", nil, false))
			out(tag("/pre", nil, false))
			cr()
		default:
			panic("Unknown node type " + node.Type.String())
		}
	})
	return buff.Bytes()
}
