// Lute - 一款对中文语境优化的 Markdown 引擎，支持 Go 和 JavaScript
// Copyright (c) 2019-present, b3log.org
//
// Lute is licensed under Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//         http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PSL v2 for more details.

package render

import (
	"unicode"
	"unicode/utf8"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/util"
)

// Space 会把文本节点 textNode 中的中西文之间加上空格。
func (r *BaseRenderer) Space(textNode *ast.Node) {
	text := util.BytesToStr(textNode.Tokens)
	text = Space0(text)
	textNode.Tokens = util.StrToBytes(text)
}

func Space0(text string) (ret string) {
	runes := []rune(text)
	length := len(runes)
	var r rune
	for i := 0; i < length; {
		r = runes[i]
		if i < length-3 && 'i' == runes[i+1] && 'n' == runes[i+2] && 'g' == runes[i+3] && unicode.Is(unicode.Han, runes[i]) {
			// ing 前不需要空格，如 打码ing https://github.com/88250/lute/issues/9
			ret += string(r) + "ing"
			i += 4
			continue
		}
		ret = addSpaceAtBoundary(ret, r)
		i++
	}
	return
}

func addSpaceAtBoundary(prefix string, nextChar rune) string {
	if 0 == len(prefix) {
		return string(nextChar)
	}

	if "1" <= prefix && "9" >= prefix && 65039 == nextChar { // Emoji 1-9
		// 在这里处理并不是太合适，应该在 emoji.go 中直接将 Unicode Emoji 解析为节点
		return prefix + string(nextChar)
	}

	currentChar, _ := utf8.DecodeLastRuneInString(prefix)
	if allowSpace(currentChar, nextChar) {
		return prefix + " " + string(nextChar)
	}
	return prefix + string(nextChar)
}

func allowSpace(currentChar, nextChar rune) bool {
	if unicode.IsSpace(currentChar) || unicode.IsSpace(nextChar) ||
		(util.CaretRune == currentChar) || (util.CaretRune == nextChar) ||
		!unicode.IsPrint(currentChar) || !unicode.IsPrint(nextChar) {
		return false
	}

	currentIsHan := unicode.Is(unicode.Han, currentChar)
	nextIsPunct := '%' != nextChar && (unicode.IsPunct(nextChar) || '~' == nextChar || '=' == nextChar || '#' == nextChar)
	if currentIsHan && nextIsPunct {
		return false
	}

	currentIsPunct := '%' != currentChar && (unicode.IsPunct(currentChar) || '~' == currentChar || '=' == currentChar || '#' == currentChar)
	nextIsHan := unicode.Is(unicode.Han, nextChar)
	if nextIsHan && currentIsPunct {
		return false
	}

	if (!currentIsHan && !nextIsHan) || (currentIsHan && nextIsHan) {
		return false
	}
	return true
}
