package main

import (
	"batchApprover/bitbucket"
	"fmt"
	"os"
)

const v = "1.0.0"

func main() {
	argsWithoutProg := os.Args[1:]

	if len(argsWithoutProg) == 0 {
		fmt.Println("No arguments provided. Use --help to see available commands")
		return
	}

	if argsWithoutProg[0] == "--help" || argsWithoutProg[0] == "-h" {
		printHelp()
		return
	}

	if argsWithoutProg[0] == "--version" || argsWithoutProg[0] == "-v" {
		fmt.Printf("Version %s\n", v)
		return
	}

	if argsWithoutProg[0] == "--approve-pr" || argsWithoutProg[0] == "-ap" {
		if len(argsWithoutProg) > 1 {
			fmt.Println("Approving pull request...")
			err := bitbucket.Approve(argsWithoutProg[1])
			if err != nil {
				fmt.Println("Error approving pull request:", err)
				return
			}
			fmt.Println("Pull request approved successfully")
		}
	}
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  -h: Show this help message")
	fmt.Println("  --help: Show this help message")
	fmt.Println("  -v: Show version information")
	fmt.Println("  --version: Show version information")
	fmt.Println("  -ap: Approve a pull request or a batch of them")
	fmt.Println("  --approve-pr: Approve a pull request or a batch of them")
}
