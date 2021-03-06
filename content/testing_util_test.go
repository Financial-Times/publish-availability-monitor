package content

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

//helper func
func loadBytesForFile(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
