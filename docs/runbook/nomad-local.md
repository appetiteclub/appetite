# Nomad Local Runbook

This short guide walks through running the Appetite stack on a local Nomad dev agent. It mirrors the Docker Compose setup and assumes Docker and Nomad are installed on your machine.

## 1. Build the Docker images

Nomad does not build images; it only asks Docker to run whatever tag you reference in the job. Make sure the service images exist locally before submitting the jobs:

```bash
docker compose -f deployments/docker/compose/docker-compose.yml build
```

After the build, confirm the tags are available:

```bash
docker images | grep appetite-
```

You should see `appetite-authn:latest`, `appetite-authz:latest`, `appetite-table:latest`, and `appetite-admin:latest`. If any tag is missing, rebuild that service manually, for example:

```bash
docker build -t appetite-authn:latest -f services/authn/Dockerfile .
```

## 2. Start a Nomad dev agent

Open a dedicated terminal and run:

```bash
nomad agent -dev -bind=0.0.0.0
```

Keep this process running; it exposes the Nomad API and UI on `http://127.0.0.1:4646`.

## 3. Point the CLI to the agent

In your working terminal export the address (adjust if the agent binds elsewhere):

```bash
export NOMAD_ADDR=http://127.0.0.1:4646
```

## 4. Submit the jobs

Register MongoDB and the application services:

```bash
make nomad-run
```

The `Makefile` passes the image tags built in step 1. If you need custom tags or a registry, override `NOMAD_AUTHN_IMAGE`, `NOMAD_AUTHZ_IMAGE`, `NOMAD_TABLE_IMAGE`, or `NOMAD_ADMIN_IMAGE` when invoking `make`.

## 5. Verify allocations

Check that the jobs are running:

```bash
nomad status appetite-services
nomad status mongodb
```

For more detail on a specific allocation:

```bash
nomad alloc status <alloc-id>
nomad alloc logs -f <alloc-id>
```

When `appetite-services` is healthy, the endpoints match the Compose ports. For example, `http://localhost:8081/list-users` should respond, and the Nomad UI shows the allocations under the *Services* tab.

## 6. Tear down

Stop the Nomad jobs from your working terminal:

```bash
make nomad-stop
```

Then stop the dev agent (Ctrl+C in the agent terminal). If you started it in the background, kill the process manually.

## 7. Troubleshooting tips

- **Connection refused errors** usually mean the dev agent is not running or Nomad is bound to a different address; re-check step 2/3.
- **Image not found errors** (`pull access denied`) indicate the service images were not built or the tag variables point to non-existent images. Re-run step 1 or override the `NOMAD_*_IMAGE` variables.
- **Port conflicts** arise when other processes listen on 8081–8084 or 27017. Stop them or adjust the Dockerfiles and job definitions before retrying.

This flow also works in CI/CD: build and push your images, set `NOMAD_ADDR` plus the image variables, and call `make nomad-run` from the pipeline.
