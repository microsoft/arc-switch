# gNMI Collector — TLS Security Design

## Current State

The gnmi-collector connects to network switches over gRPC/TLS:

- **TLS is enabled** (`tls.enabled: true` in both Cisco and SONiC configs)
- **Secure by default** — when TLS is enabled, the config must explicitly set either `skip_verify: true` or `ca_file`. This prevents accidentally running without any security posture.
- **`skip_verify: true`** logs a warning at startup about MITM risk
- **Authentication** uses gRPC metadata headers (`username`/`password`) read from environment variables
- **`ca_file`** loads a PEM cert and verifies the server against it. When a pinned cert is loaded, `skip_verify` is automatically overridden to `false`.
- **`cert_auto_fetch`** enables TOFU: auto-fetches the server cert on first connect (if `ca_file` is missing) and self-heals on cert rotation at runtime

## Threat Model

| Threat | Current mitigation | Risk level |
|--------|-------------------|------------|
| Eavesdropping on gNMI data | TLS encryption (even with skip_verify) | **Low** — encrypted |
| Credential theft via MITM | None — skip_verify doesn't verify server | **Medium** — requires network position |
| Credential storage | Environment variables (not on disk) | **Low** — standard practice |
| Replay attacks | gRPC over TLS prevents replay | **Low** |

In a datacenter environment, MITM risk is lower (controlled network), but defense-in-depth is important for production.

## Options Considered

### Option 1: `skip_verify` only (current state)

**How it works:** TLS encrypts the connection, but the server's certificate is not verified.

| ✅ Benefits | ❌ Risks |
|------------|---------|
| Zero configuration — works immediately | No server identity verification |
| No cert management needed | Vulnerable to MITM if attacker has network position |
| Compatible with any switch cert | Cannot detect cert rotation or tampering |

**Verdict:** Acceptable for dev/lab. Not sufficient for production. **Status:** Supported — requires explicit `skip_verify: true`. Logs a security warning at startup.

---

### Option 2: Manual cert pinning (`ca_file` loaded from disk)

**How it works:** Operator manually copies the switch's self-signed cert to the collector host and configures `ca_file`. The collector verifies the server cert against this pinned CA.

| ✅ Benefits | ❌ Risks |
|------------|---------|
| Full server identity verification | Manual process — error-prone at scale |
| No trust-on-first-use risk | Collector breaks silently on cert rotation |
| Operator explicitly trusts the cert | Requires re-deployment for every switch |

**Verdict:** Most secure, but operationally expensive. Not suitable for automated onboarding. **Status:** Supported — set `ca_file` with `cert_auto_fetch: false`.

---

### Option 3: Trust-on-first-use (TOFU) with auto-fetch ✅ Selected

**How it works:** On first connection, the collector probes the switch's TLS endpoint, fetches the server certificate, saves it to `ca_file`, and uses it for all subsequent connections. If the cert later changes (rotation, regeneration), the collector detects the TLS failure, re-fetches the new cert, and reconnects automatically.

| ✅ Benefits | ❌ Risks |
|------------|---------|
| Zero-touch setup — works with onboarding scripts | First connection trusts whatever cert is presented |
| Self-heals on cert rotation | Auto re-fetch on failure could trust a MITM cert |
| Compatible with self-signed certs | Brief window of unauthenticated trust during re-fetch |
| No manual cert distribution | |

**Mitigations for TOFU risks:**
- Log SHA-256 fingerprint of every fetched cert (old → new) at WARN level
- Only re-fetch when the failure is specifically a certificate verification error (not any TLS error)
- Atomic file writes prevent corruption during re-fetch
- Operators can disable auto-fetch and pin manually for high-security environments

**Verdict:** Best balance of security and operability for datacenter switch management. **Status:** Implemented — set `ca_file` + `cert_auto_fetch: true`.

---

### Option 4: Mutual TLS (mTLS)

**How it works:** Both the collector and the switch present certificates. Requires a PKI or certificate distribution system.

| ✅ Benefits | ❌ Risks |
|------------|---------|
| Strongest authentication model | Requires PKI infrastructure |
| Both sides verified | Complex certificate lifecycle management |
| Industry standard for production | Not all switches support client certs for gNMI |

**Verdict:** Ideal long-term goal. Out of scope for initial release — requires PKI infrastructure that doesn't exist today. **Status:** Not implemented.

## Selected Approach: TOFU + Self-Healing

### Behavior

```
Startup:
  if ca_file configured and file exists → load and verify server cert
  if ca_file configured and file missing + cert_auto_fetch → probe switch, save cert, then connect
  if ca_file not configured → use skip_verify (backward compatible)

Runtime (subscribe reconnect loop):
  if TLS cert verification fails → probe switch for current cert
    if cert changed → save new cert, log old/new fingerprint, reconnect
    if cert same → real TLS error, don't overwrite, propagate error
```

### Config Changes

```yaml
target:
  tls:
    enabled: true
    skip_verify: true           # Overridden to false automatically when ca_file is loaded
    ca_file: /etc/gnmi/server.pem
    cert_auto_fetch: true       # Enable TOFU + self-healing
```

### Security Logging

Every cert fetch logs at WARN level:
```
WARN: auto-fetched server certificate from 10.0.0.1:50051 (SHA-256: ab:cd:ef:...)
WARN: server certificate changed — old fingerprint: ab:cd:ef:..., new fingerprint: 12:34:56:...
```

## Cert Lifecycle on Switches

| Platform | Cert generation | Auto-rotation | Rotation trigger |
|----------|----------------|--------------|-----------------|
| Cisco NX-OS | Generated once on gNMI enable | No | Manual delete or hostname change |
| SONiC (Dell) | Generated by telemetry container | No | Container rebuild or manual delete |

Both platforms persist their self-signed certs across reboots. Cert changes are rare, manual events.

## Future Considerations

- **mTLS**: When PKI is available, add client cert support for bidirectional authentication
- **Cert expiry monitoring**: Log warnings when the pinned cert is approaching expiry
- **Azure Key Vault integration**: Store/retrieve certs from Key Vault instead of local files
