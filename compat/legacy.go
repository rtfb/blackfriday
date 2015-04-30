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
package compat

import (
	"gopkg.in/rtfb/blackfriday.v2"
)

const VERSION = blackfriday.Version

const (
	EXTENSION_NO_INTRA_EMPHASIS          = int(blackfriday.NoIntraEmphasis)
	EXTENSION_TABLES                     = int(blackfriday.Tables)
	EXTENSION_FENCED_CODE                = int(blackfriday.FencedCode)
	EXTENSION_AUTOLINK                   = int(blackfriday.Autolink)
	EXTENSION_STRIKETHROUGH              = int(blackfriday.Strikethrough)
	EXTENSION_LAX_HTML_BLOCKS            = int(blackfriday.LaxHTMLBlocks)
	EXTENSION_SPACE_HEADERS              = int(blackfriday.SpaceHeaders)
	EXTENSION_HARD_LINE_BREAK            = int(blackfriday.HardLineBreak)
	EXTENSION_TAB_SIZE_EIGHT             = int(blackfriday.TabSizeEight)
	EXTENSION_FOOTNOTES                  = int(blackfriday.Footnotes)
	EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK = int(blackfriday.NoEmptyLineBeforeBlock)
	EXTENSION_HEADER_IDS                 = int(blackfriday.HeaderIDs)
	EXTENSION_TITLEBLOCK                 = int(blackfriday.Titleblock)
	EXTENSION_AUTO_HEADER_IDS            = int(blackfriday.AutoHeaderIDs)
)

const (
	LINK_TYPE_NOT_AUTOLINK = int(blackfriday.LinkTypeNotAutolink)
	LINK_TYPE_NORMAL       = int(blackfriday.LinkTypeNormal)
	LINK_TYPE_EMAIL        = int(blackfriday.LinkTypeEmail)
)

const (
	LIST_TYPE_ORDERED           = int(blackfriday.ListTypeOrdered)
	LIST_ITEM_CONTAINS_BLOCK    = int(blackfriday.ListItemContainsBlock)
	LIST_ITEM_BEGINNING_OF_LIST = int(blackfriday.ListItemBeginningOfList)
	LIST_ITEM_END_OF_LIST       = int(blackfriday.ListItemEndOfList)
)

const (
	TABLE_ALIGNMENT_LEFT   = int(blackfriday.TableAlignmentLeft)
	TABLE_ALIGNMENT_RIGHT  = int(blackfriday.TableAlignmentRight)
	TABLE_ALIGNMENT_CENTER = int(blackfriday.TableAlignmentCenter)
)

const (
	TAB_SIZE_DEFAULT = blackfriday.TabSizeDefault
	TAB_SIZE_EIGHT   = blackfriday.TabSizeDouble
)

const (
	HTML_SKIP_HTML                 = int(blackfriday.SkipHTML)
	HTML_SKIP_STYLE                = int(blackfriday.SkipStyle)
	HTML_SKIP_IMAGES               = int(blackfriday.SkipImages)
	HTML_SKIP_LINKS                = int(blackfriday.SkipLinks)
	HTML_SAFELINK                  = int(blackfriday.Safelink)
	HTML_NOFOLLOW_LINKS            = int(blackfriday.NofollowLinks)
	HTML_NOREFERRER_LINKS          = int(blackfriday.NoreferrerLinks)
	HTML_HREF_TARGET_BLANK         = int(blackfriday.HrefTargetBlank)
	HTML_TOC                       = int(blackfriday.Toc)
	HTML_OMIT_CONTENTS             = int(blackfriday.OmitContents)
	HTML_COMPLETE_PAGE             = int(blackfriday.CompletePage)
	HTML_USE_XHTML                 = int(blackfriday.UseXHTML)
	HTML_USE_SMARTYPANTS           = int(blackfriday.UseSmartypants)
	HTML_SMARTYPANTS_FRACTIONS     = int(blackfriday.SmartypantsFractions)
	HTML_SMARTYPANTS_LATEX_DASHES  = int(blackfriday.SmartypantsLatexDashes)
	HTML_SMARTYPANTS_ANGLED_QUOTES = int(blackfriday.SmartypantsAngledQuotes)
	HTML_FOOTNOTE_RETURN_LINKS     = int(blackfriday.FootnoteReturnLinks)
)

func MarkdownBasic(input []byte) []byte {
	return blackfriday.MarkdownBasic(input)
}

func MarkdownCommon(input []byte) []byte {
	return blackfriday.MarkdownCommon(input)
}

func Markdown(input []byte, renderer blackfriday.Renderer, extensions int) []byte {
	return blackfriday.Markdown(input, renderer, blackfriday.Extensions(extensions))
}

func HtmlRenderer(flags int, title string, css string) blackfriday.Renderer {
	return blackfriday.HtmlRendererWithParameters(blackfriday.HtmlFlags(flags),
		title, css, blackfriday.HtmlRendererParameters{})
}

func HtmlRendererWithParameters(flags int, title string,
	css string,
	renderParameters blackfriday.HtmlRendererParameters) blackfriday.Renderer {
	return blackfriday.HtmlRendererWithParameters(blackfriday.HtmlFlags(flags),
		title, css, renderParameters)
}
