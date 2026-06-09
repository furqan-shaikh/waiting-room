# Control Plane API Authentication

This document captures the authentication technical spike for Waiting Room Control Plane APIs and the intended implementation direction.

## Problem

Control Plane APIs are currently unauthenticated. This is acceptable for the early MVP, but it becomes unsafe before adding broader admin APIs such as `LIST /waitingRooms`, because any caller could read or mutate waiting room configuration.

The first goal is authentication:

- Identify that the caller is a trusted API client.
- Reject unsigned, invalid, expired, or unverifiable requests.
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

The first implementation will not include:

- Resource-level authorization.
- Tenant ownership checks.
- Browser login.
- OAuth/OIDC integration.
- Full key management APIs.
- Multi-key rotation workflows.
- Replay prevention using nonce storage.

These can be added in later iterations.

## Authentication Model

The client owns a private key. The Control Plane knows the corresponding public key.

Each signed request includes:

- `Signature-Input`: describes which request components were signed and includes signature parameters.
- `Signature`: contains the cryptographic signature bytes.
- `Content-Digest`: required for requests with a body.

The server verifies:

1. Required signature headers are present.
2. `created` is within the allowed clock skew.
3. `keyid` maps to a trusted public key.
4. For body requests, the request body matches `Content-Digest`.
5. The server can reconstruct the same signature base from the request.
6. The signature verifies with the public key.

If verification succeeds, the request is authenticated.

## Initial Signing Profile

For the first implementation, Waiting Room will use a small project-specific RFC 9421 profile.

### Algorithm

Use `rsa-v1_5-sha256` for the first implementation.

This matches the current proof of concept in:

```text
src/controlplane/authn/client.go
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

The first implementation should include:

```text
created
keyid
alg
```

Example:

```text
Signature-Input: sig1=("@method" "@target-uri" "@authority" "content-type" "content-length" "content-digest");created=1717930000;keyid="key-1";alg="rsa-v1_5-sha256"
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
"@signature-params": ("@method" "@target-uri" "@authority" "content-type" "content-length" "content-digest");created=1717930000;keyid="key-1";alg="rsa-v1_5-sha256"
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

The first server-side implementation should be middleware around Control Plane routes.

Request flow:

1. Middleware receives the HTTP request.
2. Middleware parses `Signature-Input` and `Signature`.
3. Middleware extracts `keyid`.
4. Middleware loads the public key for `keyid`.
5. Middleware verifies `Content-Digest` when the request has a body.
6. Middleware reconstructs the signature base.
7. Middleware verifies the signature.
8. Middleware attaches an authenticated principal to request context.
9. Route handlers continue only for authenticated requests.

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
