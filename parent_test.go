package mustache

import (
	"os"
	"path"
	"testing"
)

func TestParent1(t *testing.T) {
	filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "parent1.mustache")
	expected := "default content"
	tmpl, err := ParseFile(filename)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := tmpl.Render(map[string]string{})
	if err != nil {
		t.Error(err)
		return
	} else if output != expected {
		t.Errorf("testparent1 expected %q got %q", expected, output)
		return
	}

	expectedTags := []tag{
		{
			Type: Parent,
			Name: "include1",
			Tags: []tag{},
		},
	}
	compareTags(t, tmpl.Tags(), expectedTags)
}
