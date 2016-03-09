package blackfriday

import "bytes"

type Parser struct {
	doc                  *Node
	tip                  *Node // = doc
	oldTip               *Node
	refmap               RefMap
	lineNumber           uint32
	lastLineLength       uint32
	offset               uint32
	column               uint32
	nextNonspace         uint32
	nextNonspaceColumn   uint32
	lastMatchedContainer *Node // = doc
	currentLine          []byte
	lines                [][]byte // input document.split(newlines)
	indent               uint32
	indented             bool
	blank                bool
	allClosed            bool
	//inlineParser         *InlineParser
}

type Ref struct {
	Dest  []byte
	Title []byte
}

type RefMap map[string]*Ref

func NewParser() *Parser {
	docNode := NewNode(Document)
	return &Parser{
		doc:                  docNode,
		tip:                  docNode,
		oldTip:               docNode,
		lineNumber:           0,
		lastLineLength:       0,
		offset:               0,
		column:               0,
		lastMatchedContainer: docNode,
		currentLine:          []byte{},
		lines:                nil,
		allClosed:            true,
		//inlineParser:         NewInlineParser(),
	}
}

func (p *Parser) incorporateLine(line []byte) {
	allMatched := true
	container := p.doc
	p.oldTip = p.tip
	p.offset = 0
	p.lineNumber += 1
	p.currentLine = line
	dbg("%3d: %q\n", p.lineNumber, string(line))
	lastChild := container.lastChild
	for lastChild != nil && lastChild.open {
		container = lastChild
		lastChild = container.lastChild
		p.findNextNonspace()
		switch blockHandlers[container.Type].Continue(p, container) {
		case Matched: // matched, keep going
			break
		case NotMatched: // failed to match a block
			allMatched = false
			break
		case Completed: // we've hit end of line for fenced code close and can return
			p.lastLineLength = ulen(line)
			return
		default:
			panic("Continue returned illegal value, must be 0, 1, or 2")
		}
		if !allMatched {
			container = container.parent // back up to last matching block
			break
		}
	}
	dbg("incorporateLine -- xx\n")
	p.allClosed = container == p.oldTip
	p.lastMatchedContainer = container
	matchedLeaf := container.Type != Paragraph && blockHandlers[container.Type].AcceptsLines()
	for !matchedLeaf {
		p.findNextNonspace()
		//if !p.indented && reMaybeSpecial.Find(line[p.nextNonspace:]) == nil {
		//	p.advanceNextNonspace()
		//	break
		//}
		nothingMatched := true
		for _, trigger := range blockTriggers {
			st := trigger(p, container)
			if st != NoMatch {
				container = p.tip
				nothingMatched = false
				if st == LeafMatch {
					matchedLeaf = true
				}
				break
			}
		}
		if nothingMatched {
			p.advanceNextNonspace()
			break
		}
	}
	dbg("incorporateLine -- yy\n")
	if !p.allClosed && !p.blank && p.tip.Type == Paragraph {
		p.addLine()
	} else {
		p.closeUnmatchedBlocks()
		if p.blank && container.lastChild != nil {
			container.lastChild.lastLineBlank = true
		}
		t := container.Type
		lastLineBlank := p.blank /* &&
		!(t == BlockQuote || (t == CodeBlock && container.isFenced) ||
			(t == Item && container.firstChild == nil && container.sourcePos.line == p.lineNumber))
		*/
		cont := container
		for cont != nil {
			cont.lastLineBlank = lastLineBlank
			cont = cont.parent
		}
		if blockHandlers[t].AcceptsLines() {
			p.addLine()
			if t == HtmlBlock && canCloseHtmlBlock(container, p) {
				p.finalize(container, p.lineNumber)
			}
		} else if p.offset < ulen(line) && !p.blank {
			container = p.addChild(Paragraph, p.offset)
			p.advanceNextNonspace()
			p.addLine()
		}
	}
	dbg("incorporateLine -- zz\n")
	p.lastLineLength = ulen(line)
}

func canCloseHtmlBlock(container *Node, p *Parser) bool {
	if container.htmlBlockType < 1 || container.htmlBlockType > 5 {
		return false
	}
	s := p.currentLine[p.offset:]
	match := reHtmlBlockClose[container.htmlBlockType].Find(s)
	return match != nil
}

func (p *Parser) finalize(block *Node, lineNumber uint32) {
	above := block.parent
	block.open = false
	blockHandlers[block.Type].Finalize(p, block)
	p.tip = above
}

func (p *Parser) processInlines(ast *Node) {
	//p.inlineParser.refmap = p.refmap
	//p.inlineParser.options = p.options
	/*
		walker := NewNodeWalker(ast)
		for node := ast; node != nil; node, _ = walker.next() {
			if node.Type == Paragraph || node.Type == Header {
				// TODO
				//p.inlineParser.parse(node)
			}
		}
	*/
	forEachNode(ast, func(node *Node, entering bool) {
		if node.Type == Paragraph || node.Type == Header {
			//p.inlineParser.parse(node)
		}
	})
}

func (p *Parser) addLine() {
	p.tip.content = append(p.tip.content, p.currentLine[p.offset:]...)
	p.tip.content = append(p.tip.content, '\n')
}

func (p *Parser) addChild(node NodeType, offset uint32) *Node {
	for !blockHandlers[p.tip.Type].CanContain(node) {
		p.finalize(p.tip, p.lineNumber-1)
	}
	newNode := NewNode(node)
	newNode.content = []byte{}
	p.tip.appendChild(newNode)
	p.tip = newNode
	return newNode
}

func (p *Parser) advanceOffset(count uint32, columns bool) {
	var i uint32 = 0
	var cols uint32 = 0
	for p.offset+i < ulen(p.currentLine) {
		if columns {
			if cols >= count {
				break
			}
		} else {
			if i >= count {
				break
			}
		}
		if p.currentLine[p.offset+i] == '\t' {
			cols += (4 - ((p.column + cols) % 4))
		} else {
			cols += 1
		}
		i += 1
	}
	p.offset += i
	p.column += cols
}

func (p *Parser) advanceNextNonspace() {
	p.offset = p.nextNonspace
	p.column = p.nextNonspaceColumn
}

func (p *Parser) closeUnmatchedBlocks() {
	if !p.allClosed {
		for p.oldTip != p.lastMatchedContainer {
			parent := p.oldTip.parent
			p.finalize(p.oldTip, p.lineNumber-1)
			p.oldTip = parent
		}
		p.allClosed = true
	}
}

func (p *Parser) findNextNonspace() {
	i := p.offset
	cols := p.column
	var c byte
	for i < ulen(p.currentLine) {
		c = p.currentLine[i]
		if c == ' ' {
			i += 1
			cols += 1
		} else if c == '\t' {
			i += 1
			cols += (4 - (cols % 4))
		} else {
			break
		}
	}
	p.blank = c == '\n' || c == '\r' || i == ulen(p.currentLine)
	p.nextNonspace = i
	p.nextNonspaceColumn = cols
	p.indent = p.nextNonspaceColumn - p.column
	p.indented = p.indent >= 4
}

func (p *Parser) parse(input []byte) *Node {
	p.lines = bytes.Split(input, []byte{'\n'})
	var numLines uint32 = uint32(len(p.lines))
	if input[len(input)-1] == '\n' {
		// ignore last blank line created by final newline
		numLines -= 1
	}
	var i uint32
	for i = 0; i < numLines; i += 1 {
		p.incorporateLine(p.lines[i])
	}
	for p.tip != nil {
		p.finalize(p.tip, numLines)
	}
	p.processInlines(p.doc)
	return p.doc
}
