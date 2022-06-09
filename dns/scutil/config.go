// Package scutil provides a complete model and parser for 'scutil --dns' output.
package scutil

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config is a parsed scutil DNS config
type Config struct {
	Resolvers []Resolver
}

// Resolver is an scutil resolver entry
type Resolver struct {
	Domain         string
	Flags          []Flag
	InterfaceIndex int
	InterfaceName  string
	MulticastDNS   bool
	Nameservers    []string
	Order          int
	Port           int
	Reach          []Reach
	reachable      bool // cached status from Reach
	SearchDomain   []string
	Timeout        time.Duration
}

// ReadMacOSDNS reads and parses the current macOS scutil DNS settings
func ReadMacOSDNS(ctx context.Context) (Config, error) {
	return readMacOSDNS(ctx, runSCUtilDNS)
}

func runSCUtilDNS(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/sbin/scutil", "--dns")
	return cmd.CombinedOutput()
}

type scutilExecutor func(ctx context.Context) ([]byte, error)

func readMacOSDNS(ctx context.Context, getSCUtilDNS scutilExecutor) (Config, error) {
	output, err := getSCUtilDNS(ctx)
	if err != nil {
		return Config{}, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	var config Config
	for scanner.Scan() {
		key, value := splitKeyValue(scanner.Text())
		if strings.Contains(key, "resolver") {
			config.Resolvers = append(config.Resolvers, Resolver{})
			continue
		}
		if len(config.Resolvers) == 0 {
			continue
		}

		currentResolver := &config.Resolvers[len(config.Resolvers)-1]
		parseAndApplyResolverKey(currentResolver, key, value)
	}

	return config, nil
}

func parseAndApplyResolverKey(r *Resolver, key, value string) {
	const (
		decimalBase = 10
		maxIntBits  = 64
	)
	switch {
	case strings.Contains(key, "search domain"):
		r.SearchDomain = append(r.SearchDomain, value)
	case strings.Contains(key, "domain"):
		r.Domain = value
	case strings.Contains(key, "flags"):
		for _, flag := range strings.Split(value, ",") {
			r.Flags = append(r.Flags, Flag(strings.TrimSpace(flag)))
		}
	case strings.Contains(key, "if_index"):
		tokens := strings.Fields(value)
		const ifIndexTokenCount = 2
		if len(tokens) == ifIndexTokenCount {
			r.InterfaceName = strings.Trim(tokens[1], "()")
		}

		i, err := strconv.ParseInt(tokens[0], decimalBase, maxIntBits)
		if err == nil {
			r.InterfaceIndex = int(i)
		}
	case strings.Contains(key, "options"):
		r.MulticastDNS = strings.Contains(value, "mdns")
	case strings.Contains(key, "port"):
		i, err := strconv.ParseInt(value, decimalBase, maxIntBits)
		if err == nil {
			r.Port = int(i)
		}
	case strings.Contains(key, "nameserver"):
		r.Nameservers = append(r.Nameservers, value)
	case strings.Contains(key, "order"):
		i, err := strconv.ParseInt(value, decimalBase, maxIntBits)
		if err == nil {
			r.Order = int(i)
		}
	case strings.Contains(key, "reach"):
		tokens := strings.Fields(value)
		const reachTokenCount = 2
		if len(tokens) == reachTokenCount {
			reach := strings.Trim(tokens[1], "()")
			for _, statusStr := range strings.Split(reach, ",") {
				status := Reach(statusStr)
				if status == Reachable {
					r.reachable = true
				}
				r.Reach = append(r.Reach, status)
			}
		}
	case strings.Contains(key, "timeout"):
		i, err := strconv.ParseInt(value, decimalBase, maxIntBits)
		if err == nil {
			r.Timeout = time.Duration(i) * time.Second
		}
	}
}

func splitKeyValue(line string) (key, value string) {
	const keyValueSplits = 2
	tokens := strings.SplitN(line, ":", keyValueSplits)
	switch len(tokens) {
	case 1:
		return strings.TrimSpace(tokens[0]), ""
	default:
		return strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1])
	}
}

// Reachable returns whether this resolver is reachable.
//
// scutil defines it as: The specified nodename/address can be reached using the current network configuration.
func (r Resolver) Reachable() bool {
	return r.reachable
}
