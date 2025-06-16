package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "strings"
    "time"
)

// LLDPData represents the structure of the LLDP data for syslog.
type LLDPData struct {
    Hostname        string `json:"hostname"`
    LocalPortID     string `json:"local_port_id"`
    RemoteSystemName string `json:"remote_system_name"`
    RemotePortID    string `json:"remote_port_id"`
    RemoteChassisID string `json:"remote_chassis_id"`
    RemotePortDesc  string `json:"remote_port_description"`
    RemoteMTU       string `json:"remote_mtu"`
    Timestamp       string `json:"timestamp"`
}

// getLLDPDataFromClish executes the clish command to retrieve LLDP data.
func getLLDPDataFromClish() (string, error) {
    fmt.Println("[VERBOSE] Executing clish command to retrieve LLDP data...")
    cmd := exec.Command("clish", "-c", "show lldp neighbors detail")
    var out bytes.Buffer
    var stderr bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &stderr

    err := cmd.Run()
    if err != nil {
        return "", fmt.Errorf("clish command failed: %v, stderr: %s", err, stderr.String())
    }

    fmt.Println("[VERBOSE] Successfully retrieved LLDP data from clish.")
    return out.String(), nil
}

// parseLLDPData parses the LLDP data from the input string.
func parseLLDPData(data string) ([]LLDPData, error) {
    fmt.Println("[VERBOSE] Parsing LLDP data...")
    var lldpEntries []LLDPData
    var currentEntry LLDPData

    scanner := bufio.NewScanner(strings.NewReader(data))
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())

        // Parse fields
        if strings.HasPrefix(line, "Local Port ID:") {
            currentEntry.LocalPortID = strings.TrimSpace(strings.TrimPrefix(line, "Local Port ID:"))
        } else if strings.HasPrefix(line, "Remote System Name:") {
            currentEntry.RemoteSystemName = strings.TrimSpace(strings.TrimPrefix(line, "Remote System Name:"))
        } else if strings.HasPrefix(line, "Remote Port ID:") {
            currentEntry.RemotePortID = strings.TrimSpace(strings.TrimPrefix(line, "Remote Port ID:"))
        } else if strings.HasPrefix(line, "Remote Chassis ID:") {
            currentEntry.RemoteChassisID = strings.TrimSpace(strings.TrimPrefix(line, "Remote Chassis ID:"))
        } else if strings.HasPrefix(line, "Remote Port Description:") {
            currentEntry.RemotePortDesc = strings.TrimSpace(strings.TrimPrefix(line, "Remote Port Description:"))
        } else if strings.HasPrefix(line, "Remote Max Frame Size:") {
            currentEntry.RemoteMTU = strings.TrimSpace(strings.TrimPrefix(line, "Remote Max Frame Size:"))
        }

        // Detect end of a block
        if line == "---------------------------------------------------------------------------" {
            currentEntry.Timestamp = time.Now().Format("2006-01-02T15:04:05Z07:00")
            lldpEntries = append(lldpEntries, currentEntry)
            currentEntry = LLDPData{} // Reset for the next block
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("error parsing LLDP data: %v", err)
    }

    fmt.Println("[VERBOSE] Successfully parsed LLDP data.")
    return lldpEntries, nil
}

// generateJSONOutput generates the JSON output for an LLDP entry as a single-line string.
func generateJSONOutput(entry LLDPData) (string, error) {
    jsonData, err := json.Marshal(entry) // Use json.Marshal for compact JSON
	
    if err != nil {
        return "", fmt.Errorf("error generating JSON: %v", err)
    }
    return string(jsonData), nil
}

// processLLDPData processes the LLDP data and outputs JSON or makes a system call to the logger command.
func processLLDPData(entries []LLDPData, testMode bool) {
    for _, entry := range entries {
        jsonOutput, err := generateJSONOutput(entry)
        if err != nil {
            fmt.Printf("[ERROR] Error generating JSON: %v\n", err)
            continue
        }

        if testMode {
            // Print the logger command for debugging
            fmt.Printf("[VERBOSE] logger -p local0.info -t LLDPNeighbor '%s'\n", jsonOutput)
        } else {
            // Execute the logger command
            cmd := exec.Command("logger", "-p", "local0.info", "-t", "LLDPNeighbor", jsonOutput, ";")
            err := cmd.Run()
            if err != nil {
                fmt.Printf("[ERROR] Failed to execute logger command: %v\n", err)
            } else {
                fmt.Println("[INFO] Successfully logged LLDP data.")
            }
        }
    }
}

// readFile reads the content of a file.
func readFile(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", fmt.Errorf("failed to open file: %v", err)
    }
    defer file.Close()

    data, err := ioutil.ReadAll(file)
    if err != nil {
        return "", fmt.Errorf("failed to read file: %v", err)
    }

    return string(data), nil
}

func main() {
    // Define flags
    inputFile := flag.String("file", "", "Path to the input LLDP data file")
    testMode := flag.Bool("test", false, "Print the logger command instead of executing it")
    flag.Parse()

    var lldpData string
    var err error

    if *inputFile != "" {
        fmt.Printf("[VERBOSE] Reading LLDP data from file: %s\n", *inputFile)
        lldpData, err = readFile(*inputFile)
        if err != nil {
            fmt.Printf("[ERROR] Failed to read input file: %v\n", err)
            os.Exit(1)
        }
    } else {
        fmt.Println("[VERBOSE] No input file provided. Retrieving LLDP data using clish...")
        lldpData, err = getLLDPDataFromClish()
        if err != nil {
            fmt.Printf("[ERROR] Failed to retrieve LLDP data using clish: %v\n", err)
            os.Exit(1)
        }
    }

    // Parse the LLDP data
    entries, err := parseLLDPData(lldpData)
    if err != nil {
        fmt.Printf("[ERROR] Error parsing LLDP data: %v\n", err)
        os.Exit(1)
    }

    // Process the LLDP data
    processLLDPData(entries, *testMode)
}