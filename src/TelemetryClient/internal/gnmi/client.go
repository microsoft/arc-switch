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
	}

	if cfg.Target.TLS.Enabled {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: cfg.Target.TLS.SkipVerify,
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

	encoding := gpb.Encoding_JSON
	if strings.EqualFold(c.cfg.Collection.Encoding, "PROTO") {
		encoding = gpb.Encoding_PROTO
	}

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

// SubscriptionPath defines a single path to subscribe to.
type SubscriptionPath struct {
	YANGPath       string
	Mode           string // "sample" or "on_change"
	SampleInterval time.Duration
	Name           string // Path config name for routing updates
	Table          string
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
		} else {
			sub.Mode = gpb.SubscriptionMode_SAMPLE
			sub.SampleInterval = uint64(p.SampleInterval.Nanoseconds())
		}

		subs = append(subs, sub)
	}

	encoding := gpb.Encoding_JSON
	if strings.EqualFold(c.cfg.Collection.Encoding, "PROTO") {
		encoding = gpb.Encoding_PROTO
	}

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
// reusing the same decoding logic as Get responses.
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
