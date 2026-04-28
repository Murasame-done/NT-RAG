$ErrorActionPreference = "Stop"

$envFile = ".env.deploy"

if (-not (Test-Path $envFile)) {
    throw ".env.deploy not found"
}

docker compose --env-file $envFile -f .\docker-compose.deploy.yml pull
docker compose --env-file $envFile -f .\docker-compose.deploy.yml up -d
docker compose --env-file $envFile -f .\docker-compose.deploy.yml ps

$goHealth = Invoke-WebRequest http://localhost:8080/health -TimeoutSec 5
if ($goHealth.StatusCode -ne 200 -or $goHealth.Content.Trim() -ne "OK") {
    throw "go-service /health failed after deploy"
}

$pythonHealth = Invoke-RestMethod http://localhost:8001/health -TimeoutSec 5
if ($pythonHealth.status -ne "ok") {
    throw "python-ai /health failed after deploy"
}

Write-Host "deploy local passed"