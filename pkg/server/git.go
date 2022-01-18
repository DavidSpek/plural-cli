package server

import (
	"os"
	"io/ioutil"
	"path/filepath"
	homedir "github.com/mitchellh/go-homedir"
)

func setupGit(setup *SetupRequest) error {
	p, err := homedir.Expand("~/.ssh")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(p, 0700); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(p, "id_rsa"), []byte(setup.SshPrivateKey), 0600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(p, "id_rsa.pub"), []byte(setup.SshPublicKey), 0644); err != nil {
		return err
	}

	if err := execCmd("ssh-add", filepath.Join(p, "id_rsa")); err != nil {
		return err
	}

	dir, err := homedir.Expand("~/workspace")
	if err != nil {
		return err
	}

	if err := execCmd("git", "clone", setup.GitUrl, dir); err != nil {
		return err
	}

	os.Chdir(dir)
	if err := execCmd("plural", "crypto", "init"); err != nil {
		return err
	}

	return execCmd("plural", "crypto", "unlock")
}