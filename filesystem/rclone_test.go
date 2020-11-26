package filesystem

import (
	"sync"
	"testing"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fstest/mockfs"
	"github.com/rclone/rclone/fstest/mockobject"
	"github.com/stretchr/testify/require"
)

func TestRcloneNodeFromPath(t *testing.T) {
	tests := []struct {
		name          string
		giveNewFsFunc func(string) (fs.Fs, error)
		givePathStrs  []string
		runTest       func([]*RcloneNode)
	}{
		{
			name: "test multiple remotes with same name",
			giveNewFsFunc: func(name string) (fs.Fs, error) {
				mfs := mockfs.NewFs(name, "")
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
