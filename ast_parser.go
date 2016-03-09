package blackfriday

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
	}
}

func (p *Parser) finalize(block *Node, lineNumber uint32) {
	above := block.parent
	block.open = false
	blockHandlers[block.Type].Finalize(p, block)
	p.tip = above
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
