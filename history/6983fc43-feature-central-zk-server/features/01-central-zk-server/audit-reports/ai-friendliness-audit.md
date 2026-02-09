# AI-Friendliness Audit Report

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 3 |
| MAJOR | 5 |
| MINOR | 5 |

## CRITICAL Findings

### C1: TLS Configuration Not Specified
Spec mandates TLS but provides no guidance on:
- Certificate source (file paths, env vars)
- Certificate format (PEM, PKCS12)
- mTLS requirements
- How clients trust server cert

**Impact:** Cannot proceed without making security architecture decisions.

### C2: API Key Authentication Undefined
"Basic API key only" mentioned but zero details:
- Where key is configured server-side
- How client provides it (header, metadata)
- Key name/format
- Single vs multiple keys

**Impact:** Cannot secure the server.

### C3: Rate Limiting Strategy Unresolved
Open question blocks implementation - strategy varies dramatically:
- Per-client requires client identification
- Global requires shared state
- Both requires additional complexity

**Impact:** Cannot implement acceptance criterion.

## MAJOR Findings

1. LLM provider configuration unclear (routing, credentials)
2. Profile storage location ambiguous (database vs filesystem)
3. Database schema not referenced (existing tables, migrations)
4. Configuration hot-reload contradiction (proto has UpdateConfig, spec says no hot-reload)
5. Profile server-side validation unresolved

## MINOR Findings

1. Service table incomplete (missing HealthService)
2. Terminology inconsistency (initiative vs workflow)
3. Missing connection address configuration
4. "Conversation importer" referenced without context
5. Out of scope items reference unknown tickets
