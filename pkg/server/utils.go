package server

import (
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func serverFunc(f func(c *gin.Context) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := f(c); err != nil {
			_ = c.Error(err)
		}
	}
}

func toProvider(prov string) string {
	prov = strings.ToLower(prov)
	if prov == "gcp" {
		return "google"
	}
	return prov
}

func execCmd(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
