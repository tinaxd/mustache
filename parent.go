package mustache

import "regexp"

func getParent(partials PartialProvider, name, indent string, elems []interface{}) (*Template, []blockOverride, error) {
	data, err := partials.Get(name)
	if err != nil {
		return nil, nil, err
	}

	// indent non empty lines
	r := regexp.MustCompile(`(?m:^(.+)$)`)
	data = r.ReplaceAllString(data, indent+"$1")

	// find block elements in the children
	overrides := make([]blockOverride, 0)
	for _, child := range elems {
		if blk, ok := child.(*blockElement); ok {
			overrides = append(overrides, blockOverride{
				name:  blk.name,
				elems: blk.elems,
			})
		}
	}

	tmpl, err := ParseStringPartials(data, partials)
	return tmpl, overrides, err
}
