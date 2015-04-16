//
// Blackfriday Markdown Processor
// Available at http://github.com/russross/blackfriday
//
// Copyright © 2011 Russ Ross <russ@russross.com>.
// Distributed under the Simplified BSD License.
// See README.md for details.
//

//
// This file contains legacy constants to maintain backward compatibility. The
// constants are all initialized to their new counterparts.
//
package blackfriday

const VERSION = Version

const (
	EXTENSION_NO_INTRA_EMPHASIS          = int(NoIntraEmphasis)
	EXTENSION_TABLES                     = int(Tables)
	EXTENSION_FENCED_CODE                = int(FencedCode)
	EXTENSION_AUTOLINK                   = int(Autolink)
	EXTENSION_STRIKETHROUGH              = int(Strikethrough)
	EXTENSION_LAX_HTML_BLOCKS            = int(LaxHTMLBlocks)
	EXTENSION_SPACE_HEADERS              = int(SpaceHeaders)
	EXTENSION_HARD_LINE_BREAK            = int(HardLineBreak)
	EXTENSION_TAB_SIZE_EIGHT             = int(TabSizeEight)
	EXTENSION_FOOTNOTES                  = int(Footnotes)
	EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK = int(NoEmptyLineBeforeBlock)
	EXTENSION_HEADER_IDS                 = int(HeaderIDs)
	EXTENSION_TITLEBLOCK                 = int(Titleblock)
	EXTENSION_AUTO_HEADER_IDS            = int(AutoHeaderIDs)
)

const (
	LINK_TYPE_NOT_AUTOLINK = int(LinkTypeNotAutolink)
	LINK_TYPE_NORMAL       = int(LinkTypeNormal)
	LINK_TYPE_EMAIL        = int(LinkTypeEmail)
)

const (
	LIST_TYPE_ORDERED           = int(ListTypeOrdered)
	LIST_ITEM_CONTAINS_BLOCK    = int(ListItemContainsBlock)
	LIST_ITEM_BEGINNING_OF_LIST = int(ListItemBeginningOfList)
	LIST_ITEM_END_OF_LIST       = int(ListItemEndOfList)
)

const (
	TABLE_ALIGNMENT_LEFT   = int(TableAlignmentLeft)
	TABLE_ALIGNMENT_RIGHT  = int(TableAlignmentRight)
	TABLE_ALIGNMENT_CENTER = int(TableAlignmentCenter)
)

const (
	TAB_SIZE_DEFAULT = TabSizeDefault
	TAB_SIZE_EIGHT   = TabSizeDouble
)

const (
	HTML_SKIP_HTML                 = int(SkipHTML)
	HTML_SKIP_STYLE                = int(SkipStyle)
	HTML_SKIP_IMAGES               = int(SkipImages)
	HTML_SKIP_LINKS                = int(SkipLinks)
	HTML_SAFELINK                  = int(Safelink)
	HTML_NOFOLLOW_LINKS            = int(NofollowLinks)
	HTML_NOREFERRER_LINKS          = int(NoreferrerLinks)
	HTML_HREF_TARGET_BLANK         = int(HrefTargetBlank)
	HTML_TOC                       = int(Toc)
	HTML_OMIT_CONTENTS             = int(OmitContents)
	HTML_COMPLETE_PAGE             = int(CompletePage)
	HTML_USE_XHTML                 = int(UseXHTML)
	HTML_USE_SMARTYPANTS           = int(UseSmartypants)
	HTML_SMARTYPANTS_FRACTIONS     = int(SmartypantsFractions)
	HTML_SMARTYPANTS_LATEX_DASHES  = int(SmartypantsLatexDashes)
	HTML_SMARTYPANTS_ANGLED_QUOTES = int(SmartypantsAngledQuotes)
	HTML_FOOTNOTE_RETURN_LINKS     = int(FootnoteReturnLinks)
)

func MarkdownBasic(input []byte) []byte {
	// set up the HTML renderer
	htmlFlags := HTML_USE_XHTML
	renderer := HtmlRenderer(htmlFlags, "", "")

	// set up the parser
	extensions := 0

	return Markdown(input, renderer, extensions)
}

func MarkdownCommon(input []byte) []byte {
	// set up the HTML renderer
	renderer := HtmlRenderer(commonHtmlFlags, "", "")
	return Markdown(input, renderer, commonExtensions)
}

func Markdown(input []byte, renderer Renderer, extensions int) []byte {
	return nil
}
