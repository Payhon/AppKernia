package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	platformconfig "github.com/appkernia/appkernia/server/internal/platform/config"
)

func configCommand(program string, args []string) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintf(os.Stdout, "usage: %s config init|validate|show\n", program)
		return nil
	}
	if len(args) == 0 {
		return &UsageError{Message: fmt.Sprintf("usage: %s config init|validate|show", program)}
	}
	switch args[0] {
	case "init":
		flags := flag.NewFlagSet("config init", flag.ContinueOnError)
		output := flags.String("output", "", "configuration file path")
		force := flags.Bool("force", false, "replace an existing configuration file")
		if err := parseCommandFlags(flags, args[1:], fmt.Sprintf("usage: %s config init [--output FILE] [--force]", program)); err != nil {
			return err
		}
		path := strings.TrimSpace(*output)
		if path == "" {
			path = strings.TrimSpace(os.Getenv("AK_CONFIG_FILE"))
		}
		written, err := platformconfig.WriteDefault(path, *force)
		if err != nil {
			return err
		}
		fmt.Printf("configuration created path=%s\n", written)
		return nil
	case "validate", "show":
		flags := flag.NewFlagSet("config "+args[0], flag.ContinueOnError)
		path := flags.String("file", "", "configuration file path")
		if err := parseCommandFlags(flags, args[1:], fmt.Sprintf("usage: %s config %s [--file FILE]", program, args[0])); err != nil {
			return err
		}
		cfg, err := platformconfig.Load(platformconfig.Options{Path: strings.TrimSpace(*path)})
		if err != nil {
			return err
		}
		if args[0] == "validate" {
			path := cfg.FilePath
			if path == "" {
				path = "defaults"
			}
			fmt.Printf("configuration valid source=%s\n", path)
			return nil
		}
		content, err := platformconfig.RedactedYAML(cfg)
		if err != nil {
			return fmt.Errorf("render configuration: %w", err)
		}
		_, err = os.Stdout.Write(content)
		return err
	default:
		return &UsageError{Message: fmt.Sprintf("usage: %s config init|validate|show", program)}
	}
}
