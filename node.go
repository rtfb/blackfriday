package blackfriday

import (
	"bytes"
	"fmt"
)

type NodeType int

const (
	Document NodeType = iota
	BlockQuote
	List
	Item
	Paragraph
	Header
	HorizontalRule
	Emph
	Strong
	Link
	Image
	Text
	HtmlBlock
)

var nodeTypeNames = []string{
	Document:       "Document",
	BlockQuote:     "BlockQuote",
	List:           "List",
	Item:           "Item",
	Paragraph:      "Paragraph",
	Header:         "Header",
	HorizontalRule: "HorizontalRule",
	Emph:           "Emph",
	Strong:         "Strong",
	Link:           "Link",
	Image:          "Image",
	Text:           "Text",
	HtmlBlock:      "HtmlBlock",
}

func (t NodeType) String() string {
	return nodeTypeNames[t]
}

type Node struct {
	Type          NodeType
	parent        *Node
	firstChild    *Node
	lastChild     *Node
	prev          *Node // prev sibling
	next          *Node // next sibling
	content       []byte
	level         uint32
	open          bool
	lastLineBlank bool
	literal       []byte
	htmlBlockType int // In case Type == HtmlBlock, this holds its type
}

func NewNode(typ NodeType) *Node {
	return &Node{
		Type:          typ,
		parent:        nil,
		firstChild:    nil,
		lastChild:     nil,
		prev:          nil,
		next:          nil,
		content:       nil,
		level:         0,
		open:          true,
		lastLineBlank: false,
		literal:       nil,
	}
}

func (n *Node) unlink() {
	if n.prev != nil {
		n.prev.next = n.next
	} else if n.parent != nil {
		n.parent.firstChild = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else if n.parent != nil {
		n.parent.lastChild = n.prev
	}
	n.parent = nil
	n.next = nil
	n.prev = nil
}

func (n *Node) appendChild(child *Node) {
	child.unlink()
	child.parent = n
	if n.lastChild != nil {
		n.lastChild.next = child
		child.prev = n.lastChild
		n.lastChild = child
	} else {
		n.firstChild = child
		n.lastChild = child
	}
}

func (n *Node) isContainer() bool {
	switch n.Type {
	case Document:
		fallthrough
	case BlockQuote:
		fallthrough
	case List:
		fallthrough
	case Item:
		fallthrough
	case Paragraph:
		fallthrough
	case Header:
		fallthrough
	case Emph:
		fallthrough
	case Strong:
		fallthrough
	case Link:
		fallthrough
	case Image:
		return true
	default:
		return false
	}
	return false
}

type NodeWalker struct {
	current  *Node
	root     *Node
	entering bool
}

func NewNodeWalker(root *Node) *NodeWalker {
	return &NodeWalker{
		current:  root,
		root:     nil,
		entering: true,
	}
}

func (nw *NodeWalker) next() (*Node, bool) {
	if nw.current == nil {
		return nil, false
	}
	if nw.root == nil {
		nw.root = nw.current
		return nw.current, nw.entering
	}
	if nw.entering && nw.current.isContainer() {
		if nw.current.firstChild != nil {
			nw.current = nw.current.firstChild
			nw.entering = true
		} else {
			nw.entering = false
		}
	} else if nw.current.next == nil {
		nw.current = nw.current.parent
		nw.entering = false
	} else {
		nw.current = nw.current.next
		nw.entering = true
	}
	if nw.current == nw.root {
		return nil, false
	}
	return nw.current, nw.entering
}

func (nw *NodeWalker) resumeAt(node *Node, entering bool) {
	nw.current = node
	nw.entering = entering
}

// XXX: this is broken as of now. It seems like it should start working when
// inline parser starts working and producing non-container leave nodes. For
// now, explicit recursive tree walk should do the job.
func forEachNode(root *Node, f func(node *Node, entering bool)) {
	walker := NewNodeWalker(root)
	node, entering := walker.next()
	for node != nil {
		f(node, entering)
		node, entering = walker.next()
	}
}

func dump(ast *Node) {
	fmt.Println(dumpString(ast))
}

/*
TODO: use this one when forEachNode starts working
func dumpString(ast *Node) string {
	result := ""
	forEachNode(ast, func(node *Node, entering bool) {
		indent := ""
		tmp := node.parent
		for tmp != nil {
			indent += "\t"
			tmp = tmp.parent
		}
		content := node.literal
		if content == nil {
			content = node.content
		}
		result += fmt.Sprintf("%s%s(%q)\n", indent, node.Type, content)
	})
	return result
}
*/

func dump_r(ast *Node, depth int) string {
	indent := bytes.Repeat([]byte("\t"), depth)
	content := ast.literal
	if content == nil {
		content = ast.content
	}
	result := fmt.Sprintf("%s%s(%q)\n", indent, ast.Type, content)
	for n := ast.firstChild; n != nil; n = n.next {
		result += dump_r(n, depth+1)
	}
	return result
}

func dumpString(ast *Node) string {
	return dump_r(ast, 0)
}
