# Client Registration

The scheduler uses API keys as client credentials. A user does not need a password account to use the API.

## Register from the dashboard

Open `/dashboard/` and select **Register**.

Enter:

- Name
- Email

The dashboard calls:

```http
POST /api/v1/auth/register
Content-Type: application/json
```

Example request:

```json
{
  "name": "Demo User",
  "email": "demo@example.com"
}
```

The response contains a newly generated client API key:

```json
{
  "user_id": "user_<uuid>",
  "name": "Demo User",
  "email": "demo@example.com",
  "api_key": "client_<random-token>",
  "type": "client",
  "scopes": [
    "jobs:submit",
    "jobs:read",
    "keys:read",
    "keys:revoke"
  ],
  "message": "Registration successful. Save your API key; it is your scheduler credential."
}
```

The dashboard stores the key in browser local storage and uses it as a Bearer token for subsequent scheduler requests.

## Register with curl

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo User","email":"demo@example.com"}'
```

Then use the returned `api_key`:

```bash
export API_KEY='client_...'
curl -s http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $API_KEY"
```

## Security model

Registration is intentionally the only public account-creation endpoint. Project, queue, job, worker, metrics, and key-management operations continue to require an authenticated API key.

The generated client key is the user's credential. It is returned during registration and should be saved securely. There is currently no password reset flow.
