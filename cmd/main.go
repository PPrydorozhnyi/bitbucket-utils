package main

import (
	"batchApprover/bitbucket"
	"fmt"
	"io"
	"os"
	"strings"
)

const version = "1.0.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type command string

const (
	commandHelp    command = "help"
	commandVersion command = "version"
	commandApprove command = "approve"
)

type cliOptions struct {
	Command command
	URL     string
	DryRun  bool
}

func run(args []string, output, errorOutput io.Writer) int {
	options, err := parseCLI(args)
	if err != nil {
		fmt.Fprintf(errorOutput, "Error: %v\n", err)
		return 2
	}

	switch options.Command {
	case commandHelp:
		printHelp(output)
		return 0
	case commandVersion:
		fmt.Fprintf(output, "Version %s\n", version)
		return 0
	case commandApprove:
		_, err := bitbucket.ApproveWithOptions(options.URL, bitbucket.ApprovalOptions{
			DryRun: options.DryRun,
			Output: output,
		})
		if err != nil {
			fmt.Fprintf(errorOutput, "Error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(errorOutput, "Error: unsupported command %q\n", options.Command)
		return 2
	}
}

func parseCLI(args []string) (cliOptions, error) {
	if len(args) == 0 {
		return cliOptions{}, fmt.Errorf("no command provided; use --help for usage")
	}

	switch args[0] {
	case "--help", "-h":
		if len(args) != 1 {
			return cliOptions{}, fmt.Errorf("%s does not accept arguments", args[0])
		}
		return cliOptions{Command: commandHelp}, nil
	case "--version", "-v":
		if len(args) != 1 {
			return cliOptions{}, fmt.Errorf("%s does not accept arguments", args[0])
		}
		return cliOptions{Command: commandVersion}, nil
	case "--approve-pr", "-ap":
		options := cliOptions{Command: commandApprove}
		for _, arg := range args[1:] {
			switch {
			case arg == "--dry-run":
				if options.DryRun {
					return cliOptions{}, fmt.Errorf("--dry-run must appear at most once")
				}
				options.DryRun = true
			case strings.HasPrefix(arg, "-"):
				return cliOptions{}, fmt.Errorf("unknown option %q", arg)
			case options.URL == "":
				options.URL = arg
			default:
				return cliOptions{}, fmt.Errorf("unexpected argument %q", arg)
			}
		}
		if options.URL == "" {
			return cliOptions{}, fmt.Errorf("%s requires a Bitbucket URL", args[0])
		}
		return options, nil
	default:
		return cliOptions{}, fmt.Errorf("unknown command %q; use --help for usage", args[0])
	}
}

func printHelp(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  console-utils --approve-pr <bitbucket-url> [--dry-run]")
	fmt.Fprintln(output, "  console-utils -ap <bitbucket-url> [--dry-run]")
	fmt.Fprintln(output, "  console-utils --help")
	fmt.Fprintln(output, "  console-utils --version")
	fmt.Fprintln(output)
	fmt.Fprintln(output, "Authentication:")
	fmt.Fprintln(output, "  BITBUCKET_EMAIL")
	fmt.Fprintln(output, "  BITBUCKET_API_TOKEN")
}
