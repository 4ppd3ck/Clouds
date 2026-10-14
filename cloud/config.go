package cloud

import (
	"os"
	"path/filepath"
	"strings"
)

// ConfigPath returns the location of the optional config file:
// os.UserConfigDir()/clouds/config — e.g. ~/.config/clouds/config on Linux,
// ~/Library/Application Support/clouds/config on macOS,
// %AppData%\clouds\config on Windows.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clouds", "config"), nil
}

// LoadConfig reads the config file from the default location. It is a thin
// wrapper over LoadConfigFrom; see there for the file format.
func LoadConfig() (Options, error) {
	path, err := ConfigPath()
	if err != nil {
		return Options{}, nil // no usable config dir; run on flags alone
	}
	return LoadConfigFrom(path)
}

// LoadConfigFrom reads the optional key=value config file at path into Options.
// A missing file is not an error (all fields stay zero). Lines starting
// with # or ; are comments. Recognized keys:
//
//	profile = prod
//	region = us-west-2
//	project = my-gcp-project
//	download-dir = /data/s3
//	all-regions = true
//	demo = false
//
// Explicit command-line flags override the config file.
func LoadConfigFrom(path string) (Options, error) {
	var o Options
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return o, nil
		}
		return o, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(strings.ToLower(k))
		v = strings.TrimSpace(v)
		switch k {
		case "profile":
			o.Profile = v
		case "region":
			o.Region = v
		case "project":
			o.Project = v
		case "download-dir", "download_dir", "downloaddir":
			o.DownloadDir = v
		case "all-regions", "all_regions", "allregions":
			o.AllRegions = parseConfigBool(v)
		case "demo":
			o.Demo = parseConfigBool(v)
		}
	}
	return o, nil
}

func parseConfigBool(v string) bool {
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
