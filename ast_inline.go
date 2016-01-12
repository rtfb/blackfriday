package blackfriday

import (
	"bytes"
	"regexp"
)

var (
	reMain           = regexp.MustCompile("^[^\\n`\\\\[\\]\\!<&*_'\"]+")
	reWhitespaceChar = regexp.MustCompile("^\\s")
	reWhitespace     = regexp.MustCompile("\\s+")
	rePunctuation    = regexp.MustCompile("^[\u2000-\u206F\u2E00-\u2E7F\\'!\"#\\$%&\\(\\)\\*\\+,\\-\\.\\/:;<=>\\?@\\[\\]\\^_`\\{\\|\\}~]")
	reFinalSpace     = regexp.MustCompile(" *$")
	reInitialSpace   = regexp.MustCompile("^ *")
	reEscapable      = regexp.MustCompile("^" + Escapable)
	reTicksHere      = regexp.MustCompile("^`+")
	reTicks          = regexp.MustCompile("`+")
)

type InlineParser struct {
	subject    []byte
	pos        int
	delimiters *Delimiter
}

type Delimiter struct {
	ch        byte
	numDelims int
	node      *Node
	prev      *Delimiter
	next      *Delimiter
	canOpen   bool
	canClose  bool
	active    bool
}

func NewInlineParser() *InlineParser {
	return &InlineParser{
		subject:    []byte{},
		pos:        0,
		delimiters: nil,
	}
}

func text(s []byte) *Node {
	//node := NewNode(Text, NewSourceRange())
	node := NewNode(Text)
	node.literal = s
	return node
}

func (p *InlineParser) peek() byte {
	if p.pos < len(p.subject) {
		return p.subject[p.pos]
	}
	return 255 // XXX: figure out invalid values
}

// peekSlice() is the same as peek(), but returns a slice
func (p *InlineParser) peekSlice() []byte {
	return []byte{p.peek()}
}

func (p *InlineParser) scanDelims(ch byte) (numDelims int, canOpen, canClose bool) {
	numDelims = 0
	startPos := p.pos
	if ch == '\'' || ch == '"' {
		numDelims += 1
		p.pos += 1
	} else {
		for p.peek() == ch {
			numDelims += 1
			p.pos += 1
		}
	}
	if numDelims == 0 {
		return 0, false, false
	}
	charBefore := byte('\n')
	if startPos > 0 {
		charBefore = p.subject[startPos-1]
	}
	charAfter := p.peek()
	//cc_after = this.peek();
	//if (cc_after === -1) {
	//    char_after = '\n';
	//} else {
	//    char_after = fromCodePoint(cc_after);
	//}
	afterIsWhitespace := reWhitespaceChar.Match([]byte{charAfter})
	afterIsPunctuation := rePunctuation.Match([]byte{charAfter})
	beforeIsWhitespace := reWhitespaceChar.Match([]byte{charBefore})
	beforeIsPunctuation := rePunctuation.Match([]byte{charBefore})
	leftFlanking := !afterIsWhitespace && !(afterIsPunctuation && !beforeIsWhitespace && !beforeIsPunctuation)
	rightFlanking := !beforeIsWhitespace && !(beforeIsPunctuation && !afterIsPunctuation && !afterIsPunctuation)
	if ch == '_' {
		canOpen = leftFlanking && (!rightFlanking || beforeIsPunctuation)
		canClose = rightFlanking && (!leftFlanking || afterIsPunctuation)
	} else if ch == '\'' || ch == '"' {
		canOpen = leftFlanking && !rightFlanking
		canClose = rightFlanking
	} else {
		canOpen = leftFlanking
		canClose = rightFlanking
	}
	p.pos = startPos
	return
}

func (p *InlineParser) pushDelim(delim *Delimiter) {
	delim.prev = p.delimiters
	delim.next = nil
	p.delimiters = delim
	if p.delimiters.prev != nil {
		p.delimiters.prev.next = p.delimiters
	}
}

func (p *InlineParser) removeDelimiter(delim *Delimiter) {
	if delim.prev != nil {
		delim.prev.next = delim.next
	}
	if delim.next == nil {
		// top of stack
		p.delimiters = delim.prev
	} else {
		delim.next.prev = delim.prev
	}
}

func removeDelimitersBetween(bottom, top *Delimiter) {
	for bottom.next != top {
		bottom.next = top
		top.prev = bottom
	}
}

func (p *InlineParser) handleDelim(ch byte, block *Node) bool {
	numDelims, canOpen, canClose := p.scanDelims(ch)
	if numDelims < 1 {
		return false
	}
	startPos := p.pos
	p.pos += numDelims
	var contents []byte
	if ch == '\'' || ch == '"' {
		contents = []byte{ch}
	} else {
		contents = p.subject[startPos:p.pos]
	}
	node := text(contents)
	block.appendChild(node)
	p.pushDelim(&Delimiter{
		ch:        ch,
		numDelims: numDelims,
		node:      node,
		canOpen:   canOpen,
		canClose:  canClose,
		active:    true,
	})
	return true
}

func (p *InlineParser) parseString(block *Node) bool {
	match := reMain.Find(p.subject[p.pos:])
	if match == nil {
		return false
	}
	p.pos += len(match)
	block.appendChild(text(match))
	return true
}

// Returns str[pos] with Pythonesque semantics for negative pos
func peekPos(str []byte, pos int) byte {
	if pos >= 0 {
		return str[pos]
	}
	if len(str)+pos < 0 {
		return 255 // XXX: figure out invalid values
	}
	return str[len(str)+pos]
}

// If re matches at current position in the subject, advance
// position in subject and return the match; otherwise return nil.
func (p *InlineParser) match(re *regexp.Regexp) []byte {
	m := re.FindIndex(p.subject[p.pos:])
	if m == nil {
		return nil
	}
	ret := p.subject[p.pos+m[0] : p.pos+m[1]]
	p.pos += m[1]
	return ret
}

func (p *InlineParser) parseNewline(block *Node) bool {
	p.pos += 1 // assume we're at a \n
	// check previous node for trailing spaces
	lastChild := block.lastChild
	if lastChild != nil && lastChild.Type == Text && peekPos(lastChild.literal, -1) == ' ' {
		hardBreak := peekPos(lastChild.literal, -2) == ' '
		lastChild.literal = reFinalSpace.ReplaceAll(lastChild.literal, []byte{})
		childType := Softbreak
		if hardBreak {
			childType = Hardbreak
		}
		block.appendChild(NewNode(childType))
	} else {
		block.appendChild(NewNode(Softbreak))
	}
	p.match(reInitialSpace) // gobble leading spaces in next line
	return true
}

// Parse a backslash-escaped special character, adding either the escaped
// character, a hard line break (if the backslash is followed by a newline),
// or a literal backslash to the block's children.  Assumes current character
// is a backslash.
func (p *InlineParser) parseBackslash(block *Node) bool {
	p.pos += 1
	if p.peek() == '\n' {
		block.appendChild(NewNode(Hardbreak))
		p.pos += 1
	} else if reEscapable.Match(p.peekSlice()) {
		block.appendChild(text(p.peekSlice()))
		p.pos += 1
	} else {
		block.appendChild(text([]byte{'\\'}))
	}
	return true
}

// Attempt to parse backticks, adding either a backtick code span or a
// literal sequence of backticks.
func (p *InlineParser) parseBackticks(block *Node) bool {
	ticks := p.match(reTicksHere)
	if ticks == nil {
		return false
	}
	afterOpenTicks := p.pos
	matched := p.match(reTicks)
	for matched != nil {
		if bytes.Equal(matched, ticks) {
			node := NewNode(Code)
			node.literal = reWhitespace.ReplaceAll(bytes.TrimSpace(p.subject[afterOpenTicks:p.pos-len(ticks)]), []byte{' '})
			block.appendChild(node)
			return true
		}
		matched = p.match(reTicks)
	}
	// If we got here, we didn't match a closing backtick sequence.
	p.pos = afterOpenTicks
	block.appendChild(text(ticks))
	return true
}

func (p *InlineParser) parseInline(block *Node) bool {
	res := false
	ch := p.peek()
	if ch == 255 { // XXX: invalid char
		return false
	}
	switch ch {
	case '\n':
		res = p.parseNewline(block)
	case '\\':
		res = p.parseBackslash(block)
	case '`':
		res = p.parseBackticks(block)
	case '*', '_':
		res = p.handleDelim(ch, block)
		break
	default:
		res = p.parseString(block)
		break
	}
	if !res {
		p.pos += 1
		block.appendChild(text([]byte{ch}))
	}
	return true
}

func isEmphasisChar(ch byte) bool {
	return bytes.IndexByte([]byte{'_', '*', '\'', '"'}, ch) >= 0
}

func (p *InlineParser) processEmphasis(stackBottom *Delimiter) {
	openersBottom := make(map[byte]*Delimiter)
	openersBottom['_'] = stackBottom
	openersBottom['*'] = stackBottom
	openersBottom['\''] = stackBottom
	openersBottom['"'] = stackBottom
	// find first closer above stackBottom:
	closer := p.delimiters
	for closer != nil && closer.prev != stackBottom {
		closer = closer.prev
	}
	// move forward, looking for closers, and handling each
	for closer != nil {
		if !(closer.canClose && isEmphasisChar(closer.ch)) {
			closer = closer.next
		} else {
			// found emphasis closer. Now look back for first matching opener:
			opener := closer.prev
			openerFound := false
			for opener != nil && opener != stackBottom && opener != openersBottom[closer.ch] {
				if opener.ch == closer.ch && opener.canOpen {
					openerFound = true
					break
				}
				opener = opener.prev
			}
			oldCloser := closer
			if closer.ch == '*' || closer.ch == '_' {
				if !openerFound {
					closer = closer.next
				} else {
					useDelims := 0
					// calculate actual number of delimiters used from closer
					if closer.numDelims < 3 || opener.numDelims < 3 {
						useDelims = opener.numDelims
						if closer.numDelims <= opener.numDelims {
							useDelims = closer.numDelims
						}
					} else {
						useDelims = 1
						if closer.numDelims%2 == 0 {
							useDelims = 2
						}
					}
					openerInl := opener.node
					closerInl := closer.node
					// remove used delimiters from stack elts and inlines
					opener.numDelims -= useDelims
					closer.numDelims -= useDelims
					openerInl.literal = openerInl.literal[:len(openerInl.literal)-useDelims]
					closerInl.literal = closerInl.literal[:len(closerInl.literal)-useDelims]
					// build contents for new emph element
					nodeType := Strong
					if useDelims == 1 {
						nodeType = Emph
					}
					emph := NewNode(nodeType)
					tmp := openerInl.next
					for tmp != nil && tmp != closerInl {
						next := tmp.next
						tmp.unlink()
						emph.appendChild(tmp)
						tmp = next
					}
					openerInl.insertAfter(emph)
					// remove elts between opener and closer in delimiters stack
					removeDelimitersBetween(opener, closer)
					// if opener has 0 delims, remove it and the inline
					if opener.numDelims == 0 {
						openerInl.unlink()
						p.removeDelimiter(opener)
					}
					if closer.numDelims == 0 {
						closerInl.unlink()
						tempStack := closer.next
						p.removeDelimiter(closer)
						closer = tempStack
					}
				}
			} else if closer.ch == '\'' {
				closer.node.literal = []byte("\u2019")
				if openerFound {
					opener.node.literal = []byte("\u2018")
				}
				closer = closer.next
			} else if closer.ch == '"' {
				closer.node.literal = []byte("\u201D")
				if openerFound {
					opener.node.literal = []byte("\u201C")
				}
				closer = closer.next
			}
			if !openerFound {
				// Set lower bound for future searches for openers:
				if closer != nil {
					openersBottom[closer.ch] = oldCloser.prev
				}
				if !oldCloser.canOpen {
					// We can remove a closer that can't be an opener,
					// once we've seen there's no matching opener:
					p.removeDelimiter(oldCloser)
				}
			}
		}
	}
	// remove all delimiters
	for p.delimiters != nil && p.delimiters != stackBottom {
		p.removeDelimiter(p.delimiters)
	}
}

func (p *InlineParser) parse(block *Node) {
	p.subject = bytes.Trim(block.content, " \n\r")
	p.pos = 0
	p.delimiters = nil
	for p.parseInline(block) {
	}
	block.content = nil // allow raw string to be garbage collected
	p.processEmphasis(nil)
}
