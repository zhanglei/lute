// Lute - A structured markdown engine.
// Copyright (C) 2019-present, b3log.org
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lute

func (t *Tree) parseInlines() {
	t.context.Delimiters = nil
	t.context.Brackets = nil
	t.parseBlockInlines(t.Root.Children())
}

func (t *Tree) parseBlockInlines(blocks []Node) {
	for _, block := range blocks {
		cType := block.Type()
		switch cType {
		case NodeCode, NodeThematicBreak, NodeHTML:
			continue
		}

		cs := block.Children()
		if 0 < len(cs) {
			t.parseBlockInlines(cs)

			continue
		}

		tokens := block.Tokens()
		if nil == tokens {
			continue
		}

		t.context.Pos = 0
		for {
			token := tokens[t.context.Pos]
			var n Node
			switch token.typ {
			case itemBackslash:
				n = t.parseBackslash(tokens)
			case itemBacktick:
				n = t.parseInlineCode(tokens)
			case itemAsterisk, itemUnderscore:
				t.handleDelim(block, tokens)
			case itemNewline:
				n = t.parseNewline(block, tokens)
			case itemLess:
				n = t.parseInlineHTML(tokens)
			case itemOpenBracket:
				n = t.parseOpenBracket(tokens)
			case itemCloseBracket:
				t.parseCloseBracket(block, tokens)
			default:
				n = t.parseText(tokens)
			}

			if nil != n {
				block.AppendChild(block, n)
			}

			len := len(tokens)
			if 1 > len || t.context.Pos >= len || tokens[t.context.Pos].isEOF() {
				break
			}
		}

		t.processEmphasis(nil)
	}
}

func (t *Tree) parseCloseBracket(block Node, tokens items) {
	var startPos int
	var isImage bool
	matched := false
	var dest, title, reflabel string
	var opener *delimiter

	startPos = t.context.Pos

	// get last [ or ![
	opener = t.context.Brackets

	if nil == opener {
		// no matched opener, just return a literal
		block.AppendChild(block, &Text{&BaseNode{typ: NodeText}, "]"})

		return
	}

	if !opener.active {
		// no matched opener, just return a literal
		block.AppendChild(block, &Text{&BaseNode{typ: NodeText}, "]"})

		// take opener off brackets stack
		t.removeBracket()
		return
	}

	// If we got here, open is a potential opener
	isImage = opener.image

	// Check to see if we have a link/image

	savepos := t.context.Pos

	// Inline link?
	if itemOpenParen == tokens[t.context.Pos].typ {
		t.context.Pos++

		tmp := tokens[t.context.Pos:]
		isLink, tmp := tmp.spnl()

		if isLink {
			_, tmp, dest := t.parseLinkDest(tmp)
			if "" != dest {
				isLink, tmp = tmp.spnl()
				if isLink {
					if tmp[0].isWhitespace() { // make sure there's a space before the title
						_, tmp, title := t.parseLinkTitle(tmp)
						if "" != title {
							isLink, tmp = tmp.spnl()
							if isLink && itemCloseParen == tmp[0].typ {
								t.context.Pos++
								matched = true
							}
						}
					}
				}
			}
		}
	}

	if !matched {
		t.context.Pos = savepos
	}

	if !matched {
		// Next, see if there's a link label
		//var beforelabel = t.context.Pos
		_, _, label := t.parseLinkLabel(tokens[:t.context.Pos+1])
		var n = len(label)
		if n > 2 {
			//reflabel = this.subject.slice(beforelabel, beforelabel+n)
			reflabel = label
		} else if !opener.bracketAfter {
			// Empty or missing second label means to use the first label as the reference.
			// The reference must not contain a bracket. If we know there's a bracket, we don't even bother checking it.
			//reflabel = this.subject.slice(opener.index, startpos)
			reflabel = tokens[opener.index:startPos].rawText()
		}
		if n == 0 {
			// If shortcut reference link, rewind before spaces we skipped.
			t.context.Pos = savepos
		}

		if "" != reflabel {
			// lookup rawlabel in refmap
			var link = t.context.LinkRefDef[reflabel]
			if nil != link {
				dest = link.URL
				title = link.Title
				matched = true
			}
		}
	}

	if matched {
		var node Node
		if isImage {
			node = &Image{&BaseNode{typ: NodeImage}, dest, title}
		} else {
			node = &Link{&BaseNode{typ: NodeLink}, dest, title}
		}

		var tmp, next Node
		tmp = opener.node.Next()
		for nil != tmp {
			next = tmp.Next()
			tmp.Unlink()
			node.AppendChild(node, tmp)
			tmp = next
		}

		block.AppendChild(block, node)
		t.processEmphasis(opener.previousDelimiter)
		t.removeBracket()
		opener.node.Unlink()

		// We remove this bracket and processEmphasis will remove later delimiters.
		// Now, for a link, we also deactivate earlier link openers.
		// (no links in links)
		if !isImage {
			opener = t.context.Brackets
			for nil != opener {
				if !opener.image {
					opener.active = false // deactivate this opener
				}
				opener = opener.previous
			}
		}

		t.context.Pos++

		return
	} else { // no match
		t.removeBracket() // remove this opener from stack
		t.context.Pos = startPos
		block.AppendChild(block, &Text{&BaseNode{typ: NodeText}, "]"})

		return
	}
}

func (t *Tree) parseOpenBracket(tokens items) (ret Node) {
	startPos := t.context.Pos
	t.context.Pos++

	ret = &Text{&BaseNode{typ: NodeText}, "["}

	// Add entry to stack for this opener
	t.addBracket(ret, startPos, false)

	return
}

func (t *Tree) addBracket(node Node, index int, image bool) {
	if nil != t.context.Brackets {
		t.context.Brackets.bracketAfter = true
	}

	t.context.Brackets = &delimiter{
		node:              node,
		previous:          t.context.Brackets,
		previousDelimiter: t.context.Delimiters,
		index:             index,
		image:             image,
		active:            true,
	}
}

func (t *Tree) removeBracket() {
	t.context.Brackets = t.context.Brackets.previous
}

func (t *Tree) parseBackslash(tokens items) (ret Node) {
	if len(tokens)-1 > t.context.Pos {
		t.context.Pos++
	}
	token := tokens[t.context.Pos]
	if token.isNewline() {
		ret = &HardBreak{&BaseNode{typ: NodeHardBreak}}
		t.context.Pos++
	} else if token.isASCIIPunct() {
		ret = &Text{&BaseNode{typ: NodeText}, token.val}
		t.context.Pos++
	} else {
		ret = &Text{&BaseNode{typ: NodeText}, "\\"}
	}

	return
}

func (t *Tree) processEmphasis(stackBottom *delimiter) {
	var opener, closer, old_closer *delimiter
	var opener_inl, closer_inl Node
	var tempstack *delimiter
	var use_delims int
	var opener_found bool
	var openers_bottom = map[itemType]*delimiter{}
	var odd_match = false

	openers_bottom[itemUnderscore] = stackBottom
	openers_bottom[itemAsterisk] = stackBottom

	// find first closer above stack_bottom:
	closer = t.context.Delimiters
	for closer != nil && closer.previous != stackBottom {
		closer = closer.previous
	}

	// move forward, looking for closers, and handling each
	for nil != closer {
		var closercc = closer.typ
		if !closer.canClose {
			closer = closer.next
			continue
		}

		// found emphasis closer. now look back for first matching opener:
		opener = closer.previous
		opener_found = false
		for nil != opener && opener != stackBottom && opener != openers_bottom[closercc] {
			odd_match = (closer.canOpen || opener.canClose) && closer.originalNum%3 != 0 && (opener.originalNum+closer.originalNum)%3 == 0
			if opener.typ == closer.typ && opener.canOpen && !odd_match {
				opener_found = true
				break
			}
			opener = opener.previous
		}
		old_closer = closer

		if !opener_found {
			closer = closer.next
		} else {
			// calculate actual number of delimiters used from closer
			if closer.num >= 2 && opener.num >= 2 {
				use_delims = 2
			} else {
				use_delims = 1
			}

			opener_inl = opener.node
			closer_inl = closer.node

			// remove used delimiters from stack elts and inlines
			opener.num -= use_delims
			closer.num -= use_delims

			text := opener_inl.RawText()[0 : len(opener_inl.RawText())-use_delims]
			opener_inl.SetRawText(text)

			text = closer_inl.RawText()[0 : len(closer_inl.RawText())-use_delims]
			closer_inl.SetRawText(text)

			// build contents for new emph element
			var emph Node
			if 1 == use_delims {
				emph = &Emphasis{&BaseNode{typ: NodeEmphasis}}
			} else {
				emph = &Strong{&BaseNode{typ: NodeStrong}}
			}

			tmp := opener_inl.Next()
			for nil != tmp && tmp != closer_inl {
				next := tmp.Next()
				tmp.Unlink()
				emph.AppendChild(emph, tmp)
				tmp = next
			}

			opener_inl.InsertAfter(opener_inl, emph)

			// remove elts between opener and closer in delimiters stack
			if opener.next != closer {
				opener.next = closer
				closer.previous = opener
			}

			// if opener has 0 delims, remove it and the inline
			if opener.num == 0 {
				opener_inl.Unlink()
				t.removeDelimiter(opener)
			}

			if closer.num == 0 {
				closer_inl.Unlink()
				tempstack = closer.next
				t.removeDelimiter(closer)
				closer = tempstack
			}
		}

		if !opener_found && !odd_match {
			// Set lower bound for future searches for openers:
			// We don't do this with odd_match because a **
			// that doesn't match an earlier * might turn into
			// an opener, and the * might be matched by something
			// else.
			openers_bottom[closercc] = old_closer.previous
			if !old_closer.canOpen {
				// We can remove a closer that can't be an opener,
				// once we've seen there's no matching opener:
				t.removeDelimiter(old_closer)
			}
		}
	}

	// remove all delimiters
	for t.context.Delimiters != nil && t.context.Delimiters != stackBottom {
		t.removeDelimiter(t.context.Delimiters)
	}
}

func (t *Tree) handleDelim(block Node, tokens items) {
	startPos := t.context.Pos
	delim := t.scanDelims(tokens)

	subTokens, text := t.extractTokens(tokens, startPos, t.context.Pos)
	baseNode := &BaseNode{typ: NodeText, rawText: text, tokens: subTokens}
	node := &Text{baseNode, text}
	block.AppendChild(block, node)

	// Add entry to stack for this opener
	t.context.Delimiters = &delimiter{
		typ:         delim.typ,
		num:         delim.num,
		originalNum: delim.num,
		node:        node,
		previous:    t.context.Delimiters,
		next:        nil,
		canOpen:     delim.canOpen,
		canClose:    delim.canClose,
	}
	if t.context.Delimiters.previous != nil {
		t.context.Delimiters.previous.next = t.context.Delimiters
	}
}

func (t *Tree) extractTokens(tokens items, startPos, endPos int) (subTokens items, text string) {
	for i := startPos; i < endPos; i++ {
		text += tokens[i].val
		subTokens = append(subTokens, tokens[i])
	}

	return
}

func (t *Tree) scanDelims(tokens items) *delimiter {
	startPos := t.context.Pos
	token := tokens[startPos]
	delimitersCount := 0
	for i := t.context.Pos; i < len(tokens) && token.val == tokens[i].val; i++ {
		delimitersCount++
		t.context.Pos++
	}

	tokenBefore, tokenAfter := tNewLine, tNewLine
	index := startPos - 1
	if 0 < index {
		tokenBefore = tokens[index]
	}
	index = t.context.Pos
	if len(tokens) > index {
		tokenAfter = tokens[index]
	}

	var beforeIsPunct, beforeIsWhitespace, afterIsPunct, afterIsWhitespace, canOpen, canClose bool
	if nil != tokenBefore {
		beforeIsWhitespace = tokenBefore.isWhitespace()
		beforeIsPunct = tokenBefore.isPunct()
	}
	if nil != tokenAfter {
		afterIsWhitespace = tokenAfter.isWhitespace()
		afterIsPunct = tokenAfter.isPunct()
	}

	isLeftFlanking := !afterIsWhitespace && (!afterIsPunct || beforeIsWhitespace || beforeIsPunct)
	isRightFlanking := !beforeIsWhitespace && (!beforeIsPunct || afterIsWhitespace || afterIsPunct)
	if itemUnderscore == token.typ {
		canOpen = isLeftFlanking && (!isRightFlanking || beforeIsPunct)
		canClose = isRightFlanking && (!isLeftFlanking || afterIsPunct)
	} else {
		canOpen = isLeftFlanking
		canClose = isRightFlanking
	}

	return &delimiter{typ: token.typ, num: delimitersCount, active: true, canOpen: canOpen, canClose: canClose}
}

func (t *Tree) parseInlineCode(tokens items) (ret Node) {
	startPos := t.context.Pos
	marker := tokens[startPos]
	n := tokens[startPos:].accept(marker.typ)
	endPos := t.matchEnd(tokens[startPos+n:], marker, n)
	if 1 > endPos {
		marker.typ = itemStr
		t.context.Pos++

		baseNode := &BaseNode{typ: NodeText, rawText: marker.val}
		ret = &Text{baseNode, marker.val}

		return
	}
	endPos = startPos + endPos + n

	var textTokens = items{}
	for i := startPos + n; i < len(tokens) && i < endPos; i++ {
		token := tokens[i]
		if token.isNewline() {
			textTokens = append(textTokens, tSpace)
		} else {
			textTokens = append(textTokens, token)
		}
	}

	if 2 < len(textTokens) && textTokens[0].isSpace() && textTokens[len(textTokens)-1].isSpace() && !textTokens.isBlankLine() {
		textTokens = textTokens[1:]
		textTokens = textTokens[:len(textTokens)-1]
	}

	baseNode := &BaseNode{typ: NodeInlineCode, tokens: textTokens}
	ret = &InlineCode{baseNode, textTokens.rawText()}
	t.context.Pos = endPos + n

	return
}

func (t *Tree) parseText(tokens items) (ret Node) {
	token := tokens[t.context.Pos]
	t.context.Pos++

	baseNode := &BaseNode{typ: NodeText, rawText: token.val}
	ret = &Text{baseNode, escapeHTML(token.val)}

	return
}

func (t *Tree) parseInlineHTML(tokens items) (ret Node) {
	tag := tokens[t.context.Pos:]
	tag = tag[:tag.index(itemGreater)+1]
	if 1 > len(tag) {
		token := tokens[t.context.Pos]
		baseNode := &BaseNode{typ: NodeText, rawText: token.val}
		ret = &Text{baseNode, escapeHTML(token.val)}
		t.context.Pos++

		return
	}

	codeTokens := items{}
	for _, token := range tag {
		codeTokens = append(codeTokens, token)
	}
	t.context.Pos += len(codeTokens)

	baseNode := &BaseNode{typ: NodeInlineHTML, tokens: codeTokens}
	ret = &InlineHTML{baseNode, codeTokens.rawText()}

	return
}

func (t *Tree) parseNewline(block Node, tokens items) (ret Node) {
	t.context.Pos++
	// check previous node for trailing spaces
	var lastc = block.LastChild()
	len := len(lastc.RawText())
	rawText := lastc.RawText()
	if nil != lastc && lastc.Type() == NodeText && rawText[len-1] == ' ' {
		var hardbreak = rawText[len-2] == ' '
		rawText = rawText[:len-1]
		lastc.SetRawText(rawText)
		if hardbreak {
			ret = &HardBreak{&BaseNode{typ: NodeHardBreak}}
		} else {
			ret = &SoftBreak{&BaseNode{typ: NodeSoftBreak}}
		}
	} else {
		ret = &SoftBreak{&BaseNode{typ: NodeSoftBreak}}
	}

	return
}

func (t *Tree) matchEnd(tokens items, openMarker *item, num int) (pos int) {
	for ; pos < len(tokens); pos++ {
		len := tokens[pos:].accept(openMarker.typ)
		if num <= len {
			return pos
		}
	}

	return -1
}
