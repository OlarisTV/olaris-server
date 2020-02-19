package helpers

import (
	"fmt"
	"strconv"
	"syscall"
)

func GetXattrInts(FileName string, XattrNames []string) (XattrMap map[string]int, err error) {
	XattrMap = make(map[string]int)
	for _, XattrName := range XattrNames {
		sz, err := syscall.Getxattr(FileName, XattrName, nil)
		if err != nil {
			return nil, fmt.Errorf("couldn't access xattr", XattrName)
		}

		// Arbitrary limit
		if sz > 32 {
			sz = 32
		}

		dest := make([]byte, sz)
		_, err = syscall.Getxattr(FileName, XattrName, dest)
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(string(dest))
		if err != nil {
			return nil, err
		}
		XattrMap[XattrName] = i
	}
	return XattrMap, nil
}
