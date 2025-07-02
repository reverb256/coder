# Networking & Security for OpenCode and Agent-Zero Workspace Services

## Service Isolation

- Each workspace runs OpenCode and Agent-Zero in isolated containers.
- Services are only accessible within the workspace network by default.

## Port Exposure

- OpenCode: port 8081 (internal)
- Agent-Zero: port 8090 (internal)
- Only expose ports externally if required, and use authentication.

## Security Best Practices

- Store API keys and secrets using the `secrets` section in your template.
- Never hardcode secrets in scripts or configs.
- Use HTTPS for any external communication.
- Restrict network access to trusted endpoints.

## Health Checks

- Both services should expose `/health` endpoints.
- The template includes health checks for automatic monitoring.

## IDE Integration

- Use the provided VS Code extension recommendations for secure agent communication.
- Configure endpoints to use `localhost` or the workspace network.

## Troubleshooting

- Check service logs for startup or health check failures.
- Ensure secrets are correctly injected.
- Verify network policies if services are unreachable.
