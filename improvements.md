I want to improve the `credissuance` package. Here are several suggestions to improve its understandability and robustness.

  1. Robustness & Reliability

   * Consistent HTTP Client Usage:
       * Currently, LEARIssuance defines a custom httpClient with a 10-second timeout, but methods like TokenRequest still use http.DefaultClient.
       * Suggestion: Update all network-calling functions to use the httpClient from the LEARIssuance struct. This ensures that timeouts and other configurations are applied consistently across all requests.

   * Context Propagation:
       * The package makes several external network calls (Token Endpoint, TMF API, Issuance Endpoint).
       * Suggestion: Add context.Context as the first parameter to all functions that perform I/O. This allows the caller to cancel requests or enforce deadlines, which is critical for a robust backend system.

   * Error Handling in Request Creation:
       * Review code to aslways check for errors (when it makes sense according to the logic).

  2. Understandability & Maintainability

   * Decompose organization.go:
       * Review the iles composing the package for length or complexity, and make suggestions to improve the organization.

   * Consistent Naming Conventions:
       * Some functions use a TMF prefix (e.g., TMFListOrganizations), while others do not (e.g., TokenRequest).
       * Suggestion: Standardize the naming. Using the TMF prefix for all methods that interact with the TM Forum Party API helps distinguish them from local logic or other API calls.

   * Improve LEARIssuance Initialization:
       * NewLEARIssuance performs complex manual derivation of a did:key from a raw hex private key.
       * Suggestion: Consider moving this cryptographic derivation to a utility function or package (e.g., internal/crypto). This keeps the initialization logic focused on configuration and setup.

  3. Suggestions for access_token.go

   * Helper Method for Nested JWTs:
       * The logic for creating the nested JWT (Outer Assertion + Inner VP Token) is clear but repetitive in its claim setup (timestamps, nonces, audience).
       * Suggestion: Create a private helper method for setting standard registered claims to ensure consistency and reduce boilerplate.

   * Validation of Parameters:
       * Suggestion: Add basic validation for inputs like didkey and verifierURL before starting the expensive JWT signing process. For example, ensuring
         didkey starts with the expected prefix.

  4. Summary of Suggested Refactor (Strategic Approach)

   1. Step 1: Add context.Context to all API-calling methods.
   2. Step 2: Refactor LEARIssuance methods to use the internal httpClient.
   3. Step 3: Split organization.go to separate data models from API logic.
   4. Step 4: Replace hardcoded values and global variables with configuration-driven values.