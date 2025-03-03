package pluralfile

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pluralsh/plural/pkg/api"
	"github.com/pluralsh/plural/pkg/executor"
	"github.com/pluralsh/plural/pkg/utils"
)

type Stack struct {
	File string
}

func (a *Stack) Type() ComponentName {
	return STACK
}

func (a *Stack) Key() string {
	return a.File
}

func (a *Stack) Push(repo string, sha string) (string, error) {
	newsha, err := executor.MkHash(a.File, []string{})
	if err != nil || newsha == sha {
		utils.Highlight("No change for %s\n", a.File)
		return sha, err
	}

	fullPath, _ := filepath.Abs(a.File)
	contents, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	utils.Highlight("pushing stack %s", a.File)
	client := api.NewClient()
	attrs, err := api.ConstructStack(contents)
	if err != nil {
		return "", err
	}

	_, err = client.CreateStack(attrs)
	if err == nil {
		utils.Success("\u2713\n")
	}

	return newsha, err
}
