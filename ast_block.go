package blackfriday

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
