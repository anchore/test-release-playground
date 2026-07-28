// Package cli wires up the traceart command-line entrypoint.
package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/anchore/release-playground/internal/art"
)

// Identification carries build-time identity injected via -ldflags.
type Identification struct {
	Version        string
	GitCommit      string
	GitDescription string
	BuildDate      string
}

// App is the runnable traceart CLI.
type App struct {
	id Identification
}

// New constructs a new App from the provided build identification.
func New(id Identification) *App { return &App{id: id} }

const usage = `traceart — visualise a simulated packet's grand tour on an ASCII world map

Usage:
  traceart [flags]

Flags:
  --simulate           run an offline simulated traceroute (currently the only mode)
  --destination HOST   pick a known target (substring match) or supply a custom hostname
  --seed N             seed the simulation for reproducible output
  --width N            map width in columns (default: 104, min: 100)
  --no-color           disable ANSI colour output
  --version            print version information

Examples:
  traceart --simulate
  traceart --simulate --destination tokyo
  traceart --simulate --seed 42 --width 120
`

// Run parses args and dispatches the request. Returns an error suitable for an exit code.
func (a *App) Run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("traceart", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }

	var (
		simulate    bool
		destination string
		noColor     bool
		showVersion bool
		seed        int64
		width       int
	)
	fs.BoolVar(&simulate, "simulate", true, "run the offline simulated traceroute (the only mode today)")
	fs.StringVar(&destination, "destination", "", "destination host (substring of a known target, or any custom name)")
	fs.BoolVar(&noColor, "no-color", false, "disable ANSI colour output")
	fs.BoolVar(&showVersion, "version", false, "print version information")
	fs.Int64Var(&seed, "seed", 0, "seed the simulation (0 = time-based)")
	fs.IntVar(&width, "width", 0, "map width in columns (0 = default)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if showVersion {
		a.printVersion(stdout)
		return nil
	}

	if !simulate {
		// live traceroute would need raw sockets plus a geoip database; for
		// now the simulation is the only mode. Tell the user instead of
		// silently substituting.
		fmt.Fprintln(stderr, "traceart: live mode is not implemented; running --simulate")
	}

	return art.Render(stdout, art.Options{
		NoColor:     noColor,
		Seed:        seed,
		Destination: destination,
		Width:       width,
	})
}

func (a *App) printVersion(w io.Writer) {
	fmt.Fprintf(w, "traceart %s\n", a.id.Version)
	fmt.Fprintf(w, "  commit:      %s\n", a.id.GitCommit)
	fmt.Fprintf(w, "  description: %s\n", a.id.GitDescription)
	fmt.Fprintf(w, "  built:       %s\n", a.id.BuildDate)
}
