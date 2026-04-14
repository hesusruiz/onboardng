# Onboarding System

A simple, secure onboarding system designed for user registration and automated credential issuance.

## Overview

The Onboarding System allows users to register their data via a web form. The process includes:
1. **Email Validation**: Users enter their email and receive a validation code.
2. **Data Entry**: Users provide registration details (VAT ID, company info, etc.).
3. **Automated Processing**:
   - Data is stored in a SQLite database.
   - An external API (TMForum/Credential Issuance) is called to register the data.
   - Emails are sent to the user and administrators.
4. **Completion**: Users are presented with a welcome page.

## Architectural Integrity

- **Decoupled Frontend**: Strictly static HTML (in `dist/browser`) which can be served from a CDN or static directory. No Go template rendering at runtime.
- **API-First Backend**: The Go server is a pure JSON API following the structure: `{ "success": boolean, "message": string, "data": ... }`.
- **Security**: 
  - CSRF protection via `X-Requested-With` header.
  - Post-Quantum Encryption (MLKEM768-X25519) for sensitive configuration.

## Commands

The application provides several commands for managing the lifecycle of the system.

### Running the Server
Starts the API backend and optionally watches for frontend changes.

```bash
./onboardng run [options]
```
**Options:**
- `-config`: Path to config file (`.yaml` for dev, `.age` for production). Default: `config.age`.
- `-watch`: Watch for changes in `src/` and automatically regenerate static files.
- `-env`: Environment to serve (`dev`, `pre`, or `pro`). Default: `dev`.
- `-port`: Port for the server. Default: `7777`.

### Generating Frontend
Regenerates the static HTML files from the templates found in the source directory.

```bash
./onboardng generate
```
This command reads the configuration and processes files in `src/pages/` and `src/layouts/` to produce static output in the destination directory (e.g., `docs/`).

### Sealing Configuration
Encrypts a plaintext YAML configuration file using Post-Quantum encryption.

```bash
./onboardng seal [options]
```
**Options:**
- `-in`: Plaintext YAML file to encrypt. Default: `config/config.yaml`.
- `-out`: Target encrypted file path. Default: `config.age`.

This command generates a new identity key, saves it to `config/age_secret_key.txt` (ignored by git), and produces the `.age` file.

## Configuration

- **`config/config.yaml`**: The primary configuration file containing environment settings, SMTP details, and API endpoints.
- **`AGE_SECRET_KEY`**: Environment variable used to decrypt the `.age` configuration file in production.
- **`docs/`**: The target directory for generated static files.
- **`src/`**: The source directory containing Alpine.js templates and assets.

## Getting Started

1. **Build**: Build the Go binary.
   ```bash
   go build -o onboardng main.go
   ```
2. **Generate**: Create the initial set of static files.
   ```bash
   ./onboardng generate
   ```
3. **Run**: Start the server in development mode.
   ```bash
   ./onboardng run -env dev -watch
   ```