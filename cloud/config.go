package cloud

import (
	"fmt"
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

// SaveDownloadDir updates only the download-dir setting in the user's Clouds
// config file, preserving any other options and comments.
func SaveDownloadDir(path, configPath string) error {
	if strings.ContainsAny(path, "\r\n") {
		return fmt.Errorf("download directory must be a single-line path")
	}
	if configPath == "" {
		var err error
		configPath, err = ConfigPath()
		if err != nil {
			return err
		}
	} else {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return err
		}
		configPath = abs
	}
	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	out := make([]string, 0, len(lines)+1)
	updated := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		key, _, ok := strings.Cut(trimmed, "=")
		key = strings.ToLower(strings.TrimSpace(key))
		if ok && (key == "download-dir" || key == "download_dir" || key == "downloaddir") {
			if !updated {
				out = append(out, "download-dir = "+path)
				updated = true
			}
			continue
		}
		out = append(out, line)
	}
	if !updated {
		out = append(out, "download-dir = "+path)
	}
	content := strings.Join(out, "\n") + "\n"
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(configPath), ".config-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, configPath)
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
