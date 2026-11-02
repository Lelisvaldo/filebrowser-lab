package indexing

import "testing"

func TestIsLabTrashPath(t *testing.T) {
	for p, want := range map[string]bool{"/.lixeira": true, "/.lixeira/": true, "/a/.lixeira/b.txt": true, `\.lixeira\x`: true, "/.lixeira2": false, "/a/lixeira": false, "/Filmes/a.txt": false, "/": false} {
		if got := IsLabTrashPath(p); got != want {
			t.Errorf("IsLabTrashPath(%q) = %v, esperava %v", p, got, want)
		}
	}
}
