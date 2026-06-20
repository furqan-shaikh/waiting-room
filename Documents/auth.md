# Control Plane API Authentication

This document captures the authentication design for Waiting Room Control Plane APIs.

## Problem

Control Plane APIs mutate waiting room configuration and should not be callable by arbitrary clients. This became especially important before adding broader admin APIs such as `LIST /waitingRooms`, because unauthenticated callers could read or mutate all waiting room resources.

The first goal is authentication:

- Identify that the caller is a trusted API client.
- Reject unsigned, invalid, expired, replayed, or unverifiable requests.
- Keep authorization out of scope for the first implementation.

Authorization will be handled later. For now, any authenticated caller is treated as an admin client.

## Options Considered

### Password / Shared Secret

- Basic Authentication
- API key
- HMAC-based API key

### Token-Based Authentication

- JWT bearer token
- OAuth 2.0 / OpenID Connect

### Cryptographic Request Signing

- Cavage HTTP Signatures
- RFC 9421 HTTP Message Signatures

### Certificate-Based Authentication

- mTLS client certificates

## Decision

Use **RFC 9421 HTTP Message Signatures** as the initial authentication method for Control Plane APIs.

The earlier idea was to use Cavage HTTP Signatures because OCI APIs use a Cavage-style request signature scheme. Cavage is historical now. RFC 9421 HTTP Message Signatures is the modern standards-track successor and evolved from the earlier Cavage draft.

## Why RFC 9421

- It is a current standard rather than an expired draft.
- It teaches the same core request-signing concepts as Cavage.
- It has a clearer model for covered components, derived components, signature parameters, `Signature-Input`, `Signature`, and body integrity through `Content-Digest`.
- It works well for API/CLI-style Control Plane clients.
- It can coexist with other authentication methods later if the server maps all methods to a common authenticated principal.

## Non-Goals

The first implementation does not include:

- Resource-level authorization.
- Tenant ownership checks.
- Browser login.
- OAuth/OIDC integration.
- Full key management APIs.
- Multi-key rotation workflows.

These can be added in later iterations.

## Authentication Model

The client owns a private key. The Control Plane knows the corresponding public key.

Each signed request includes:

- `Signature-Input`: describes which request components were signed and includes signature parameters.
- `Signature`: contains the cryptographic signature bytes.
- `Content-Digest`: required for requests with a body.

The server verifies:

1. Required signature headers are present.
2. `created`, `expires`, `keyid`, `alg`, and `nonce` are present in `Signature-Input`.
3. `expires` has not passed, is greater than `created`, and is within the maximum signature lifetime.
4. `keyid` maps to a trusted public key.
5. For body requests, the request body matches `Content-Digest`.
6. The server can reconstruct the same signature base from the request.
7. The signature verifies with the public key.
8. The `nonce` has not already been used for that `keyid`.

If verification succeeds, the request is authenticated.

## Initial Signing Profile

The first implementation uses a small project-specific RFC 9421 profile.

### Algorithm

Use `rsa-v1_5-sha256` for the first implementation.

This matches the current proof of concept in:

```text
src/controlplane/testdriver/client.go
```

### Covered Components

For requests without a body:

```text
"@method"
"@target-uri"
"@authority"
```

For requests with a body:

```text
"@method"
"@target-uri"
"@authority"
"content-type"
"content-length"
"content-digest"
```

### Signature Parameters

The first implementation includes:

```text
created
expires
keyid
alg
nonce
```

Example:

```text
Signature-Input: sig1=("@method" "@target-uri" "@authority" "content-type" "content-length" "content-digest");created=1717930000;keyid="key-1";alg="rsa-v1_5-sha256";nonce="b3k2pp5k7z-50gnwp.yemd";expires=1717930600
Signature: sig1=:Base64SignatureGoesHere:
```

## Signature Base

The signature is not generated over the `Signature-Input` header alone.

The client signs the RFC 9421 signature base. The signature base contains the canonical covered request components and a final `@signature-params` line.

Example for a `POST /waitingRooms` request:

```text
"@method": POST
"@target-uri": http://localhost:3000/waitingRooms
"@authority": localhost:3000
"content-type": application/json
"content-length": 64
"content-digest": sha-512=:Base64BodyDigest:
"@signature-params": ("@method" "@target-uri" "@authority" "content-type" "content-length" "content-digest");created=1717930000;keyid="key-1";alg="rsa-v1_5-sha256";nonce="b3k2pp5k7z-50gnwp.yemd";expires=1717930600
```

`@signature-params` must be the final line in the signature base.

The value used in the `Signature-Input` header must match the serialized value used in the `@signature-params` line.

## Content-Digest

`Content-Digest` is required for signed requests with a body.

Without `Content-Digest`, the signature can protect metadata such as method, URL, content type, and content length, but it does not prove the actual request body bytes.

For body requests, the client computes a digest of the body and sends it as a header:

```text
Content-Digest: sha-512=:Base64BodyDigest:
```

The server must verify that the received body matches the digest. Since `content-digest` is also included in the signature base, changing either the body or the digest breaks verification.

## Key Lookup

`keyid` is only an identifier. The server needs a trusted mapping:

```text
keyid -> public key
```

The first implementation can use a simple static key store so the authn verifier can be built without solving key management immediately.

Later, this should become a proper key management model, for example:

```text
api_clients
- client_id
- key_id
- public_key
- status
- created_at
- expires_at
```

## Bootstrap And Key Management

Full public key registration is out of scope for the first implementation.

Eventually, the project needs a way for clients to register or rotate public keys. The Control Plane should never store client private keys.

Open bootstrap options:

- Manual DB insert for local MVP.
- Static local development key configured at startup.
- One-time setup token.
- Admin-only key registration endpoint.

This will be handled in a later issue.

## Server-Side Design

The first server-side implementation is middleware around Control Plane routes.

Request flow:

1. Middleware receives the HTTP request.
2. Middleware parses `Signature-Input` and `Signature`.
3. Middleware extracts `created`, `expires`, `keyid`, `alg`, and `nonce`.
4. Middleware rejects expired signatures or signatures whose lifetime is greater than the configured maximum.
5. Middleware verifies `Content-Digest` when the request has a body.
6. Middleware loads the public key for `keyid`.
7. Middleware reconstructs the signature base.
8. Middleware verifies the signature.
9. Middleware atomically persists the nonce for `keyid`; duplicate nonce persistence fails authentication.
10. Middleware attaches an authenticated principal to request context.
11. Route handlers continue only for authenticated requests.

Rejected requests should return `401 Unauthorized`.

The authenticated principal should be generic enough to support more authentication methods later:

```text
principal.id
principal.auth_method
principal.key_id
```

This allows a future OIDC implementation to produce the same kind of internal identity:

```text
RFC 9421 signature -> principal: api-client-123
OIDC bearer token  -> principal: user@example.com
```

## Preventing Replay Attacks

Since HTTP message signatures allow sub-portions of the HTTP message to be signed, it is possible for two different HTTP messages to validate against the same signature. The most extreme form of this would be a signature over no message components. If such a signature were intercepted, it could be replayed at will by an attacker, attached to any HTTP message. Even with sufficient component coverage, a given signature could be applied to two similar HTTP messages, allowing a message to be replayed by an attacker with the signature intact.

To counteract these kinds of attacks, follow the below practices:

- Signer must cover sufficient portions of the message to differentiate it from other messages.
- Signature uses the `nonce` signature parameter to provide a per-message unique value.
- Signature uses `created` and `expires` to limit the useful lifetime of a captured signature.

### Using Nonce

- `nonce` must be a random, unique string generated by the signer.
- `nonce` must be sent as part of `Signature-Input`.
- `nonce` is covered by `@signature-params`, so changing it invalidates the signature.
- `nonce` is one-time use per `keyid`.

Example:

```text
Signature-Input: sig1=("@method" "@target-uri" "@authority" "content-type" "content-length" "content-digest");created=1717930000;keyid="key-1";alg="rsa-v1_5-sha256";nonce="b3k2pp5k7z-50gnwp.yemd";expires=1717930600
```

### Nonce Persistence

Control Plane stores nonce values in Postgres:

```sql
CREATE TABLE IF NOT EXISTS nonces(
    key_id TEXT NOT NULL,
    nonce_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (key_id, nonce_value)
);
```

The middleware calls:

```text
TryUseNonce(ctx context.Context, request models.Nonce) (bool, error)
```

`true` means the nonce was inserted and accepted. `false` means the nonce was already used or could not be accepted.

The implementation uses one insert and relies on the database uniqueness constraint on `(key_id, nonce_value)`. This makes duplicate nonce detection atomic even when the same signed request is replayed concurrently.

The nonce is persisted only after:

1. `Signature-Input` is parsed.
2. `expires` is validated.
3. `Content-Digest` is verified.
4. The message signature is verified.

This prevents invalid signatures from filling the nonce table.

## Signature Expiry

The signer includes `expires` in `Signature-Input`. `expires` is a Unix timestamp representing when the signer considers the signature invalid.

The verifier rejects the request when:

- `expires` is missing.
- current time is greater than `expires`.
- `expires <= created`.
- `expires - created` is greater than the configured maximum signature lifetime.

The current maximum signature lifetime is:

```text
15 minutes
```

Because `expires` is part of `@signature-params`, an attacker cannot extend the expiry without breaking signature verification.

## Local Test Driver

The local signed client lives in:

```text
src/controlplane/testdriver/client.go
```

Run it from the testdriver directory:

```bash
cd src/controlplane/testdriver
go run .
```

The driver generates a key pair, writes the public key file used by the local filesystem key repository, signs a `POST /waitingRooms` request, and sends it twice. Use an `expires` value within 15 minutes of `created` to test the success plus replay path; use an `expires` value beyond 15 minutes to test maximum-lifetime rejection.

## Testing

Important cases:

1. Send the exact same signed POST twice. First succeeds, second returns `401`.
2. Missing nonce returns `401`.
3. Bad signature with a new nonce does not persist the nonce.
4. Concurrent replay of the same signed request allows only one request to succeed.
5. Missing `expires` returns `401`.
6. Expired signature returns `401`.
7. `expires <= created` returns `401`.
8. Signature lifetime over the configured maximum returns `401`.

## Future Authentication Methods

RFC 9421 is the initial method, not necessarily the only method forever.

Future additions could include:

- OpenID Connect for browser/admin UI use cases.
- JWT bearer tokens for service integrations.
- mTLS for service-to-service deployments.

The route handlers should not depend on which authentication method produced the principal.

## References

- RFC 9421 HTTP Message Signatures: https://www.rfc-editor.org/info/rfc9421
- OAuth.net HTTP Signatures overview: https://oauth.net/http-signatures/
- OCI request signing documentation: https://docs.oracle.com/iaas/Content/API/Concepts/signingrequests.htm
