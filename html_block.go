// Lute - A structured markdown engine.
// Copyright (c) 2019-present, b3log.org
//
// Lute is licensed under the Mulan PSL v1.
// You can use this software according to the terms and conditions of the Mulan PSL v1.
// You may obtain a copy of Mulan PSL v1 at:
//     http://license.coscl.org.cn/MulanPSL
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v1 for more details.

package lute

import (
	"bytes"
)

func (html *Node) htmlBlockContinue(context *Context) int {
	if context.blank && (html.htmlBlockType == 6 || html.htmlBlockType == 7) {
		return 1
	}
	return 0
}

func (html *Node) htmlBlockFinalize(context *Context) {
	html.tokens = bytes.TrimRight(html.tokens.replaceNewlineSpace(), " \t\n")
}

var (
	htmlBlockTags1      = []items{items("<script"), items("<pre"), items("<style")}
	htmlBlockCloseTags1 = []items{items("</script>"), items("</pre>"), items("</style>")}
	htmlBlockTags6      = []items{
		items("<address"), items("<article"), items("<aside"), items("<base"), items("<basefont"), items("<blockquote"), items("<body"), items("<caption"), items("<center"), items("<col"), items("<colgroup"), items("<dd"), items("<details"), items("<dialog"), items("<dir"), items("<div"), items("<dl"), items("<dt"), items("<fieldset"), items("<figcaption"), items("<figure"), items("<footer"), items("<form"), items("<frame"), items("<frameset"), items("<h1"), items("<h2"), items("<h3"), items("<h4"), items("<h5"), items("<h6"), items("<head"), items("<header"), items("<hr"), items("<html"), items("<iframe"), items("<legend"), items("<li"), items("<link"), items("<main"), items("<menu"), items("<menuitem"), items("<nav"), items("<noframes"), items("<ol"), items("<optgroup"), items("<option"), items("<p"), items("<param"), items("<section"), items("<source"), items("<summary"), items("<table"), items("<tbody"), items("<td"), items("<tfoot"), items("<th"), items("<thead"), items("<title"), items("<tr"), items("<track"), items("<ul"),
		items("</address"), items("</article"), items("</aside"), items("</base"), items("</basefont"), items("</blockquote"), items("</body"), items("</caption"), items("</center"), items("</col"), items("</colgroup"), items("</dd"), items("</details"), items("</dialog"), items("</dir"), items("</div"), items("</dl"), items("</dt"), items("</fieldset"), items("</figcaption"), items("</figure"), items("</footer"), items("</form"), items("</frame"), items("</frameset"), items("</h1"), items("</h2"), items("</h3"), items("</h4"), items("</h5"), items("</h6"), items("</head"), items("</header"), items("</hr"), items("</html"), items("</iframe"), items("</legend"), items("</li"), items("</link"), items("</main"), items("</menu"), items("</menuitem"), items("</nav"), items("</noframes"), items("</ol"), items("</optgroup"), items("</option"), items("</p"), items("</param"), items("</section"), items("</source"), items("</summary"), items("</table"), items("</tbody"), items("</td"), items("</tfoot"), items("</th"), items("</thead"), items("</title"), items("</tr"), items("</track"), items("</ul"),
	}
	htmlBlockEqual       = items{itemEqual}
	htmlBlockSinglequote = items{itemSinglequote}
	htmlBlockDoublequote = items{itemDoublequote}
	htmlBlockGreater     = items{itemGreater}
)

func (t *Tree) isHTMLBlockClose(tokens items, htmlType int) bool {
	length := len(tokens)
	switch htmlType {
	case 1:
		if pos := tokens.acceptTokenss(htmlBlockCloseTags1); 0 <= pos {
			return true
		}
		return false
	case 2:
		for i := 0; i < length-3; i++ {
			if itemHyphen == tokens[i] && itemHyphen == tokens[i+1] && itemGreater == tokens[i+2] {
				return true
			}
		}
	case 3:
		for i := 0; i < length-2; i++ {
			if itemQuestion == tokens[i] && itemGreater == tokens[i+1] {
				return true
			}
		}
	case 4:
		return bytes.Contains(tokens, htmlBlockGreater)
	case 5:
		for i := 0; i < length-2; i++ {
			if itemCloseBracket == tokens[i] && itemCloseBracket == tokens[i+1] {
				return true
			}
		}
	}

	return false
}

func (t *Tree) parseHTML(tokens items) (ret *Node) {
	tokens = bytes.TrimLeft(tokens, " \t\n")
	length := len(tokens)
	if 3 > length { // at least <? and a newline
		return nil
	}

	if itemLess != tokens[0] {
		return nil
	}

	ret = &Node{typ: NodeHTMLBlock, tokens: make(items, 0, 256), htmlBlockType: 1}

	if pos := tokens.acceptTokenss(htmlBlockTags1); 0 <= pos {
		if isWhitespace(tokens[pos]) || itemGreater == tokens[pos] {
			return
		}
	}

	if pos := tokens.acceptTokenss(htmlBlockTags6); 0 <= pos {
		if isWhitespace(tokens[pos]) || itemGreater == tokens[pos] {
			ret.htmlBlockType = 6
			return
		}
		if itemSlash == tokens[pos] && itemGreater == tokens[pos+1] {
			ret.htmlBlockType = 6
			return
		}
	}

	tag := bytes.TrimSpace(tokens)
	isOpenTag := t.isOpenTag(tag)
	if isOpenTag && t.context.tip.typ != NodeParagraph {
		ret.htmlBlockType = 7
		return
	}
	isCloseTag := t.isCloseTag(tag)
	if isCloseTag && t.context.tip.typ != NodeParagraph {
		ret.htmlBlockType = 7
		return
	}

	if 0 == bytes.Index(tokens, toItems("<!--")) {
		ret.htmlBlockType = 2
		return
	}

	if 0 == bytes.Index(tokens, toItems("<?")) {
		ret.htmlBlockType = 3
		return
	}

	if 2 < len(tokens) && 0 == bytes.Index(tokens, toItems("<!")) {
		following := tokens[2:]
		if 'A' <= following[0] && 'Z' >= following[0] {
			ret.htmlBlockType = 4
			return
		}
		if 0 == bytes.Index(following, toItems("[CDATA[")) {
			ret.htmlBlockType = 5
			return
		}
	}

	return nil
}

// tokenize 在 init 函数中调用，可以认为是静态分配，所以使用拷贝字符不会有性能问题。
// 另外，这里也必须要拷贝，因为调用点的 str 是局部变量，地址上的值会被覆盖。
func tokenize(str string) (ret items) {
	for _, r := range str {
		ret = append(ret, byte(r))
	}

	return
}

func (t *Tree) isOpenTag(tokens items) (isOpenTag bool) {
	length := len(tokens)
	if 3 > length {
		return
	}

	if itemLess != tokens[0] {
		return
	}
	if itemGreater != tokens[length-1] {
		return
	}
	if itemSlash == tokens[length-2] {
		tokens = tokens[1 : length-2]
	} else {
		tokens = tokens[1 : length-1]
	}

	length = len(tokens)
	if 0 == length {
		return
	}

	if isWhitespace(tokens[0]) { // < 后面不能跟空白
		return
	}

	nameAndAttrs := tokens.splitWhitespace()
	name := nameAndAttrs[0]
	if !isASCIILetter(name[0]) {
		return
	}
	if 1 < len(name) {
		name = name[1:]
		for _, n := range name {
			if !isASCIILetterNumHyphen(n) {
				return
			}
		}
	}

	attrs := nameAndAttrs[1:]
	for _, attr := range attrs {
		if 1 >= len(attr) {
			continue
		}

		nameAndValue := bytes.Split(attr, htmlBlockEqual)
		name := nameAndValue[0]
		if 1 > len(name) { // 等号前面空格的情况
			continue
		}
		if !isASCIILetter(name[0]) && itemUnderscore != name[0] && itemColon != name[0] {
			return
		}

		if 1 < len(name) {
			name = name[1:]
			for _, n := range name {
				if !isASCIILetter(n) && !isDigit(n) && itemUnderscore != n && itemDot != n && itemColon != n && itemHyphen != n {
					return
				}
			}
		}

		if 1 < len(nameAndValue) {
			value := nameAndValue[1]
			if bytes.HasPrefix(value, htmlBlockSinglequote) && bytes.HasSuffix(value, htmlBlockSinglequote) {
				value = value[1:]
				value = value[:len(value)-1]
				return !bytes.Contains(value, htmlBlockSinglequote)
			}
			if bytes.HasPrefix(value, htmlBlockDoublequote) && bytes.HasSuffix(value, htmlBlockDoublequote) {
				value = value[1:]
				value = value[:len(value)-1]
				return !bytes.Contains(value, htmlBlockDoublequote)
			}
			return !bytes.ContainsAny(value, " \t\n") && !bytes.ContainsAny(value, "\"'=<>`")
		}
	}
	return true
}

func (t *Tree) isCloseTag(tokens items) bool {
	tokens = bytes.TrimSpace(tokens)
	length := len(tokens)
	if 4 > length {
		return false
	}

	if itemLess != tokens[0] || itemSlash != tokens[1] {
		return false
	}
	if itemGreater != tokens[length-1] {
		return false
	}

	tokens = tokens[2 : length-1]
	length = len(tokens)
	if 0 == length {
		return false
	}

	name := tokens[0:]
	if !isASCIILetter(name[0]) {
		return false
	}
	if 1 < len(name) {
		name = name[1:]
		for _, n := range name {
			if !isASCIILetterNumHyphen(n) {
				return false
			}
		}
	}

	return true
}
