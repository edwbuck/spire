package cli

import (
	"context"
	"flag"
	"io"
	stdlog "log"
	"os"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/cli"
	"github.com/spiffe/spire/cmd/spire-agent/cli/api"
	"github.com/spiffe/spire/cmd/spire-agent/cli/healthcheck"
	"github.com/spiffe/spire/cmd/spire-agent/cli/logger"
	"github.com/spiffe/spire/cmd/spire-agent/cli/run"
	"github.com/spiffe/spire/cmd/spire-agent/cli/validate"
	"github.com/spiffe/spire/pkg/common/log"
	"github.com/spiffe/spire/pkg/common/version"
)

const (
	defaultConfigPath = "conf/agent/agent.conf"
)

type Config struct {
	Agent *agentConfig `hcl:"agent"`
}

type agentConfig struct {
	AdminSocketPath string `hcl:"admin_socket_path"`
}

type CLI struct {
	LogOptions         []log.Option
	AllowUnknownConfig bool
}

func ConfigFilePath(args []string) string {
	for len(args) > 0 && !strings.HasSuffix(args[0], "-config") {
		args = args[1:]
	}

	var configPath string
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&configPath, "config", defaultConfigPath, "Path to a SPIRE config file")
	flags.SetOutput(io.Discard)
	flags.Parse(args)
	return configPath
}

func ExpandEnv(args []string) bool {
	for len(args) > 0 && !strings.HasSuffix(args[0], "-expandEnv") {
		args = args[1:]
	}

	var expandEnv bool
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.BoolVar(&expandEnv, "expandEnv", false, "Expand environment variable in SPIRE config file")
	flags.SetOutput(io.Discard)
	flags.Parse(args)
	return expandEnv
}

func AdminEnabled(path string, expandEnv bool) bool {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	config := string(bytes)
	if expandEnv {
		config = os.ExpandEnv(config)
	}
	c := Config{}
	err = hcl.Decode(&c, config)
	if err != nil {
		return false
	}
	return c.Agent.AdminSocketPath != ""
}

func (cc *CLI) Run(ctx context.Context, args []string) int {
	configPath := ConfigFilePath(args)
	expandEnv := ExpandEnv(args)
	adminEnabled := AdminEnabled(configPath, expandEnv)

	c := cli.NewCLI("spire-agent", version.Version())
	c.Args = args

	c.Commands = map[string]cli.CommandFactory{
		"api fetch": func() (cli.Command, error) {
			return api.NewFetchX509Command(), nil
		},
		"api fetch x509": func() (cli.Command, error) {
			return api.NewFetchX509Command(), nil
		},
		"api fetch jwt": func() (cli.Command, error) {
			return api.NewFetchJWTCommand(), nil
		},
		"api validate jwt": func() (cli.Command, error) {
			return api.NewValidateJWTCommand(), nil
		},
		"api watch": func() (cli.Command, error) {
			return &api.WatchCLI{}, nil
		},
		"run": func() (cli.Command, error) {
			return run.NewRunCommand(ctx, cc.LogOptions, cc.AllowUnknownConfig), nil
		},
		"healthcheck": func() (cli.Command, error) {
			return healthcheck.NewHealthCheckCommand(), nil
		},
		"validate": func() (cli.Command, error) {
			return validate.NewValidateCommand(), nil
		},
	}
	if adminEnabled {
		c.Commands["logger get"] = func() (cli.Command, error) {
			return logger.NewGetCommand(), nil
		}
		c.Commands["logger set"] = func() (cli.Command, error) {
			return validate.NewValidateCommand(), nil
		}
		c.Commands["logger reset"] = func() (cli.Command, error) {
			return validate.NewValidateCommand(), nil
		}
	}

	exitStatus, err := c.Run()
	if err != nil {
		stdlog.Println(err)
	}
	return exitStatus
}
