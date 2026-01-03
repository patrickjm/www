package app

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

var Version = "dev"

func Execute(args []string, out io.Writer, errOut io.Writer) int {
	app := App{Out: out, Err: errOut}
	flags := GlobalFlags{}
	var showVersion bool

	root := &cobra.Command{
		Use:           "www",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.SetOut(out)
	root.SetErr(errOut)

	root.PersistentFlags().BoolVarP(&showVersion, "version", "V", false, "version")
	root.PersistentFlags().StringVarP(&flags.Profile, "profile", "p", "", "profile name")
	root.PersistentFlags().StringVarP(&flags.ProfileDir, "profile-dir", "D", "", "profile directory")
	root.PersistentFlags().BoolVarP(&flags.JSON, "json", "j", false, "json output")
	root.PersistentFlags().BoolVarP(&flags.Plain, "plain", "P", false, "plain output")
	root.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "quiet output")
	root.PersistentFlags().BoolVarP(&flags.Verbose, "verbose", "v", false, "verbose output")
	root.PersistentFlags().BoolVarP(&flags.NoStart, "no-start", "N", false, "do not auto-start")
	root.PersistentFlags().BoolVarP(&flags.Save, "save", "s", false, "persist overrides to profile")
	root.PersistentFlags().StringVarP(&flags.Browser, "browser", "b", "", "browser type")
	root.PersistentFlags().StringVarP(&flags.Channel, "channel", "c", "", "browser channel")
	root.PersistentFlags().BoolVarP(&flags.Headless, "headless", "H", false, "run headless")
	root.PersistentFlags().BoolVarP(&flags.Headed, "headed", "E", false, "run headed")
	root.PersistentFlags().IntVarP(&flags.Tab, "tab", "T", 0, "tab id")
	root.PersistentFlags().StringVarP(&flags.TTL, "ttl", "L", "", "profile ttl")
	root.PersistentFlags().StringVarP(&flags.Selector, "selector", "S", "", "selector")
	root.PersistentFlags().BoolVarP(&flags.Main, "main", "m", false, "use main content")
	root.PersistentFlags().StringVarP(&flags.Timeout, "timeout", "t", "", "action timeout")

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if showVersion {
			fmt.Fprintln(out, Version)
			return exitError{code: exitSuccess}
		}
		return nil
	}

	root.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install Playwright driver and browsers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			code := app.runInstall(flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Check install and environment health",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, _, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runDoctor(cfg, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "start",
		Short: "Start a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runStart(store, mgr, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "stop",
		Short: "Stop a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runStop(mgr, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "ps",
		Short: "List running profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runPs(mgr, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, _, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runList(store, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "show NAME",
		Short: "Show a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, _, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runShow(store, flags, args)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "rm NAME...",
		Short: "Remove profiles",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runRemove(store, mgr, flags, args)
			return exitOrNil(code)
		},
	})

	pruneCmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove expired profiles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runPrune(store, mgr, flags, dryRun, force)
			return exitOrNil(code)
		},
	}
	pruneCmd.Flags().BoolP("dry-run", "n", false, "preview")
	pruneCmd.Flags().BoolP("force", "f", false, "force removal")
	root.AddCommand(pruneCmd)

	tabCmd := &cobra.Command{
		Use:   "tab",
		Short: "Manage tabs",
	}
	tabNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new tab",
		RunE: func(cmd *cobra.Command, _ []string) error {
			url, _ := cmd.Flags().GetString("url")
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runTabNew(store, mgr, flags, url)
			return exitOrNil(code)
		},
	}
	tabNewCmd.Flags().StringP("url", "u", "", "navigate url")
	tabCmd.AddCommand(tabNewCmd)

	tabCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List tabs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runTabList(store, mgr, flags)
			return exitOrNil(code)
		},
	})

	tabCmd.AddCommand(&cobra.Command{
		Use:   "close",
		Short: "Close a tab",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if flags.Tab == 0 {
				return exitError{code: exitUsage}
			}
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runTabClose(store, mgr, flags, flags.Tab)
			return exitOrNil(code)
		},
	})

	tabCmd.AddCommand(&cobra.Command{
		Use:   "switch",
		Short: "Switch active tab",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if flags.Tab == 0 {
				return exitError{code: exitUsage}
			}
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runTabSwitch(store, mgr, flags, flags.Tab)
			return exitOrNil(code)
		},
	})

	root.AddCommand(tabCmd)

	root.AddCommand(&cobra.Command{
		Use:   "goto URL",
		Short: "Navigate the active tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runGoto(store, mgr, flags, args[0])
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "click TEXT|SELECTOR",
		Short: "Click an element",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runClick(store, mgr, flags, args[0])
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "fill SELECTOR VALUE",
		Short: "Fill an input",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runFill(store, mgr, flags, args[0], args[1])
			return exitOrNil(code)
		},
	})

	shotCmd := &cobra.Command{
		Use:   "shot PATH",
		Short: "Take a screenshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fullPage, _ := cmd.Flags().GetBool("full-page")
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runShot(store, mgr, flags, args[0], fullPage, flags.Selector)
			return exitOrNil(code)
		},
	}
	shotCmd.Flags().BoolP("full-page", "F", false, "full page")
	root.AddCommand(shotCmd)

	root.AddCommand(&cobra.Command{
		Use:   "extract",
		Short: "Extract page info",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runExtract(store, mgr, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "read",
		Short: "Read main content",
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags.Main = true
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runRead(store, mgr, flags)
			return exitOrNil(code)
		},
	})

	root.AddCommand(&cobra.Command{
		Use:   "url",
		Short: "Print current tab URL",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runURL(store, mgr, flags)
			return exitOrNil(code)
		},
	})

	linksCmd := &cobra.Command{
		Use:   "links",
		Short: "List visible links",
		RunE: func(cmd *cobra.Command, _ []string) error {
			filter, _ := cmd.Flags().GetString("filter")
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runLinks(store, mgr, flags, filter)
			return exitOrNil(code)
		},
	}
	linksCmd.Flags().StringP("filter", "f", "", "filter")
	root.AddCommand(linksCmd)

	root.AddCommand(&cobra.Command{
		Use:   "eval JS",
		Short: "Evaluate JavaScript",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, store, mgr, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runEval(store, mgr, flags, strings.Join(args, " "))
			return exitOrNil(code)
		},
	})

	serveCmd := &cobra.Command{
		Use:    "serve",
		Short:  "Internal daemon entrypoint",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, store, _, err := app.prepare(flags)
			if err != nil {
				fmt.Fprintln(errOut, err)
				return exitError{code: exitFailure}
			}
			code := app.runServe(store, flags)
			return exitOrNil(code)
		},
	}
	root.AddCommand(serveCmd)

	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		var exit exitError
		if errors.As(err, &exit) {
			return exit.code
		}
		fmt.Fprintln(errOut, err)
		return exitUsage
	}
	return exitSuccess
}

func exitOrNil(code int) error {
	if code == exitSuccess {
		return nil
	}
	return exitError{code: code}
}
