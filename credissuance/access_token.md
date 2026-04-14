# Client Assertion Specification

This document specifies the structure and generation process for the Client Assertion used to authenticate against the Token Endpoint. This assertion follows the **JSON Web Token (JWT) Profile for OAuth 2.0 Client Authentication** ([RFC 7523](https://datatracker.ietf.org/doc/html/rfc7523)).

## Overview

The authentication mechanism uses a nested JWT structure. A "Client Assertion" (the outer JWT) is sent to the token endpoint. This assertion includes a specific claim, `vp_token`, which contains a Base64URL-encoded string representation of a "Verifiable Presentation Token" (the inner JWT).

## 1. Outer JWT: Client Assertion

The Client Assertion MUST be a signed JSON Web Signature (JWS) as defined in [RFC 7515](https://datatracker.ietf.org/doc/html/rfc7515).

### 1.1 Header

- **alg**: MUST be `ES256` (ECDSA using P-256 and SHA-256).
- **kid**: The Key ID, which SHOULD be the Decentralized Identifier (DID) of the client.
- **typ**: SHOULD be `JWT`.

### 1.2 Payload (Claims)

The payload MUST contain the following claims as defined in [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519#section-4.1) and [RFC 7523 Section 3](https://datatracker.ietf.org/doc/html/rfc7523#section-3):

- **iss** (Issuer): The DID of the client, specifically the `did:key` of the caller.
- **sub** (Subject): The DID of the client (usually the same as `iss`).
- **aud** (Audience): The URL of the Token Endpoint or Verifier.
- **exp** (Expiration Time): The time after which the JWT MUST NOT be accepted.
- **iat** (Issued At): The time at which the JWT was issued.
- **nbf** (Not Before): The time before which the JWT MUST NOT be accepted.
- **jti** (JWT ID): A unique identifier to prevent replay attacks.
- **vp_token**: A Base64URL-encoded string of the signed **Verifiable Presentation Token** (see Section 2).

## 2. Inner JWT: Verifiable Presentation Token

The `vp_token` claim contains a second signed JWT.

### 2.1 Header

- **alg**: MUST be `ES256`.
- **kid**: The DID of the client.

### 2.2 Payload (Claims)

- **iss** (Issuer): The DID of the client.
- **aud** (Audience): The URL of the Verifier.
- **exp**, **iat**, **nbf**, **jti**: Standard JWT claims.
- **nonce**: A unique string value to mitigate replay attacks.
- **vp**: A Verifiable Presentation object as defined by the [W3C Verifiable Credentials Data Model](https://www.w3.org/TR/vc-data-model/).
    - **@context**: MUST include `https://www.w3.org/2018/credentials/v1`.
    - **type**: MUST include `VerifiablePresentation`.
    - **holder**: The DID of the client.
    - **verifiableCredential**: An array containing the raw string representation of the Verifiable Credential(s) being presented (e.g., the LEARCredentialMachine).

## 3. The LEARCredentialMachine

The specific type of Verifiable Credential used in this process is the **LEARCredentialMachine**. It is used within the eIDAS framework to delegate specific powers to machines or workloads operated by a legal person (organization).

### 3.1 Structure Example
```json
{
  "sub": "did:key:zDnaeajw...",
  "iss": "did:elsi:VATES-B60645900",
  "vc": {
    "type": ["LEARCredentialMachine", "VerifiableCredential"],
    "credentialSubject": {
      "mandate": {
        "mandator": { "organization": "ALTIA CONSULTORES SA" },
        "mandatee": { "id": "did:key:zDnaeajw..." },
        "power": [{ "action": ["Execute"], "domain": "DOME", "function": "Onboarding" }]
      }
    }
  }
}
```

## 4. Access Token Request

Once the Client Assertion is generated, it is used to request an access token from the OAuth 2.0 Token Endpoint.

### 4.1 HTTP Request
The request MUST be a `POST` request with the `application/x-www-form-urlencoded` content type.

**Endpoint URL**: Provided by the verifier (e.g., `https://tmf.sbx.evidenceledger.eu/token`).

**Headers**:
```http
Content-Type: application/x-www-form-urlencoded
```

**Body Parameters**:
- `grant_type`: MUST be `client_credentials`.
- `client_id`: The DID of the client (the same used in `iss` and `sub`, which is the `did:key` of the machine).
- `client_assertion_type`: MUST be `urn:ietf:params:oauth:client-assertion-type:jwt-bearer`.
- `client_assertion`: The final signed string of the **Outer JWT (Client Assertion)**.

### 4.2 Example Request Body
```text
client_id=did:key:zDnae...&
grant_type=client_credentials&
client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer&
client_assertion=eyJhbGciOiJFUzI1NiIs...
```

### 4.3 Response
A successful response returns an `application/json` body containing the access token. Example:

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

## 5. Generation Summary

1.  **Construct the VP**: Assemble the Verifiable Presentation including the LEARCredentialMachine.
2.  **Generate VP Token (Inner JWT)**: Sign the VP as a JWT using `ES256`.
3.  **Encode VP Token**: Base64URL-encode the signed inner JWT string.
4.  **Generate Client Assertion (Outer JWT)**: Place the encoded VP Token in the `vp_token` claim and sign using `ES256`.
5.  **Request Token**: Send the Client Assertion to the Token Endpoint via a `POST` request.

## 6. Cryptographic Requirements

- **Signing Algorithm**: `ES256` ([RFC 7518 Section 3.4](https://datatracker.ietf.org/doc/html/rfc7518#section-3.4)).
- **Encoding**: All Base64 encoding MUST follow the "base64url" alphabet and MUST omit any trailing padding (`=`) characters ([RFC 4648 Section 5](https://datatracker.ietf.org/doc/html/rfc4648#section-5)).
