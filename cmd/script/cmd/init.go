package cmd

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/scripttoken/script/common"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Script node configuration.",
	Long:  ``,
	Run:   runInit,
}

func init() {
	RootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) {
fmt.Println("Init cfgPath=", cfgPath)

	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		log.WithFields(log.Fields{"err": err, "path": cfgPath}).Fatal("Directory already exists!")
	}

	if err := os.Mkdir(cfgPath, 0700); err != nil {
		log.WithFields(log.Fields{"err": err, "path": cfgPath}).Fatal("Failed to create config directory")
	}

	if err := common.WriteInitialConfig(path.Join(cfgPath, "config.yaml")); err != nil {
		log.WithFields(log.Fields{"err": err, "path": cfgPath}).Fatal("Failed to write config")
	}
}
