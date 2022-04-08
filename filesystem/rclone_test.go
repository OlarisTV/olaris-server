package filesystem

import (
	"context"
	"sync"
	"testing"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fstest/mockfs"
	"github.com/rclone/rclone/fstest/mockobject"
	"github.com/stretchr/testify/require"
)

/* I can't get this to work, sad panda :-(
func TestRcloneListDir(t *testing.T) {
	ctx := context.Background()
	mfs := mockfs.NewFs("rclone::/", "/")
	mfs.AddObject(mockobject.New("test-dir"))
	mfs.AddObject(mockobject.New("test-other-dir"))
	mfs.AddObject(mockobject.New("test-yet-an-other-dir"))
	vfsCache["rclone:"] = vfs.New(mfs, &vfscommon.DefaultOpt)
	dir, err := mfs.List(ctx, "")
	require.NoError(t, err)
	fmt.Println("List:", dir)
	path := "rclone:/"
	node, err := RcloneNodeFromPath(path)
	require.NoError(t, err)
	dirs, err := node.ListDir()
	firstLevel := []string{"test-dir", "test-other-dir", "test-yet-an-other-dir"}
	if !reflect.DeepEqual(dirs, firstLevel) {
		t.Errorf("Did not get the correct folders back from ListDir() for first level: %s:%s", dirs, firstLevel)
	}
}*/

func TestRcloneNodeFromPath(t *testing.T) {
	tests := []struct {
		name          string
		giveNewFsFunc func(context.Context, string) (fs.Fs, error)
		givePathStrs  []string
		runTest       func([]*RcloneNode)
	}{
		{
			name: "test multiple remotes with same name",
			giveNewFsFunc: func(ctx context.Context, name string) (fs.Fs, error) {
				mfs := mockfs.NewFs(ctx, name, "")
				mfs.AddObject(mockobject.New("test-dir"))
				mfs.AddObject(mockobject.New("test-other-dir"))
				return mfs, nil
			},
			givePathStrs: []string{
				"rclone:/test-dir",
				"rclone:/test-other-dir",
			},
			runTest: func(nodes []*RcloneNode) {
				nodes[0].FileLocator()
				nodes[1].FileLocator()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(tst *testing.T) {
			for k := range vfsCache {
				delete(vfsCache, k)
			}

			if tt.giveNewFsFunc != nil {
				oldNewFsFunc := newFsFunc
				newFsFunc = tt.giveNewFsFunc
				defer func() {
					newFsFunc = oldNewFsFunc
				}()
			}

			var nodes []*RcloneNode
			var nodesLock sync.Mutex
			var pathStrWg sync.WaitGroup
			pathStrWg.Add(len(tt.givePathStrs))

			for _, pathStr := range tt.givePathStrs {
				go func(p string) {
					defer pathStrWg.Done()

					node, err := RcloneNodeFromPath(p)
					require.NoError(tst, err)

					nodesLock.Lock()
					nodes = append(nodes, node)
					nodesLock.Unlock()
				}(pathStr)
			}

			pathStrWg.Wait()
			tt.runTest(nodes)
		})
	}
}
