package inflection

import "testing"

func TestSingular(t *testing.T) {
	testSingular(t, "campus")
	testSingular(t, "Campus")
	testSingular(t, "meta")
	testSingular(t, "Meta")
}

func testSingular(t *testing.T, name string) {
	params := SingularParams{
		Name:       name,
		Exclusions: make([]string, 0),
	}
	result := Singular(params)
	expected := name

	if result != expected {
		t.Errorf("TestSingular(%s) failed: expected %s, got %s", name, expected, result)
	}
}
