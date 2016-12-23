package cli

import (
	"log"

	"github.com/jessevdk/go-flags"
)

type mainConfig struct {
	ShortDesc string
	Commands  []*Command
}

// Main is the entry point for programs provided by this package.
func Main(opts ...Option) {
	log.SetFlags(0)

	var cfg mainConfig
	for _, o := range opts {
		o.apply(&cfg)
	}

	var gcfg globalConfig
	parser := flags.NewParser(&gcfg, flags.HelpFlag|flags.PassDoubleDash)
	for _, cmd := range cfg.Commands {
		_, err := parser.AddCommand(
			cmd.Name, cmd.ShortDesc, "", cmd.Build(gcfg.Build))
		if err != nil {
			log.Fatalf("Could not register command %q: %v", cmd.Name, err)
		}
	}

	if _, err := parser.Parse(); err != nil {
		log.Fatal(err)
	}
}
