// Lute - 一款对中文语境优化的 Markdown 引擎，支持 Go 和 JavaScript
// Copyright (c) 2019-present, b3log.org
//
// Lute is licensed under Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//         http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PSL v2 for more details.

package lute

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/html"
	"github.com/88250/lute/html/atom"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	"github.com/88250/lute/util"
)

// SpinVditorIRBlockDOM 自旋 Vditor Instant-Rendering Block DOM，用于即时渲染块模式下的编辑。
func (lute *Lute) SpinVditorIRBlockDOM(ivHTML string) (ovHTML string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	// 替换插入符
	ivHTML = strings.ReplaceAll(ivHTML, "<wbr>", util.Caret)

	markdown := lute.vditorIRBlockDOM2Md(ivHTML)
	tree := parse.Parse("", []byte(markdown), lute.Options)

	ovHTML = lute.Tree2VditorIRBlockDOM(tree)
	// 替换插入符
	ovHTML = strings.ReplaceAll(ovHTML, util.Caret, "<wbr>")
	return
}

// HTML2VditorIRBlockDOM 将 HTML 转换为 Vditor Instant-Rendering Block DOM，用于即时渲染块模式下粘贴。
func (lute *Lute) HTML2VditorIRBlockDOM(sHTML string) (vHTML string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	markdown, err := lute.HTML2Markdown(sHTML)
	if nil != err {
		vHTML = err.Error()
		return
	}

	tree := parse.Parse("", []byte(markdown), lute.Options)
	renderer := render.NewVditorIRBlockRenderer(tree)
	for nodeType, rendererFunc := range lute.HTML2VditorIRBlockDOMRendererFuncs {
		renderer.ExtRendererFuncs[nodeType] = rendererFunc
	}
	output := renderer.Render()
	if renderer.Option.Footnotes && 0 < len(renderer.Tree.Context.FootnotesDefs) {
		output = renderer.RenderFootnotesDefs(renderer.Tree.Context)
	}
	vHTML = string(output)
	return
}

// VditorIRBlockDOM2HTML 将 Vditor Instant-Rendering Block DOM 转换为 HTML，用于 Vditor.getHTML() 接口。
func (lute *Lute) VditorIRBlockDOM2HTML(vhtml string) (sHTML string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	markdown := lute.vditorIRBlockDOM2Md(vhtml)
	sHTML = lute.Md2HTML(markdown)
	return
}

// Md2VditorIRBlockDOM 将 markdown 转换为 Vditor Instant-Rendering Block DOM，用于从源码模式切换至即时渲染块模式。
func (lute *Lute) Md2VditorIRBlockDOM(markdown string) (vHTML string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	tree := parse.Parse("", []byte(markdown), lute.Options)
	renderer := render.NewVditorIRBlockRenderer(tree)
	for nodeType, rendererFunc := range lute.Md2VditorIRBlockDOMRendererFuncs {
		renderer.ExtRendererFuncs[nodeType] = rendererFunc
	}
	output := renderer.Render()
	if renderer.Option.Footnotes && 0 < len(renderer.Tree.Context.FootnotesDefs) {
		output = renderer.RenderFootnotesDefs(renderer.Tree.Context)
	}
	vHTML = string(output)
	return
}

// VditorIRBlockDOM2Md 将 Vditor Instant-Rendering DOM 转换为 markdown，用于从即时渲染块模式切换至源码模式。
func (lute *Lute) VditorIRBlockDOM2Md(htmlStr string) (markdown string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	htmlStr = strings.ReplaceAll(htmlStr, parse.Zwsp, "")
	markdown = lute.vditorIRBlockDOM2Md(htmlStr)
	markdown = strings.ReplaceAll(markdown, parse.Zwsp, "")
	return
}

func (lute *Lute) VditorIRBlockDOM2Text(htmlStr string) (text string) {
	lute.VditorIR = true
	lute.VditorWYSIWYG = false
	lute.VditorSV = false

	tree, err := lute.VditorIRBlockDOM2Tree(htmlStr)
	if nil != err {
		return ""
	}
	return tree.Root.Text()
}

func (lute *Lute) VditorIRBlockDOMHeadings(htmlStr string) (ret string) {
	tree, err := lute.VditorIRBlockDOM2Tree(htmlStr)
	if nil != err {
		return ""
	}
	headings := tree.Root.ChildrenByType(ast.NodeHeading)
	buf := &bytes.Buffer{}
	for _, heading := range headings {
		if tree.Root != heading.Parent {
			continue
		}

		padding := 16 + 14*(heading.HeadingLevel-1)
		buf.WriteString(`<div data-id="ir-` + render.HeadingID(heading) + `" style="padding-left:` + strconv.Itoa(padding) +
			`px" class="b3-list-item">` + headingText(heading) + `</div>`)
	}
	ret = buf.String()
	return
}

func headingText(n *ast.Node) (ret string) {
	ast.Walk(n, func(n *ast.Node, entering bool) ast.WalkStatus {
		if (ast.NodeText == n.Type || ast.NodeCodeSpanContent == n.Type ||
			ast.NodeLinkText == n.Type || ast.NodeBlockRefText == n.Type || ast.NodeBlockEmbedText == n.Type) && entering {
			ret += util.BytesToStr(n.Tokens)
		}
		return ast.WalkContinue
	})
	return
}

func VditorIRBlockDOMHeadings(htmlStr string) (ret string) {
	luteEngine := New()
	luteEngine.KramdownIAL = true
	return luteEngine.VditorIRBlockDOMHeadings(htmlStr)
}

func (lute *Lute) Tree2VditorIRBlockDOM(tree *parse.Tree) (vHTML string) {
	renderer := render.NewVditorIRBlockRenderer(tree)
	output := renderer.Render()
	if renderer.Option.Footnotes && 0 < len(renderer.Tree.Context.FootnotesDefs) {
		output = renderer.RenderFootnotesDefs(renderer.Tree.Context)
	}
	vHTML = string(output)
	return
}

func (lute *Lute) VditorIRBlockDOM2Tree(htmlStr string) (ret *parse.Tree, err error) {
	// 删掉插入符
	htmlStr = strings.ReplaceAll(htmlStr, "<wbr>", "")

	// 替换结尾空白，否则 HTML 解析会产生冗余节点导致生成空的代码块
	htmlStr = strings.ReplaceAll(htmlStr, "\t\n", "\n")
	htmlStr = strings.ReplaceAll(htmlStr, "    \n", "  \n")

	// 将字符串解析为 DOM 树
	reader := strings.NewReader(htmlStr)
	htmlRoot := &html.Node{Type: html.ElementNode}
	htmlNodes, err := html.ParseFragment(reader, htmlRoot)
	if nil != err {
		return
	}

	// 调整 DOM 结构
	lute.adjustVditorDOM(htmlNodes)

	// 将 HTML 树转换为 Markdown AST

	ret = &parse.Tree{Name: "", Root: &ast.Node{Type: ast.NodeDocument}, Context: &parse.Context{Option: lute.Options}}
	ret.Context.Tip = ret.Root
	for _, htmlNode := range htmlNodes {
		lute.genASTByVditorIRBlockDOM(htmlNode, ret)
	}

	// 调整树结构
	ast.Walk(ret.Root, func(n *ast.Node, entering bool) ast.WalkStatus {
		if entering {
			switch n.Type {
			case ast.NodeInlineHTML, ast.NodeCodeSpan, ast.NodeInlineMath, ast.NodeHTMLBlock, ast.NodeCodeBlockCode, ast.NodeMathBlockContent:
				n.Tokens = html.UnescapeHTML(n.Tokens)
				if nil != n.Next && ast.NodeCodeSpan == n.Next.Type && n.CodeMarkerLen == n.Next.CodeMarkerLen {
					// 合并代码节点 https://github.com/Vanessa219/vditor/issues/167
					n.FirstChild.Next.Tokens = append(n.FirstChild.Next.Tokens, n.Next.FirstChild.Next.Tokens...)
					n.Next.Unlink()
				}
			case ast.NodeList:
				// 浏览器生成的子列表是 ul.ul 形式，需要将其调整为 ul.li.ul
				if nil != n.Parent && ast.NodeList == n.Parent.Type {
					if previousLi := n.Previous; nil != previousLi {
						if ast.NodeKramdownBlockIAL == previousLi.Type {
							previousLi = previousLi.Previous
						}
						listIAL := n.Next
						previousLi.AppendChild(n)
						previousLi.AppendChild(listIAL)
					}
				}
			}
		}
		return ast.WalkContinue
	})
	return
}

func (lute *Lute) vditorIRBlockDOM2Md(htmlStr string) (markdown string) {
	tree, err := lute.VditorIRBlockDOM2Tree(htmlStr)
	if nil != err {
		return err.Error()
	}

	// 将 AST 进行 Markdown 格式化渲染
	renderer := render.NewFormatRenderer(tree)
	formatted := renderer.Render()
	markdown = string(formatted)
	return
}

// genASTByVditorIRBlockDOM 根据指定的 Vditor IR DOM 节点 n 进行深度优先遍历并逐步生成 Markdown 语法树 tree。
func (lute *Lute) genASTByVditorIRBlockDOM(n *html.Node, tree *parse.Tree) {
	dataRender := lute.domAttrValue(n, "data-render")
	if "1" == dataRender || "2" == dataRender { // 1：浮动工具栏，2：preview 代码块、数学公式块或者不解析的节点
		return
	}

	dataType := lute.domAttrValue(n, "data-type")

	if atom.Div == n.DataAtom {
		// TODO: 细化节点 https://github.com/siyuan-note/siyuan-src/issues/27
		if "link-ref-defs-block" == dataType {
			text := lute.domText(n)
			node := &ast.Node{Type: ast.NodeText, Tokens: []byte(text)}
			tree.Context.Tip.AppendChild(node)
			return
		} else if "footnotes-def" == dataType {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				lute.genASTByVditorIRBlockDOM(c, tree)
			}
			return
		} else if "footnotes-block" == dataType {
			for def := n.FirstChild; nil != def; def = def.NextSibling {
				originalHTML := &bytes.Buffer{}
				if err := html.Render(originalHTML, def); nil == err {
					md := lute.vditorIRBlockDOM2Md(originalHTML.String())
					lines := strings.Split(md, "\n")
					md = ""
					for i, line := range lines {
						if 0 < i {
							md += "    " + line
						} else {
							md = line
						}
						md += "\n"
					}
					node := &ast.Node{Type: ast.NodeText, Tokens: []byte(md)}
					tree.Context.Tip.AppendChild(node)
				}
			}
			return
		} else if "toc-block" == dataType {
			node := &ast.Node{Type: ast.NodeText, Tokens: []byte("[toc]\n\n")}
			tree.Context.Tip.AppendChild(node)
			return
		}
	}

	class := lute.domAttrValue(n, "class")
	content := strings.ReplaceAll(n.Data, parse.Zwsp, "")
	nodeID := lute.domAttrValue(n, "data-node-id")
	node := &ast.Node{ID: nodeID, Type: ast.NodeText, Tokens: []byte(content)}
	if "" == nodeID {
		if "p" == dataType || "ul" == dataType || "ol" == dataType || "blockquote" == dataType ||
			"math-block" == dataType || "code-block" == dataType || "table" == dataType || "h" == dataType {
			nodeID = ast.NewNodeID()
		}
	}
	if "" != nodeID {
		node.KramdownIAL = [][]string{{"id", nodeID}}
		ialTokens := []byte("{: id=\"" + nodeID + "\"")
		bookmark := lute.domAttrValue(n, "bookmark")
		if "" != bookmark {
			node.SetIALAttr("bookmark", bookmark)
			ialTokens = append(ialTokens, []byte(" bookmark=\""+bookmark+"\"")...)
		}
		ialTokens = append(ialTokens, '}')
		ial := &ast.Node{Type: ast.NodeKramdownBlockIAL, Tokens: ialTokens}
		defer tree.Context.TipAppendChild(ial)
	}

	switch n.DataAtom {
	case 0, atom.Scope:
		if "" == content {
			return
		}

		if html.ElementNode == n.Type {
			break
		}

		checkIndentCodeBlock := strings.ReplaceAll(content, util.Caret, "")
		checkIndentCodeBlock = strings.ReplaceAll(checkIndentCodeBlock, "\t", "    ")
		if (!lute.isInline(n.PrevSibling)) && strings.HasPrefix(checkIndentCodeBlock, "    ") {
			node.Type = ast.NodeCodeBlock
			node.IsFencedCodeBlock = true
			node.AppendChild(&ast.Node{Type: ast.NodeCodeBlockFenceOpenMarker, Tokens: []byte("```"), CodeBlockFenceLen: 3})
			node.AppendChild(&ast.Node{Type: ast.NodeCodeBlockFenceInfoMarker})
			startCaret := strings.HasPrefix(content, util.Caret)
			if startCaret {
				content = strings.ReplaceAll(content, util.Caret, "")
			}
			content = strings.TrimSpace(content)
			if startCaret {
				content = util.Caret + content
			}
			content := &ast.Node{Type: ast.NodeCodeBlockCode, Tokens: []byte(content)}
			node.AppendChild(content)
			node.AppendChild(&ast.Node{Type: ast.NodeCodeBlockFenceCloseMarker, Tokens: []byte("```"), CodeBlockFenceLen: 3})
			tree.Context.Tip.AppendChild(node)
			return
		}
		if nil != n.Parent && atom.A == n.Parent.DataAtom {
			node.Type = ast.NodeLinkText
		}

		if ast.NodeCodeBlock == tree.Context.Tip.Type {
			// 开始代码块标记符后退格的情况
			tree.Context.Tip.Type = ast.NodeParagraph
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(content)})
			if nil != n.NextSibling && nil != n.NextSibling.NextSibling && nil != n.NextSibling.NextSibling.NextSibling &&
				nil != n.NextSibling.NextSibling.NextSibling.NextSibling {
				n.Parent.RemoveChild(n.NextSibling.NextSibling.NextSibling.NextSibling)
			}
			break
		}

		tokens := make([]byte, len(node.Tokens))
		copy(tokens, node.Tokens)

		// 尝试块级解析，处理列表代码块
		subTree := parse.Parse("", tokens, tree.Context.Option)
		if nil != subTree.Root.FirstChild && ast.NodeCodeBlock == subTree.Root.FirstChild.Type {
			node.Tokens = bytes.TrimPrefix(node.Tokens, []byte("\n"))
			tree.Context.Tip.AppendChild(node)
		} else {
			// 尝试行级解析，处理段落图片文本节点转换为图片节点
			subTree = parse.Inline("", tokens, tree.Context.Option)
			if ast.NodeSoftBreak == subTree.Root.FirstChild.FirstChild.Type || // 软换行
				(ast.NodeParagraph == subTree.Root.FirstChild.Type &&
					(ast.NodeImage == subTree.Root.FirstChild.FirstChild.Type ||
						(ast.NodeSoftBreak == subTree.Root.FirstChild.FirstChild.Type && nil != subTree.Root.FirstChild.FirstChild.Next &&
							(ast.NodeText == subTree.Root.FirstChild.FirstChild.Next.Type ||
								ast.NodeEmphasis == subTree.Root.FirstChild.FirstChild.Next.Type ||
								ast.NodeStrong == subTree.Root.FirstChild.FirstChild.Next.Type ||
								ast.NodeStrikethrough == subTree.Root.FirstChild.FirstChild.Next.Type ||
								ast.NodeCodeSpan == subTree.Root.FirstChild.FirstChild.Next.Type ||
								ast.NodeMark == subTree.Root.FirstChild.FirstChild.Next.Type))) && // 软换行后跟普通文本
					nil == subTree.Root.FirstChild.Next) {
				appendNextToTip(subTree.Root.FirstChild.FirstChild, tree)
			} else {
				node.Tokens = bytes.TrimPrefix(node.Tokens, []byte("\n"))
				tree.Context.Tip.AppendChild(node)
			}
		}
	case atom.P:
		node.Type = ast.NodeParagraph
		text := lute.domText(n)
		if "\n" == text && ast.NodeBlockquote == tree.Context.Tip.Type && nil == tree.Context.Tip.FirstChild.Next {
			// 不允许在 bq 第一个节点前换行
			return
		} else {
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		}
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		text := lute.domText(n)
		if "" == strings.TrimSpace(text) {
			return
		}
		node.Type = ast.NodeHeading
		marker := lute.domAttrValue(n, "data-marker")
		node.HeadingSetext = "=" == marker || "-" == marker
		if !node.HeadingSetext {
			marker := lute.domText(n.FirstChild)
			node.HeadingLevel = bytes.Count([]byte(marker), []byte("#"))
		} else {
			if "" == strings.TrimSpace(strings.ReplaceAll(lute.domText(n.LastChild), util.Caret, "")) {
				node.Type = ast.NodeText
				node.Tokens = []byte(text)
				tree.Context.Tip.AppendChild(node)
				tree.Context.Tip = node
				break
			}

			if "=" == marker {
				node.HeadingLevel = 1
			} else {
				node.HeadingLevel = 2
			}
			if nil != n.LastChild.PrevSibling {
				n.LastChild.PrevSibling.Data = strings.TrimSuffix(n.LastChild.PrevSibling.Data, "\n")
			}
		}
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Hr:
		node.Type = ast.NodeThematicBreak
		tree.Context.Tip.AppendChild(node)
	case atom.Blockquote:
		content := strings.TrimSpace(lute.domText(n))
		if "" == content || "&gt;" == content {
			return
		}
		if util.Caret == content {
			node.Type = ast.NodeText
			node.Tokens = []byte(content)
			tree.Context.Tip.AppendChild(node)
		}

		node.Type = ast.NodeBlockquote
		node.AppendChild(&ast.Node{Type: ast.NodeBlockquoteMarker, Tokens: []byte(">")})
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Ol, atom.Ul:
		if nil == n.FirstChild {
			return
		}

		node.Type = ast.NodeList
		node.ListData = &ast.ListData{}
		marker := lute.domAttrValue(n, "data-marker")
		if "" == marker {
			marker = "*"
		}
		if atom.Ol == n.DataAtom {
			node.ListData.Typ = 1
			start := lute.domAttrValue(n, "start")
			if "" == start {
				start = "1"
			}
			node.ListData.Start, _ = strconv.Atoi(start)
		} else {
			node.ListData.BulletChar = marker[0]
		}
		node.ListData.Marker = []byte(marker)
		tight := lute.domAttrValue(n, "data-tight")
		if "true" == tight || "" == tight {
			node.Tight = true
		}
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Li:
		if ast.NodeList != tree.Context.Tip.Type {
			parent := &ast.Node{}
			parent.Type = ast.NodeList
			parent.ListData = &ast.ListData{Tight: true}
			marker := lute.domAttrValue(n, "data-marker")
			if "" == marker {
				marker = "*"
			}
			tree.Context.Tip.AppendChild(parent)
			tree.Context.Tip = parent
		}

		if p := n.FirstChild; nil != p && atom.P == p.DataAtom && nil != p.NextSibling && atom.P == p.NextSibling.DataAtom {
			tree.Context.Tip.Tight = false
		}

		node.Type = ast.NodeListItem
		marker := lute.domAttrValue(n, "data-marker")
		var bullet byte
		if "" == marker {
			if nil != n.Parent && atom.Ol == n.Parent.DataAtom {
				firstLiMarker := lute.domAttrValue(n.Parent.FirstChild, "data-marker")
				if startAttr := lute.domAttrValue(n.Parent, "start"); "" == startAttr {
					marker = "1"
				} else {
					marker = startAttr
				}
				if "" != firstLiMarker {
					marker += firstLiMarker[len(firstLiMarker)-1:]
				} else {
					marker += "."
				}
			} else {
				marker = lute.domAttrValue(n.Parent, "data-marker")
				if "" == marker {
					marker = "*"
				}
				bullet = marker[0]
			}
		} else {
			if nil != n.Parent {
				if atom.Ol == n.Parent.DataAtom {
					if "*" == marker || "-" == marker || "+" == marker {
						marker = "1."
					}
					if "1." != marker && "1)" != marker && nil != n.PrevSibling && atom.Li != n.PrevSibling.DataAtom &&
						nil != n.Parent.Parent && (atom.Ol == n.Parent.Parent.DataAtom || atom.Ul == n.Parent.Parent.DataAtom) {
						// 子有序列表第一项必须从 1 开始
						marker = "1."
					}
					if "1." != marker && "1)" != marker && atom.Ol == n.Parent.DataAtom && n.Parent.FirstChild == n && "" == lute.domAttrValue(n.Parent, "start") {
						marker = "1."
					}
				} else {
					if "*" != marker && "-" != marker && "+" != marker {
						marker = "*"
					}
					bullet = marker[0]
				}
			} else {
				marker = lute.domAttrValue(n, "data-marker")
				if "" == marker {
					marker = "*"
				}
				bullet = marker[0]
			}
		}
		node.ListData = &ast.ListData{Marker: []byte(marker), BulletChar: bullet}
		if 0 == bullet {
			node.ListData.Num, _ = strconv.Atoi(marker[:len(marker)-1])
			node.ListData.Delimiter = marker[len(marker)-1]
		}
		if "vditor-task" == lute.domAttrValue(n, "class") {
			node.ListData.Typ = 3
			tree.Context.Tip.ListData.Typ = 3
		}

		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Pre:
		if atom.Code == n.FirstChild.DataAtom {
			var codeTokens []byte
			if nil != n.FirstChild.FirstChild {
				codeTokens = []byte(n.FirstChild.FirstChild.Data)
			}

			divDataType := lute.domAttrValue(n.Parent, "data-type")
			switch divDataType {
			case "math-block":
				node.Type = ast.NodeMathBlockContent
				node.Tokens = codeTokens
				tree.Context.Tip.AppendChild(node)
			case "html-block":
				tree.Context.Tip.Tokens = codeTokens
			case "yaml-front-matter":
				node.Type = ast.NodeYamlFrontMatterContent
				node.Tokens = codeTokens
				tree.Context.Tip.AppendChild(node)
			default:
				node.Type = ast.NodeCodeBlockCode
				node.Tokens = codeTokens
				tree.Context.Tip.AppendChild(node)
			}
		}
		return
	case atom.Em, atom.I:
		if nil == n.FirstChild || atom.Br == n.FirstChild.DataAtom {
			return
		}
		if lute.starstWithNewline(n.FirstChild) {
			n.FirstChild.Data = strings.TrimLeft(n.FirstChild.Data, parse.Zwsp+"\n")
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp + "\n")})
		}
		text := strings.TrimSpace(lute.domText(n))
		if lute.isEmptyText(n) {
			return
		}
		if util.Caret == text {
			node.Tokens = util.CaretTokens
			tree.Context.Tip.AppendChild(node)
			return
		}

		node.Type = ast.NodeEmphasis
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Strong, atom.B:
		if nil == n.FirstChild || atom.Br == n.FirstChild.DataAtom {
			return
		}
		if lute.starstWithNewline(n.FirstChild) {
			n.FirstChild.Data = strings.TrimLeft(n.FirstChild.Data, parse.Zwsp+"\n")
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp + "\n")})
		}
		text := strings.TrimSpace(lute.domText(n))
		if lute.isEmptyText(n) {
			return
		}
		if util.Caret == text {
			node.Tokens = util.CaretTokens
			tree.Context.Tip.AppendChild(node)
			return
		}

		node.Type = ast.NodeStrong
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Del, atom.S, atom.Strike:
		if nil == n.FirstChild || atom.Br == n.FirstChild.DataAtom {
			return
		}
		if lute.starstWithNewline(n.FirstChild) {
			n.FirstChild.Data = strings.TrimLeft(n.FirstChild.Data, parse.Zwsp+"\n")
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp + "\n")})
		}
		text := strings.TrimSpace(lute.domText(n))
		if lute.isEmptyText(n) {
			return
		}
		if util.Caret == text {
			node.Tokens = util.CaretTokens
			tree.Context.Tip.AppendChild(node)
			return
		}

		node.Type = ast.NodeStrikethrough
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Mark:
		if nil == n.FirstChild || atom.Br == n.FirstChild.DataAtom {
			return
		}
		if lute.starstWithNewline(n.FirstChild) {
			n.FirstChild.Data = strings.TrimLeft(n.FirstChild.Data, parse.Zwsp+"\n")
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp + "\n")})
		}
		text := strings.TrimSpace(lute.domText(n))
		if lute.isEmptyText(n) {
			return
		}
		if util.Caret == text {
			node.Tokens = util.CaretTokens
			tree.Context.Tip.AppendChild(node)
			return
		}

		node.Type = ast.NodeMark
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Code:
		if nil == n.FirstChild {
			return
		}
		contentStr := strings.ReplaceAll(n.FirstChild.Data, parse.Zwsp, "")
		if util.Caret == contentStr {
			node.Tokens = util.CaretTokens
			tree.Context.Tip.AppendChild(node)
			return
		}
		if "" == contentStr {
			return
		}
		codeTokens := []byte(contentStr)
		content := &ast.Node{Type: ast.NodeCodeSpanContent, Tokens: codeTokens}
		node.Type = ast.NodeCodeSpan
		node.AppendChild(content)
		tree.Context.Tip.AppendChild(node)
		return
	case atom.Br:
		if nil != n.Parent {
			if lute.parentIs(n, atom.Td, atom.Th) {
				if (nil == n.PrevSibling || util.Caret == n.PrevSibling.Data) && (nil == n.NextSibling || util.Caret == n.NextSibling.Data) {
					return
				}
				if nil == n.NextSibling {
					return // 删掉表格中结尾的 br
				}

				node.Type = ast.NodeInlineHTML
				node.Tokens = []byte("<br />")
				tree.Context.Tip.AppendChild(node)
				return
			}
			if atom.P == n.Parent.DataAtom {
				if nil != n.Parent.NextSibling && (atom.Ul == n.Parent.NextSibling.DataAtom || atom.Ol == n.Parent.NextSibling.DataAtom || atom.Blockquote == n.Parent.NextSibling.DataAtom) {
					tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp)})
					return
				}
				if nil != n.Parent.Parent && nil != n.Parent.Parent.NextSibling && atom.Li == n.Parent.Parent.NextSibling.DataAtom {
					// TODO 暂不确定是否能彻底移除
					//tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeText, Tokens: []byte(parse.Zwsp)})
					//return
				}
			}
		}

		node.Type = ast.NodeHardBreak
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.A:
		node.Type = ast.NodeLink
		node.AppendChild(&ast.Node{Type: ast.NodeOpenBracket})
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Img:
		imgClass := class
		imgAlt := lute.domAttrValue(n, "alt")
		if "emoji" == imgClass {
			node.Type = ast.NodeEmoji
			emojiImg := &ast.Node{Type: ast.NodeEmojiImg, Tokens: tree.EmojiImgTokens(imgAlt, lute.domAttrValue(n, "src"))}
			emojiImg.AppendChild(&ast.Node{Type: ast.NodeEmojiAlias, Tokens: []byte(":" + imgAlt + ":")})
			node.AppendChild(emojiImg)
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		} else {
			return
		}
	case atom.Input:
		if nil == n.Parent || (atom.P != n.Parent.DataAtom && atom.Li != n.Parent.DataAtom) {
			// 仅允许 input 出现在任务列表中
			return
		}
		if nil != n.NextSibling && atom.Span == n.NextSibling.DataAtom {
			// 在任务列表前退格
			n.NextSibling.FirstChild.Data = strings.TrimSpace(n.NextSibling.FirstChild.Data)
			break
		}
		node.Type = ast.NodeTaskListItemMarker
		if lute.hasAttr(n, "checked") {
			node.TaskListItemChecked = true
		}
		tree.Context.Tip.AppendChild(node)
		if nil != node.Parent.Parent.Parent && nil != node.Parent.Parent.Parent.ListData { // ul.li.p.input
			node.Parent.Parent.Parent.ListData.Typ = 3
			node.Parent.Parent.ListData.Typ = 3
		}
	case atom.Table:
		node.Type = ast.NodeTable
		var tableAligns []int
		for th := n.FirstChild.FirstChild.FirstChild; nil != th; th = th.NextSibling {
			align := lute.domAttrValue(th, "align")
			switch align {
			case "left":
				tableAligns = append(tableAligns, 1)
			case "center":
				tableAligns = append(tableAligns, 2)
			case "right":
				tableAligns = append(tableAligns, 3)
			default:
				tableAligns = append(tableAligns, 0)
			}
		}
		node.TableAligns = tableAligns
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Thead:
		node.Type = ast.NodeTableHead
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Tbody:
	case atom.Tr:
		node.Type = ast.NodeTableRow
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Th, atom.Td:
		node.Type = ast.NodeTableCell
		align := lute.domAttrValue(n, "align")
		var tableAlign int
		switch align {
		case "left":
			tableAlign = 1
		case "center":
			tableAlign = 2
		case "right":
			tableAlign = 3
		default:
			tableAlign = 0
		}
		node.TableCellAlign = tableAlign
		tree.Context.Tip.AppendChild(node)
		tree.Context.Tip = node
		defer tree.Context.ParentTip()
	case atom.Sup:
		if nil == n.FirstChild {
			break
		}
		if "footnotes-ref" == dataType {
			node.Type = ast.NodeText
			node.Tokens = []byte(lute.domText(n))
			tree.Context.Tip.AppendChild(node)
		}
		return
	case atom.Span:
		if nil == n.FirstChild {
			break
		}

		switch dataType {
		case "block-ref":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if textNode := t.Root.FirstChild.FirstChild; nil != textNode && ast.NodeText == textNode.Type &&
				nil != textNode.Next && ast.NodeBlockRef == textNode.Next.Type {
				content := textNode.Text()
				if ("！"+util.Caret == content) || ("!"+util.Caret == content) {
					text = strings.Replace(text, content, "!", 1) + util.Caret
					t = parse.Parse("", []byte(text), lute.Options)
				}
			}
			if blockRef := t.Root.FirstChild.FirstChild; nil != blockRef && ast.NodeBlockRef == blockRef.Type {
				node = blockRef
				next := blockRef.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			if blockEmbed := t.Root.FirstChild; nil != blockEmbed && ast.NodeBlockEmbed == blockEmbed.Type {
				node = blockEmbed
				next := blockEmbed.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}

			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "heading-id":
			node.Type = ast.NodeHeadingID
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			if !strings.HasSuffix(text, "}") {
				node.Type = ast.NodeText
				node.Tokens = []byte(text)
				tree.Context.Tip.AppendChild(node)
				return
			}
			text = strings.TrimSpace(text)
			node.Tokens = []byte(text[1 : len(text)-1])
			tree.Context.Tip.AppendChild(node)
			return
		case "em", "strong", "s", "mark", "code", "inline-math", "tag":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if inlineNode := t.Root.FirstChild.FirstChild; nil != inlineNode && (ast.NodeEmphasis == inlineNode.Type ||
				ast.NodeStrong == inlineNode.Type || ast.NodeStrikethrough == inlineNode.Type || ast.NodeMark == inlineNode.Type ||
				ast.NodeCodeSpan == inlineNode.Type || ast.NodeInlineMath == inlineNode.Type || ast.NodeTag == inlineNode.Type) {
				node = inlineNode
				next := inlineNode.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "a", "link-ref", "img":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if inlineNode := t.Root.FirstChild.FirstChild; nil != inlineNode && (ast.NodeLink == inlineNode.Type || ast.NodeImage == inlineNode.Type) {
				node = inlineNode
				next := inlineNode.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "html-inline":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			node.Type = ast.NodeInlineHTML
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "html-entity":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if inlineNode := t.Root.FirstChild.FirstChild; nil != inlineNode && (ast.NodeHTMLEntity == inlineNode.Type) {
				node = inlineNode
				next := inlineNode.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "emoji":
			text := lute.domText(n.FirstChild.NextSibling)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if inlineNode := t.Root.FirstChild.FirstChild; nil != inlineNode && (ast.NodeEmoji == inlineNode.Type) {
				node = inlineNode
				next := inlineNode.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "backslash":
			text := lute.domText(n)
			if "" == strings.TrimSpace(text) {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if inlineNode := t.Root.FirstChild.FirstChild; nil != inlineNode && (ast.NodeBackslash == inlineNode.Type) {
				node = inlineNode
				next := inlineNode.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "inline-node":
			text := lute.domText(n)
			node.Type = ast.NodeText
			if "</font>" == text {
				node.Type = ast.NodeInlineHTML
			}
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			return
		case "math-block-close-marker":
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeMathBlockCloseMarker, Tokens: parse.MathBlockMarker})
			return
		case "math-block-open-marker":
			node.Type = ast.NodeMathBlockOpenMarker
			node.Tokens = parse.MathBlockMarker
			tree.Context.Tip.AppendChild(node)
			return
		case "yaml-front-matter-close-marker":
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeYamlFrontMatterCloseMarker, Tokens: parse.YamlFrontMatterMarker})
			return
		case "yaml-front-matter-open-marker":
			node.Type = ast.NodeYamlFrontMatterOpenMarker
			node.Tokens = parse.MathBlockMarker
			tree.Context.Tip.AppendChild(node)
			return
		case "code-block-open-marker":
			if atom.Pre == n.NextSibling.DataAtom { // DOM 后缺少 info span 节点
				n.InsertAfter(&html.Node{DataAtom: atom.Span, Attr: []*html.Attribute{{Key: "data-type", Val: "code-block-info"}}})
			}
			marker := []byte(lute.domText(n))
			lastBacktick := bytes.LastIndex(marker, []byte("`")) + 1
			if 0 < lastBacktick {
				// 把 ` 后面的字符调整到 info 节点
				n.NextSibling.AppendChild(&html.Node{Data: string(marker[lastBacktick:])})
				marker = marker[:lastBacktick]
			}
			tree.Context.Tip.IsFencedCodeBlock = true
			node.Type = ast.NodeCodeBlockFenceOpenMarker
			node.Tokens = marker
			node.CodeBlockFenceLen = len(marker)
			tree.Context.Tip.AppendChild(node)
			return
		case "code-block-info":
			info := []byte(lute.domText(n))
			info = bytes.ReplaceAll(info, []byte(parse.Zwsp), nil)
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeCodeBlockFenceInfoMarker, CodeBlockInfo: info})
			return
		case "code-block-close-marker":
			marker := []byte(lute.domText(n))
			lastBacktick := bytes.LastIndex(marker, []byte("`")) + 1
			if 0 < lastBacktick {
				marker = marker[:lastBacktick]
			}
			if 0 == len(marker) {
				marker = []byte("```")
			}
			tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeCodeBlockFenceCloseMarker, Tokens: marker, CodeBlockFenceLen: len(marker)})
			return
		case "heading-marker":
			text := lute.domText(n)
			if caretInMarker := strings.Contains(text, util.Caret); caretInMarker {
				caret := &html.Node{Type: html.TextNode, Data: util.Caret}
				n.InsertAfter(caret)
				text = strings.ReplaceAll(text, "#", "")
				text = strings.ReplaceAll(text, util.Caret, "")
				text = strings.TrimSpace(text)
				if 0 < len(text) {
					caret.Data = text + caret.Data
				}
			}
			return
		}

		text := lute.domText(n)
		node.Type = ast.NodeText
		node.Tokens = []byte(text)
		tree.Context.Tip.AppendChild(node)
		return
	case atom.Div:
		switch dataType {
		case "math-block":
			node.Type = ast.NodeMathBlock
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		case "code-block":
			node.Type = ast.NodeCodeBlock
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		case "html-block":
			node.Type = ast.NodeHTMLBlock
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		case "yaml-front-matter":
			node.Type = ast.NodeYamlFrontMatter
			tree.Context.Tip.AppendChild(node)
			tree.Context.Tip = node
			defer tree.Context.ParentTip()
		case "block-ref-embed":
			text := lute.domText(n)
			if "" == text {
				return
			}

			t := parse.Parse("", []byte(text), lute.Options)
			if blockEmbed := t.Root.FirstChild; nil != blockEmbed && ast.NodeBlockEmbed == blockEmbed.Type {
				ial, id := node.KramdownIAL, node.ID
				node = blockEmbed
				node.KramdownIAL, node.ID = ial, id
				next := blockEmbed.Next
				tree.Context.Tip.AppendChild(node)
				appendNextToTip(next, tree)
				return
			}
			node.Type = ast.NodeText
			node.Tokens = []byte(text)
			tree.Context.Tip.AppendChild(node)
			defer tree.Context.ParentTip()
			return
		}
	case atom.Font:
		text := lute.domText(n)
		inlineTree := parse.Inline("", []byte(text), tree.Context.Option)
		appendNextToTip(inlineTree.Root.FirstChild.FirstChild, tree)
		return
	case atom.Details:
		node.Type = ast.NodeHTMLBlock
		node.Tokens = lute.domHTML(n)
		node.Tokens = bytes.SplitAfter(node.Tokens, []byte("</summary>"))[0]
		tree.Context.Tip.AppendChild(node)
	case atom.Kbd:
		// kbd 标签由 code 标签构成节点
	case atom.Summary:
		return
	default:
		node.Type = ast.NodeHTMLBlock
		node.Tokens = lute.domHTML(n)
		tree.Context.Tip.AppendChild(node)
		return
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		lute.genASTByVditorIRBlockDOM(c, tree)
	}

	switch n.DataAtom {
	case atom.A:
		node.AppendChild(&ast.Node{Type: ast.NodeCloseBracket})
		node.AppendChild(&ast.Node{Type: ast.NodeOpenParen})
		href := lute.domAttrValue(n, "href")
		if "" != lute.LinkBase {
			href = strings.ReplaceAll(href, lute.LinkBase, "")
		}
		if "" != lute.LinkPrefix {
			href = strings.ReplaceAll(href, lute.LinkPrefix, "")
		}
		node.AppendChild(&ast.Node{Type: ast.NodeLinkDest, Tokens: []byte(href)})
		linkTitle := lute.domAttrValue(n, "title")
		if "" != linkTitle {
			node.AppendChild(&ast.Node{Type: ast.NodeLinkSpace})
			node.AppendChild(&ast.Node{Type: ast.NodeLinkTitle, Tokens: []byte(linkTitle)})
		}
		node.AppendChild(&ast.Node{Type: ast.NodeCloseParen})
	case atom.Details:
		tree.Context.Tip.AppendChild(&ast.Node{Type: ast.NodeHTMLBlock, Tokens: []byte("</details>")})
	}
}

func appendNextToTip(next *ast.Node, tree *parse.Tree) {
	var nodes []*ast.Node
	for n := next; nil != n; n = n.Next {
		nodes = append(nodes, n)
	}
	for _, n := range nodes {
		tree.Context.Tip.AppendChild(n)
	}
}

func FilePath(p string) string {
	id := FileID(p)
	if "" == id {
		return p
	}
	ret := strings.TrimSuffix(p, ".sy.md")
	ret = strings.TrimSuffix(ret, "_"+id)
	return ret
}

func FileID(p string) (ret string) {
	if 0 <= strings.Index(ret, "/") {
		p = p[strings.LastIndex(p, "/"):]
	}
	p = strings.TrimSuffix(p, ".md")
	p = strings.TrimSuffix(p, ".sy")
	if !strings.Contains(p, "_") {
		return
	}
	idx := strings.LastIndex(p, "_")
	if len(p)-1 == idx {
		return
	}

	ret = p[idx+1:]
	if len(ret) != len("20201001000000-1949101") {
		return ""
	}
	return
}

func NormalizeLinkBase(linkBase string) string {
	ret := linkBase
	if !strings.HasPrefix(ret, "http://127.0.0.1") {
		// 远程 WebDAV，走本地反代
		ret = "/remote?url=" + ret + "/"
		return ret
	}

	ret = ret[len("http://127.0.0.1:6806"):]
	if !strings.HasSuffix(ret, "/") {
		ret += "/"
	}
	return ret
}
