package main

import (
	"dell-os10-parser/parsers"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

var parserRegistry = map[string]parsers.Parser{
	"version":          &parsers.VersionParser{},
	"lldp":             &parsers.LldpParser{},
	"interface-status": &parsers.InterfaceStatusParser{},
	"inventory":        &parsers.InventoryParser{},
	"environment":      &parsers.EnvironmentParser{},
	"system":           &parsers.SystemParser{},
	"processes-cpu":    &parsers.ProcessesCpuParser{},
	"uptime":           &parsers.UptimeParser{},
	"bgp-summary":      &parsers.BgpSummaryParser{},
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
		fmt.Println("Dell OS10 Switch Output Parser v2.0.0")
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
