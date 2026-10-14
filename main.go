// Command clouds is a k9s-style terminal UI for browsing AWS and GCP
// resources — a single static binary with no runtime dependencies.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"clouds/cloud"
	"clouds/providers"
	"clouds/ui"
)

func main() {
	// Undocumented: print the embedded authorship seal (ciphertext) and exit.
	// Intercepted before flag.Parse so it never shows in -h.
	for _, a := range os.Args[1:] {
		if a == "--seal" {
			fmt.Println(ui.SealBlob())
			return
		}
	}

	profile := flag.String("profile", "", "AWS shared-config profile (default: AWS default chain)")
	region := flag.String("region", "", "AWS region override")
	project := flag.String("project", "", "GCP project override")
	downloadDir := flag.String("download-dir", "", "download target dir (default: ~/Downloads/clouds)")
	demo := flag.Bool("demo", false, "explore built-in sample data (no credentials needed)")
	allRegions := flag.Bool("all-regions", false, "AWS: scan every enabled region instead of just the configured one (S3 is always global)")
	showVersion := flag.Bool("version", false, "print version and exit")
	listServices := flag.Bool("list-services", false, "list providers/services and exit")
	configPath := flag.String("config", "", "config file to load (default: the OS config dir)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("clouds %s", ui.Version)
		if ui.Commit != "" {
			fmt.Printf(" (%s, built %s)", ui.Commit, ui.Date)
		}
		fmt.Println()
		return
	}

	// Option precedence: explicit flags > config file > built-in defaults.
	var fileCfg cloud.Options
	var cfgErr error
	if *configPath != "" {
		fileCfg, cfgErr = cloud.LoadConfigFrom(*configPath)
	} else {
		fileCfg, cfgErr = cloud.LoadConfig()
	}
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "clouds: config: %v\n", cfgErr)
	}
	flagSet := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })
	strOpt := func(flagVal, cfgVal string) string {
		if flagVal != "" {
			return flagVal
		}
		return cfgVal
	}
	boolOpt := func(name string, flagVal, cfgVal bool) bool {
		if flagSet[name] {
			return flagVal
		}
		return cfgVal
	}

	opts := cloud.Options{
		Profile:     strOpt(*profile, fileCfg.Profile),
		Region:      strOpt(*region, fileCfg.Region),
		Project:     strOpt(*project, fileCfg.Project),
		DownloadDir: strOpt(*downloadDir, fileCfg.DownloadDir),
		Demo:        boolOpt("demo", *demo, fileCfg.Demo),
		AllRegions:  boolOpt("all-regions", *allRegions, fileCfg.AllRegions),
	}

	impls := providers.BuildAll(opts)

	if *listServices {
		for _, impl := range impls {
			cat := impl.Catalog()
			fmt.Printf("\n%s  (%s)\n", cat.ID, cat.Name)
			for _, s := range cat.Services {
				alias := ""
				if s.Alias != "" && s.Alias != s.ID {
					alias = fmt.Sprintf("   alias: :%s", s.Alias)
				}
				fmt.Printf("  :%-10s %s%s\n", s.ID, s.Name, alias)
			}
		}
		return
	}

	p := tea.NewProgram(ui.NewModel(opts, impls), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "clouds:", err)
		os.Exit(1)
	}
}
