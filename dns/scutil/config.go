package scutil

// 'scutil' provides a complete model and parser for 'scutil --dns' output.

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Resolvers []Resolver
}

type Resolver struct {
	Domain         string
	Flags          []Flag
	InterfaceIndex int
	InterfaceName  string
	MulticastDNS   bool
	Nameservers    []string
	Order          int
	Reach          []Reach
	reachable      bool // cached status from Reach
	SearchDomain   []string
	Timeout        time.Duration
}

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
		switch {
		case strings.Contains(key, "search domain"):
			currentResolver.SearchDomain = append(currentResolver.SearchDomain, value)
		case strings.Contains(key, "domain"):
			currentResolver.Domain = value
		case strings.Contains(key, "flags"):
			for _, flag := range strings.Split(value, ",") {
				currentResolver.Flags = append(currentResolver.Flags, Flag(strings.TrimSpace(flag)))
			}
		case strings.Contains(key, "if_index"):
			tokens := strings.SplitN(value, " ", 2)
			if len(tokens) == 2 {
				currentResolver.InterfaceName = strings.Trim(tokens[1], "()")
			}

			i, err := strconv.ParseInt(tokens[0], 10, 64)
			if err == nil {
				currentResolver.InterfaceIndex = int(i)
			}
		case strings.Contains(key, "options"):
			currentResolver.MulticastDNS = strings.Contains(value, "mdns")
		case strings.Contains(key, "nameserver"):
			currentResolver.Nameservers = append(currentResolver.Nameservers, value)
		case strings.Contains(key, "order"):
			i, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				currentResolver.Order = int(i)
			}
		case strings.Contains(key, "reach"):
			tokens := strings.SplitN(value, " ", 2)
			if len(tokens) == 2 {
				reach := strings.Trim(tokens[1], "()")
				for _, statusStr := range strings.Split(reach, ",") {
					status := Reach(statusStr)
					if status == Reachable {
						currentResolver.reachable = true
					}
					currentResolver.Reach = append(currentResolver.Reach, status)
				}
			}
		case strings.Contains(key, "timeout"):
			i, err := strconv.ParseInt(value, 10, 64)
			if err == nil {
				currentResolver.Timeout = time.Duration(i) * time.Second
			}
		}
	}

	return config, nil
}

func splitKeyValue(line string) (key, value string) {
	tokens := strings.SplitN(line, ":", 2)
	switch len(tokens) {
	case 1:
		return strings.TrimSpace(tokens[0]), ""
	default:
		return strings.TrimSpace(tokens[0]), strings.TrimSpace(tokens[1])
	}
}

func (r Resolver) Reachable() bool {
	return r.reachable
}
