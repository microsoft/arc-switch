// SNMP Poll - Tool to retrieve SNMP MIB data from network devices
// Author: emarq
// Date: June 2025

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/sleepinggenius2/gosmi"
)

// Device represents an SNMP-enabled device
type Device struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      uint16 `json:"port"`
	Version   string `json:"version"`
	Community string `json:"community"`
	Timeout   int    `json:"timeout"`
}

// OIDConfig represents a configured OID to poll
type OIDConfig struct {
	Name        string `json:"name"`
	OID         string `json:"oid"`
	Description string `json:"description"`
}

// DeviceConfig contains all device configurations
type DeviceConfig struct {
	Devices []Device `json:"devices"`
}

// OIDList contains all OIDs to poll
type OIDList struct {
	OIDs []OIDConfig `json:"oids"`
}

// SNMPValue represents an SNMP value result
type SNMPValue struct {
	Name  string      `json:"name"`
	OID   string      `json:"oid"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

// SNMPResult represents poll results for a device
type SNMPResult struct {
	Device    string      `json:"device"`
	Timestamp string      `json:"timestamp"`
	Values    []SNMPValue `json:"values"`
}

func main() {
	configPath := flag.String("config", "config", "Path to configuration directory")
	mibsPath := flag.String("mibs", "mibs", "Path to MIB files directory")
	interval := flag.Int("interval", 30, "Polling interval in seconds")
	flag.Parse()

	// Initialize MIB system
	if err := initMIBs(*mibsPath); err != nil {
		log.Fatalf("Failed to initialize MIBs: %v", err)
	}

	// Load configurations
	devices, err := loadDevices(filepath.Join(*configPath, "devices.json"))
	if err != nil {
		log.Fatalf("Failed to load device config: %v", err)
	}

	oids, err := loadOIDs(filepath.Join(*configPath, "oids.json"))
	if err != nil {
		log.Fatalf("Failed to load OID config: %v", err)
	}

	// Prepare OIDs for polling
	var oidList []string
	for _, oid := range oids.OIDs {
		oidList = append(oidList, oid.OID)
	}

	// Main polling loop
	for {
		for _, device := range devices.Devices {
			log.Printf("Polling device: %s (%s)", device.Name, device.IP)
			result, err := pollDevice(device, oidList, oids.OIDs)
			if err != nil {
				log.Printf("Error polling device %s: %v", device.Name, err)
				continue
			}

			// Output results as JSON
			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				log.Printf("Error marshaling JSON: %v", err)
				continue
			}
			fmt.Println(string(jsonData))
		}

		// Wait for next polling interval
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

// initMIBs initializes the MIB system by loading MIBs from the specified directory
func initMIBs(mibsPath string) error {
	err := gosmi.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize gosmi: %v", err)
	}

	// Add the MIB path
	_, err = gosmi.LoadModule("")
	if err != nil {
		return fmt.Errorf("failed to load MIBs: %v", err)
	}

	return nil
}

// loadDevices loads device configuration from a JSON file
func loadDevices(filePath string) (DeviceConfig, error) {
	var config DeviceConfig

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return config, fmt.Errorf("failed to read device config file: %v", err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse device config: %v", err)
	}

	return config, nil
}

// loadOIDs loads OID configuration from a JSON file
func loadOIDs(filePath string) (OIDList, error) {
	var oids OIDList

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return oids, fmt.Errorf("failed to read OID config file: %v", err)
	}

	err = json.Unmarshal(data, &oids)
	if err != nil {
		return oids, fmt.Errorf("failed to parse OID config: %v", err)
	}

	return oids, nil
}

// pollDevice performs SNMP polling of a single device
func pollDevice(device Device, oids []string, oidConfigs []OIDConfig) (SNMPResult, error) {
	result := SNMPResult{
		Device:    device.Name,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Configure SNMP client
	client := &gosnmp.GoSNMP{
		Target:    device.IP,
		Port:      device.Port,
		Transport: "udp",
		Community: device.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(device.Timeout) * time.Second,
	}

	// Set SNMP version
	switch device.Version {
	case "v1":
		client.Version = gosnmp.Version1
	case "v3":
		client.Version = gosnmp.Version3
	default:
		client.Version = gosnmp.Version2c
	}

	// Connect to device
	err := client.Connect()
	if err != nil {
		return result, fmt.Errorf("failed to connect to device: %v", err)
	}
	defer client.Conn.Close()

	// Perform SNMP get
	response, err := client.Get(oids)
	if err != nil {
		return result, fmt.Errorf("SNMP get failed: %v", err)
	}

	// Process results
	for _, variable := range response.Variables {
		var value SNMPValue
		value.OID = variable.Name

		// Find the OID name from config
		for _, oid := range oidConfigs {
			if oid.OID == variable.Name {
				value.Name = oid.Name
				break
			}
		}

		// Set value based on type
		switch variable.Type {
		case gosnmp.OctetString:
			value.Value = string(variable.Value.([]byte))
			value.Type = "OctetString"
		case gosnmp.ObjectIdentifier:
			value.Value = variable.Value.(string)
			value.Type = "ObjectIdentifier"
		case gosnmp.Integer:
			value.Value = variable.Value.(int)
			value.Type = "Integer"
		case gosnmp.TimeTicks:
			value.Value = variable.Value.(uint)
			value.Type = "TimeTicks"
		case gosnmp.Gauge32:
			value.Value = variable.Value.(uint)
			value.Type = "Gauge32"
		case gosnmp.Counter32:
			value.Value = variable.Value.(uint)
			value.Type = "Counter32"
		case gosnmp.Counter64:
			value.Value = variable.Value.(uint64)
			value.Type = "Counter64"
		default:
			value.Value = fmt.Sprintf("%v", variable.Value)
			value.Type = variable.Type.String()
		}

		result.Values = append(result.Values, value)
	}

	return result, nil
}
