package blackfriday

import (
	"bytes"
	"html"
	"regexp"
)

const (
	Entity    = "&(?:#x[a-f0-9]{1,8}|#[0-9]{1,8}|[a-z][a-z0-9]{1,31});"
	Escapable = "[!\"#$%&'()*+,./:;<=>?@[\\\\\\]^_`{|}~-]"
)

var (
	blockHandlers = map[NodeType]BlockHandler{
		Document:       &DocumentBlockHandler{},
		Header:         &HeaderBlockHandler{},
		HorizontalRule: &HorizontalRuleBlockHandler{},
		BlockQuote:     &BlockQuoteBlockHandler{},
		Paragraph:      &ParagraphBlockHandler{},
		HtmlBlock:      &HtmlBlockHandler{},
		List:           &ListBlockHandler{},
		Item:           &ItemBlockHandler{},
		CodeBlock:      &CodeBlockHandler{},
	}

	reBackslashOrAmp      = regexp.MustCompile("[\\&]")
	reEntityOrEscapedChar = regexp.MustCompile("(?i)\\\\" + Escapable + "|" + Entity)
	reTrailingWhitespace  = regexp.MustCompile("(\n *)+$")
)

type BlockHandler interface {
	Finalize(block *Node)
	CanContain(t NodeType) bool
}

type HeaderBlockHandler struct {
}

func (h *HeaderBlockHandler) Finalize(block *Node) {
}

func (h *HeaderBlockHandler) CanContain(t NodeType) bool {
	return false
}

type DocumentBlockHandler struct {
}

func (h *DocumentBlockHandler) Finalize(block *Node) {
}

func (h *DocumentBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type HorizontalRuleBlockHandler struct {
}

func (h *HorizontalRuleBlockHandler) Finalize(block *Node) {
}

func (h *HorizontalRuleBlockHandler) CanContain(t NodeType) bool {
	return false
}

type BlockQuoteBlockHandler struct {
}

func (h *BlockQuoteBlockHandler) Finalize(block *Node) {
}

func (h *BlockQuoteBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type ParagraphBlockHandler struct {
}

func (h *ParagraphBlockHandler) Finalize(block *Node) {
	/*
		TODO:
			hasReferenceDefs := false
			for peek(block.content, 0) == '[' &&
				(pos := p.inlineParser.parseReference(block.content, p.refmap); pos != 0) {
				block.content = block.content[pos:]
				hasReferenceDefs = true
			}
			if hasReferenceDefs && isBlank(block.content) {
				block.unlink()
			}
	*/
}

func (h *ParagraphBlockHandler) CanContain(t NodeType) bool {
	return false
}

type HtmlBlockHandler struct {
}

func (h *HtmlBlockHandler) Finalize(block *Node) {
	block.literal = reTrailingWhitespace.ReplaceAll(block.content, []byte{})
	block.content = []byte{}
}

func (h *HtmlBlockHandler) CanContain(t NodeType) bool {
	return false
}

type ListBlockHandler struct {
}

func (h *ListBlockHandler) Finalize(block *Node) {
	item := block.firstChild
	for item != nil {
		// check for non-final list item ending with blank line:
		if endsWithBlankLine(item) && item.next != nil {
			block.listData.tight = false
			break
		}
		// recurse into children of list item, to see if there are spaces
		// between any of them:
		subItem := item.firstChild
		for subItem != nil {
			if endsWithBlankLine(subItem) && (item.next != nil || subItem.next != nil) {
				block.listData.tight = false
				break
			}
			subItem = subItem.next
		}
		item = item.next
	}
}

func (h *ListBlockHandler) CanContain(t NodeType) bool {
	return t == Item
}

type ItemBlockHandler struct {
}

func (h *ItemBlockHandler) Finalize(block *Node) {
}

func (h *ItemBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type CodeBlockHandler struct {
}

func unescapeChar(str []byte) []byte {
	if str[0] == '\\' {
		return []byte{str[1]}
	}
	return []byte(html.UnescapeString(string(str)))
}

func unescapeString(str []byte) []byte {
	if reBackslashOrAmp.Match(str) {
		return reEntityOrEscapedChar.ReplaceAllFunc(str, unescapeChar)
	} else {
		return str
	}
}

func (h *CodeBlockHandler) Finalize(block *Node) {
	if block.isFenced {
		newlinePos := bytes.IndexByte(block.content, '\n')
		firstLine := block.content[:newlinePos]
		rest := block.content[newlinePos+1:]
		block.info = unescapeString(bytes.Trim(firstLine, "\n"))
		block.literal = rest
	} else {
		block.literal = reTrailingWhitespace.ReplaceAll(block.content, []byte{'\n'})
	}
	block.content = nil
}

func (h *CodeBlockHandler) CanContain(t NodeType) bool {
	return false
}

// XXX: there's already ListType in blackfriday, so name it somewhat
// differently for now. See if both types are necessary later.
type ASTListType int

const (
	BulletList ASTListType = iota
	OrderedList
)

type ListData struct {
	Type         ASTListType
	tight        bool // skip <p>s around list item data if true
	bulletChar   byte // '*', '+' or '-' in bullet lists
	start        uint32
	delimiter    byte // '.' or ')' after the number in ordered lists
	padding      uint32
	markerOffset uint32
}

// Returns true if block ends with a blank line, descending if needed
// into lists and sublists.
func endsWithBlankLine(block *Node) bool {
	for block != nil {
		if block.lastLineBlank {
			return true
		}
		t := block.Type
		if t == List || t == Item {
			block = block.lastChild
		} else {
			break
		}
	}
	return false
}
