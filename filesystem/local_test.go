package filesystem

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestListDirNodeFromPath(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "olaris-rename-test")
	err = os.MkdirAll(path.Join(tmp, "first"), 0755)
	err = os.MkdirAll(path.Join(tmp, "second"), 0755)
	err = os.MkdirAll(path.Join(tmp, "second/second-one"), 0755)
	err = os.MkdirAll(path.Join(tmp, "third/third-one"), 0755)
	err = os.MkdirAll(path.Join(tmp, "third/third-two"), 0755)
	defer os.RemoveAll(tmp)

	require.NoError(t, err)

	node, err := LocalNodeFromPath(tmp)
	dirs, err := node.ListDir()
	require.NoError(t, err)
	firstLevel := []string{"first", "second", "third"}
	if !reflect.DeepEqual(dirs, firstLevel) {
		t.Errorf("Did not get the correct folders back from ListDir() for first level: %s:%s", dirs, firstLevel)
	}

	secondLevel := []string{"third-one", "third-two"}
	node, err = LocalNodeFromPath(path.Join(tmp, "third"))
	dirs, err = node.ListDir()
	require.NoError(t, err)
	if !reflect.DeepEqual(dirs, secondLevel) {
		t.Errorf("Did not get the correct folders back from ListDir() for second level: %s:%s", dirs, secondLevel)
	}
}
