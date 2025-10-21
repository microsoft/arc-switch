package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"bgp_all_summary_parser"
	"class_map_parser"
	"environment_power_parser"
	"interface_counters_parser"
	"interface_counters_error_parser"
	"inventory_parser"
	"ip_arp_parser"
	"ip_route_parser"
	"lldp_neighbor_parser"
	"mac_address_parser"
	"system_uptime_parser"
	"transceiver_parser"
)

const version = "v1.0.0"

type Parser interface {
	Parse(input []byte) (interface{}, error)
	GetDescription() string
}

var parsers = map[string]func() Parser{
	"bgp-all-summary":          func() Parser { return &bgp_all_summary_parser.UnifiedParser{} },
	"class-map":                func() Parser { return &class_map_parser.UnifiedParser{} },
	"environment-power":        func() Parser { return &environment_power_parser.UnifiedParser{} },
	"interface-counters":       func() Parser { return &interface_counters_parser.UnifiedParser{} },
	"interface-error-counters": func() Parser { return &interface_counters_error_parser.UnifiedParser{} },
	"inventory":                func() Parser { return &inventory_parser.UnifiedParser{} },
	"ip-arp":                   func() Parser { return &ip_arp_parser.UnifiedParser{} },
	"ip-route":                 func() Parser { return &ip_route_parser.UnifiedParser{} },
	"lldp-neighbor":            func() Parser { return &lldp_neighbor_parser.UnifiedParser{} },
	"mac-address":              func() Parser { return &mac_address_parser.UnifiedParser{} },
	"system-uptime":            func() Parser { return &system_uptime_parser.UnifiedParser{} },
	"transceiver":              func() Parser { return &transceiver_parser.UnifiedParser{} },
}

func main() {
	var (
		parserType  string
		inputFile   string
		outputFile  string
		listParsers bool
		showVersion bool
	)

	flag.StringVar(&parserType, "parser", "", "Parser type to use")
	flag.StringVar(&parserType, "p", "", "Parser type to use (shorthand)")
	flag.StringVar(&inputFile, "input", "", "Input file path")
	flag.StringVar(&inputFile, "i", "", "Input file path (shorthand)")
	flag.StringVar(&outputFile, "output", "", "Output file path")
	flag.StringVar(&outputFile, "o", "", "Output file path (shorthand)")
	flag.BoolVar(&listParsers, "list", false, "List available parsers")
	flag.BoolVar(&showVersion, "version", false, "Show version")

	flag.Parse()

	if showVersion {
		fmt.Printf("cisco-parser %s\n", version)
		os.Exit(0)
	}

	if listParsers {
		fmt.Println("Available parsers:")
		for name, parserFunc := range parsers {
			parser := parserFunc()
			fmt.Printf("  %-20s %s\n", name, parser.GetDescription())
		}
		os.Exit(0)
	}

	if parserType == "" || inputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	parserFunc, exists := parsers[strings.ToLower(parserType)]
	if !exists {
		log.Fatalf("Unknown parser type: %s", parserType)
	}
	parser := parserFunc()

	input, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	result, err := parser.Parse(input)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if outputFile != "" {
		err = ioutil.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Successfully parsed %s\n", filepath.Base(inputFile))
		fmt.Printf("Output written to: %s\n", outputFile)
	} else {
		fmt.Println(string(jsonData))
	}
}