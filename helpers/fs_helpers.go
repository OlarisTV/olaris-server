package helpers

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"os/user"
	"path"
)

func GetHome() string {
	usr, err := user.Current()
	if err != nil {
		panic(fmt.Sprintf("Failed to determine user's home directory, error: '%s'\n", err.Error()))
	}
	return usr.HomeDir
}

func EnsurePath(pathName string) error {
	glog.Infof("Ensuring folder %s exists.\n", pathName)
	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		glog.Infof("Path %s does not exist, creating", pathName)
		err = os.MkdirAll(pathName, 0755)
		if err != nil {
			glog.Errorf("Could not create %s", pathName)
			return err
		}
	}
	return nil
}

func FileExists(pathName string) bool {
	_, err := os.Stat(pathName)
	return err == nil
}

func BaseConfigPath() string {
	return path.Join(GetHome(), ".config", "olaris")
}

func MetadataConfigPath() string {
	return path.Join(BaseConfigPath(), "metadb")
}
