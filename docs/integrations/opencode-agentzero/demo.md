# OpenCode + Agent-Zero Integration Demo Guide

This guide walks you through running a complete demonstration of the OpenCode and Agent-Zero integration using local mock services.

---

## Prerequisites

- Go 1.20+ installed
- Bash shell
- Ports 8081 and 8082 available

---

## 1. Start Mock Services

Open two terminals and run:

```bash
go run ./agentic/testdata/mock_opencode_server.go
go run ./agentic/testdata/mock_agentzero_server.go
```

Or use the helper script:

```bash
./scripts/demo_start.sh
```

---

## 2. Configure the Demo Environment

Copy the demo environment and config files:

```bash
cp agentic/.env.demo agentic/.env
cp agentic/config.demo.yaml agentic/config.yaml
```

---

## 3. Run the Integration Demo

Start the integration service or run tests:

```bash
go run ./agentic/example.go
# or
./scripts/demo_test.sh
```

---

## 4. Stopping the Demo

To stop all mock/demo services:

```bash
./scripts/demo_stop.sh
```

---

## 5. Testing

- Run unit and integration tests:
  ```bash
  go test ./agentic/...
  ```
- Test API endpoints using curl or Postman:
  - OpenCode: `curl http://localhost:8081/api/v1/users`
  - Agent-Zero: `curl http://localhost:8082/api/v1/status`

- Verify UI components load and interact with the mock APIs.

---

## 6. Troubleshooting

- **Port in use**: Ensure 8081 and 8082 are not occupied.
- **Mock server not responding**: Check terminal output for errors.
- **Integration errors**: Confirm `.env` and `config.yaml` point to the correct mock URLs.
- **Test failures**: Review logs and ensure all services are running.

---

## 7. Additional Notes

- The mock servers provide realistic API responses for integration testing.
- All scripts are located in `scripts/`.
- For advanced debugging, increase log verbosity in `config.demo.yaml`.
