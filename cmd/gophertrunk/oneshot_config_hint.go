package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MattCheramie/GopherTrunk/internal/config"
)

// The one-shot tools (`power`, `capture`) open a radio from their own flags and
// read no config file, so an `sdr.iio` / `sdr.rtl_tcp` entry in config.yaml
// does nothing for them. The #1230 reporter wrote a PlutoSky into config.yaml,
// ran `power`, and got only "no SDRs found" with no way to tell why. When the
// tool finds no local SDR and was given no network radio, it now checks the
// config the daemon would load and, if that file lists network radios, says
// which flag reaches them.

// noSDRWithConfigHint returns err, extended with the config hint when err is
// the "no SDRs found" failure and the discovered config lists network radios.
// The original message stays the prefix: callers (wmbus's reconnect wait)
// match on it.
func noSDRWithConfigHint(tool string, err error) error {
	return noSDRWithConfigHintAt(tool, config.Discover(), err)
}

// noSDRWithConfigHintAt is noSDRWithConfigHint against an explicit config path.
func noSDRWithConfigHintAt(tool, path string, err error) error {
	if err == nil || !strings.Contains(err.Error(), "no SDRs found") {
		return err
	}
	if hint := oneShotConfigHint(tool, path); hint != "" {
		return fmt.Errorf("%w\n%s", err, hint)
	}
	return err
}

// oneShotConfigHint reads the sdr section of the config at path and returns
// operator-facing lines naming the flag for each network radio it lists, or ""
// when path is empty, unreadable, or lists none. It parses only the sdr
// section, so a config that would fail the daemon's full validation (a
// half-written one, as the reporter's was) still gets the hint.
func oneShotConfigHint(tool, path string) string {
	if path == "" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		SDR struct {
			IIO []struct {
				Addr string `yaml:"addr"`
			} `yaml:"iio"`
			RTLTCP []struct {
				Addr string `yaml:"addr"`
			} `yaml:"rtl_tcp"`
			SoapyRemote []struct {
				Addr string `yaml:"addr"`
			} `yaml:"soapy_remote"`
		} `yaml:"sdr"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return ""
	}
	var lines []string
	for _, e := range doc.SDR.IIO {
		if a := strings.TrimSpace(e.Addr); a != "" {
			lines = append(lines, fmt.Sprintf("  sdr.iio lists a Pluto / PlutoSky at %s: run `gophertrunk %s -iio %s ...`", a, tool, a))
		}
	}
	for _, e := range doc.SDR.RTLTCP {
		if a := strings.TrimSpace(e.Addr); a != "" {
			lines = append(lines, fmt.Sprintf("  sdr.rtl_tcp lists a server at %s: run `gophertrunk %s -rtltcp %s ...`", a, tool, a))
		}
	}
	for _, e := range doc.SDR.SoapyRemote {
		if a := strings.TrimSpace(e.Addr); a != "" {
			lines = append(lines, fmt.Sprintf("  sdr.soapy_remote lists a SoapySDRServer at %s: `%s` cannot open it; only the daemon (`gophertrunk -config <file>`) can", a, tool))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("`%s` reads no config file, so the network radios in %s are not used:\n%s",
		tool, path, strings.Join(lines, "\n"))
}
