//
// Blackfriday Markdown Processor
// Available at http://github.com/russross/blackfriday
//
// Copyright © 2011 Russ Ross <russ@russross.com>.
// Distributed under the Simplified BSD License.
// See README.md for details.
//

//
// Functions to parse inline elements.
//

package blackfriday

import (
	"bytes"
	"regexp"
	"strconv"
)

var (
	urlRe    = `((https?|ftp):\/\/|\/)[-A-Za-z0-9+&@#\/%?=~_|!:,.;\(\)]+`
	anchorRe = regexp.MustCompile(`^(<a\shref="` + urlRe + `"(\stitle="[^"<>]+")?\s?>` + urlRe + `<\/a>)`)
)

// Functions to parse text within a block
// Each function returns the number of chars taken care of
// data is the complete block being rendered
// offset is the number of valid chars before the current cursor

func (p *parser) inline(data []byte) {
	// this is called recursively: enforce a maximum depth
	if p.nesting >= p.maxNesting {
		return
	}
	p.nesting++

	i, end := 0, 0
	for i < len(data) {
		// Copy inactive chars into the output, but first check for one quirk:
		// 'h', 'm' and 'f' all might trigger a check for autolink processing
		// and end this run of inactive characters. However, there's one nasty
		// case where breaking this run would be bad: in smartypants fraction
		// detection, we expect things like "1/2th" to be in a single run. So
		// we check here if an 'h' is followed by 't' (from 'http') and if it's
		// not, we short circuit the 'h' into the run of inactive characters.
		for end < len(data) {
			if p.inlineCallback[data[end]] != nil {
				if end+1 < len(data) && data[end] == 'h' && data[end+1] != 't' {
					end++
				} else {
					break
				}
			} else {
				end++
			}
		}

		p.r.NormalText(data[i:end])

		if end >= len(data) {
			break
		}
		i = end

		// call the trigger
		handler := p.inlineCallback[data[end]]
		if consumed := handler(p, data, i); consumed == 0 {
			// no action from the callback; buffer the byte for later
			end = i + 1
		} else {
			// skip past whatever the callback used
			i += consumed
			end = i
		}
	}

	p.nesting--
}

// single and double emphasis parsing
func emphasis(p *parser, data []byte, offset int) int {
	data = data[offset:]
	c := data[0]
	ret := 0

	if len(data) > 2 && data[1] != c {
		// whitespace cannot follow an opening emphasis;
		// strikethrough only takes two characters '~~'
		if c == '~' || isspace(data[1]) {
			return 0
		}
		if ret = helperEmphasis(p, data[1:], c); ret == 0 {
			return 0
		}

		return ret + 1
	}

	if len(data) > 3 && data[1] == c && data[2] != c {
		if isspace(data[2]) {
			return 0
		}
		if ret = helperDoubleEmphasis(p, data[2:], c); ret == 0 {
			return 0
		}

		return ret + 2
	}

	if len(data) > 4 && data[1] == c && data[2] == c && data[3] != c {
		if c == '~' || isspace(data[3]) {
			return 0
		}
		if ret = helperTripleEmphasis(p, data, 3, c); ret == 0 {
			return 0
		}

		return ret + 3
	}

	return 0
}

func codeSpan(p *parser, data []byte, offset int) int {
	data = data[offset:]

	nb := 0

	// count the number of backticks in the delimiter
	for nb < len(data) && data[nb] == '`' {
		nb++
	}

	// find the next delimiter
	i, end := 0, 0
	for end = nb; end < len(data) && i < nb; end++ {
		if data[end] == '`' {
			i++
		} else {
			i = 0
		}
	}

	// no matching delimiter?
	if i < nb && end >= len(data) {
		return 0
	}

	// trim outside whitespace
	fBegin := nb
	for fBegin < end && data[fBegin] == ' ' {
		fBegin++
	}

	fEnd := end - nb
	for fEnd > fBegin && data[fEnd-1] == ' ' {
		fEnd--
	}

	// render the code span
	if fBegin != fEnd {
		p.r.CodeSpan(data[fBegin:fEnd])
	}

	return end

}

// newline preceded by two spaces becomes <br>
func maybeLineBreak(p *parser, data []byte, offset int) int {
	origOffset := offset
	for offset < len(data) && data[offset] == ' ' {
		offset++
	}
	if offset < len(data) && data[offset] == '\n' {
		if offset-origOffset >= 2 {
			p.r.LineBreak()
			return offset - origOffset + 1
		}
		return offset - origOffset
	}
	return 0
}

// newline without preceding spaces works when extension HardLineBreak is enabled
func lineBreak(p *parser, data []byte, offset int) int {
	// should there be a hard line break here?
	if p.flags&HardLineBreak != 0 {
		p.r.LineBreak()
		return 1
	}
	return 0
}

type linkType int

const (
	linkNormal linkType = iota
	linkImg
	linkDeferredFootnote
	linkInlineFootnote
)

func maybeImage(p *parser, data []byte, offset int) int {
	if offset < len(data)-1 && data[offset+1] == '[' {
		return link(p, data, offset)
	}
	return 0
}

func maybeInlineFootnote(p *parser, data []byte, offset int) int {
	if offset < len(data)-1 && data[offset+1] == '[' {
		return link(p, data, offset)
	}
	return 0
}

// '[': parse a link or an image or a footnote
func link(p *parser, data []byte, offset int) int {
	// no links allowed inside regular links, footnote, and deferred footnotes
	if p.insideLink && (offset > 0 && data[offset-1] == '[' || len(data)-1 > offset && data[offset+1] == '^') {
		return 0
	}

	// [text] == regular link
	// ![alt] == image
	// ^[text] == inline footnote
	// [^refId] == deferred footnote
	var t linkType
	if offset >= 0 && data[offset] == '!' {
		offset += 1
		t = linkImg
	} else if p.flags&Footnotes != 0 {
		if offset >= 0 && data[offset] == '^' {
			offset += 1
			t = linkInlineFootnote
		} else if len(data)-1 > offset && data[offset+1] == '^' {
			t = linkDeferredFootnote
		}
	}

	data = data[offset:]

	var (
		i           = 1
		noteId      int
		title, link []byte
		textHasNl   = false
	)

	if t == linkDeferredFootnote {
		i++
	}

	// look for the matching closing bracket
	for level := 1; level > 0 && i < len(data); i++ {
		switch {
		case data[i] == '\n':
			textHasNl = true

		case data[i-1] == '\\':
			continue

		case data[i] == '[':
			level++

		case data[i] == ']':
			level--
			if level <= 0 {
				i-- // compensate for extra i++ in for loop
			}
		}
	}

	if i >= len(data) {
		return 0
	}

	txtE := i
	i++

	// skip any amount of whitespace or newline
	// (this is much more lax than original markdown syntax)
	for i < len(data) && isspace(data[i]) {
		i++
	}

	// inline style link
	switch {
	case i < len(data) && data[i] == '(':
		// skip initial whitespace
		i++

		for i < len(data) && isspace(data[i]) {
			i++
		}

		linkB := i

		// look for link end: ' " )
	findlinkend:
		for i < len(data) {
			switch {
			case data[i] == '\\':
				i += 2

			case data[i] == ')' || data[i] == '\'' || data[i] == '"':
				break findlinkend

			default:
				i++
			}
		}

		if i >= len(data) {
			return 0
		}
		linkE := i

		// look for title end if present
		titleB, titleE := 0, 0
		if data[i] == '\'' || data[i] == '"' {
			i++
			titleB = i

		findtitleend:
			for i < len(data) {
				switch {
				case data[i] == '\\':
					i += 2

				case data[i] == ')':
					break findtitleend

				default:
					i++
				}
			}

			if i >= len(data) {
				return 0
			}

			// skip whitespace after title
			titleE = i - 1
			for titleE > titleB && isspace(data[titleE]) {
				titleE--
			}

			// check for closing quote presence
			if data[titleE] != '\'' && data[titleE] != '"' {
				titleB, titleE = 0, 0
				linkE = i
			}
		}

		// remove whitespace at the end of the link
		for linkE > linkB && isspace(data[linkE-1]) {
			linkE--
		}

		// remove optional angle brackets around the link
		if data[linkB] == '<' {
			linkB++
		}
		if data[linkE-1] == '>' {
			linkE--
		}

		// build escaped link and title
		if linkE > linkB {
			link = data[linkB:linkE]
		}

		if titleE > titleB {
			title = data[titleB:titleE]
		}

		i++

	// reference style link
	case i < len(data)-1 && data[i] == '[' && data[i+1] != '^':
		var id []byte

		// look for the id
		i++
		linkB := i
		for i < len(data) && data[i] != ']' {
			i++
		}
		if i >= len(data) {
			return 0
		}
		linkE := i

		// find the reference
		if linkB == linkE {
			if textHasNl {
				var b bytes.Buffer

				for j := 1; j < txtE; j++ {
					switch {
					case data[j] != '\n':
						b.WriteByte(data[j])
					case data[j-1] != ' ':
						b.WriteByte(' ')
					}
				}

				id = b.Bytes()
			} else {
				id = data[1:txtE]
			}
		} else {
			id = data[linkB:linkE]
		}

		// find the reference with matching id (ids are case-insensitive)
		key := string(bytes.ToLower(id))
		lr, ok := p.refs[key]
		if !ok {
			return 0

		}

		// keep link and title from reference
		link = lr.link
		title = lr.title
		i++

	// shortcut reference style link or reference or inline footnote
	default:
		var id []byte

		// craft the id
		if textHasNl {
			var b bytes.Buffer

			for j := 1; j < txtE; j++ {
				switch {
				case data[j] != '\n':
					b.WriteByte(data[j])
				case data[j-1] != ' ':
					b.WriteByte(' ')
				}
			}

			id = b.Bytes()
		} else {
			if t == linkDeferredFootnote {
				id = data[2:txtE] // get rid of the ^
			} else {
				id = data[1:txtE]
			}
		}

		key := string(bytes.ToLower(id))
		if t == linkInlineFootnote {
			// create a new reference
			noteId = len(p.notes) + 1

			var fragment []byte
			if len(id) > 0 {
				if len(id) < 16 {
					fragment = make([]byte, len(id))
				} else {
					fragment = make([]byte, 16)
				}
				copy(fragment, slugify(id))
			} else {
				fragment = append([]byte("footnote-"), []byte(strconv.Itoa(noteId))...)
			}

			ref := &reference{
				noteId:   noteId,
				hasBlock: false,
				link:     fragment,
				title:    id,
			}

			p.notes = append(p.notes, ref)

			link = ref.link
			title = ref.title
		} else {
			// find the reference with matching id
			lr, ok := p.refs[key]
			if !ok {
				return 0
			}

			if t == linkDeferredFootnote {
				lr.noteId = len(p.notes) + 1
				p.notes = append(p.notes, lr)
			}

			// keep link and title from reference
			link = lr.link
			// if inline footnote, title == footnote contents
			title = lr.title
			noteId = lr.noteId
		}

		// rewind the whitespace
		i = txtE + 1
	}

	// build content: img alt is escaped, link content is parsed
	var content bytes.Buffer
	if txtE > 1 {
		if t == linkImg {
			content.Write(data[1:txtE])
		} else {
			// links cannot contain other links, so turn off link parsing temporarily
			insideLink := p.insideLink
			p.insideLink = true
			content.Write(p.r.captureWrites(func() {
				p.inline(data[1:txtE])
			}))
			p.insideLink = insideLink
		}
	}

	var uLink []byte
	if t == linkNormal || t == linkImg {
		if len(link) > 0 {
			var uLinkBuf bytes.Buffer
			unescapeText(&uLinkBuf, link)
			uLink = uLinkBuf.Bytes()
		}

		// links need something to click on and somewhere to go
		if len(uLink) == 0 || (t == linkNormal && content.Len() == 0) {
			return 0
		}
	}

	// call the relevant rendering function
	switch t {
	case linkNormal:
		p.r.Link(uLink, title, content.Bytes())

	case linkImg:
		p.r.Image(uLink, title, content.Bytes())
		i += 1

	case linkInlineFootnote:
		p.r.FootnoteRef(link, noteId)
		i += 1

	case linkDeferredFootnote:
		p.r.FootnoteRef(link, noteId)

	default:
		return 0
	}

	return i
}

// '<' when tags or autolinks are allowed
func leftAngle(p *parser, data []byte, offset int) int {
	data = data[offset:]
	altype := LinkTypeNotAutolink
	end := tagLength(data, &altype)

	if end > 2 {
		if altype != LinkTypeNotAutolink {
			var uLink bytes.Buffer
			unescapeText(&uLink, data[1:end+1-2])
			if uLink.Len() > 0 {
				p.r.AutoLink(uLink.Bytes(), altype)
			}
		} else {
			p.r.RawHtmlTag(data[:end])
		}
	}

	return end
}

// '\\' backslash escape
var escapeChars = []byte("\\`*_{}[]()#+-.!:|&<>~")

func escape(p *parser, data []byte, offset int) int {
	data = data[offset:]

	if len(data) > 1 {
		if bytes.IndexByte(escapeChars, data[1]) < 0 {
			return 0
		}

		p.r.NormalText(data[1:2])
	}

	return 2
}

func unescapeText(ob *bytes.Buffer, src []byte) {
	i := 0
	for i < len(src) {
		org := i
		for i < len(src) && src[i] != '\\' {
			i++
		}

		if i > org {
			ob.Write(src[org:i])
		}

		if i+1 >= len(src) {
			break
		}

		ob.WriteByte(src[i+1])
		i += 2
	}
}

// '&' escaped when it doesn't belong to an entity
// valid entities are assumed to be anything matching &#?[A-Za-z0-9]+;
func entity(p *parser, data []byte, offset int) int {
	data = data[offset:]

	end := 1

	if end < len(data) && data[end] == '#' {
		end++
	}

	for end < len(data) && isalnum(data[end]) {
		end++
	}

	if end < len(data) && data[end] == ';' {
		end++ // real entity
	} else {
		return 0 // lone '&'
	}

	p.r.Entity(data[:end])

	return end
}

func linkEndsWithEntity(data []byte, linkEnd int) bool {
	entityRanges := htmlEntity.FindAllIndex(data[:linkEnd], -1)
	return entityRanges != nil && entityRanges[len(entityRanges)-1][1] == linkEnd
}

func maybeAutoLink(p *parser, data []byte, offset int) int {
	// quick check to rule out most false hits
	if p.insideLink || len(data) < offset+6 { // 6 is the len() of the shortest prefix below
		return 0
	}
	prefixes := []string{
		"http://",
		"https://",
		"ftp://",
		"file://",
		"mailto:",
	}
	for _, prefix := range prefixes {
		endOfHead := offset + 8 // 8 is the len() of the longest prefix
		if endOfHead > len(data) {
			endOfHead = len(data)
		}
		head := bytes.ToLower(data[offset:endOfHead])
		if bytes.HasPrefix(head, []byte(prefix)) {
			return autoLink(p, data, offset)
		}
	}
	return 0
}

func autoLink(p *parser, data []byte, offset int) int {
	// Now a more expensive check to see if we're not inside an anchor element
	anchorStart := offset
	offsetFromAnchor := 0
	for anchorStart > 0 && data[anchorStart] != '<' {
		anchorStart--
		offsetFromAnchor++
	}

	anchorStr := anchorRe.Find(data[anchorStart:])
	if anchorStr != nil {
		//out.Write(anchorStr[offsetFromAnchor:]) // XXX: write in parser?
		p.r.Write(anchorStr[offsetFromAnchor:])
		return len(anchorStr) - offsetFromAnchor
	}

	// scan backward for a word boundary
	rewind := 0
	for offset-rewind > 0 && rewind <= 7 && isletter(data[offset-rewind-1]) {
		rewind++
	}
	if rewind > 6 { // longest supported protocol is "mailto" which has 6 letters
		return 0
	}

	origData := data
	data = data[offset-rewind:]

	if !isSafeLink(data) {
		return 0
	}

	linkEnd := 0
	for linkEnd < len(data) && !isEndOfLink(data[linkEnd]) {
		linkEnd++
	}

	// Skip punctuation at the end of the link
	if (data[linkEnd-1] == '.' || data[linkEnd-1] == ',') && data[linkEnd-2] != '\\' {
		linkEnd--
	}

	// But don't skip semicolon if it's a part of escaped entity:
	if data[linkEnd-1] == ';' && data[linkEnd-2] != '\\' && !linkEndsWithEntity(data, linkEnd) {
		linkEnd--
	}

	// See if the link finishes with a punctuation sign that can be closed.
	var copen byte
	switch data[linkEnd-1] {
	case '"':
		copen = '"'
	case '\'':
		copen = '\''
	case ')':
		copen = '('
	case ']':
		copen = '['
	case '}':
		copen = '{'
	default:
		copen = 0
	}

	if copen != 0 {
		bufEnd := offset - rewind + linkEnd - 2

		openDelim := 1

		/* Try to close the final punctuation sign in this same line;
		 * if we managed to close it outside of the URL, that means that it's
		 * not part of the URL. If it closes inside the URL, that means it
		 * is part of the URL.
		 *
		 * Examples:
		 *
		 *      foo http://www.pokemon.com/Pikachu_(Electric) bar
		 *              => http://www.pokemon.com/Pikachu_(Electric)
		 *
		 *      foo (http://www.pokemon.com/Pikachu_(Electric)) bar
		 *              => http://www.pokemon.com/Pikachu_(Electric)
		 *
		 *      foo http://www.pokemon.com/Pikachu_(Electric)) bar
		 *              => http://www.pokemon.com/Pikachu_(Electric))
		 *
		 *      (foo http://www.pokemon.com/Pikachu_(Electric)) bar
		 *              => foo http://www.pokemon.com/Pikachu_(Electric)
		 */

		for bufEnd >= 0 && origData[bufEnd] != '\n' && openDelim != 0 {
			if origData[bufEnd] == data[linkEnd-1] {
				openDelim++
			}

			if origData[bufEnd] == copen {
				openDelim--
			}

			bufEnd--
		}

		if openDelim == 0 {
			linkEnd--
		}
	}

	var uLink bytes.Buffer
	unescapeText(&uLink, data[:linkEnd])

	if uLink.Len() > 0 {
		p.r.AutoLink(uLink.Bytes(), LinkTypeNormal)
	}

	return linkEnd
}

func isEndOfLink(char byte) bool {
	return isspace(char) || char == '<'
}

var validUris = [][]byte{[]byte("http://"), []byte("https://"), []byte("ftp://"), []byte("mailto://"), []byte("/")}

func isSafeLink(link []byte) bool {
	for _, prefix := range validUris {
		// TODO: handle unicode here
		// case-insensitive prefix test
		if len(link) > len(prefix) && bytes.Equal(bytes.ToLower(link[:len(prefix)]), prefix) && isalnum(link[len(prefix)]) {
			return true
		}
	}

	return false
}

// return the length of the given tag, or 0 is it's not valid
func tagLength(data []byte, autolink *LinkType) int {
	var i, j int

	// a valid tag can't be shorter than 3 chars
	if len(data) < 3 {
		return 0
	}

	// begins with a '<' optionally followed by '/', followed by letter or number
	if data[0] != '<' {
		return 0
	}
	if data[1] == '/' {
		i = 2
	} else {
		i = 1
	}

	if !isalnum(data[i]) {
		return 0
	}

	// scheme test
	*autolink = LinkTypeNotAutolink

	// try to find the beginning of an URI
	for i < len(data) && (isalnum(data[i]) || data[i] == '.' || data[i] == '+' || data[i] == '-') {
		i++
	}

	if i > 1 && i < len(data) && data[i] == '@' {
		if j = isMailtoAutoLink(data[i:]); j != 0 {
			*autolink = LinkTypeEmail
			return i + j
		}
	}

	if i > 2 && i < len(data) && data[i] == ':' {
		*autolink = LinkTypeNormal
		i++
	}

	// complete autolink test: no whitespace or ' or "
	switch {
	case i >= len(data):
		*autolink = LinkTypeNotAutolink
	case *autolink != 0:
		j = i

		for i < len(data) {
			if data[i] == '\\' {
				i += 2
			} else if data[i] == '>' || data[i] == '\'' || data[i] == '"' || isspace(data[i]) {
				break
			} else {
				i++
			}

		}

		if i >= len(data) {
			return 0
		}
		if i > j && data[i] == '>' {
			return i + 1
		}

		// one of the forbidden chars has been found
		*autolink = LinkTypeNotAutolink
	}

	// look for something looking like a tag end
	for i < len(data) && data[i] != '>' {
		i++
	}
	if i >= len(data) {
		return 0
	}
	return i + 1
}

// look for the address part of a mail autolink and '>'
// this is less strict than the original markdown e-mail address matching
func isMailtoAutoLink(data []byte) int {
	nb := 0

	// address is assumed to be: [-@._a-zA-Z0-9]+ with exactly one '@'
	for i := 0; i < len(data); i++ {
		if isalnum(data[i]) {
			continue
		}

		switch data[i] {
		case '@':
			nb++

		case '-', '.', '_':
			break

		case '>':
			if nb == 1 {
				return i + 1
			} else {
				return 0
			}
		default:
			return 0
		}
	}

	return 0
}

// look for the next emph char, skipping other constructs
func helperFindEmphChar(data []byte, c byte) int {
	i := 1

	for i < len(data) {
		for i < len(data) && data[i] != c && data[i] != '`' && data[i] != '[' {
			i++
		}
		if i >= len(data) {
			return 0
		}
		if data[i] == c {
			return i
		}

		// do not count escaped chars
		if i != 0 && data[i-1] == '\\' {
			i++
			continue
		}

		if data[i] == '`' {
			// skip a code span
			tmpI := 0
			i++
			for i < len(data) && data[i] != '`' {
				if tmpI == 0 && data[i] == c {
					tmpI = i
				}
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			i++
		} else if data[i] == '[' {
			// skip a link
			tmpI := 0
			i++
			for i < len(data) && data[i] != ']' {
				if tmpI == 0 && data[i] == c {
					tmpI = i
				}
				i++
			}
			i++
			for i < len(data) && (data[i] == ' ' || data[i] == '\n') {
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			if data[i] != '[' && data[i] != '(' { // not a link
				if tmpI > 0 {
					return tmpI
				} else {
					continue
				}
			}
			cc := data[i]
			i++
			for i < len(data) && data[i] != cc {
				if tmpI == 0 && data[i] == c {
					return i
				}
				i++
			}
			if i >= len(data) {
				return tmpI
			}
			i++
		}
	}
	return 0
}

func helperEmphasis(p *parser, data []byte, c byte) int {
	i := 0

	// skip one symbol if coming from emph3
	if len(data) > 1 && data[0] == c && data[1] == c {
		i = 1
	}

	for i < len(data) {
		length := helperFindEmphChar(data[i:], c)
		if length == 0 {
			return 0
		}
		i += length
		if i >= len(data) {
			return 0
		}

		if i+1 < len(data) && data[i+1] == c {
			i++
			continue
		}

		if data[i] == c && !isspace(data[i-1]) {

			if p.flags&NoIntraEmphasis != 0 {
				if !(i+1 == len(data) || isspace(data[i+1]) || ispunct(data[i+1])) {
					continue
				}
			}

			work := p.r.captureWrites(func() {
				p.inline(data[:i])
			})
			p.r.Emphasis(work)
			return i + 1
		}
	}

	return 0
}

func helperDoubleEmphasis(p *parser, data []byte, c byte) int {
	i := 0

	for i < len(data) {
		length := helperFindEmphChar(data[i:], c)
		if length == 0 {
			return 0
		}
		i += length

		if i+1 < len(data) && data[i] == c && data[i+1] == c && i > 0 && !isspace(data[i-1]) {
			var work bytes.Buffer
			work.Write(p.r.captureWrites(func() {
				p.inline(data[:i])
			}))

			if work.Len() > 0 {
				// pick the right renderer
				if c == '~' {
					p.r.StrikeThrough(work.Bytes())
				} else {
					p.r.DoubleEmphasis(work.Bytes())
				}
			}
			return i + 2
		}
		i++
	}
	return 0
}

func helperTripleEmphasis(p *parser, data []byte, offset int, c byte) int {
	i := 0
	origData := data
	data = data[offset:]

	for i < len(data) {
		length := helperFindEmphChar(data[i:], c)
		if length == 0 {
			return 0
		}
		i += length

		// skip whitespace preceded symbols
		if data[i] != c || isspace(data[i-1]) {
			continue
		}

		switch {
		case i+2 < len(data) && data[i+1] == c && data[i+2] == c:
			// triple symbol found
			var work bytes.Buffer
			work.Write(p.r.captureWrites(func() {
				p.inline(data[:i])
			}))
			if work.Len() > 0 {
				p.r.TripleEmphasis(work.Bytes())
			}
			return i + 3
		case (i+1 < len(data) && data[i+1] == c):
			// double symbol found, hand over to emph1
			length = helperEmphasis(p, origData[offset-2:], c)
			if length == 0 {
				return 0
			} else {
				return length - 2
			}
		default:
			// single symbol found, hand over to emph2
			length = helperDoubleEmphasis(p, origData[offset-1:], c)
			if length == 0 {
				return 0
			} else {
				return length - 1
			}
		}
	}
	return 0
}
