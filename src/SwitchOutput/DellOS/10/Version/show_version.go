package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// OSInfo represents the structure of the JSON output.
type OSInfo struct {
	OS struct {
		Name        string `json:"Name"`
		Copyright   string `json:"Copyright"`
		Version     string `json:"Version"`
		BuildVersion string `json:"BuildVersion"`
		BuildTime   string `json:"BuildTime"`
		SystemType  string `json:"SystemType"`
		Architecture string `json:"Architecture"`
		UpTime      string `json:"UpTime"`
	} `json:"OS"`
}

// parseFile processes the input file and converts the data to JSON format.
func parseFile(filePath string, verbose bool) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	osInfo := OSInfo{}

	for scanner.Scan() {
		line := scanner.Text()
		if verbose {
			fmt.Println("Processing line:", line)
		}

		if strings.HasPrefix(line, "Dell SmartFabric") {
			osInfo.OS.Name = line
		} else if strings.HasPrefix(line, "Copyright") {
			osInfo.OS.Copyright = line
		} else if strings.HasPrefix(line, "OS Version:") {
			osInfo.OS.Version = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Build Version:") {
			osInfo.OS.BuildVersion = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Build Time:") {
			osInfo.OS.BuildTime = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "System Type:") {
			osInfo.OS.SystemType = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Architecture:") {
			osInfo.OS.Architecture = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Up Time:") {
			osInfo.OS.UpTime = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	jsonData, err := json.Marshal(osInfo)
	if err != nil {
		return "", fmt.Errorf("failed to convert to JSON: %v", err)
	}

	return string(jsonData), nil
}

// logToSyslogger sends the JSON data to the syslogger.
func logToSyslogger(jsonData string, verbose bool) error {
	if verbose {
		fmt.Printf("Logger command: logger -p local0.info -t showVersion '%s';\n", jsonData)
		return nil
	}

	cmd := exec.Command("logger", "--size", "4096", "-p", "local0.info", "-t", "Version", jsonData)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send data to syslogger: %v", err)
	}
	return nil
}

func main() {
	// Define command-line flags.
	filePath := flag.String("file", "", "Path to the input file (e.g., show_version.txt)")
	verbose := flag.Bool("test", false, "Enable verbose output for testing")
	flag.Parse()

	// Validate input when the test flag is used.
	if *verbose && *filePath == "" {
		log.Fatalf("Error: The -test flag requires an input file specified with -file.")
	}

	var jsonData string
	var err error

	if *filePath != "" {
		// Process the input file.
		jsonData, err = parseFile(*filePath, *verbose)
		if err != nil {
			log.Fatalf("Error processing file: %v", err)
		}
	} else {
		// If no file is provided and not in test mode, execute the clish command.
		output, err := exec.Command("/opt/dell/os10/bin/clish", "-c", "show version").Output()
		if err != nil {
			log.Fatalf("Error executing clish command: %v", err)
		}

		// Write the output to a temporary file for processing.
		tempFile := "temp_show_version.txt"
		err = os.WriteFile(tempFile, output, 0644)
		if err != nil {
			log.Fatalf("Error writing temporary file: %v", err)
		}
		defer os.Remove(tempFile)

		jsonData, err = parseFile(tempFile, *verbose)
		if err != nil {
			log.Fatalf("Error processing clish output: %v", err)
		}
	}

	// Print the JSON data in verbose mode.
	if *verbose {
		fmt.Println("Generated JSON:", jsonData)
	}

	// Send the JSON data to the syslogger.
	err = logToSyslogger(jsonData, *verbose)
	if err != nil {
		log.Fatalf("Error logging to syslogger: %v", err)
	}

	fmt.Println("Data successfully processed and logged.")
}