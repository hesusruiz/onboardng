# Frontend-Backend API Interface Documentation

This document describes the API interface between the frontend and the Go backend server. It is intended for developers and maintainers working on the frontend layer.

## General Information

### Base URL
All API endpoints are prefixed with `/api`.

### Common Response Structure
All API responses follow a consistent JSON structure as defined in `APIResponse`:

```json
{
  "success": boolean,
  "message": "Human-readable description of the result",
  "data": null | object | array
}
```

- `success`: `true` if the operation was successful, `false` otherwise.
- `message`: Provides context or error details.
- `data`: Optional payload containing relevant information (e.g., verification code during testing).

### Security Requirements

#### CSRF Protection
All `/api/*` requests (excluding `/health`) MUST include the following HTTP header to pass the security check:
```http
X-Requested-With: XMLHttpRequest
```
Failure to provide this header will result in a `403 Forbidden` response.

#### CORS
The backend enables a permissive CORS policy only in development (`dev`). In preproduction and production, CORS headers are omitted because the frontend is served from the same origin, allowing the browser's Same-Origin Policy to block cross-origin requests regardless of the hosted domain.

---

## API Endpoints

### 1. Validate Email
Starts the onboarding process by validating the email format and sending a verification code.

- **URL**: `/api/validate-email`
- **Method**: `POST`
- **Request Body**:
  ```json
  {
    "email": "user@example.com"
  }
  ```
- **Success Response (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Validation code sent to your email",
    "data": {
      "code": "123456"
    }
  }
  ```
  *Note: During development/testing, the code is returned in the `data` object for convenience.*
- **Error Responses**:
  - `400 Bad Request`: Email is missing or invalid.
  - `403 Forbidden`: Missing `X-Requested-With` header.
  - `429 Too Many Requests`: Rate limit reached (both IP-based and email-based).

---

### 2. Verify Code
Verifies the 6-digit code sent to the user's email.

- **URL**: `/api/verify-code`
- **Method**: `POST`
- **Request Body**:
  ```json
  {
    "email": "user@example.com",
    "code": "123456"
  }
  ```
- **Success Response (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Email verified successfully",
    "data": null
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: Invalid verification code or missing fields.
  - `403 Forbidden`: Missing `X-Requested-With` header.

---

### 3. Register
Submits the final registration form data to create a record and trigger the external issuance process.

- **URL**: `/api/register`
- **Method**: `POST`
- **Request Body**:
  ```json
  {
    "firstName": "John",
    "lastName": "Doe",
    "companyName": "Acme Corp",
    "country": "ES",
    "vatId": "B12345678",
    "streetAddress": "Calle Mayor 1",
    "postalCode": "28001",
    "email": "john.doe@example.com",
    "code": "123456",
    "website": ""
  }
  ```
  - **Security**: The `code` used for email verification MUST be sent again to verify the authenticity of the registration request.
  - **Important**: The `website` field is a **honeypot**. It must be left empty by real users. If it is populated, the server will treat the request as a bot, log it, and return a fake success without processing the data.
  - **Country**: Must be a valid 2-letter ISO country code.

- **Success Response (200 OK)**:
  ```json
  {
    "success": true,
    "message": "Registration successful",
    "data": null
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: Validation failure (missing required fields, invalid country, etc.).
  - `403 Forbidden`: Missing `X-Requested-With` header.
  - `500 Internal Server Error`: Server-side failure (e.g., database error).

