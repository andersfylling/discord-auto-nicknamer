package nicknamer

import (
	"strings"
	"testing"
)

func TestReadOnlyDictionary_RemoveGenericUnsafe(t *testing.T) {
	d := &ReadOnlyDictionary{}
	l := []string{"a", "b", "c", "d"}
	d.RemoveGenericUnsafe(&l, "b")

	if strings.Contains(strings.Join(l, ""), "b") {
		t.Errorf("b should have been deleted")
	}

	d.RemoveGenericUnsafe(&l, "a")
	if strings.Contains(strings.Join(l, ""), "a") {
		t.Errorf("a should have been deleted")
	}

	d.RemoveGenericUnsafe(&l, "d")
	if strings.Contains(strings.Join(l, ""), "d") {
		t.Errorf("d should have been deleted")
	}
}
