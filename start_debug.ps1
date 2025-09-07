#!/usr/bin/env pwsh

# Script para iniciar WuzAPI com logs de debug habilitados
# Uso: .\start_debug.ps1

Write-Host "=== Iniciando WuzAPI com Debug Logs ===" -ForegroundColor Green

# Configurar nível de log para debug
$env:LOG_LEVEL = "debug"

# Verificar se o executável existe
if (-not (Test-Path "wuzapi.exe")) {
    if (Test-Path "wuzapi_debug.exe") {
        Write-Host "Usando wuzapi_debug.exe" -ForegroundColor Yellow
        .\wuzapi_debug.exe
    } else {
        Write-Host "Executável não encontrado. Compilando..." -ForegroundColor Yellow
        go build -o wuzapi_debug.exe .
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Compilação concluída!" -ForegroundColor Green
            .\wuzapi_debug.exe
        } else {
            Write-Host "Erro na compilação!" -ForegroundColor Red
            exit 1
        }
    }
} else {
    Write-Host "Usando wuzapi.exe" -ForegroundColor Yellow
    .\wuzapi.exe
}
