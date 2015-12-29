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

var (
	blockTriggers = []func(p *Parser, container *Node) BlockStatus{
		atxHeaderTrigger,
		hruleTrigger,
		blockquoteTrigger,
	}

	blockHandlers = map[NodeType]BlockHandler{
		Document:       &DocumentBlockHandler{},
		Header:         &HeaderBlockHandler{},
		HorizontalRule: &HorizontalRuleBlockHandler{},
		BlockQuote:     &BlockQuoteBlockHandler{},
		Paragraph:      &ParagraphBlockHandler{},
	}

	reATXHeaderMarker = regexp.MustCompile("^#{1,6}(?: +|$)")
	reHrule           = regexp.MustCompile("^(?:(?:\\* *){3,}|(?:_ *){3,}|(?:- *){3,}) *$")
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
