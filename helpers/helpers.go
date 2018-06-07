package helpers

import (
	"fmt"
	"os"
	"os/user"
)

func GetHome() string {
	usr, err := user.Current()
	if err != nil {
		panic(fmt.Sprintf("Failed to determine user's home directory, error: '%s'\n", err.Error()))
	}
	return usr.HomeDir
}

func EnsurePath(pathName string) error {
	fmt.Printf("Ensuring folder %s exists.\n", pathName)
	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		fmt.Println("Path does not exist, creating", pathName)
		err = os.MkdirAll(pathName, 0755)
		if err != nil {
			fmt.Println("Could not create path.")
			return err
		}
	}
	return nil
}

func FileExists(pathName string) bool {
	fmt.Println("Checking if path", pathName, "exists")
	if _, err := os.Stat(pathName); err == nil {
		return true
	} else {
		return false
	}
}
