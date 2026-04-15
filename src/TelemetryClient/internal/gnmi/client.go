package gnmi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gnmi-collector/internal/config"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// Notification represents a single gNMI notification with a timestamp and
// a list of path-keyed updates containing decoded JSON values.
type Notification struct {
	Timestamp int64    `json:"timestamp"`
	Updates   []Update `json:"updates"`
}

// Update represents a single gNMI update with its path and decoded JSON value.
type Update struct {
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// Client wraps a gNMI gRPC connection with credential injection and
// convenience methods for Get requests.
type Client struct {
	cfg      *config.Config
	conn     *grpc.ClientConn
	gnmi     gpb.GNMIClient
	username string
	password string
}

// NewClient creates a new gNMI client and establishes a gRPC connection.
func NewClient(cfg *config.Config) (*Client, error) {
	username, password := cfg.ResolveCredentials()

	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64 * 1024 * 1024)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	if cfg.Target.TLS.Enabled {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.Target.TLS.SkipVerify,
		}
		if cfg.Target.TLS.SkipVerify {
			log.Printf("WARN: TLS skip_verify is enabled — server certificate is NOT verified. " +
				"An attacker with network access could intercept gNMI credentials and telemetry data. " +
				"Set tls.ca_file with cert_auto_fetch for production use.")
		}
		// Load pinned CA cert if configured (TOFU bootstrap or manual pin).
		if cfg.Target.TLS.CAFile != "" {
			pool, err := BootstrapCert(cfg.TargetAddr(), cfg.Target.TLS.CAFile, cfg.Target.TLS.CertAutoFetch)
			if err != nil {
				return nil, fmt.Errorf("TLS cert setup: %w", err)
			}
			if pool != nil {
				tlsCfg.RootCAs = pool
				tlsCfg.InsecureSkipVerify = false // enforce verification when we have a pinned cert
			}
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Collection.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.TargetAddr(), opts...)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial %s: %w", cfg.TargetAddr(), err)
	}

	return &Client{
		cfg:      cfg,
		conn:     conn,
		gnmi:     gpb.NewGNMIClient(conn),
		username: username,
		password: password,
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// authContext returns a context with gNMI username/password metadata attached.
func (c *Client) authContext(ctx context.Context) context.Context {
	if c.username != "" || c.password != "" {
		md := metadata.Pairs("username", c.username, "password", c.password)
		return metadata.NewOutgoingContext(ctx, md)
	}
	return ctx
}

// Capabilities fetches the gNMI server capabilities.
func (c *Client) Capabilities(ctx context.Context) (*gpb.CapabilityResponse, error) {
	ctx = c.authContext(ctx)
	return c.gnmi.Capabilities(ctx, &gpb.CapabilityRequest{})
}

// Get performs a gNMI Get request for the given YANG path and returns the
// response as a list of decoded Notifications.
func (c *Client) Get(ctx context.Context, yangPath string) ([]Notification, error) {
	ctx = c.authContext(ctx)

	pathElems, err := parsePath(yangPath)
	if err != nil {
		return nil, fmt.Errorf("parsing YANG path %q: %w", yangPath, err)
	}

	encoding := resolveEncoding(c.cfg.Collection.Encoding)

	req := &gpb.GetRequest{
		Path:     []*gpb.Path{pathElems},
		Type:     gpb.GetRequest_STATE,
		Encoding: encoding,
	}

	resp, err := c.gnmi.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gNMI Get %q: %w", yangPath, err)
	}

	return decodeNotifications(resp)
}

// GetWithTimeout performs a Get with the configured timeout.
func (c *Client) GetWithTimeout(yangPath string) ([]Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.Collection.Timeout)
	defer cancel()
	return c.Get(ctx, yangPath)
}

// SubscribeOnce performs a gNMI Subscribe with ONCE mode, which requests
// the target to send the current state of all matching data and then close
// the stream. This is useful for devices (like SONiC) that return empty
// responses for bulk Get requests on list paths but respond correctly to
// Subscribe ONCE requests.
//
// The returned notifications are normalized to match Get-style output:
// leaf-level updates are merged into nested maps, and the notification
// prefix path (containing entity keys like [name=Ethernet0]) is preserved.
func (c *Client) SubscribeOnce(ctx context.Context, yangPath string) ([]Notification, error) {
	ctx = c.authContext(ctx)

	pathElems, err := parsePath(yangPath)
	if err != nil {
		return nil, fmt.Errorf("parsing YANG path %q: %w", yangPath, err)
	}

	encoding := resolveEncoding(c.cfg.Collection.Encoding)

	req := &gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			Subscribe: &gpb.SubscriptionList{
				Subscription: []*gpb.Subscription{
					{
						Path: pathElems,
						Mode: gpb.SubscriptionMode_TARGET_DEFINED,
					},
				},
				Mode:     gpb.SubscriptionList_ONCE,
				Encoding: encoding,
			},
		},
	}

	stream, err := c.gnmi.Subscribe(ctx)
	if err != nil {
		return nil, fmt.Errorf("opening subscribe-once stream: %w", err)
	}

	if err := stream.Send(req); err != nil {
		return nil, fmt.Errorf("sending subscribe-once request: %w", err)
	}

	// Collect raw subscribe responses, preserving the notification prefix
	const maxNotifications = 1000
	var allNotifs []Notification
	for {
		resp, err := stream.Recv()
		if err != nil {
			return nil, fmt.Errorf("subscribe-once recv: %w", err)
		}

		notif := decodeSubscribeResponseWithPrefix(resp)
		if notif == nil {
			// SyncResponse — ONCE stream is complete
			break
		}
		if len(allNotifs) >= maxNotifications {
			log.Printf("WARN: subscribe-once for %s reached %d notification cap — dropping further updates to prevent memory exhaustion", yangPath, maxNotifications)
			continue
		}
		allNotifs = append(allNotifs, *notif)
	}

	// Normalize: merge leaf-level scalar updates into tree-structured maps.
	// Subscribe responses send individual leaf values, but transformers
	// expect nested maps (the same structure that Get returns).
	return NormalizeSubscribeNotifications(allNotifs), nil
}

// SubscribeOnceWithTimeout performs a SubscribeOnce with the configured timeout.
func (c *Client) SubscribeOnceWithTimeout(yangPath string) ([]Notification, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.Collection.Timeout)
	defer cancel()
	return c.SubscribeOnce(ctx, yangPath)
}

// SubscriptionPath defines a single path to subscribe to.
type SubscriptionPath struct {
	YANGPath          string
	Mode              string // "sample" or "on_change"
	SampleInterval    time.Duration
	HeartbeatInterval time.Duration
	Name              string // Path config name for routing updates
	Table             string
}

// Subscribe opens a gNMI Subscribe stream for the given paths and calls
// the handler for each received SubscribeResponse. It blocks until the
// context is cancelled or the stream errors out.
func (c *Client) Subscribe(ctx context.Context, paths []SubscriptionPath, handler func(*gpb.SubscribeResponse) error) error {
	ctx = c.authContext(ctx)

	subs := make([]*gpb.Subscription, 0, len(paths))
	for _, p := range paths {
		pathElems, err := parsePath(p.YANGPath)
		if err != nil {
			return fmt.Errorf("parsing YANG path %q: %w", p.YANGPath, err)
		}

		sub := &gpb.Subscription{
			Path: pathElems,
		}

		if strings.EqualFold(p.Mode, "on_change") {
			sub.Mode = gpb.SubscriptionMode_ON_CHANGE
			// Default heartbeat for on_change: server-side liveness signal
			// so a silent connection is detected without waiting for data changes.
			if p.HeartbeatInterval > 0 {
				sub.HeartbeatInterval = uint64(p.HeartbeatInterval.Nanoseconds())
			} else {
				sub.HeartbeatInterval = uint64((2 * time.Minute).Nanoseconds())
			}
		} else {
			sub.Mode = gpb.SubscriptionMode_SAMPLE
			sub.SampleInterval = uint64(p.SampleInterval.Nanoseconds())
			// Always suppress unchanged values in sample mode to avoid
			// shipping redundant data every interval.
			sub.SuppressRedundant = true
			if p.HeartbeatInterval > 0 {
				sub.HeartbeatInterval = uint64(p.HeartbeatInterval.Nanoseconds())
			}
		}

		subs = append(subs, sub)
	}

	encoding := resolveEncoding(c.cfg.Collection.Encoding)

	req := &gpb.SubscribeRequest{
		Request: &gpb.SubscribeRequest_Subscribe{
			Subscribe: &gpb.SubscriptionList{
				Subscription: subs,
				Mode:         gpb.SubscriptionList_STREAM,
				Encoding:     encoding,
			},
		},
	}

	stream, err := c.gnmi.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("opening subscribe stream: %w", err)
	}

	if err := stream.Send(req); err != nil {
		return fmt.Errorf("sending subscribe request: %w", err)
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("subscribe recv: %w", err)
		}
		if err := handler(resp); err != nil {
			return fmt.Errorf("handler: %w", err)
		}
	}
}

// DecodeSubscribeResponse decodes a SubscribeResponse into Notifications,
// reusing the same decoding logic as Get responses. Does NOT preserve the
// notification prefix — use DecodeSubscribeResponseWithPrefix for full
// entity path reconstruction in streaming mode.
func DecodeSubscribeResponse(resp *gpb.SubscribeResponse) ([]Notification, error) {
	switch r := resp.GetResponse().(type) {
	case *gpb.SubscribeResponse_Update:
		notif := Notification{
			Timestamp: r.Update.GetTimestamp(),
		}
		for _, u := range r.Update.GetUpdate() {
			update := Update{
				Path: pathToString(u.GetPath()),
			}
			if val := u.GetVal(); val != nil {
				decoded, err := decodeTypedValue(val)
				if err != nil {
					log.Printf("WARN: decode subscribe value at %s: %v", update.Path, err)
					continue
				}
				update.Value = decoded
			}
			notif.Updates = append(notif.Updates, update)
		}
		return []Notification{notif}, nil
	case *gpb.SubscribeResponse_SyncResponse:
		// Sync indicates initial dump is complete; no data to process
		return nil, nil
	default:
		return nil, nil
	}
}

// DecodeSubscribeResponseWithPrefix decodes a SubscribeResponse while
// preserving the notification prefix path (entity keys like [name=Ethernet0]).
// Returns nil for SyncResponse. This produces full paths that transformers
// need for entity identification (e.g., interface name extraction).
func DecodeSubscribeResponseWithPrefix(resp *gpb.SubscribeResponse) ([]Notification, error) {
	notif := decodeSubscribeResponseWithPrefix(resp)
	if notif == nil {
		return nil, nil
	}
	return []Notification{*notif}, nil
}

// parsePath converts a YANG path string like
// "/openconfig-interfaces:interfaces/interface/state/counters"
// into a gnmi.Path with typed path elements.
func parsePath(path string) (*gpb.Path, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	elems := []*gpb.PathElem{}
	for _, seg := range strings.Split(path, "/") {
		if seg == "" {
			continue
		}

		elem := &gpb.PathElem{}

		// Handle key selectors: interface[name=eth1/1]
		if idx := strings.Index(seg, "["); idx != -1 {
			elem.Name = seg[:idx]
			keyStr := seg[idx:]
			elem.Key = map[string]string{}
			for _, kv := range parseKeys(keyStr) {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) == 2 {
					elem.Key[parts[0]] = parts[1]
				}
			}
		} else {
			elem.Name = seg
		}

		// Strip module prefix (openconfig-interfaces:interfaces → interfaces)
		if colonIdx := strings.Index(elem.Name, ":"); colonIdx != -1 {
			elem.Name = elem.Name[colonIdx+1:]
		}

		elems = append(elems, elem)
	}

	return &gpb.Path{Elem: elems}, nil
}

// parseKeys extracts key-value pairs from "[key1=val1][key2=val2]".
func parseKeys(s string) []string {
	var keys []string
	for s != "" {
		start := strings.Index(s, "[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "]")
		if end == -1 {
			break
		}
		keys = append(keys, s[start+1:start+end])
		s = s[start+end+1:]
	}
	return keys
}

// decodeNotifications converts a gNMI GetResponse into a list of Notifications.
func decodeNotifications(resp *gpb.GetResponse) ([]Notification, error) {
	var notifications []Notification

	for _, n := range resp.GetNotification() {
		notif := Notification{
			Timestamp: n.GetTimestamp(),
		}

		for _, u := range n.GetUpdate() {
			update := Update{
				Path: pathToString(u.GetPath()),
			}

			val := u.GetVal()
			if val != nil {
				decoded, err := decodeTypedValue(val)
				if err != nil {
					log.Printf("WARN: decode value at %s: %v", update.Path, err)
					continue
				}
				update.Value = decoded
			}

			notif.Updates = append(notif.Updates, update)
		}

		notifications = append(notifications, notif)
	}

	return notifications, nil
}

// decodeTypedValue converts a gNMI TypedValue into a Go value.
func decodeTypedValue(val *gpb.TypedValue) (interface{}, error) {
	switch v := val.GetValue().(type) {
	case *gpb.TypedValue_JsonVal:
		var result interface{}
		if err := json.Unmarshal(v.JsonVal, &result); err != nil {
			return string(v.JsonVal), nil
		}
		return result, nil
	case *gpb.TypedValue_JsonIetfVal:
		var result interface{}
		if err := json.Unmarshal(v.JsonIetfVal, &result); err != nil {
			return string(v.JsonIetfVal), nil
		}
		// RFC 7951 JSON_IETF encoding adds module prefixes to map keys
		// (e.g., "openconfig-interfaces:counters" instead of "counters").
		// Strip these prefixes so transformers can use plain YANG leaf names.
		result = stripModulePrefixes(result)
		// RFC 7951 also wraps data in a top-level container named after the
		// YANG module:node. After prefix stripping this becomes a single-key
		// map (e.g., {"counters": {...}}). Unwrap it so transformers see the
		// same flat structure they expect from JSON encoding.
		if m, ok := result.(map[string]interface{}); ok && len(m) == 1 {
			for _, inner := range m {
				return inner, nil
			}
		}
		return result, nil
	case *gpb.TypedValue_StringVal:
		return v.StringVal, nil
	case *gpb.TypedValue_IntVal:
		return v.IntVal, nil
	case *gpb.TypedValue_UintVal:
		return v.UintVal, nil
	case *gpb.TypedValue_BoolVal:
		return v.BoolVal, nil
	case *gpb.TypedValue_FloatVal:
		return v.FloatVal, nil
	case *gpb.TypedValue_BytesVal:
		return v.BytesVal, nil
	default:
		return nil, fmt.Errorf("unsupported TypedValue type: %T", v)
	}
}

// stripModulePrefixes recursively removes YANG module prefixes from all map
// keys in a decoded JSON_IETF structure. Per RFC 7951, JSON_IETF encodes
// keys as "module-name:leaf-name" for the top-level node and for any node
// from a different YANG module (augmentations). This function strips the
// "module-name:" prefix so that downstream code can use plain leaf names.
//
// Only map keys are stripped — string values (e.g., enum identifiers like
// "openconfig-bgp-types:IPV4_UNICAST") are left untouched.
func stripModulePrefixes(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		stripped := make(map[string]interface{}, len(val))
		for k, child := range val {
			newKey := k
			if idx := strings.Index(k, ":"); idx != -1 {
				newKey = k[idx+1:]
			}
			stripped[newKey] = stripModulePrefixes(child)
		}
		return stripped
	case []interface{}:
		for i, item := range val {
			val[i] = stripModulePrefixes(item)
		}
		return val
	default:
		return v
	}
}

// decodeSubscribeResponseWithPrefix decodes a single SubscribeResponse,
// preserving the notification prefix path. Returns nil for SyncResponse
// (which signals end of ONCE stream).
func decodeSubscribeResponseWithPrefix(resp *gpb.SubscribeResponse) *Notification {
	r, ok := resp.GetResponse().(*gpb.SubscribeResponse_Update)
	if !ok {
		return nil // SyncResponse or unknown
	}

	prefix := pathToString(r.Update.GetPrefix())

	notif := &Notification{
		Timestamp: r.Update.GetTimestamp(),
	}

	for _, u := range r.Update.GetUpdate() {
		relPath := pathToString(u.GetPath())
		fullPath := joinPaths(prefix, relPath)

		update := Update{Path: fullPath}
		if val := u.GetVal(); val != nil {
			decoded, err := decodeTypedValue(val)
			if err != nil {
				log.Printf("WARN: decode subscribe-once value at %s: %v", fullPath, err)
				continue
			}
			update.Value = decoded
		}
		notif.Updates = append(notif.Updates, update)
	}

	return notif
}

// joinPaths combines a prefix path and a relative path into a full path.
func joinPaths(prefix, relative string) string {
	prefix = strings.TrimSuffix(prefix, "/")
	relative = strings.TrimPrefix(relative, "/")
	if prefix == "" {
		return "/" + relative
	}
	if relative == "" {
		return prefix
	}
	return prefix + "/" + relative
}

// NormalizeSubscribeNotifications converts Subscribe-style notifications
// (many leaf-level scalar updates per notification) into Get-style
// notifications (one update per notification with a nested map value).
//
// Subscribe responses for list paths look like:
//
//	notification prefix: /interfaces/interface[name=Ethernet0]/state
//	updates: [{path: /counters/in-octets, val: 100}, {path: /admin-status, val: "UP"}, ...]
//
// Get responses look like:
//
//	notification updates: [{path: /interfaces/interface[name=Ethernet0]/state, val: {"counters":{"in-octets":100}, "admin-status":"UP"}}]
//
// This function converts from the Subscribe format to the Get format so
// that all existing transformers work without modification.
func NormalizeSubscribeNotifications(notifs []Notification) []Notification {
	var result []Notification
	for _, n := range notifs {
		if len(n.Updates) == 0 {
			continue
		}

		// If there's only one update and its value is already a map,
		// it's already in tree format (some servers send tree values).
		if len(n.Updates) == 1 {
			if _, ok := n.Updates[0].Value.(map[string]interface{}); ok {
				result = append(result, n)
				continue
			}
		}

		// Check if updates are leaf-level (scalar values) by looking
		// at the first few. If they're all scalars, build a tree.
		if hasLeafUpdates(n.Updates) {
			tree := buildTreeFromUpdates(n.Updates)
			// Use the common prefix from all update paths as the entity path
			entityPath := commonPathPrefix(n.Updates)
			result = append(result, Notification{
				Timestamp: n.Timestamp,
				Updates: []Update{{
					Path:  entityPath,
					Value: tree,
				}},
			})
		} else {
			result = append(result, n)
		}
	}
	return result
}

// hasLeafUpdates returns true if the majority of updates contain scalar
// (non-map, non-slice) values, indicating Subscribe leaf-level format.
func hasLeafUpdates(updates []Update) bool {
	if len(updates) <= 1 {
		return false
	}
	scalarCount := 0
	for _, u := range updates {
		switch u.Value.(type) {
		case map[string]interface{}, []interface{}:
			// tree or array value
		default:
			scalarCount++
		}
	}
	return scalarCount > len(updates)/2
}

// buildTreeFromUpdates constructs a nested map from leaf-level updates.
// Each update path is split into segments and used to create the tree
// structure, with the value placed at the leaf.
func buildTreeFromUpdates(updates []Update) map[string]interface{} {
	root := map[string]interface{}{}
	// Find the common prefix to strip from paths
	prefix := commonPathPrefix(updates)
	prefixLen := len(strings.Split(strings.Trim(prefix, "/"), "/"))
	if prefix == "/" || prefix == "" {
		prefixLen = 0
	}

	for _, u := range updates {
		path := strings.Trim(u.Path, "/")
		parts := splitPathSegments(path)
		if len(parts) <= prefixLen {
			continue
		}
		// Use only the relative parts (after the common prefix)
		relParts := parts[prefixLen:]
		setNestedValue(root, relParts, u.Value)
	}
	return root
}

// splitPathSegments splits a path into segments, handling key selectors.
// "interfaces/interface[name=Ethernet0]/state/counters" →
// ["interfaces", "interface[name=Ethernet0]", "state", "counters"]
func splitPathSegments(path string) []string {
	var segments []string
	var current strings.Builder
	depth := 0
	for _, ch := range path {
		switch ch {
		case '[':
			depth++
			current.WriteRune(ch)
		case ']':
			depth--
			current.WriteRune(ch)
		case '/':
			if depth == 0 {
				if s := current.String(); s != "" {
					segments = append(segments, s)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if s := current.String(); s != "" {
		segments = append(segments, s)
	}
	return segments
}

// setNestedValue sets a value deep inside a nested map structure,
// creating intermediate maps as needed. When a path segment contains
// a YANG list key selector (e.g., "cpu[index=0]"), it creates an array
// at that key and uses separate elements per list entity.
func setNestedValue(m map[string]interface{}, parts []string, value interface{}) {
	for i, part := range parts {
		mapKey := part
		bracketIdx := strings.Index(part, "[")

		// Non-list segment: simple map navigation
		if bracketIdx == -1 {
			if i == len(parts)-1 {
				m[mapKey] = value
			} else {
				if next, ok := m[mapKey]; ok {
					if nextMap, ok := next.(map[string]interface{}); ok {
						m = nextMap
					} else {
						nextMap := map[string]interface{}{}
						m[mapKey] = nextMap
						m = nextMap
					}
				} else {
					nextMap := map[string]interface{}{}
					m[mapKey] = nextMap
					m = nextMap
				}
			}
			continue
		}

		// List segment: create array entries per entity key.
		// e.g., "cpu[index=0]" → mapKey="cpu", keyName="index", keyVal="0"
		mapKey = part[:bracketIdx]
		keyName, keyVal := parseListKeySelector(part[bracketIdx:])

		// Get or create array at this map key
		arr, _ := m[mapKey].([]interface{})
		if arr == nil {
			arr = []interface{}{}
		}

		// Find existing element with matching key, or create new one
		elem := findArrayElement(arr, keyName, keyVal)
		if elem == nil {
			elem = map[string]interface{}{}
			if keyName != "" {
				elem[keyName] = keyVal
			}
			arr = append(arr, elem)
			m[mapKey] = arr
		}

		if i == len(parts)-1 {
			// Leaf at list element: merge value if map, else set key
			if valMap, ok := value.(map[string]interface{}); ok {
				for k, v := range valMap {
					elem[k] = v
				}
			} else {
				// Keep existing element data, just overwrite this key's value
				elem[keyName] = value
			}
		} else {
			m = elem
		}
	}
}

// parseListKeySelector extracts the key name and value from a bracket selector.
// "[index=0]" → ("index", "0"), "[name=Ethernet0]" → ("name", "Ethernet0")
func parseListKeySelector(bracket string) (string, string) {
	// Remove outer brackets
	inner := strings.TrimPrefix(bracket, "[")
	// Only parse first key if multiple selectors exist
	if idx := strings.Index(inner, "]"); idx != -1 {
		inner = inner[:idx]
	}
	parts := strings.SplitN(inner, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return inner, ""
}

// findArrayElement finds an element in an array with matching key field value.
func findArrayElement(arr []interface{}, keyName, keyVal string) map[string]interface{} {
	for _, item := range arr {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if v, ok := itemMap[keyName]; ok {
				if fmt.Sprintf("%v", v) == keyVal {
					return itemMap
				}
			}
		}
	}
	return nil
}

// commonPathPrefix finds the longest common path prefix across all updates.
func commonPathPrefix(updates []Update) string {
	if len(updates) == 0 {
		return "/"
	}

	firstParts := splitPathSegments(strings.Trim(updates[0].Path, "/"))

	commonLen := len(firstParts)
	for _, u := range updates[1:] {
		parts := splitPathSegments(strings.Trim(u.Path, "/"))
		maxLen := commonLen
		if len(parts) < maxLen {
			maxLen = len(parts)
		}
		newCommon := 0
		for i := 0; i < maxLen; i++ {
			if parts[i] == firstParts[i] {
				newCommon++
			} else {
				break
			}
		}
		commonLen = newCommon
	}

	if commonLen == 0 {
		return "/"
	}
	return "/" + strings.Join(firstParts[:commonLen], "/")
}

// HasNonEmptyValues returns true if at least one notification update contains
// a non-empty value. Used to detect when a gNMI Get returns structurally
// valid but content-empty responses (e.g., SONiC returns {} or "" for bulk
// list queries).
func HasNonEmptyValues(notifs []Notification) bool {
	for _, n := range notifs {
		for _, u := range n.Updates {
			switch v := u.Value.(type) {
			case map[string]interface{}:
				if len(v) > 0 {
					return true
				}
			case []interface{}:
				if len(v) > 0 {
					return true
				}
			case string:
				if v != "" {
					return true
				}
			case nil:
				// skip
			default:
				// Numeric/bool values are always considered non-empty
				return true
			}
		}
	}
	return false
}

// resolveEncoding maps the config encoding string to a gNMI Encoding enum.
func resolveEncoding(cfgEncoding string) gpb.Encoding {
	switch {
	case strings.EqualFold(cfgEncoding, "JSON_IETF"):
		return gpb.Encoding_JSON_IETF
	case strings.EqualFold(cfgEncoding, "PROTO"):
		return gpb.Encoding_PROTO
	default:
		return gpb.Encoding_JSON
	}
}

// pathToString converts a gnmi.Path to a human-readable string.
func pathToString(p *gpb.Path) string {
	if p == nil {
		return ""
	}
	var parts []string
	for _, elem := range p.GetElem() {
		s := elem.GetName()
		for k, v := range elem.GetKey() {
			s += fmt.Sprintf("[%s=%s]", k, v)
		}
		parts = append(parts, s)
	}
	return "/" + strings.Join(parts, "/")
}
