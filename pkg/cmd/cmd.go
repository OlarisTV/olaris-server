package cmd

import (
	"github.com/spf13/cobra"
)

type CobraCommand struct {
	Command *cobra.Command
}

func (c *CobraCommand) GetCobraCommand() *cobra.Command {
	return c.Command
}

type Command interface {
	GetCobraCommand() *cobra.Command
}

var (
	_ Command = (*CobraCommand)(nil)
)
