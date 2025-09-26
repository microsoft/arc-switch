package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// InterfaceData represents the structure of the JSON output.
type InterfaceData struct {
    Interfaces []Interface `json:"interfaces"`
}

// Interface represents a single interface's details.
type Interface struct {
    Name                string          `json:"name"`
    Status              string          `json:"status"`
    LineProtocol        string          `json:"line_proto"`
    MTU                 int             `json:"mtu"`
    HardwareEthAddr     string          `json:"HardwareAddr"`
    Description         string          `json:"description"`
    WaveLength          int             `json:"WaveLength"`
    MediaFecOption      string          `json:"MediaFecOption"`
    InternetAddrIPv6    string          `json:"Iaddr_ipv6"`
    InternetAddrIPv4    string          `json:"Iaddr_ipv4"`
    LineSpeed           string          `json:"LineSpd"`
    AutoNeg             string          `json:"AutoNeg"`
    LastClearedCounter  string          `json:"LastClearedCoutner"`
    LastStatusChange    string          `json:"LastStatusChange"`
    QueuingStrategy     string          `json:"Queuing"`
    InputStatistics     InputStatistics `json:"input_stat"`
    OutputStatistics    OutputStatistics `json:"output_stat"`
    RateInfo            RateInfo        `json:"rate_info"`
}

// InputStatistics represents the input statistics of an interface.
type InputStatistics struct {
    Packets    int64 `json:"pkts"`
    Bytes      int64 `json:"bytes"`
    Multicasts int64 `json:"multicasts"`
    Broadcasts int64 `json:"broadcasts"`
    Unicasts   int64 `json:"unicasts"`
    Runts      int64 `json:"runts"`
    Giants     int64 `json:"giants"`
    Throttles  int64 `json:"throttles"`
    CRC        int64 `json:"crc"`
    Overrun    int64 `json:"overrun"`
    Discarded  int64 `json:"discarded"`
}

// OutputStatistics represents the output statistics of an interface.
type OutputStatistics struct {
    Packets    int64 `json:"pkts"`
    Bytes      int64 `json:"bytes"`
    Multicasts int64 `json:"multicasts"`
    Broadcasts int64 `json:"broadcasts"`
    Unicasts   int64 `json:"unicasts"`
    Throttles  int64 `json:"throttles"`
    Discarded  int64 `json:"discarded"`
    Collisions int64 `json:"collisions"`
    WREDDrops  int64 `json:"wred_drops"`
}

// RateInfo represents rate information of an interface.
type RateInfo struct {
    InputRateMbps      int `json:"inRateMbps"`
    InputPacketsPerSec int `json:"inPktsPerSec"`
    InputLineRatePct   int `json:"inLineRatePct"`
    OutputRateMbps     int `json:"outRateMbps"`
    OutputPacketsPerSec int `json:"outPktsPerSec"`
    OutputLineRatePct   int `json:"outLineRatePct"`
}

func main() {
    // Define command-line flags
    inputfile := flag.String("inputfile", "", "Path to the input file containing 'show interface' data")
    test := flag.Bool("test", false, "Enable test mode (no OS commands will be executed, verbose output enabled)")
    flag.Parse()

    // Validate input parameters
    if *test && *inputfile == "" {
        log.Fatal("Error: When test mode is enabled, an inputfile must be provided.")
    }

    // Fetch interface data
    data, err := getInterfaceData(*inputfile, *test)
    if err != nil {
        log.Fatalf("Error fetching interface data: %v", err)
    }

    // Process data into JSON format
    jsonData, err := processToJSON(data)
    if err != nil {
        log.Fatalf("Error processing data to JSON: %v", err)
    }

    // Log JSON data to syslog
    err = logToSyslog(jsonData, *test)
    if err != nil {
        log.Fatalf("Error logging data to syslog: %v", err)
    }

    if *test {
        fmt.Println("Operation completed successfully in test mode.")
    }
}

// getInterfaceData fetches the interface data from the specified source.
func getInterfaceData(inputfile string, test bool) (string, error) {
    if inputfile != "" {
        if test {
            fmt.Printf("Reading data from file: %s\n", inputfile)
        }
        data, err := os.ReadFile(inputfile)
        if err != nil {
            return "", fmt.Errorf("failed to read file: %w", err)
        }
        return string(data), nil
    }

    if test {
        return "", errors.New("test mode is enabled, but no inputfile was provided")
    }

    // Use CLI command to fetch data
    cmd := exec.Command("/opt/dell/os10/bin/clish", "-c", "show interface")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to execute CLI command: %w", err)
    }
    return string(output), nil
}

// processToJSON processes the raw interface data into JSON format.
func processToJSON(data string) ([]byte, error) {
    interfaces := []Interface{}
    lines := strings.Split(data, "\n")
    var currentInterface *Interface

    // Regular expressions for parsing
    reInterface := regexp.MustCompile(`^(\S+ \S+) is (\S+), line protocol is (\S+)$`)
    reDescription := regexp.MustCompile(`^Description: (.+)$`)
    reHardware := regexp.MustCompile(`^Hardware is (\S+), address is (\S+)$`)
    reWaveLength := regexp.MustCompile(`^Wavelength is (\d+)$`)
    reMediaFecOption := regexp.MustCompile(`^Configured media fec option is (.+)$`)
    reIPv4 := regexp.MustCompile(`^Internet address is (.+)$`)
    reIPv6 := regexp.MustCompile(`^Mode of IPv4 Address Assignment: (.+)$`)
    reMTU := regexp.MustCompile(`^MTU (\d+) bytes, IP MTU (\d+) bytes$`)
    reLineSpeed := regexp.MustCompile(`^LineSpeed (\S+), Auto-Negotiation (\S+), Link-Training (\S+)$`)
    reLastClearedCounter := regexp.MustCompile(`^Last clearing of "show interface" counters: (.+)$`)
    reLastStatusChange := regexp.MustCompile(`^Time since last interface status change: (.+)$`)
    reQueuingStrategy := regexp.MustCompile(`^Queuing strategy: (.+)$`)
    reInputRate := regexp.MustCompile(`^Input (\d+) Mbits/sec, (\d+) packets/sec, (\d+)% (.+)$`)
    reOutputRate := regexp.MustCompile(`^Output (\d+) Mbits/sec, (\d+) packets/sec, (\d+)% (.+)$`)
    parsingInputStats := false
    parsingOutputStats := false

    for _, line := range lines {
        line = strings.TrimSpace(line)

        if parsingInputStats {
            // Parse Input Statistics
            if strings.HasPrefix(line, "Output statistics:") {
                parsingInputStats = false
                parsingOutputStats = true
                continue
            }
            parseInputStatistics(line, currentInterface)
            continue
        }

        if parsingOutputStats {
            // Parse Output Statistics
            if line == "" || strings.HasPrefix(line, "Rate Info") {
                parsingOutputStats = false
                continue
            }
            parseOutputStatistics(line, currentInterface)
            continue
        }

        if strings.HasPrefix(line, "Input statistics:") {
            parsingInputStats = true
            continue
        }

        if match := reInterface.FindStringSubmatch(line); match != nil {
            // Start of a new interface
            if currentInterface != nil {
                interfaces = append(interfaces, *currentInterface)
            }
            currentInterface = &Interface{
                Name:         match[1],
                Status:       match[2],
                LineProtocol: match[3],
                RateInfo:     RateInfo{}, // Initialize RateInfo to avoid undefined errors
            }
        } else if currentInterface != nil {
            // Parse additional fields
            if match := reDescription.FindStringSubmatch(line); match != nil {
                currentInterface.Description = match[1]
            } else if match := reHardware.FindStringSubmatch(line); match != nil {
                currentInterface.HardwareEthAddr = match[2]
            } else if match := reWaveLength.FindStringSubmatch(line); match != nil {
                currentInterface.WaveLength, _ = strconv.Atoi(match[1])
            } else if match := reMediaFecOption.FindStringSubmatch(line); match != nil {
                currentInterface.MediaFecOption = match[1]
            } else if match := reIPv4.FindStringSubmatch(line); match != nil {
                currentInterface.InternetAddrIPv4 = match[1]
            } else if match := reIPv6.FindStringSubmatch(line); match != nil {
                currentInterface.InternetAddrIPv6 = match[1]
            } else if match := reMTU.FindStringSubmatch(line); match != nil {
                currentInterface.MTU, _ = strconv.Atoi(match[1])
            } else if match := reLineSpeed.FindStringSubmatch(line); match != nil {
                currentInterface.LineSpeed = match[1]
                currentInterface.AutoNeg = match[2]
            } else if match := reLastClearedCounter.FindStringSubmatch(line); match != nil {
                currentInterface.LastClearedCounter = match[1]
            } else if match := reLastStatusChange.FindStringSubmatch(line); match != nil {
                currentInterface.LastStatusChange = match[1]
            } else if match := reQueuingStrategy.FindStringSubmatch(line); match != nil {
                currentInterface.QueuingStrategy = match[1]
            } else if match := reInputRate.FindStringSubmatch(line); match != nil {
                currentInterface.RateInfo.InputRateMbps, _ = strconv.Atoi(match[1])
                currentInterface.RateInfo.InputPacketsPerSec, _ = strconv.Atoi(match[2])
                currentInterface.RateInfo.InputLineRatePct, _ = strconv.Atoi(match[3])
            } else if match := reOutputRate.FindStringSubmatch(line); match != nil {
                currentInterface.RateInfo.OutputRateMbps, _ = strconv.Atoi(match[1])
                currentInterface.RateInfo.OutputPacketsPerSec, _ = strconv.Atoi(match[2])
                currentInterface.RateInfo.OutputLineRatePct, _ = strconv.Atoi(match[3]) 
            }
            
        }
    }

    // Add the last interface
    if currentInterface != nil {
        interfaces = append(interfaces, *currentInterface)
    }

    // Convert to JSON
    interfaceData := InterfaceData{Interfaces: interfaces}
    jsonData, err := json.Marshal(interfaceData)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal data to JSON: %w", err)
    }
    return jsonData, nil
}

// Helper function to parse Input Statistics
func parseInputStatistics(line string, iface *Interface) {
    if iface == nil {
        return
    }
    fields := strings.Fields(line)
    if len(fields) >= 2 {
        switch {
        case strings.HasSuffix(fields[1], "packets,"):
            iface.InputStatistics.Packets = atoi64(fields[0])
        case strings.HasSuffix(fields[1], "64-byte"):
            iface.InputStatistics.Bytes = atoi64(fields[0])
        case strings.HasSuffix(fields[1], "Multicasts,") && len(fields) >= 6 && strings.HasSuffix(fields[3], "Broadcasts,") && strings.HasSuffix(fields[5], "Unicasts"):
            iface.InputStatistics.Multicasts = atoi64(fields[0])
            iface.InputStatistics.Broadcasts = atoi64(fields[2])
            iface.InputStatistics.Unicasts = atoi64(fields[4])
        case strings.HasSuffix(fields[1], "runts,") && strings.HasSuffix(fields[3], "giants,") && strings.HasSuffix(fields[5], "throttles"):
            iface.InputStatistics.Runts = atoi64(fields[0])
            iface.InputStatistics.Giants = atoi64(fields[2])
            iface.InputStatistics.Throttles = atoi64(fields[4])
        case strings.HasSuffix(fields[1], "CRC,") && strings.HasSuffix(fields[3], "overrun,") && strings.HasSuffix(fields[5], "discarded"):
            iface.InputStatistics.CRC = atoi64(fields[0])
            iface.InputStatistics.Overrun = atoi64(fields[2])
            iface.InputStatistics.Discarded = atoi64(fields[4])
        }
    }
}

// Helper function to parse Output Statistics
func parseOutputStatistics(line string, iface *Interface) {
    if iface == nil {
        return
    }
    fields := strings.Fields(line)
    if len(fields) >= 2 {
        switch {
        case strings.HasSuffix(fields[1], "packets,"):
            iface.OutputStatistics.Packets = atoi64(fields[0])
        case strings.HasSuffix(fields[1], "64-byte"):
            iface.OutputStatistics.Bytes = atoi64(fields[0])
        case strings.HasSuffix(fields[1], "Multicasts,") && len(fields) >= 6 && strings.HasSuffix(fields[3], "Broadcasts,") && strings.HasSuffix(fields[5], "Unicasts"):
            iface.OutputStatistics.Multicasts = atoi64(fields[0])
            iface.OutputStatistics.Broadcasts = atoi64(fields[2])
            iface.OutputStatistics.Unicasts = atoi64(fields[4])
        case strings.HasSuffix(fields[1], "throttles,") && strings.HasSuffix(fields[3], "discarded,") && strings.HasSuffix(fields[5], "Collisions"):
            iface.OutputStatistics.Throttles = atoi64(fields[0])
            iface.OutputStatistics.Discarded = atoi64(fields[2])
            iface.OutputStatistics.Collisions = atoi64(fields[4])
        }
    }
}

// Helper function to safely convert strings to int64
func atoi64(s string) int64 {
    value, _ := strconv.ParseInt(s, 10, 64)
    return value
}


// logToSyslog logs the JSON data to syslog in a compact format.
func logToSyslog(jsonData []byte, test bool) error {
    // Unmarshal the JSON data into the InterfaceData struct
    var interfaceData InterfaceData
    err := json.Unmarshal(jsonData, &interfaceData)
    if err != nil {
        return fmt.Errorf("failed to unmarshal JSON data: %w", err)
    }

    // Iterate over each interface and log it individually
    for _, iface := range interfaceData.Interfaces {
        // Marshal the individual interface to JSON
        ifaceJSON, err := json.Marshal(iface)
        if err != nil {
            return fmt.Errorf("failed to marshal interface to JSON: %w", err)
        }

        // Prepare the logger command
        loggerCmd := exec.Command("logger", "--size", "4096", "-p", "local0.info", "-t", "interface", string(ifaceJSON)+";")

        if test {
            // Print the logger command and JSON data in test mode
            fmt.Printf("Logger command: %s\n", strings.Join(loggerCmd.Args, " "))
            fmt.Printf("Compact JSON data: %s\n", string(ifaceJSON))
        } else {
            // Execute the logger command
            err := loggerCmd.Run()
            if err != nil {
                return fmt.Errorf("failed to log data to syslog: %w", err)
            }
        }
    }

    return nil
}