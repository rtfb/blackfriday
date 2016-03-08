package blackfriday

import (
	"bytes"
	"html"
	"regexp"
	"strconv"
)

type BlockStatus int

const (
	NoMatch BlockStatus = iota
	ContainerMatch
	LeafMatch
)

type ContinueStatus int

const (
	Matched ContinueStatus = iota
	NotMatched
	Completed
)

type HtmlBlockType int

const (
	TagName            = "[A-Za-z][A-Za-z0-9-]*"
	AttributeName      = "[a-zA-Z_:][a-zA-Z0-9:._-]*"
	UnquotedValue      = "[^\"'=<>`\\x00-\\x20]+"
	SingleQuotedValue  = "'[^']*'"
	DoubleQuotedValue  = "\"[^\"]*\""
	AttributeValue     = "(?:" + UnquotedValue + "|" + SingleQuotedValue + "|" + DoubleQuotedValue + ")"
	AttributeValueSpec = "(?:" + "\\s*=" + "\\s*" + AttributeValue + ")"
	Attribute          = "(?:" + "\\s+" + AttributeName + AttributeValueSpec + "?)"
	OpenTag            = "<" + TagName + Attribute + "*" + "\\s*/?>"
	CloseTag           = "</" + TagName + "\\s*[>]"
	Entity             = "&(?:#x[a-f0-9]{1,8}|#[0-9]{1,8}|[a-z][a-z0-9]{1,31});"
	Escapable          = "[!\"#$%&'()*+,./:;<=>?@[\\\\\\]^_`{|}~-]"

	CodeIndent = 4
)

var (
	blockTriggers = []func(p *Parser, container *Node) BlockStatus{
		atxHeaderTrigger,
		setextHeaderTrigger,
		hruleTrigger,
		blockquoteTrigger,
		htmlBlockTrigger,
		listItemTrigger,
		fencedCodeTrigger,
		indentedCodeTrigger,
	}

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

	reATXHeaderMarker = regexp.MustCompile("^#{1,6}(?: +|$)")
	reHrule           = regexp.MustCompile("^(?:(?:\\* *){3,}|(?:_ *){3,}|(?:- *){3,}) *$")
	reHtmlBlockOpen   = []*regexp.Regexp{
		regexp.MustCompile("."), // dummy for 0
		regexp.MustCompile("(?i)^<(?:script|pre|style)(?:\\s|>|$)"),
		regexp.MustCompile("^<!--"),
		regexp.MustCompile("^<[?]"),
		regexp.MustCompile("^<![A-Z]"),
		regexp.MustCompile("^<!\\[CDATA\\["),
		regexp.MustCompile("(?i)^<[/]?(?:address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h1|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|meta|nav|noframes|ol|optgroup|option|p|param|section|source|title|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul)(?:\\s|[/]?[>]|$)"),
		regexp.MustCompile("(?i)^(?:" + OpenTag + "|" + CloseTag + ")\\s*$"),
	}
	reHtmlBlockClose = []*regexp.Regexp{
		regexp.MustCompile("."), // dummy for 0
		regexp.MustCompile("(?i)<\\/(?:script|pre|style)>"),
		regexp.MustCompile("-->"),
		regexp.MustCompile("\\?>"),
		regexp.MustCompile(">"),
		regexp.MustCompile("\\]\\]>"),
	}
	reBulletListMarker    = regexp.MustCompile("^[*+-]( +|$)")
	reOrderedListMarker   = regexp.MustCompile("^(\\d{1,9})([.)])( +|$)")
	reSetextHeaderLine    = regexp.MustCompile("^(?:=+|-+) *$")
	reBackslashOrAmp      = regexp.MustCompile("[\\&]")
	reEntityOrEscapedChar = regexp.MustCompile("(?i)\\\\" + Escapable + "|" + Entity)
	reClosingCodeFence    = regexp.MustCompile("^(?:`{3,}|~{3,})(?: *$)")

	//reCodeFence           = regexp.MustCompile("^`{3,}(?!.*`)|^~{3,}(?!.*~)")
	// XXX: The above regexp has a negative lookahead bit (the one that goes
	// (?!...)) and Go doesn't support negative lookahead. Need to figure out a
	// way to work around that, but for now I'm using a simplified regexp below
	reCodeFence          = regexp.MustCompile("^`{3,}|^~{3,}")
	reTrailingWhitespace = regexp.MustCompile("(\n *)+$")
)

type BlockHandler interface {
	Continue(p *Parser, container *Node) ContinueStatus
	Finalize(p *Parser, block *Node)
	CanContain(t NodeType) bool
	AcceptsLines() bool
}

type HeaderBlockHandler struct {
}

func (h *HeaderBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	// a header can never contain > 1 line, so fail to match:
	return NotMatched
}

func (h *HeaderBlockHandler) Finalize(p *Parser, block *Node) {
}

func (h *HeaderBlockHandler) CanContain(t NodeType) bool {
	return false
}

func (h *HeaderBlockHandler) AcceptsLines() bool {
	return false
}

type DocumentBlockHandler struct {
}

func (h *DocumentBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	return Matched
}

func (h *DocumentBlockHandler) Finalize(p *Parser, block *Node) {
}

func (h *DocumentBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

func (h *DocumentBlockHandler) AcceptsLines() bool {
	return false
}

type HorizontalRuleBlockHandler struct {
}

func (h *HorizontalRuleBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	// an hrule can never container > 1 line, so fail to match:
	return NotMatched
}

func (h *HorizontalRuleBlockHandler) Finalize(p *Parser, block *Node) {
}

func (h *HorizontalRuleBlockHandler) CanContain(t NodeType) bool {
	return false
}

func (h *HorizontalRuleBlockHandler) AcceptsLines() bool {
	return false
}

type BlockQuoteBlockHandler struct {
}

func (h *BlockQuoteBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	ln := p.currentLine
	if !p.indented && peek(ln, p.nextNonspace) == '>' {
		p.advanceNextNonspace()
		p.advanceOffset(1, false)
		if peek(ln, p.offset) == ' ' {
			p.offset += 1
		}
	} else {
		return NotMatched
	}
	return Matched
}

func (h *BlockQuoteBlockHandler) Finalize(p *Parser, block *Node) {
}

func (h *BlockQuoteBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

func (h *BlockQuoteBlockHandler) AcceptsLines() bool {
	return false
}

type ParagraphBlockHandler struct {
}

func (h *ParagraphBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	if p.blank {
		return NotMatched
	} else {
		return Matched
	}
}

func (h *ParagraphBlockHandler) Finalize(p *Parser, block *Node) {
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

func (h *ParagraphBlockHandler) AcceptsLines() bool {
	return true
}

type HtmlBlockHandler struct {
}

func (h *HtmlBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	if p.blank && (container.htmlBlockType == 6 || container.htmlBlockType == 7) {
		return NotMatched
	}
	return Matched
}

func (h *HtmlBlockHandler) Finalize(p *Parser, block *Node) {
	block.literal = reTrailingWhitespace.ReplaceAll(block.content, []byte{})
	block.content = []byte{}
}

func (h *HtmlBlockHandler) CanContain(t NodeType) bool {
	return false
}

func (h *HtmlBlockHandler) AcceptsLines() bool {
	return true
}

type ListBlockHandler struct {
}

func (h *ListBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	return Matched
}

func (h *ListBlockHandler) Finalize(p *Parser, block *Node) {
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

func (h *ListBlockHandler) AcceptsLines() bool {
	return false
}

type ItemBlockHandler struct {
}

func (h *ItemBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	if p.blank && container.firstChild != nil {
		p.advanceNextNonspace()
		return Matched
	}
	offsetAndPadding := container.listData.markerOffset + container.listData.padding
	if p.indent >= offsetAndPadding {
		p.advanceOffset(offsetAndPadding, true)
		return Matched
	}
	return NotMatched
}

func (h *ItemBlockHandler) Finalize(p *Parser, block *Node) {
}

func (h *ItemBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

func (h *ItemBlockHandler) AcceptsLines() bool {
	return false
}

type CodeBlockHandler struct {
}

func (h *CodeBlockHandler) Continue(p *Parser, container *Node) ContinueStatus {
	if container.isFenced {
		// Fenced
		var match [][]byte
		if p.indent <= 3 && peek(p.currentLine, p.nextNonspace) == container.fenceChar {
			match = reClosingCodeFence.FindSubmatch(p.currentLine[p.nextNonspace:])
		}
		if match != nil && ulen(match[0]) >= container.fenceLength {
			// closing fence - we're at end of line, so we can return
			p.finalize(container, p.lineNumber)
			return Completed
		} else {
			// skip optional spaces of fence offset
			for i := container.fenceOffset; i > 0 && peek(p.currentLine, p.offset) == ' '; i-- {
				p.advanceOffset(1, false)
			}
		}
	} else {
		// Indented
		if p.indent >= CodeIndent {
			p.advanceOffset(CodeIndent, true)
		} else if p.blank {
			p.advanceNextNonspace()
		} else {
			return NotMatched
		}
	}
	return Matched
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

func (h *CodeBlockHandler) Finalize(p *Parser, block *Node) {
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

func (h *CodeBlockHandler) AcceptsLines() bool {
	return true
}

func atxHeaderTrigger(p *Parser, container *Node) BlockStatus {
	match := reATXHeaderMarker.Find(p.currentLine[p.nextNonspace:])
	if !p.indented && match != nil {
		p.advanceNextNonspace()
		p.advanceOffset(uint32(len(match)), false)
		p.closeUnmatchedBlocks()
		container := p.addChild(Header, p.nextNonspace)
		container.level = uint32(len(bytes.Trim(match, " \t\n\r"))) // number of #s
		reLeft := regexp.MustCompile("^ *#+ *$")
		reRight := regexp.MustCompile(" +#+ *$")
		container.content = reRight.ReplaceAll(reLeft.ReplaceAll(p.currentLine[p.offset:], []byte{}), []byte{})
		//parser.currentLine.slice(parser.offset).replace(/^ *#+ *$/, '').replace(/ +#+ *$/, '');
		p.advanceOffset(uint32(len(p.currentLine))-p.offset, false)
		return LeafMatch
	}
	return NoMatch
}

func setextHeaderCondition(p *Parser, container *Node) (bool, byte) {
	if p.indented {
		return false, 0
	}
	if container.Type != Paragraph {
		return false, 0
	}
	if !bytes.HasSuffix(container.content, []byte("\n")) {
		return false, 0
	}
	match := reSetextHeaderLine.FindSubmatch(p.currentLine[p.nextNonspace:])
	if match != nil {
		return true, match[0][0]
	} else {
		return false, 0
	}
}

func levelFromChar(char byte) uint32 {
	if char == '=' {
		return 1
	} else {
		return 2
	}
}

func setextHeaderTrigger(p *Parser, container *Node) BlockStatus {
	if ok, char := setextHeaderCondition(p, container); ok {
		p.closeUnmatchedBlocks()
		header := NewNode(Header)
		header.level = levelFromChar(char)
		header.content = container.content
		container.insertAfter(header)
		container.unlink()
		p.tip = header
		p.advanceOffset(ulen(p.currentLine)-p.offset, false)
		return LeafMatch
	} else {
		return NoMatch
	}
}

func hruleTrigger(p *Parser, container *Node) BlockStatus {
	match := reHrule.Find(p.currentLine[p.nextNonspace:])
	if !p.indented && match != nil {
		p.closeUnmatchedBlocks()
		p.addChild(HorizontalRule, p.nextNonspace)
		p.advanceOffset(uint32(len(p.currentLine))-p.offset, false)
		return LeafMatch
	} else {
		return NoMatch
	}
}

func blockquoteTrigger(p *Parser, container *Node) BlockStatus {
	if !p.indented && peek(p.currentLine, p.nextNonspace) == '>' {
		p.advanceNextNonspace()
		p.advanceOffset(1, false)
		if peek(p.currentLine, p.offset) == ' ' {
			p.advanceOffset(1, false)
		}
		p.closeUnmatchedBlocks()
		p.addChild(BlockQuote, p.nextNonspace)
		return ContainerMatch
	} else {
		return NoMatch
	}
}

func htmlBlockTrigger(p *Parser, container *Node) BlockStatus {
	if !p.indented && peek(p.currentLine, p.nextNonspace) == '<' {
		s := p.currentLine[p.nextNonspace:]
		for blockType := 1; blockType <= 7; blockType++ {
			match := reHtmlBlockOpen[blockType].Find(s)
			if match != nil && (blockType < 7 || container.Type == Paragraph) {
				p.closeUnmatchedBlocks()
				b := p.addChild(HtmlBlock, p.offset)
				b.htmlBlockType = blockType
				return LeafMatch
			}
		}
	}
	return NoMatch
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

func parseListMarker(line []byte, offset, indent uint32) *ListData {
	rest := line[offset:]
	spacesAfterMarker := uint32(0)
	data := &ListData{
		tight:        true,
		markerOffset: indent,
	}
	match := reBulletListMarker.FindSubmatch(rest)
	if match != nil {
		spacesAfterMarker = ulen(match[1])
		data.Type = BulletList
		data.bulletChar = match[0][0]
	} else {
		match = reOrderedListMarker.FindSubmatch(rest)
		if match != nil {
			spacesAfterMarker = ulen(match[3])
			data.Type = OrderedList
			start, _ := strconv.Atoi(string(match[1]))
			data.start = uint32(start)
			data.delimiter = match[2][0]
		} else {
			return nil
		}
	}
	blankItem := len(match[0]) == len(rest)
	if spacesAfterMarker >= 5 || spacesAfterMarker < 1 || blankItem {
		data.padding = ulen(match[0]) - spacesAfterMarker + 1
	} else {
		data.padding = ulen(match[0])
	}
	return data
}

// Returns true if the two list items are of the same type, with the same
// delimiter and bullet character.  This is used in agglomerating list items
// into lists.
func listsMatch(listData, itemData *ListData) bool {
	return listData.Type == itemData.Type &&
		listData.delimiter == itemData.delimiter &&
		listData.bulletChar == itemData.bulletChar
}

func listItemTrigger(p *Parser, container *Node) BlockStatus {
	data := parseListMarker(p.currentLine, p.nextNonspace, p.indent)
	if data != nil && (!p.indented || container.Type == List) {
		p.closeUnmatchedBlocks()
		p.advanceNextNonspace()
		// recalculate data.padding, taking into account tabs:
		i := p.column
		p.advanceOffset(data.padding, false)
		data.padding = p.column - i
		// add the list if needed
		if p.tip.Type != List || !listsMatch(container.listData, data) {
			container = p.addChild(List, p.nextNonspace)
			container.listData = data
		}
		// add the list item
		container = p.addChild(Item, p.nextNonspace)
		container.listData = data
		return ContainerMatch
	} else {
		return NoMatch
	}
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

func fencedCodeTrigger(p *Parser, unused *Node) BlockStatus {
	if p.indented {
		return NoMatch
	}
	match := reCodeFence.FindSubmatch(p.currentLine[p.nextNonspace:])
	if match == nil {
		return NoMatch
	}
	fenceLength := ulen(match[0])
	p.closeUnmatchedBlocks()
	container := p.addChild(CodeBlock, p.nextNonspace)
	container.isFenced = true
	container.fenceLength = fenceLength
	container.fenceChar = match[0][0]
	container.fenceOffset = p.indent
	p.advanceNextNonspace()
	p.advanceOffset(fenceLength, false)
	return LeafMatch
}

func indentedCodeTrigger(p *Parser, unused *Node) BlockStatus {
	if p.indented && p.tip.Type != Paragraph && !p.blank {
		p.advanceOffset(CodeIndent, true)
		p.closeUnmatchedBlocks()
		p.addChild(CodeBlock, p.offset)
		return LeafMatch
	}
	return NoMatch
}
