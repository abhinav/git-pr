package main

import "github.com/abhinav/git-fu/cli"

func main() {
	cli.Main(
		cli.ShortDesc("Interact with GitHub Pull Requests from the command line."),
		&cli.Command{
			Name:      "land",
			ShortDesc: "Lands a GitHub PR.",
			Build:     newLandCommand,
		},
		&cli.Command{
			Name:      "rebase",
			ShortDesc: "Rebases a PR branch.",
			Build:     newRebaseCommand,
		},
	)
}
