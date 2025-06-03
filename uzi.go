// Copyright (c) Subtrace, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devflowinc/uzi/cmd/broadcast"
	"github.com/devflowinc/uzi/cmd/checkpoint"
	"github.com/devflowinc/uzi/cmd/kill"
	"github.com/devflowinc/uzi/cmd/ls"
	"github.com/devflowinc/uzi/cmd/prompt"
	"github.com/devflowinc/uzi/cmd/reset"
	"github.com/devflowinc/uzi/cmd/run"
	"github.com/devflowinc/uzi/cmd/watch"
	"github.com/devflowinc/uzi/cmd/web"

	"github.com/peterbourgon/ff/v3/ffcli"
)

var subcommands = []*ffcli.Command{
	prompt.CmdPrompt,
	ls.CmdLs,
	kill.CmdKill,
	reset.CmdReset,
	run.CmdRun,
	checkpoint.CmdCheckpoint,
	watch.CmdWatch,
	broadcast.CmdBroadcast,
	web.CmdWeb,
}

var commandAliases = map[string]*regexp.Regexp{
	"prompt":     regexp.MustCompile(`^p(ro(mpt)?)?$`),
	"ls":         regexp.MustCompile(`^l(s)?$`),
	"kill":       regexp.MustCompile(`^k(ill)?$`),
	"reset":      regexp.MustCompile(`^re(set)?$`),
	"checkpoint": regexp.MustCompile(`^c(heckpoint)?$`),
	"run":        regexp.MustCompile(`^r(un)?$`),
	"watch":      regexp.MustCompile(`^w(atch)?$`),
	"broadcast":  regexp.MustCompile(`^b(roadcast)?$`),
	"attach":     regexp.MustCompile(`^a(ttach)?$`),
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := new(ffcli.Command)
	c.Name = filepath.Base(os.Args[0])
	c.ShortUsage = "uzi <command>"
	c.Subcommands = subcommands

	c.FlagSet = flag.NewFlagSet("uzi", flag.ContinueOnError)
	c.FlagSet.SetOutput(os.Stdout)
	c.Exec = func(ctx context.Context, args []string) error {
		fmt.Fprintf(os.Stdout, "%s\n", c.UsageFunc(c))

		if len(os.Args) >= 2 {
			return fmt.Errorf("unknown command %q", os.Args[1])
		}
		return nil
	}

	// Resolve command aliases before parsing
	args := os.Args[1:]
	if len(args) > 0 {
		cmdName := args[0]
		for realCmd, pattern := range commandAliases {
			if pattern.MatchString(cmdName) {
				args[0] = realCmd
				break
			}
		}
	}

	switch err := c.Parse(args); {
	case err == nil:
	case errors.Is(err, flag.ErrHelp):
		return
	case strings.Contains(err.Error(), "flag provided but not defined"):
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "uzi: error: %v\n", err)
		os.Exit(1)
	}

	if err := c.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "uzi: error: %v\n", err)
		os.Exit(1)
	}
}
