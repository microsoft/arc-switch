package main

import (
	"bgp_summary_parser"
	"encoding/json"
	"environment_temperature_parser"
	"flag"
	"fmt"
	"interface_status_parser"
	"inventory_parser"
	"lldp_neighbor_parser"
	"os"
	"processes_cpu_parser"
	"sort"
	"system_parser"
	"system_uptime_parser"
	"version_parser"
)

// Parser is the interface each parser module satisfies via structural typing
type Parser interface {
	GetDescription() string
	Parse(input []byte) (interface{}, error)
}

var version = "dev"

var parserRegistry = map[string]Parser{
	"version":          &version_parser.VersionParser{},
	"lldp":             &lldp_neighbor_parser.LldpParser{},
	"interface-status": &interface_status_parser.InterfaceStatusParser{},
	"inventory":        &inventory_parser.InventoryParser{},
	"environment":      &environment_temperature_parser.EnvironmentParser{},
	"system":           &system_parser.SystemParser{},
	"processes-cpu":    &processes_cpu_parser.ProcessesCpuParser{},
	"uptime":           &system_uptime_parser.UptimeParser{},
	"bgp-summary":      &bgp_summary_parser.BgpSummaryParser{},
}

func main() {
	parserType := flag.String("parser", "", "Parser type to use")
	parserTypeShort := flag.String("p", "", "Parser type to use (shorthand)")
	inputFile := flag.String("input", "", "Input file path")
	inputFileShort := flag.String("i", "", "Input file path (shorthand)")
	outputFile := flag.String("output", "", "Output file path (default: stdout)")
	outputFileShort := flag.String("o", "", "Output file path (shorthand)")
	listParsers := flag.Bool("list", false, "List available parsers")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *showVersion {
		fmt.Printf("Dell OS10 Switch Output Parser v%s\n", version)
		os.Exit(0)
	}

	if *listParsers {
		fmt.Println("Available parsers:")
		names := make([]string, 0, len(parserRegistry))
		for name := range parserRegistry {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			p := parserRegistry[name]
			fmt.Printf("  %-20s %s\n", name, p.GetDescription())
		}
		os.Exit(0)
	}

	pType := *parserType
	if pType == "" {
		pType = *parserTypeShort
	}
	if pType == "" {
		fmt.Fprintln(os.Stderr, "Error: parser type is required. Use -p or -parser.")
		fmt.Fprintln(os.Stderr, "Use -list to see available parsers.")
		os.Exit(1)
	}

	inFile := *inputFile
	if inFile == "" {
		inFile = *inputFileShort
	}
	if inFile == "" {
		fmt.Fprintln(os.Stderr, "Error: input file is required. Use -i or -input.")
		os.Exit(1)
	}

	outFile := *outputFile
	if outFile == "" {
		outFile = *outputFileShort
	}

	p, ok := parserRegistry[pType]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown parser type: %s\n", pType)
		fmt.Fprintln(os.Stderr, "Use -list to see available parsers.")
		os.Exit(1)
	}

	data, err := os.ReadFile(inFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}

	result, err := p.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if outFile != "" {
		err = os.WriteFile(outFile, jsonBytes, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(string(jsonBytes))
	}
}
