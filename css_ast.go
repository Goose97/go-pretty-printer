package main

type CssFile []CssRule

type CssRule struct {
	selector   CssCompoundSelector
	properties []CssProperty
}

type CssCompoundSelector []CssSelector

type CssSelector struct {
	selector   string
	combinator string
}

type CssProperty struct {
	name  string
	value []string
}

func (property CssProperty) toDoc() Doc {
	valueDoc := _nil()
	for _, s := range property.value {
		valueDoc = concatWithBreak(valueDoc, text(s))
	}
	valueDoc = group(nest(valueDoc, 2))

	// name: value;
	return concatList([]Doc{
		text(property.name),
		text(": "),
		valueDoc,
		text(";"),
	})
}

func (selector *CssSelector) toDoc() Doc {
	return concatList([]Doc{
		text(selector.selector),
		text(" "),
		text(selector.combinator),
	})
}

func (selector *CssCompoundSelector) toDoc() Doc {
	selectors := []*CssSelector{}
	for i := 0; i < len(*selector); i++ {
		selectors = append(selectors, &(*selector)[i])
	}

	return fold(selectors)
}

func (rule *CssRule) toDoc() Doc {
	selectorDoc := rule.selector.toDoc()

	// selector {
	//   property1
	//   property2
	//   ...
	// }
	return concatList([]Doc{
		selectorDoc,
		text("{"),
		nest(concatList([]Doc{
			_break(),
			fold(rule.properties),
		}), 2),
		_break(),
		text("}"),
	})
}

func (file *CssFile) toDoc() Doc {
	rules := []*CssRule{}
	for i := 0; i < len(*file); i++ {
		rules = append(rules, &(*file)[i])
	}
	return fold(rules)
}
