package helpers

import (
	"fmt"
	"strconv"
	"syscall"
)

// GetXattrInts reads multiple extended attribute values from a filename,
// one for each extended attribute name passed in the input array, and
// returns the values in a map so they can be easily correlated with their names.
func GetXattrInts(fileName string, xattrNames []string) (xattrMap map[string]int, err error) {
	xattrMap = map[string]int{}
	for _, xattrName := range xattrNames {
		sz, err := syscall.Getxattr(fileName, xattrName, nil)
		if err != nil {
			return nil, fmt.Errorf("couldn't access xattr %s", xattrName)
		}

		// Arbitrary limit
		if sz > 32 {
			sz = 32
		}

		dest := make([]byte, sz)
		_, err = syscall.Getxattr(fileName, xattrName, dest)
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(string(dest))
		if err != nil {
			return nil, err
		}
		xattrMap[xattrName] = i
	}
	return xattrMap, nil
}
