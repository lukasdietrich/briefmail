// Copyright (C) 2020  Lukas Dietrich <lukas@lukasdietrich.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"github.com/ktr0731/go-fuzzyfinder"

	"github.com/lukasdietrich/briefmail/internal/storage"
)

// Shell is an interactive shell to manage briefmail domains, mailboxes and addresses.
type Shell struct {
	database *storage.Database
	commands cmdSlice
}

// NewShell creates a new shell instance.
func NewShell(database *storage.Database) *Shell {
	return &Shell{
		database: database,
		commands: cmdSlice{
			{
				name: "domain",
				help: "Manage the domains under control of briefmail.",
				children: cmdSlice{
					{
						name:   "add",
						help:   "Add a new domain.",
						action: addDomain,
					},
					{
						name:   "delete",
						help:   "Delete an existing domain.",
						action: deleteDomain,
					},
					{
						name:   "replace",
						help:   "Replace an existing domain. This also affects addresses.",
						action: replaceDomain,
					},
				},
			},
			{
				name: "mailbox",
				help: "Manage mailboxes for the people using this briefmail instance.",
				children: cmdSlice{
					{
						name:   "info",
						help:   "Show info for a mailbox.",
						action: infoMailbox,
					},
					{
						name:   "add",
						help:   "Add a new mailbox.",
						action: addMailbox,
					},
					{
						name:   "delete",
						help:   "Delete an existing mailbox.",
						action: deleteMailbox,
					},
				},
			},
			{
				name: "address",
				help: "Manage the addresses able to receive and send mails through briefmail.",
				children: cmdSlice{
					{
						name:   "add",
						help:   "Add a new address for an existing domain.",
						action: addAddress,
					},
					{
						name:   "delete",
						help:   "Delete an existing address.",
						action: deleteAddress,
					},
				},
			},
		},
	}
}

// Run starts the shell read loop.
func (s *Shell) Run() error {
	config := readline.Config{
		AutoComplete: readline.NewPrefixCompleter(s.commands.buildCompleters()...),
	}

	rl, err := readline.NewEx(&config)
	if err != nil {
		return err
	}

	defer rl.Close()

	for {
		rl.SetPrompt(">>> ")

		line, err := rl.Readline()
		if err != nil {
			if isUnimportantError(err) {
				return nil
			}

			return err
		}

		args := strings.Fields(line)
		if err := s.handleCommand(rl, args); err != nil && !isUnimportantError(err) {
			fmt.Printf("\nERROR:\n  %s\n\n", err)
		}
	}
}

func isUnimportantError(err error) bool {
	return errors.Is(err, fuzzyfinder.ErrAbort) ||
		errors.Is(err, readline.ErrInterrupt) ||
		errors.Is(err, io.EOF)
}

type cmdFunc func(*cmdContext) error

type cmdSlice []cmdDef

func (s cmdSlice) lookup(args []string) (cmdDef, bool) {
	if len(s) > 0 && len(args) > 0 {
		var (
			head = args[0]
			tail = args[1:]
		)

		for _, cmd := range s {
			if head == cmd.name {
				if len(tail) > 0 {
					return cmd.children.lookup(tail)
				}

				return cmd, true
			}
		}
	}

	return cmdDef{}, false
}

func (s cmdSlice) buildCompleters() []readline.PrefixCompleterInterface {
	var completers []readline.PrefixCompleterInterface

	for _, cmd := range s {
		cmdCompleter := readline.PcItem(cmd.name, cmd.children.buildCompleters()...)
		completers = append(completers, cmdCompleter)
	}

	return completers
}

type cmdDef struct {
	name     string
	help     string
	action   cmdFunc
	children cmdSlice
}

type cmdContext struct {
	context.Context
	rl        *readline.Instance
	tx        *storage.Tx
	infoLines []string
}

func (c *cmdContext) info(format string, v ...interface{}) {
	text := fmt.Sprintf(format, v...)
	c.infoLines = append(c.infoLines, text)
}

func (c *cmdContext) ask(prompt string) (string, error) {
	return c.askWithDefault(prompt, "")
}

func (c *cmdContext) askWithDefault(prompt, defaultValue string) (string, error) {
	c.rl.HistoryDisable()
	defer c.rl.HistoryEnable()

	c.rl.SetPrompt(prompt)

	for {
		answer, err := c.rl.ReadlineWithDefault(defaultValue)
		if err != nil || len(answer) > 0 {
			return answer, err
		}
	}
}

func (c *cmdContext) password(prompt string) ([]byte, error) {
	c.rl.HistoryDisable()
	defer c.rl.HistoryEnable()

	for {
		answer, err := c.rl.ReadPassword(prompt)
		if err != nil || len(answer) > 0 {
			return answer, err
		}
	}
}

func (s *Shell) handleCommand(rl *readline.Instance, args []string) error {
	cmd, ok := s.commands.lookup(args)
	if ok {
		if cmd.action != nil {
			return s.executeCommand(rl, cmd)
		}

		printCommandHelp(cmd)
	} else {
		printCommandUnknown(s.commands, args)
	}

	return nil
}

func (s *Shell) executeCommand(rl *readline.Instance, cmd cmdDef) error {
	ctx := context.Background()

	tx, err := s.database.BeginTx(ctx)
	if err != nil {
		return err
	}

	cmdCtx := cmdContext{
		Context: context.Background(),
		rl:      rl,
		tx:      tx,
	}

	if err := cmd.action(&cmdCtx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if len(cmdCtx.infoLines) > 0 {
		fmt.Println()

		for _, infoLine := range cmdCtx.infoLines {
			fmt.Print("  ")
			fmt.Println(infoLine)
		}

		fmt.Println()
	}

	return nil
}

func printCommandUnknown(cmds cmdSlice, args []string) {
	fmt.Printf("\n  Unknown command %q\n", strings.Join(args, " "))
	printCommandUsage(cmds)
}

func printCommandHelp(cmd cmdDef) {
	fmt.Printf("\n  %s\n", cmd.help)
	printCommandUsage(cmd.children)
}

func printCommandUsage(cmds cmdSlice) {
	if len(cmds) > 0 {
		fmt.Println()
		fmt.Println("Commands:")

		for _, cmd := range cmds {
			fmt.Printf("  %-10s  %s\n", cmd.name, cmd.help)
		}
	}

	fmt.Println()
}
