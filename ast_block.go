package blackfriday

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
)

type BlockHandler interface {
	CanContain(t NodeType) bool
}

type HeaderBlockHandler struct {
}

func (h *HeaderBlockHandler) CanContain(t NodeType) bool {
	return false
}

type DocumentBlockHandler struct {
}

func (h *DocumentBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type HorizontalRuleBlockHandler struct {
}

func (h *HorizontalRuleBlockHandler) CanContain(t NodeType) bool {
	return false
}

type BlockQuoteBlockHandler struct {
}

func (h *BlockQuoteBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type ParagraphBlockHandler struct {
}

func (h *ParagraphBlockHandler) CanContain(t NodeType) bool {
	return false
}

type HtmlBlockHandler struct {
}

func (h *HtmlBlockHandler) CanContain(t NodeType) bool {
	return false
}

type ListBlockHandler struct {
}

func (h *ListBlockHandler) CanContain(t NodeType) bool {
	return t == Item
}

type ItemBlockHandler struct {
}

func (h *ItemBlockHandler) CanContain(t NodeType) bool {
	return t != Item
}

type CodeBlockHandler struct {
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
