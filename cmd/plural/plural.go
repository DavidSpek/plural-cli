package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pluralsh/plural/pkg/api"
	"github.com/pluralsh/plural/pkg/config"
	"github.com/pluralsh/plural/pkg/crypto"
	"github.com/pluralsh/plural/pkg/kubernetes"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

func init() {
	cli.BashCompletionFlag = cli.BoolFlag{Name: "compgen", Hidden: true}
}

const ApplicationName = "plural"

type Plural struct {
	api.Client
	kubernetes.Kube
}

func (p *Plural) InitKube() error {
	if p.Kube == nil {
		kube, err := kubernetes.Kubernetes()
		if err != nil {
			return err
		}
		p.Kube = kube
	}
	return nil
}

func (p *Plural) InitPluralClient() {
	if p.Client == nil {
		p.Client = api.NewClient()
	}
}

func (p *Plural) getCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "version",
			Aliases: []string{"v", "vsn"},
			Usage:   "Gets cli version info",
			Action:  versionInfo,
		},
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "builds your workspace",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "only",
					Usage: "repository to (re)build",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "force workspace to build even if remote is out of sync",
				},
			},
			Action: tracked(owned(p.build), "cli.build"),
		},
		{
			Name:      "deploy",
			Aliases:   []string{"d"},
			Usage:     "Deploys the current workspace. This command will first sniff out git diffs in workspaces, topsort them, then apply all changes.",
			ArgsUsage: "WKSPACE",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "silence",
					Usage: "don't display notes for deployed apps",
				},
				cli.BoolFlag{
					Name:  "verbose",
					Usage: "show all command output during execution",
				},
				cli.BoolFlag{
					Name:  "ignore-console",
					Usage: "don't deploy the plural console",
				},
				cli.BoolFlag{
					Name:  "all",
					Usage: "deploy all repos irregardless of changes",
				},
				cli.StringFlag{
					Name:  "commit",
					Usage: "commits your changes with this message",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "use force push when pushing to git",
				},
			},
			Action: tracked(owned(rooted(p.deploy)), "cli.deploy"),
		},
		{
			Name:      "diff",
			Aliases:   []string{"df"},
			Usage:     "diffs the state of the current workspace with the deployed version and dumps results to diffs/",
			ArgsUsage: "WKSPACE",
			Action:    handleDiff,
		},
		{
			Name:     "create",
			Usage:    "scaffolds the resources needed to create a new plural repository",
			Action:   handleScaffold,
			Category: "WKSPACE",
		},
		{
			Name:      "watch",
			Usage:     "watches applications until they become ready",
			ArgsUsage: "REPO",
			Action:    requireArgs(handleWatch, []string{"REPO"}),
			Category:  "Debugging",
		},
		{
			Name:      "wait",
			Usage:     "waits on applications until they become ready",
			ArgsUsage: "REPO",
			Action:    requireArgs(handleWait, []string{"REPO"}),
			Category:  "Debugging",
		},
		{
			Name:      "info",
			Usage:     "generates a console dashboard for the namespace of this repo",
			ArgsUsage: "REPO",
			Action:    requireArgs(handleInfo, []string{"REPO"}),
			Category:  "Debugging",
		},
		{
			Name:  "apply",
			Usage: "applys the current pluralfile",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Usage: "pluralfile to use",
				},
			},
			Action:   apply,
			Category: "Publishing",
		},
		{
			Name:    "validate",
			Aliases: []string{"v"},
			Usage:   "validates your workspace",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "only",
					Usage: "repository to (re)build",
				},
			},
			Action:   p.validate,
			Category: "Workspace",
		},
		{
			Name:     "topsort",
			Aliases:  []string{"d"},
			Usage:    "renders a dependency-inferred topological sort of the installations in a workspace",
			Action:   p.topsort,
			Category: "Workspace",
		},
		{
			Name:      "bounce",
			Aliases:   []string{"b"},
			Usage:     "redeploys the charts in a workspace",
			ArgsUsage: "WKSPACE",
			Action:    owned(p.bounce),
		},
		{
			Name:      "destroy",
			Aliases:   []string{"b"},
			Usage:     "iterates through all installations in reverse topological order, deleting helm installations and terraform",
			ArgsUsage: "WKSPACE",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "from",
					Usage: "where to start your deploy command (useful when restarting interrupted destroys)",
				},
				cli.StringFlag{
					Name:  "commit",
					Usage: "commits your changes with this message",
				},
				cli.BoolFlag{
					Name:  "force",
					Usage: "use force push when pushing to git",
				},
			},
			Action: tracked(owned(p.destroy), "cli.destroy"),
		},
		{
			Name:  "init",
			Usage: "initializes plural within a git repo",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "the endpoint for the plural installation you're working with",
				},
				cli.StringFlag{
					Name:  "service-account",
					Usage: "email for the service account you'd like to use for this workspace",
				},
			},
			Action: tracked(handleInit, "cli.init"),
		},
		{
			Name:   "preflights",
			Usage:  "runs provider preflight checks",
			Action: preflights,
		},
		{
			Name:   "login",
			Usage:  "logs into plural and saves credentials to the current config profile",
			Action: handleLogin,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "endpoint",
					Usage: "the endpoint for the plural installation you're working with",
				},
				cli.StringFlag{
					Name:  "service-account",
					Usage: "email for the service account you'd like to use for this workspace",
				},
			},
			Category: "User Profile",
		},
		{
			Name:     "import",
			Usage:    "imports plural config from another file",
			Action:   handleImport,
			Category: "User Profile",
		},
		{
			Name:     "repair",
			Usage:    "commits any new encrypted changes in your local workspace automatically",
			Action:   handleRepair,
			Category: "WORKSPACE",
		},
		{
			Name:     "serve",
			Usage:    "launch the server",
			Action:   handleServe,
			Category: "Workspace",
		},
		{
			Name:        "shell",
			Usage:       "manages your cloud shell",
			Subcommands: shellCommands(),
			Category:    "Workspace",
		},
		{
			Name:        "repos",
			Usage:       "view and manage plural repositories",
			Subcommands: p.reposCommands(),
			Category:    "API",
		},
		{
			Name:     "test",
			Usage:    "validate a values templace",
			Action:   testTemplate,
			Category: "Publishing",
		},
		{
			Name:        "proxy",
			Usage:       "proxies into running processes in your cluster",
			Subcommands: p.proxyCommands(),
			Category:    "Debugging",
		},
		{
			Name:        "crypto",
			Usage:       "forge encryption utilities",
			Subcommands: p.cryptoCommands(),
			Category:    "User Profile",
		},
		{
			Name:        "push",
			Usage:       "utilities for pushing tf or helm packages",
			Subcommands: p.pushCommands(),
			Category:    "Publishing",
		},
		{
			Name:        "api",
			Usage:       "inspect the forge api",
			Subcommands: p.apiCommands(),
			Category:    "API",
		},
		{
			Name:        "config",
			Aliases:     []string{"conf"},
			Usage:       "reads/modifies cli configuration",
			Subcommands: configCommands(),
			Category:    "User Profile",
		},
		{
			Name:        "workspace",
			Aliases:     []string{"wkspace"},
			Usage:       "Commands for managing installations in your workspace",
			Subcommands: workspaceCommands(),
			Category:    "Workspace",
		},
		{
			Name:        "profile",
			Usage:       "Commands for managing config profiles for plural",
			Subcommands: profileCommands(),
			Category:    "User Profile",
		},
		{
			Name:        "output",
			Usage:       "Commands for generating outputs from supported tools",
			Subcommands: outputCommands(),
			Category:    "Workspace",
		},
		{
			Name:        "logs",
			Usage:       "Commands for tailing logs for specific apps",
			Subcommands: p.logsCommands(),
			Category:    "Debugging",
		},
		{
			Name:        "bundle",
			Usage:       "Commands for installing and discovering installation bundles",
			Subcommands: p.bundleCommands(),
		},
		{
			Name:        "stack",
			Usage:       "Commands for installing and discovering plural stacks",
			Subcommands: p.stackCommands(),
		},
		{
			Name:        "packages",
			Usage:       "Commands for managing your installed packages",
			Subcommands: p.packagesCommands(),
		},
		{
			Name:        "ops",
			Usage:       "Commands for simplifying cluster operations",
			Subcommands: p.opsCommands(),
			Category:    "Debugging",
		},
		{
			Name:        "utils",
			Usage:       "useful plural utilities",
			Subcommands: utilsCommands(),
			Category:    "Miscellaneous",
		},
		{
			Name:    "template",
			Aliases: []string{"tpl"},
			Usage:   "templates a helm chart to be uploaded to plural",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "values",
					Usage: "the values file",
				},
			},
			Action:   handleHelmTemplate,
			Category: "Publishing",
		},
		{
			Name:     "build-context",
			Usage:    "creates a fresh context.yaml for legacy repos",
			Action:   p.buildContext,
			Category: "Workspace",
		},
		{
			Name:     "changed",
			Usage:    "shows repos with pending changes",
			Action:   diffed,
			Category: "Workspace",
		},
		{
			Name:     "from-grafana",
			Usage:    "imports a grafana dashboard to a plural crd",
			Action:   formatDashboard,
			Category: "Publishing",
		},
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	// init Kube when k8s config exists
	plural := &Plural{}
	app := CreateNewApp(plural)
	if os.Getenv("ENABLE_COLOR") != "" {
		color.NoColor = false
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "profile-file",
			Usage:       "configure your config.yml profile `FILE`",
			EnvVar:      "PLURAL_PROFILE_FILE",
			Destination: &config.ProfileFile,
		},
		cli.StringFlag{
			Name:        "encryption-key-file",
			Usage:       "configure your encryption key `FILE`",
			EnvVar:      "PLURAL_ENCRYPTION_KEY_FILE",
			Destination: &crypto.EncryptionKeyFile,
		},
	}
}

func CreateNewApp(plural *Plural) *cli.App {
	app := cli.NewApp()
	app.Name = ApplicationName
	app.Usage = "Tooling to manage your installed plural applications"
	app.EnableBashCompletion = true
	app.Flags = globalFlags()
	app.Commands = plural.getCommands()
	links := linkCommands()
	app.Commands = append(app.Commands, links...)

	return app
}
