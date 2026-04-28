$ErrorActionPreference = "Stop"

$goHealth = Invoke-WebRequest http://localhost:8080/health -TimeoutSec 5
if ($goHealth.StatusCode -ne 200 -or $goHealth.Content.Trim() -ne "OK") {
    throw "go-service /health failed"
}

$pythonHealth = Invoke-RestMethod http://localhost:8001/health -TimeoutSec 5
if ($pythonHealth.status -ne "ok") {
    throw "python-ai /health failed"
}

$chatResp = Invoke-RestMethod -Method Post `
    -Uri http://localhost:8080/api/ai/chat `
    -ContentType "application/json" `
    -Body '{"message":"hello smoke"}' `
    -TimeoutSec 10

if ($chatResp.model -ne "mock-fastapi-v1") {
       throw "unexpected model from chat api"
   }

if ($chatResp.reply -ne "python-ai received: hello smoke") {
    throw "unexpected reply from chat api"
}

Write-Host "smoke test passed"

