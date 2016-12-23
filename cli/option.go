package cli

import "github.com/jessevdk/go-flags"

// Option configures the behavior of Main.
type Option interface {
	apply(*mainConfig)
}

// ShortDesc is an Option that specifies a short description of the program.
type ShortDesc string

func (s ShortDesc) apply(c *mainConfig) {
	c.ShortDesc = string(s)
}

// Command is an option that adds a subcommand to the program.
type Command struct {
	Name      string
	ShortDesc string
	Build     func(ConfigBuilder) flags.Commander
}

func (cmd *Command) apply(c *mainConfig) {
	c.Commands = append(c.Commands, cmd)
}
