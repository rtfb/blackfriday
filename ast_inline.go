package blackfriday

import (
	"bytes"
	"html"
	"regexp"
)

const (
	EscapedChar           = "\\\\" + Escapable
	RegChar               = "[^\\\\()\\x00-\\x20]"
	InParensNoSp          = "\\((" + RegChar + "|" + EscapedChar + "|\\\\)*\\)"
	HTMLComment           = "<!---->|<!--(?:-?[^>-])(?:-?[^-])*-->"
	ProcessingInstruction = "[<][?].*?[?][>]"
	Declaration           = "<![A-Z]+" + "\\s+[^>]*>"
	CDATA                 = "<!\\[CDATA\\[[\\s\\S]*?\\]\\]>"
	HTMLTag               = "(?:" + OpenTag + "|" + CloseTag + "|" + HTMLComment + "|" +
		ProcessingInstruction + "|" + Declaration + "|" + CDATA + ")"
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
	reEntityHere     = regexp.MustCompile("(?i)^" + Entity)
	reEmailAutolink  = regexp.MustCompile("^<([a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)>")
	reAutolink       = regexp.MustCompile("(?i)^<(?:coap|doi|javascript|aaa|aaas|about|acap|cap|cid|crid|data|dav|dict|dns|file|ftp|geo|go|gopher|h323|http|https|iax|icap|im|imap|info|ipp|iris|iris.beep|iris.xpc|iris.xpcs|iris.lwz|ldap|mailto|mid|msrp|msrps|mtqp|mupdate|news|nfs|ni|nih|nntp|opaquelocktoken|pop|pres|rtsp|service|session|shttp|sieve|sip|sips|sms|snmp|soap.beep|soap.beeps|tag|tel|telnet|tftp|thismessage|tn3270|tip|tv|urn|vemmi|ws|wss|xcon|xcon-userid|xmlrpc.beep|xmlrpc.beeps|xmpp|z39.50r|z39.50s|adiumxtra|afp|afs|aim|apt|attachment|aw|beshare|bitcoin|bolo|callto|chrome|chrome-extension|com-eventbrite-attendee|content|cvs|dlna-playsingle|dlna-playcontainer|dtn|dvb|ed2k|facetime|feed|finger|fish|gg|git|gizmoproject|gtalk|hcp|icon|ipn|irc|irc6|ircs|itms|jar|jms|keyparc|lastfm|ldaps|magnet|maps|market|message|mms|ms-help|msnim|mumble|mvn|notes|oid|palm|paparazzi|platform|proxy|psyc|query|res|resource|rmi|rsync|rtmp|secondlife|sftp|sgn|skype|smb|soldat|spotify|ssh|steam|svn|teamspeak|things|udp|unreal|ut2004|ventrilo|view-source|webcal|wtai|wyciwyg|xfire|xri|ymsgr):[^<>\x00-\x20]*>")
	reLinkTitle      = regexp.MustCompile(
		"^(?:\"(" + EscapedChar + "|[^\"\\x00])*\"" +
			"|" +
			"'(" + EscapedChar + "|[^'\\x00])*'" +
			"|" +
			"\\((" + EscapedChar + "|[^)\\x00])*\\))")
	reLinkDestinationBraces = regexp.MustCompile(
		"^(?:[<](?:[^<>\\n\\\\\\x00]" + "|" + EscapedChar + "|" + "\\\\)*[>])")
	reLinkDestination = regexp.MustCompile(
		"^(?:" + RegChar + "+|" + EscapedChar + "|\\\\|" + InParensNoSp + ")*")
	reLinkLabel        = regexp.MustCompile("^\\[(?:[^\\\\\\[\\]]|" + EscapedChar + "|\\\\){0,1000}\\]")
	reSpnl             = regexp.MustCompile("^ *(?:\n *)?")
	reSpaceAtEndOfLine = regexp.MustCompile("^ *(?:\n|$)")
	reHtmlTag          = regexp.MustCompile("(?i)^" + HTMLTag)
)

type InlineParser struct {
	subject    []byte
	pos        int
	delimiters *Delimiter
	refmap     RefMap
}

type Delimiter struct {
	ch        byte
	numDelims int
	index     int
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

func normalizeURI(s []byte) []byte {
	return s // TODO: implement
}

func normalizeReference(s []byte) []byte {
	return s // TODO: implement
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

// Attempt to parse an autolink (URL or email in pointy brackets).
func (p *InlineParser) parseAutolink(block *Node) bool {
	m := p.match(reEmailAutolink)
	if m != nil {
		dest := m[1 : len(m)-1]
		node := NewNode(Link)
		//node.destination = normalizeURI([]byte("mailto:") + dest)
		node.destination = normalizeURI(append([]byte("mailto:"), dest...))
		node.appendChild(text(dest))
		block.appendChild(node)
		return true
	}
	m = p.match(reAutolink)
	if m != nil {
		dest := m[1 : len(m)-1]
		node := NewNode(Link)
		node.destination = normalizeURI(dest)
		node.appendChild(text(dest))
		block.appendChild(node)
		return true
	}
	return false
}

// Attempt to parse a raw HTML tag.
func (p *InlineParser) parseHtmlTag(block *Node) bool {
	m := p.match(reHtmlTag)
	if m == nil {
		return false
	}
	node := NewNode(HtmlSpan)
	node.literal = m
	block.appendChild(node)
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

func (p *InlineParser) parseOpenBracket(block *Node) bool {
	startPos := p.pos
	p.pos += 1
	node := text([]byte{'['})
	block.appendChild(node)
	p.pushDelim(&Delimiter{
		ch:        '[',
		numDelims: 1,
		node:      node,
		canOpen:   true,
		canClose:  false,
		index:     startPos,
		active:    true,
	})
	return true
}

// IF next character is [, add ! delimiter to delimiter stack and
// add a text node to block's children.  Otherwise just add a text node.
func (p *InlineParser) parseBang(block *Node) bool {
	startPos := p.pos
	p.pos += 1
	if p.peek() == '[' {
		p.pos += 1
		node := text([]byte("!["))
		block.appendChild(node)
		p.pushDelim(&Delimiter{
			ch:        '!',
			numDelims: 1,
			node:      node,
			canOpen:   true,
			canClose:  false,
			index:     startPos + 1,
			active:    true,
		})
	} else {
		block.appendChild(text([]byte{'!'}))
	}
	return true
}

func (p *InlineParser) parseLinkDestAndTitle() (dest, title []byte, ok bool) {
	if !p.spnl() {
		return nil, nil, false
	}
	dest = p.parseLinkDestination()
	if dest == nil {
		return nil, nil, false
	}
	if !p.spnl() {
		return nil, nil, false
	}
	// make sure there's a space before the title:
	haveWhitespace := reWhitespaceChar.Match([]byte{p.subject[p.pos-1]})
	if haveWhitespace {
		title = p.parseLinkTitle()
	}
	return dest, title, (p.spnl() && p.peek() == ')')
}

// Try to match close bracket against an opening in the delimiter
// stack.  Add either a link or image, or a plain [ character,
// to block's children.  If there is a matching delimiter,
// remove it from the delimiter stack.
func (p *InlineParser) parseCloseBracket(block *Node) bool {
	var dest, title []byte
	var matched bool
	p.pos++
	startPos := p.pos
	// look through stack of delimiters for a [ or ![
	opener := p.delimiters
	for opener != nil {
		if opener.ch == '[' || opener.ch == '!' {
			break
		}
		opener = opener.prev
	}
	if opener == nil {
		block.appendChild(text([]byte{']'}))
		return true
	}
	if !opener.active {
		// no matched opener, just return a literal
		block.appendChild(text([]byte{']'}))
		// take opener off emphasis stack
		p.removeDelimiter(opener)
		return true
	}
	// If we got here, open is a potential opener
	isImage := opener.ch == '!'
	// Check to see if we have a link/image
	if p.peek() == '(' {
		// Inline link?
		p.pos++
		var ok bool
		dest, title, ok = p.parseLinkDestAndTitle()
		if ok {
			p.pos++
			matched = true
		}
	} else {
		// Next, see if there's a link label
		savePos := p.pos
		p.spnl()
		beforeLabel := p.pos
		n := int(p.parseLinkLabel()) // XXX: int-uint32 mismatch
		var reflabel []byte
		if n == 0 || n == 2 {
			// empty or missing second label
			reflabel = p.subject[opener.index:startPos]
		} else {
			reflabel = p.subject[beforeLabel : beforeLabel+n]
		}
		if n == 0 {
			// If shortcut reference link, rewind before spaces we skipped.
			p.pos = savePos
		}
		// lookup rawlabel in refmap
		link := p.refmap[string(normalizeReference(reflabel))]
		if link != nil {
			dest = link.Dest
			title = link.Title
			matched = true
		}
	}
	if matched {
		nodeType := Link
		if isImage {
			nodeType = Image
		}
		node := NewNode(nodeType)
		node.destination = dest
		node.title = title
		tmp := opener.node.next
		for tmp != nil {
			next := tmp.next
			tmp.unlink()
			node.appendChild(tmp)
			tmp = next
		}
		block.appendChild(node)
		p.processEmphasis(opener.prev)
		opener.node.unlink()
		// processEmphasis will remove this and later delimiters.
		// Now, for a link, we also deactivate earlier link openers.
		// (no links in links)
		if !isImage {
			opener = p.delimiters
			for opener != nil {
				if opener.ch == '[' {
					opener.active = false // deactivate this opener
				}
				opener = opener.prev
			}
		}
		return true
	} else {
		p.removeDelimiter(opener)
		p.pos = startPos
		block.appendChild(text([]byte{']'}))
		return true
	}
	return false
}

// First, unescape every HTML entity, then escape several unsafe chars back
func decodeHTML(str []byte) []byte {
	var buff bytes.Buffer
	for _, b := range []byte(html.UnescapeString(string(str))) {
		switch b {
		case '&':
			buff.Write([]byte("&amp;"))
		case '<':
			buff.Write([]byte("&lt;"))
		case '>':
			buff.Write([]byte("&gt;"))
		case '"':
			buff.Write([]byte("&quot;"))
		default:
			buff.WriteByte(b)
		}
	}
	return buff.Bytes()
}

// Attempt to parse an entity.
func (p *InlineParser) parseEntity(block *Node) bool {
	m := p.match(reEntityHere)
	if m == nil {
		return false
	}
	block.appendChild(text(decodeHTML(m)))
	return true
}

// Attempt to parse link title (sans quotes), returning the string
// or null if no match.
func (p *InlineParser) parseLinkTitle() []byte {
	title := p.match(reLinkTitle)
	if title == nil {
		return nil
	}
	// chop off quotes from title and unescape:
	return unescapeString(title[1 : len(title)-1])
}

// Attempt to parse link destination, returning the string or
// null if no match.
func (p *InlineParser) parseLinkDestination() []byte {
	res := p.match(reLinkDestinationBraces)
	if res == nil {
		res = p.match(reLinkDestination)
		if res == nil {
			return nil
		} else {
			return normalizeURI(unescapeString(res))
		}
	} else { // chop off surrounding <..>:
		return normalizeURI(unescapeString(res[1 : len(res)-2]))
	}
}

// Attempt to parse a link label, returning number of characters parsed.
func (p *InlineParser) parseLinkLabel() uint32 {
	m := p.match(reLinkLabel)
	if m == nil || len(m) > 1001 {
		return 0
	}
	return ulen(m)
}

// Parse zero or more space characters, including at most one newline
func (p *InlineParser) spnl() bool {
	p.match(reSpnl)
	return true
}

func (p *InlineParser) parseReference(s []byte, refmap RefMap) int {
	p.pos = 0
	startPos := p.pos
	// label:
	matchChars := p.parseLinkLabel()
	var rawLabel []byte
	if matchChars == 0 {
		return 0
	} else {
		rawLabel = p.subject[:matchChars]
	}
	// colon:
	if p.peek() == ':' {
		p.pos++
	} else {
		p.pos = startPos
		return 0
	}
	// link url
	p.spnl()
	dest := p.parseLinkDestination()
	if dest == nil || len(dest) == 0 {
		p.pos = startPos
		return 0
	}
	beforeTitle := p.pos
	p.spnl()
	title := p.parseLinkTitle()
	if title == nil {
		title = []byte{}
		// rewind before spaces
		p.pos = beforeTitle
	}
	// make sure we're at line end:
	atLineEnd := true
	if p.match(reSpaceAtEndOfLine) == nil {
		if bytes.Equal(title, []byte{}) {
			atLineEnd = false
		} else {
			// the potential title we found is not at the line end,
			// but it could still be a legal link reference if we
			// discard the title
			title = []byte{}
			// rewind before spaces
			p.pos = beforeTitle
			// and instead check if the link URL is at the line end
			atLineEnd = p.match(reSpaceAtEndOfLine) != nil
		}
	}
	if !atLineEnd {
		p.pos = startPos
		return 0
	}
	normLabel := string(normalizeReference(rawLabel))
	if normLabel == "" {
		// label must contain non-whitespace characters
		p.pos = startPos
		return 0
	}
	if _, ok := refmap[normLabel]; !ok {
		refmap[normLabel] = &Ref{Dest: dest, Title: title}
	}
	return p.pos - startPos
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
	case '[':
		res = p.parseOpenBracket(block)
	case '!':
		res = p.parseBang(block)
	case ']':
		res = p.parseCloseBracket(block)
	case '<':
		res = p.parseAutolink(block) || p.parseHtmlTag(block)
	case '&':
		res = p.parseEntity(block)
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
