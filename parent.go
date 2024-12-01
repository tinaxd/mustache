package mustache

import "regexp"

func getParent(partials PartialProvider, name, indent string) (*Template, error) {
	data, err := partials.Get(name)
	if err != nil {
		return nil, err
	}

	// indent non empty lines
	r := regexp.MustCompile(`(?m:^(.+)$)`)
	data = r.ReplaceAllString(data, indent+"$1")

	tmpl, err := ParseStringPartials(data, partials)
	return tmpl, err
}
