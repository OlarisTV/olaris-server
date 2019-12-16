package cmd

import (
	"archive/zip"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/helpers"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var dumpdebugCmd = &cobra.Command{
	Use:   "dumpdebug",
	Short: "Dump all data for debugging purposes",
	Run: func(cmd *cobra.Command, args []string) {

		filename := fmt.Sprintf("olaris-dumpdebug-%s.zip",
			time.Now().Format("2006-01-02-15-04-05"))
		f, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Failed to open file: %s", err)
		}
		w := zip.NewWriter(f)

		writeFilesInDir(w, helpers.LogPath(), "log/")

		fw, _ := w.Create("metadata.db.sqlite")
		// TODO(Leon Handreke): Don't hardcode-copypaste this path from metadata/db/database.go
		content, _ := ioutil.ReadFile(filepath.Join(helpers.MetadataConfigPath(), "metadata.db"))
		fw.Write(content)

		err = w.Close()
		if err != nil {
			log.Fatalf("Failed to open file: %s", err)
		}
	},
}

func writeFilesInDir(w *zip.Writer, dir string, prefix string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fw, err := w.Create(prefix + info.Name())
			if err != nil {
				return errors.Errorf("Failed to write file in archive: %s", err)
			}

			content, _ := ioutil.ReadFile(path)
			_, err = fw.Write(content)
			if err != nil {
				return errors.Errorf("Failed to write file in archive: %s", err)
			}
		}
		return nil
	})
}

func init() {
	rootCmd.AddCommand(dumpdebugCmd)
}
