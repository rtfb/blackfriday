package blackfriday

import (
	"bytes"
	"regexp"
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
)

var (
	blockTriggers = []func(p *Parser, container *Node) BlockStatus{
		atxHeaderTrigger,
		hruleTrigger,
		blockquoteTrigger,
		htmlBlockTrigger,
	}

	blockHandlers = map[NodeType]BlockHandler{
		Document:       &DocumentBlockHandler{},
		Header:         &HeaderBlockHandler{},
		HorizontalRule: &HorizontalRuleBlockHandler{},
		BlockQuote:     &BlockQuoteBlockHandler{},
		Paragraph:      &ParagraphBlockHandler{},
		HtmlBlock:      &HtmlBlockHandler{},
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
	reNewlines := regexp.MustCompile("(\n *)+$")
	block.literal = reNewlines.ReplaceAll(block.content, []byte{})
	block.content = []byte{}
}

func (h *HtmlBlockHandler) CanContain(t NodeType) bool {
	return false
}

func (h *HtmlBlockHandler) AcceptsLines() bool {
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
