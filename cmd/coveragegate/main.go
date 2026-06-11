package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	profile := flag.String("profile", "", "Go coverage profile path")
	minimum := flag.Float64("min", 85, "minimum coverage percentage")
	flag.Parse()

	if *profile == "" {
		fmt.Fprintln(os.Stderr, "coverage profile required")
		os.Exit(2)
	}
	coverage, err := coveragePercent(*profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage gate: %v\n", err)
		os.Exit(2)
	}
	if coverage+0.000001 < *minimum {
		fmt.Fprintf(os.Stderr, "coverage %.1f%% is below %.1f%%\n", coverage, *minimum)
		os.Exit(1)
	}
	fmt.Printf("coverage %.1f%% meets %.1f%% minimum\n", coverage, *minimum)
}

func coveragePercent(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var total uint64
	var covered uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return 0, fmt.Errorf("invalid coverage line %q", line)
		}
		statements, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid statement count in %q", line)
		}
		count, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid execution count in %q", line)
		}
		total += statements
		if count > 0 {
			covered += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	if total == 0 {
		return 0, fmt.Errorf("coverage profile has no statements")
	}
	return float64(covered) * 100 / float64(total), nil
}
