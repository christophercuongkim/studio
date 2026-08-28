package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/christophercuongkim/studio/internal/config"
	"github.com/christophercuongkim/studio/internal/search"
)

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: studio search <text…> [filters]")
		fmt.Fprintln(fs.Output(), "\nSearch clip descriptions across every ingested shoot.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	rating := fs.String("rating", "", "minimum rating, e.g. 4+ or 4")
	status := fs.String("status", "", "review status: pending|kept|rejected")
	cam := fs.String("cam", "", "camera code, e.g. DJI")
	since := fs.String("since", "", "on/after date: YYYY, YYYY-MM, or YYYY-MM-DD")
	until := fs.String("until", "", "on/before date: YYYY, YYYY-MM, or YYYY-MM-DD")
	dur := fs.String("dur", "", "duration range, e.g. 5s..2m")
	shoot := fs.String("shoot", "", "shoot title substring")
	asJSON := fs.Bool("json", false, "output full records as JSON")
	pathsOnly := fs.Bool("paths", false, "output original paths only (pipe-friendly)")
	play := fs.Int("play", 0, "launch the configured player on result N's proxy")
	rest, err := parseFlags(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	filters := search.NewFilters()
	for _, w := range rest {
		filters.Terms = append(filters.Terms, strings.ToLower(w))
	}
	if *rating != "" {
		if filters.MinRating, err = search.ParseRating(*rating); err != nil {
			return err
		}
	}
	filters.Status = *status
	filters.Cam = *cam
	filters.Shoot = *shoot
	if *since != "" {
		if filters.Since, err = search.ParsePeriodStart(*since); err != nil {
			return err
		}
	}
	if *until != "" {
		if filters.UntilEnd, err = search.ParsePeriodEnd(*until); err != nil {
			return err
		}
	}
	if *dur != "" {
		if filters.MinDur, filters.MaxDur, err = search.ParseDurRange(*dur); err != nil {
			return err
		}
	}

	records, warnings := search.Collect(cfg.SearchRoots)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warn: skipping unreadable manifest %s\n", w)
	}
	results := search.Apply(records, filters)

	if *play > 0 {
		return playResult(cfg.Player, results, *play)
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(results)
	}
	if *pathsOnly {
		for _, r := range results {
			fmt.Println(r.OriginalPath)
		}
		return nil
	}
	printTable(results)
	return nil
}

func playResult(player string, results []search.Record, n int) error {
	if n > len(results) {
		return fmt.Errorf("--play %d out of range (%d results)", n, len(results))
	}
	r := results[n-1]
	target := r.ProxyPath
	if target == "" {
		target = r.OriginalPath
	}
	if player == "" {
		return errors.New("no player configured (set player in config)")
	}
	cmd := exec.Command(player, target)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func printTable(results []search.Record) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\trating\tdur\tdate\tcam\tshoot\tname")
	for i, r := range results {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, stars(r.Rating), fmtDur(r.DurationSec), fmtDate(r.CreatedAt), r.Cam, r.Shoot, r.Name())
	}
	tw.Flush()
	fmt.Printf("%d result(s)\n", len(results))
}

func stars(n int) string {
	if n <= 0 {
		return "-"
	}
	return strings.Repeat("★", n)
}

func fmtDur(sec float64) string {
	d := time.Duration(sec * float64(time.Second)).Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return strconv.Itoa(m) + ":" + fmt.Sprintf("%02d", s)
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02")
}
