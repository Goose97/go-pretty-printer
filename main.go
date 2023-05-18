package main

import (
	"fmt"
	"strings"
)

type Doc interface {
	isDoc()
}

type DocNil int

type DocText struct {
	payload string
}

type DocBreak struct {
	payload string
}

type DocCons struct {
	payload []Doc
}

type DocNest struct {
	payload Doc
	indent  int
}

type DocGroup struct {
	payload Doc
}

func (d DocNil) isDoc()   {}
func (d DocText) isDoc()  {}
func (d DocBreak) isDoc() {}
func (d DocCons) isDoc()  {}
func (d DocNest) isDoc()  {}
func (d DocGroup) isDoc() {}

// Inside group, there are two modes
// 1. Flat: line break will be rendered as space (or the specified string)
// 2. Broken: line break will be rendered as new line
type BreakMode int

const (
	// We are not in a group
	None BreakMode = iota + 1
	Flat
	Broken
)

func main() {
	rule1 := CssRule{
		selector: CssCompoundSelector([]CssSelector{
			{selector: "a-tag", combinator: ">"},
			{selector: ".b-selector", combinator: "+"},
			{selector: ".c-selector.d-selector"},
		}),
		properties: []CssProperty{
			{name: "display", value: []string{"flex"}},
			{name: "color", value: []string{"yellow"}},
			{name: "transform", value: []string{"translate(10%, 10%)", "scale(1.2)", "rotate(160deg)"}},
		},
	}

	rule2 := CssRule{
		selector: CssCompoundSelector([]CssSelector{
			{selector: "c-tag", combinator: "~"},
			{selector: ".f-selector.g-selector"},
		}),
		properties: []CssProperty{
			{name: "padding", value: []string{"12px", "12px", "12px", "12px"}},
		},
	}

	var file CssFile = []CssRule{rule1, rule2}
	doc := file.toDoc()

	widths := []int{40, 50, 60}
	for _, w := range widths {
		pretty := format(doc, w)
		fmt.Printf("%s\nPrint with width %v\n%s\n", strings.Repeat("-", 32), w, pretty)
	}
}

type ToDoc interface {
	toDoc() Doc
}

// Concat everything and add breaks in between
func fold[T ToDoc](list []T) Doc {
	doc := _nil()
	for _, i := range list {
		doc = concatWithBreak(doc, i.toDoc())
	}

	return doc
}

// Doc constructors
func _nil() Doc {
	return DocNil(0)
}

func text(s string) Doc {
	return DocText{
		payload: s,
	}
}

// An optional line break. It might printed as a line break or as the given string
func _break() Doc {
	return DocBreak{
		payload: " ",
	}
}

func breakWith(s string) Doc {
	return DocBreak{
		payload: s,
	}

}

// A cons doc, represents mulitple documents combining together
func concat(d1 Doc, d2 Doc) Doc {
	docs := []Doc{}
	if d, ok := d1.(DocCons); ok {
		docs = append(docs, d.payload...)
	} else {
		docs = append(docs, d1)
	}

	if d, ok := d2.(DocCons); ok {
		docs = append(docs, d.payload...)
	} else {
		docs = append(docs, d2)
	}

	return DocCons{
		payload: docs,
	}
}

func concatList(ds []Doc) Doc {
	doc := _nil()

	for i := 0; i < len(ds); i++ {
		doc = concat(doc, ds[i])
	}

	return doc
}

// Concat two docs with a break in between
// If either of doc is a NilDoc, don't add the break
func concatWithBreak(d1 Doc, d2 Doc) Doc {
	if _, ok := d1.(DocNil); ok {
		return d2
	}

	if _, ok := d2.(DocNil); ok {
		return d1
	}

	return concat(concat(d1, _break()), d2)
}

// Increase indentation level of a doc
// Identation only takes effect when a break happens. After a break, indent the new
// line with the current indentation level
func nest(d Doc, indent int) Doc {
	return DocNest{
		payload: d,
		indent:  indent,
	}
}

// Groups in conjunction with the optional line breaks
// introduces alternative layouts context
//
// Regarding line breaks, groups have two modes, see BreakMode
// Groups will greedily try to fit everything in a same line first a.k.a flat mode. If
// it can't, fallback the broken mode
//
// Note that:
// 1. The decision will be made individually by each group. So with a group
// structured like this: group [text, group, line, text], even when the outer is rendered
// as broken mode, the inner still can be rendered in flat mode
// 2. If the outer group is rendered in flat mode, all subgroups will automatically rendered
// in flat mode
func group(d Doc) Doc {
	return DocGroup{
		payload: d,
	}
}

// Given the width of the current line returns whether this doc can fit in one single line
func (d *DocGroup) fits(width int) bool {
	// There are situations where the doc certainly can not fit in one line, but we still have to
	// consider it as fits. For example, a really long text with no break. So we need at least a break
	// to consider the document does not fit. A document with no breaks automatically fits
	encounteredBreak := false
	docs := []Doc{*d}
	c := 0
	breakMode := Flat

	for {
		if c > width && encounteredBreak {
			return false
		}

		if len(docs) == 0 {
			return true
		}

		current, remain := docs[0], docs[1:]
		docs = remain

		switch doc := current.(type) {
		case DocNil:
			continue

		case DocText:
			c += len([]rune(doc.payload))
			continue

		case DocBreak:
			encounteredBreak = true
			switch breakMode {
			case Flat:
				c += len([]rune(doc.payload))
				continue
			case Broken, None:
				continue
			}

		case DocNest:
			docs = append([]Doc{doc.payload}, docs...)
			continue

		case DocCons:
			docs = append(doc.payload, docs...)
			continue

			// A group is considered as fits if it fits in flat mode
		case DocGroup:
			docs = append(docs, doc.payload)
			breakMode = Flat
			continue
		}
	}
}

// We will work with a list of triplet (i, m, d) where:
// i: current indentation level
// m: current break mode
// d: current doc
type DocWithState struct {
	indentation int
	breakMode   BreakMode
	doc         Doc
}

// Given a doc and a maximum width, print it as string with the optimal layout
func format(d Doc, width int) string {
	docs := []DocWithState{
		{indentation: 0, breakMode: None, doc: d},
	}
	columns := 0
	output := ""

	for len(docs) > 0 {
		head, tail := docs[0], docs[1:]
		docs = tail
		current := head.doc

		switch doc := current.(type) {
		case DocNil:
			continue

		case DocText:
			output += doc.payload
			columns += len([]rune(doc.payload))
			continue

		case DocBreak:
			switch head.breakMode {
			case Flat:
				output += doc.payload
				columns += len([]rune(doc.payload))

			case Broken, None:
				// Breaks outside of groups is always rendered as new lines
				output += "\n" + strings.Repeat(" ", head.indentation)
				columns = head.indentation
			}
			continue

		case DocNest:
			nested := DocWithState{
				indentation: head.indentation + doc.indent,
				breakMode:   head.breakMode,
				doc:         doc.payload,
			}

			// The fact that Go doesn't have a builtin prepend is beyond me
			docs = append([]DocWithState{nested}, docs...)

			continue

		case DocCons:
			toPrepend := []DocWithState{}
			for _, d := range doc.payload {
				toPrepend = append(toPrepend, DocWithState{
					indentation: head.indentation,
					breakMode:   head.breakMode,
					doc:         d,
				})
			}

			docs = append(toPrepend, docs...)
			continue

		case DocGroup:
			toPrepend := DocWithState{
				indentation: head.indentation,
				doc:         doc.payload,
			}

			if doc.fits(width - columns) {
				toPrepend.breakMode = Flat
			} else {
				toPrepend.breakMode = Broken
			}

			docs = append([]DocWithState{toPrepend}, docs...)
			continue
		}
	}

	return output
}
