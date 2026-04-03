package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var openConfigPaths = []struct {
	name string
	path string
}{
	{"oc-interface-counters", "/openconfig-interfaces:interfaces/interface/state/counters"},
	{"oc-interface-status", "/openconfig-interfaces:interfaces/interface/state"},
	{"oc-if-ethernet", "/openconfig-if-ethernet:interfaces/interface/ethernet/state"},
	{"oc-bgp-neighbors", "/openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state"},
	{"oc-bgp-global", "/openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/global/state"},
	{"oc-lldp-neighbors", "/openconfig-lldp:lldp/interfaces/interface/neighbors"},
	{"oc-temperature", "/openconfig-platform:components/component/state/temperature"},
	{"oc-power-supply", "/openconfig-platform:components/component/power-supply"},
	{"oc-system-cpus", "/openconfig-system:system/cpus"},
	{"oc-system-memory", "/openconfig-system:system/memory"},
	{"oc-system-state", "/openconfig-system:system/state"},
	{"oc-platform-inventory", "/openconfig-platform:components/component"},
	{"oc-arp-table", "/openconfig-if-ip:interfaces/interface/subinterfaces/subinterface/ipv4/neighbors"},
	{"oc-mac-table", "/openconfig-network-instance:network-instances/network-instance/fdb/mac-table"},
	{"oc-transceiver", "/openconfig-platform:components/component/transceiver"},
	{"oc-transceiver-channel", "/openconfig-platform:components/component/transceiver/physical-channels"},
}

var nativeCiscoPaths = []struct {
	name string
	path string
}{
	{"nx-transceiver", "/System/intf-items/phys-items/PhysIf-list/phys-items"},
	{"nx-sys-cpu-summary", "/System/procsys-items/syscpusummary-items"},
	{"nx-sys-memory", "/System/procsys-items/sysmem-items"},
	{"nx-arp", "/System/arp-items/inst-items/dom-items/Dom-list/db-items/Db-list/adj-items/AdjEp-list"},
	{"nx-mac-table", "/System/mac-items/table-items/Table-list"},
	{"nx-bgp-peers", "/System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list"},
	{"nx-lldp", "/System/lldp-items/inst-items/if-items/If-list"},
	{"nx-env-sensor", "/System/ch-items/supslot-items/SupCSlot-list/sensor-items"},
	{"nx-env-psu", "/System/ch-items/psuslot-items/PsuSlot-list"},
	{"nx-inventory", "/System/ch-items"},
	{"nx-uptime", "/System/showversion-items"},
}

func main() {
	target := flag.String("target", "localhost:50051", "gNMI target address:port")
	user := flag.String("u", "", "username")
	pass := flag.String("p", "", "password")
	useTLS := flag.Bool("tls", false, "use TLS (skip-verify)")
	timeout := flag.Int("timeout", 10, "per-request timeout in seconds")
	capsOnly := flag.Bool("caps-only", false, "only show capabilities, skip path validation")
	dumpDir := flag.String("dump", "", "directory to dump raw JSON responses")
	interactive := flag.Bool("i", false, "interactive mode: prompt for YANG paths to query")
	nativeOnly := flag.Bool("native-only", false, "only validate native Cisco YANG paths")
	ocOnly := flag.Bool("oc-only", false, "only validate OpenConfig paths")
	queryPath := flag.String("get", "", "single gNMI Get for the given YANG path, print result and exit")
	subscribePath := flag.String("subscribe", "", "subscribe to a YANG path and print updates as JSON")
	subMode := flag.String("sub-mode", "once", "subscribe mode: once, sample, on_change")
	sampleInterval := flag.Int("interval", 10, "sample interval in seconds (for sample mode)")
	encodingStr := flag.String("encoding", "JSON", "gNMI encoding: JSON or JSON_IETF")
	flag.Parse()

	// Resolve encoding
	gnmiEncoding := gpb.Encoding_JSON
	if strings.EqualFold(*encodingStr, "JSON_IETF") {
		gnmiEncoding = gpb.Encoding_JSON_IETF
	}

	fmt.Println("============================================")
	fmt.Println("  gNMI Probe — YANG Path Validator")
	fmt.Println("============================================")
	fmt.Printf("Target: %s\n\n", *target)

	// Create dump directory if specified
	if *dumpDir != "" {
		if err := os.MkdirAll(*dumpDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: cannot create dump dir %s: %v\n", *dumpDir, err)
			os.Exit(1)
		}
		fmt.Printf("Dump directory: %s\n\n", *dumpDir)
	}

	// Connect
	conn, client, rpcCtx := connect(*target, *user, *pass, *useTLS, *timeout)
	defer conn.Close()

	// --- Capabilities ---
	fmt.Println("\n--- Capabilities ---")
	capsCtx, capsCancel := context.WithTimeout(rpcCtx, time.Duration(*timeout)*time.Second)
	defer capsCancel()

	caps, err := client.Capabilities(capsCtx, &gpb.CapabilityRequest{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Capabilities RPC failed: %v\n", err)
		os.Exit(1)
	}

	printCapabilities(caps)

	if *dumpDir != "" {
		dumpCapabilities(*dumpDir, caps)
	}

	if *capsOnly {
		fmt.Println("\n[Done] caps-only mode, skipping path validation.")
		os.Exit(0)
	}

	// --- Single path query mode ---
	if *queryPath != "" {
		fmt.Printf("\n--- Single Get: %s ---\n\n", *queryPath)
		doGet(client, rpcCtx, *queryPath, *timeout, *dumpDir, "query", gnmiEncoding)
		os.Exit(0)
	}

	// --- Subscribe mode ---
	if *subscribePath != "" {
		doSubscribe(client, rpcCtx, *subscribePath, *subMode, *sampleInterval, *dumpDir, gnmiEncoding)
		os.Exit(0)
	}

	// --- Interactive mode ---
	if *interactive {
		runInteractive(client, rpcCtx, *timeout, *dumpDir, gnmiEncoding)
		os.Exit(0)
	}

	// --- Batch path validation ---
	var paths []struct {
		name string
		path string
	}

	if !*nativeOnly {
		paths = append(paths, openConfigPaths...)
	}
	if !*ocOnly {
		paths = append(paths, nativeCiscoPaths...)
	}

	fmt.Println("\n--- YANG Path Validation ---")
	fmt.Printf("%-30s %-65s %s\n", "NAME", "PATH", "RESULT")
	fmt.Println(strings.Repeat("-", 130))

	passed := 0
	failed := 0
	for _, yp := range paths {
		status := validatePath(client, rpcCtx, yp.name, yp.path, *timeout, *dumpDir, gnmiEncoding)
		if strings.HasPrefix(status, "OK") {
			passed++
		} else {
			failed++
		}
		fmt.Printf("%-30s %-65s %s\n", yp.name, truncate(yp.path, 65), status)
	}

	fmt.Println(strings.Repeat("-", 130))
	fmt.Printf("\nResults: %d passed, %d failed, %d total\n", passed, failed, passed+failed)

	if *dumpDir != "" {
		fmt.Printf("\nJSON responses saved to: %s/\n", *dumpDir)
	}
}

// connect establishes a gRPC connection and returns the client and auth context.
func connect(target, user, pass string, useTLS bool, timeout int) (*grpc.ClientConn, gpb.GNMIClient, context.Context) {
	var opts []grpc.DialOption
	if useTLS {
		opts = append(opts, grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}),
		))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, append(opts, grpc.WithBlock())...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: cannot connect to %s: %v\n", target, err)
		os.Exit(1)
	}
	fmt.Println("[OK] Connected to gNMI server")

	client := gpb.NewGNMIClient(conn)

	rpcCtx := context.Background()
	if user != "" {
		rpcCtx = metadata.AppendToOutgoingContext(rpcCtx, "username", user, "password", pass)
	}

	return conn, client, rpcCtx
}

// printCapabilities prints a summary of the gNMI capabilities response.
func printCapabilities(caps *gpb.CapabilityResponse) {
	fmt.Printf("gNMI version: %s\n", caps.GNMIVersion)
	fmt.Printf("Supported models: %d\n", len(caps.SupportedModels))

	ocCount := 0
	nativeCount := 0
	for _, m := range caps.SupportedModels {
		if strings.HasPrefix(m.Name, "openconfig") {
			ocCount++
		} else {
			nativeCount++
		}
	}
	fmt.Printf("  OpenConfig models: %d\n", ocCount)
	fmt.Printf("  Native/other models: %d\n\n", nativeCount)

	fmt.Println("All models:")
	for _, m := range caps.SupportedModels {
		ver := m.Version
		if ver == "" {
			ver = "n/a"
		}
		prefix := "  "
		if !strings.HasPrefix(m.Name, "openconfig") {
			prefix = "* "
		}
		fmt.Printf("%s%-50s %-15s %s\n", prefix, m.Name, ver, m.Organization)
	}

	fmt.Println("\nSupported encodings:")
	for _, e := range caps.SupportedEncodings {
		fmt.Printf("  %s\n", e.String())
	}
}

// doGet performs a single gNMI Get and pretty-prints the result.
func doGet(client gpb.GNMIClient, rpcCtx context.Context, yangPath string, timeout int, dumpDir string, dumpName string, encoding gpb.Encoding) {
	pathElems := parsePath(yangPath)
	getCtx, getCancel := context.WithTimeout(rpcCtx, time.Duration(timeout)*time.Second)
	defer getCancel()

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{pathElems},
		Type:     gpb.GetRequest_STATE,
		Encoding: encoding,
	}

	resp, err := client.Get(getCtx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}

	if len(resp.Notification) == 0 {
		fmt.Println("(empty response — no notifications)")
		return
	}

	totalUpdates := 0
	for _, n := range resp.Notification {
		totalUpdates += len(n.Update)
	}
	fmt.Printf("Got %d notification(s), %d update(s)\n\n", len(resp.Notification), totalUpdates)

	// Pretty-print each update
	for ni, n := range resp.Notification {
		for ui, u := range n.Update {
			path := pathToString(u.Path)
			fmt.Printf("--- [%d/%d] %s ---\n", ni*len(n.Update)+ui+1, totalUpdates, path)

			if u.Val != nil {
				var raw json.RawMessage
				if u.Val.GetJsonVal() != nil {
					raw = u.Val.GetJsonVal()
				} else if u.Val.GetJsonIetfVal() != nil {
					raw = u.Val.GetJsonIetfVal()
				} else {
					fmt.Printf("%s\n\n", u.Val.String())
					continue
				}

				// Pretty print JSON
				var pretty interface{}
				if err := json.Unmarshal(raw, &pretty); err == nil {
					out, _ := json.MarshalIndent(pretty, "", "  ")
					fmt.Printf("%s\n\n", string(out))
				} else {
					fmt.Printf("%s\n\n", string(raw))
				}
			}
		}
	}

	if dumpDir != "" && dumpName != "" {
		dumpResponse(dumpDir, dumpName, resp)
		fmt.Printf("(saved to %s/%s.json)\n", dumpDir, dumpName)
	}
}

// doSubscribe opens a gNMI Subscribe stream and prints each response as JSON.
func doSubscribe(client gpb.GNMIClient, rpcCtx context.Context, yangPath, mode string, intervalSec int, dumpDir string, encoding gpb.Encoding) {
	pathElems := parsePath(yangPath)

	// Determine subscription mode
	var subMode gpb.SubscriptionMode
	var listMode gpb.SubscriptionList_Mode

	switch strings.ToLower(mode) {
	case "once":
		listMode = gpb.SubscriptionList_ONCE
		subMode = gpb.SubscriptionMode_TARGET_DEFINED
	case "sample":
		listMode = gpb.SubscriptionList_STREAM
		subMode = gpb.SubscriptionMode_SAMPLE
	case "on_change":
		listMode = gpb.SubscriptionList_STREAM
		subMode = gpb.SubscriptionMode_ON_CHANGE
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown sub-mode %q (use: once, sample, on_change)\n", mode)
		os.Exit(1)
	}

	sub := &gpb.Subscription{
		Path: pathElems,
		Mode: subMode,
	}
	if subMode == gpb.SubscriptionMode_SAMPLE {
		sub.SampleInterval = uint64(time.Duration(intervalSec) * time.Second)
	}

	req := &gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			Subscribe: &gpb.SubscriptionList{
				Subscription: []*gpb.Subscription{sub},
				Mode:         listMode,
				Encoding:     encoding,
			},
		},
	}

	fmt.Printf("\n--- Subscribe: %s ---\n", yangPath)
	fmt.Printf("Mode: %s", strings.ToUpper(mode))
	if subMode == gpb.SubscriptionMode_SAMPLE {
		fmt.Printf(", Interval: %ds", intervalSec)
	}
	fmt.Printf("\nPress Ctrl+C to stop\n\n")

	// Trap Ctrl+C for clean exit
	ctx, cancel := context.WithCancel(rpcCtx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\n[Interrupted] Closing subscribe stream...\n")
		cancel()
	}()

	stream, err := client.Subscribe(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: opening subscribe stream: %v\n", err)
		return
	}

	if err := stream.Send(req); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: sending subscribe request: %v\n", err)
		return
	}

	updateNum := 0
	dumpNum := 0
	for {
		resp, err := stream.Recv()
		if err != nil {
			if ctx.Err() != nil {
				// Clean shutdown via Ctrl+C
				break
			}
			if err == io.EOF {
				fmt.Println("[Stream closed by server]")
				break
			}
			fmt.Fprintf(os.Stderr, "ERROR: recv: %v\n", err)
			break
		}

		switch r := resp.GetResponse().(type) {
		case *gpb.SubscribeResponse_Update:
			updateNum++
			ts := time.Unix(0, r.Update.GetTimestamp())
			fmt.Printf("=== Update #%d @ %s ===\n", updateNum, ts.Format("15:04:05.000"))

			for _, u := range r.Update.GetUpdate() {
				path := pathToString(u.GetPath())
				fmt.Printf("  Path: %s\n", path)

				if val := u.GetVal(); val != nil {
					printTypedValue(val)
				}
				fmt.Println()
			}

			// Also show deletes if any
			for _, d := range r.Update.GetDelete() {
				fmt.Printf("  DELETE: %s\n", pathToString(d))
			}

			// Dump to file if requested
			if dumpDir != "" {
				dumpNum++
				dumpSubscribeUpdate(dumpDir, dumpNum, r.Update)
			}

		case *gpb.SubscribeResponse_SyncResponse:
			fmt.Println("--- [sync] initial state complete ---\n")
			if listMode == gpb.SubscriptionList_ONCE {
				fmt.Println("[Done] ONCE mode — received all initial data.")
				return
			}

		default:
			fmt.Printf("(unknown response type: %T)\n", resp.GetResponse())
		}
	}

	fmt.Printf("\nReceived %d update(s) total.\n", updateNum)
}

// printTypedValue pretty-prints a gNMI TypedValue to stdout.
func printTypedValue(val *gpb.TypedValue) {
	switch v := val.GetValue().(type) {
	case *gpb.TypedValue_JsonVal:
		var pretty interface{}
		if err := json.Unmarshal(v.JsonVal, &pretty); err == nil {
			out, _ := json.MarshalIndent(pretty, "", "    ")
			fmt.Printf("  Value (JSON):\n    %s\n", strings.ReplaceAll(string(out), "\n", "\n    "))
		} else {
			fmt.Printf("  Value (raw): %s\n", string(v.JsonVal))
		}
	case *gpb.TypedValue_JsonIetfVal:
		var pretty interface{}
		if err := json.Unmarshal(v.JsonIetfVal, &pretty); err == nil {
			out, _ := json.MarshalIndent(pretty, "", "    ")
			fmt.Printf("  Value (JSON_IETF):\n    %s\n", strings.ReplaceAll(string(out), "\n", "\n    "))
		} else {
			fmt.Printf("  Value (raw): %s\n", string(v.JsonIetfVal))
		}
	case *gpb.TypedValue_StringVal:
		fmt.Printf("  Value (string): %s\n", v.StringVal)
	case *gpb.TypedValue_IntVal:
		fmt.Printf("  Value (int): %d\n", v.IntVal)
	case *gpb.TypedValue_UintVal:
		fmt.Printf("  Value (uint): %d\n", v.UintVal)
	case *gpb.TypedValue_BoolVal:
		fmt.Printf("  Value (bool): %v\n", v.BoolVal)
	case *gpb.TypedValue_FloatVal:
		fmt.Printf("  Value (float): %f\n", v.FloatVal)
	case *gpb.TypedValue_BytesVal:
		fmt.Printf("  Value (bytes): %d bytes\n", len(v.BytesVal))
	default:
		fmt.Printf("  Value (unknown type): %v\n", val)
	}
}

// dumpSubscribeUpdate saves a single subscribe notification to a JSON file.
func dumpSubscribeUpdate(dir string, num int, notif *gpb.Notification) {
	type updateEntry struct {
		Path  string      `json:"path"`
		Value interface{} `json:"value"`
	}
	out := struct {
		Timestamp string        `json:"timestamp"`
		Updates   []updateEntry `json:"updates"`
		Deletes   []string      `json:"deletes,omitempty"`
	}{
		Timestamp: time.Unix(0, notif.GetTimestamp()).Format(time.RFC3339Nano),
	}

	for _, u := range notif.GetUpdate() {
		entry := updateEntry{Path: pathToString(u.GetPath())}
		if val := u.GetVal(); val != nil {
			switch v := val.GetValue().(type) {
			case *gpb.TypedValue_JsonVal:
				var parsed interface{}
				json.Unmarshal(v.JsonVal, &parsed)
				entry.Value = parsed
			case *gpb.TypedValue_JsonIetfVal:
				var parsed interface{}
				json.Unmarshal(v.JsonIetfVal, &parsed)
				entry.Value = parsed
			default:
				entry.Value = val.String()
			}
		}
		out.Updates = append(out.Updates, entry)
	}
	for _, d := range notif.GetDelete() {
		out.Deletes = append(out.Deletes, pathToString(d))
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	filename := fmt.Sprintf("subscribe-%04d.json", num)
	os.WriteFile(filepath.Join(dir, filename), data, 0644)
}

// validatePath tests a single path and returns a status string.
func validatePath(client gpb.GNMIClient, rpcCtx context.Context, name, yangPath string, timeout int, dumpDir string, encoding gpb.Encoding) string {
	pathElems := parsePath(yangPath)
	getCtx, getCancel := context.WithTimeout(rpcCtx, time.Duration(timeout)*time.Second)
	defer getCancel()

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{pathElems},
		Type:     gpb.GetRequest_STATE,
		Encoding: encoding,
	}

	resp, err := client.Get(getCtx, req)
	if err != nil {
		return fmt.Sprintf("FAIL (%s)", shortenErr(err))
	}
	if len(resp.Notification) == 0 {
		return "FAIL (empty response)"
	}

	updates := 0
	for _, n := range resp.Notification {
		updates += len(n.Update)
	}

	if dumpDir != "" {
		dumpResponse(dumpDir, name, resp)
	}

	return fmt.Sprintf("OK (%d notifications, %d updates)", len(resp.Notification), updates)
}

// runInteractive enters a REPL loop where the user types YANG paths to query.
func runInteractive(client gpb.GNMIClient, rpcCtx context.Context, timeout int, dumpDir string, encoding gpb.Encoding) {
	scanner := bufio.NewScanner(os.Stdin)
	queryNum := 0

	fmt.Println("\n============================================")
	fmt.Println("  Interactive Mode")
	fmt.Println("  Type a YANG path to query (gNMI Get)")
	fmt.Println("  Prefix with 'sub ' to subscribe (ONCE)")
	fmt.Println("  Prefix with 'stream ' to subscribe (SAMPLE, 10s)")
	fmt.Println("  Type 'quit' or Ctrl+C to exit")
	fmt.Println("============================================")
	fmt.Println("\nExamples:")
	fmt.Println("  /openconfig-interfaces:interfaces/interface/state")
	fmt.Println("  sub /openconfig-system:system/cpus")
	fmt.Println("  stream /openconfig-interfaces:interfaces/interface/state/counters")
	fmt.Println()

	for {
		fmt.Print("gnmi> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" || line == "q" {
			fmt.Println("Bye.")
			break
		}
		if line == "help" || line == "?" {
			printInteractiveHelp()
			continue
		}
		if line == "paths" {
			printAllPaths()
			continue
		}

		// Handle "sub <path>" and "stream <path>" prefixes
		if strings.HasPrefix(line, "sub ") {
			subPath := strings.TrimSpace(strings.TrimPrefix(line, "sub"))
			if !strings.HasPrefix(subPath, "/") {
				subPath = "/" + subPath
			}
			fmt.Printf("\nSubscribing (ONCE): %s\n\n", subPath)
			doSubscribe(client, rpcCtx, subPath, "once", 10, dumpDir, encoding)
			fmt.Println()
			continue
		}
		if strings.HasPrefix(line, "stream ") {
			subPath := strings.TrimSpace(strings.TrimPrefix(line, "stream"))
			if !strings.HasPrefix(subPath, "/") {
				subPath = "/" + subPath
			}
			fmt.Printf("\nSubscribing (STREAM/SAMPLE 10s): %s\n", subPath)
			fmt.Println("Press Ctrl+C to stop the stream\n")
			doSubscribe(client, rpcCtx, subPath, "sample", 10, dumpDir, encoding)
			fmt.Println()
			continue
		}

		// Ensure path starts with /
		if !strings.HasPrefix(line, "/") {
			line = "/" + line
		}

		queryNum++
		dumpName := ""
		if dumpDir != "" {
			dumpName = fmt.Sprintf("interactive-%03d", queryNum)
		}

		fmt.Printf("\nQuerying: %s\n\n", line)
		doGet(client, rpcCtx, line, timeout, dumpDir, dumpName, encoding)
		fmt.Println()
	}
}

func printInteractiveHelp() {
	fmt.Println("\nCommands:")
	fmt.Println("  <yang-path>          Do a gNMI Get on the given path")
	fmt.Println("  sub <yang-path>      Subscribe ONCE (get initial state, then exit)")
	fmt.Println("  stream <yang-path>   Subscribe STREAM/SAMPLE every 10s (Ctrl+C to stop)")
	fmt.Println("  paths                List all built-in OpenConfig and native Cisco paths")
	fmt.Println("  help / ?             Show this help")
	fmt.Println("  quit / q             Exit interactive mode")
	fmt.Println()
}

func printAllPaths() {
	fmt.Println("\n--- OpenConfig Paths ---")
	for _, p := range openConfigPaths {
		fmt.Printf("  %-30s %s\n", p.name, p.path)
	}
	fmt.Println("\n--- Native Cisco NX-OS Paths ---")
	for _, p := range nativeCiscoPaths {
		fmt.Printf("  %-30s %s\n", p.name, p.path)
	}
	fmt.Println()
}

func dumpCapabilities(dir string, caps *gpb.CapabilityResponse) {
	type modelInfo struct {
		Name         string `json:"name"`
		Organization string `json:"organization"`
		Version      string `json:"version"`
	}

	out := struct {
		GNMIVersion     string      `json:"gnmi_version"`
		SupportedModels []modelInfo `json:"supported_models"`
		Encodings       []string    `json:"supported_encodings"`
	}{
		GNMIVersion: caps.GNMIVersion,
	}

	for _, m := range caps.SupportedModels {
		out.SupportedModels = append(out.SupportedModels, modelInfo{
			Name:         m.Name,
			Organization: m.Organization,
			Version:      m.Version,
		})
	}
	for _, e := range caps.SupportedEncodings {
		out.Encodings = append(out.Encodings, e.String())
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	os.WriteFile(filepath.Join(dir, "capabilities.json"), data, 0644)
}

func dumpResponse(dir, name string, resp *gpb.GetResponse) {
	type updateEntry struct {
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	}
	type notification struct {
		Timestamp int64         `json:"timestamp"`
		Updates   []updateEntry `json:"updates"`
	}

	var notifications []notification
	for _, n := range resp.Notification {
		notif := notification{Timestamp: n.Timestamp}
		for _, u := range n.Update {
			path := pathToString(u.Path)
			var val json.RawMessage
			if u.Val != nil {
				if u.Val.GetJsonVal() != nil {
					val = u.Val.GetJsonVal()
				} else if u.Val.GetJsonIetfVal() != nil {
					val = u.Val.GetJsonIetfVal()
				} else {
					val, _ = json.Marshal(u.Val.String())
				}
			}
			notif.Updates = append(notif.Updates, updateEntry{Path: path, Value: val})
		}
		notifications = append(notifications, notif)
	}

	data, _ := json.MarshalIndent(notifications, "", "  ")
	os.WriteFile(filepath.Join(dir, name+".json"), data, 0644)
}

func pathToString(p *gpb.Path) string {
	if p == nil {
		return ""
	}
	var parts []string
	for _, e := range p.Elem {
		s := e.Name
		for k, v := range e.Key {
			s += fmt.Sprintf("[%s=%s]", k, v)
		}
		parts = append(parts, s)
	}
	return "/" + strings.Join(parts, "/")
}

func parsePath(path string) *gpb.Path {
	path = strings.TrimPrefix(path, "/")
	segments := strings.Split(path, "/")
	var elems []*gpb.PathElem
	for _, s := range segments {
		if s == "" {
			continue
		}
		// Handle key selectors: e.g., interface[name=eth1/1]
		elem := &gpb.PathElem{}
		if idx := strings.Index(s, "["); idx != -1 {
			elem.Name = s[:idx]
			// Strip module prefix from name
			if ci := strings.Index(elem.Name, ":"); ci != -1 {
				elem.Name = elem.Name[ci+1:]
			}
			elem.Key = map[string]string{}
			keyPart := s[idx:]
			for keyPart != "" {
				start := strings.Index(keyPart, "[")
				end := strings.Index(keyPart, "]")
				if start == -1 || end == -1 {
					break
				}
				kv := keyPart[start+1 : end]
				eqIdx := strings.Index(kv, "=")
				if eqIdx != -1 {
					elem.Key[kv[:eqIdx]] = kv[eqIdx+1:]
				}
				keyPart = keyPart[end+1:]
			}
		} else {
			// Strip module prefix (e.g., "openconfig-interfaces:interfaces" → "interfaces")
			parts := strings.SplitN(s, ":", 2)
			if len(parts) == 2 {
				elem.Name = parts[1]
			} else {
				elem.Name = s
			}
		}
		elems = append(elems, elem)
	}
	return &gpb.Path{Elem: elems}
}

func shortenErr(err error) string {
	s := err.Error()
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
