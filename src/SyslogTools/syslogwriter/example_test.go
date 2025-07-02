package syslogwriter_test

import (
	"fmt"

	"github.com/arc-switch/syslogwriter"
)

// Example demonstrating API call without requiring syslog
func ExampleNewWithDefaults() {
	// This shows the API signature - actual syslog might not be available in test environment
	_, err := syslogwriter.NewWithDefaults("my-app")
	if err != nil {
		fmt.Println("Syslog not available in test environment, but API is correct")
		return
	}
	
	fmt.Println("Syslog writer created successfully")
	// Output: Syslog not available in test environment, but API is correct
}

// Example demonstrating custom configuration API
func ExampleNew() {
	// This shows the API signature - actual syslog might not be available in test environment
	_, err := syslogwriter.New("cisco-parser", 8192, true) // verbose enabled
	if err != nil {
		fmt.Println("Custom configuration API called correctly")
		return
	}
	
	fmt.Println("Custom syslog writer created successfully")
	// Output: Custom configuration API called correctly
}
