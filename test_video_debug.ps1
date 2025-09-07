#!/usr/bin/env pwsh

# Script para testar endpoint de video status com logging detalhado
# Uso: .\test_video_debug.ps1

$API_URL = "http://localhost:3000"
$TOKEN = "seu_token_aqui"  # Substitua pelo seu token

Write-Host "=== Teste de Status de Video com Debug ===" -ForegroundColor Green
Write-Host ""

# Função para fazer request
function Test-VideoStatus {
    param(
        [string]$Video,
        [string]$Source,
        [string]$Caption,
        [string]$TestName
    )
    
    Write-Host "Testando: $TestName" -ForegroundColor Yellow
    
    $body = @{
        video = $Video
        source = $Source
        caption = $Caption
    } | ConvertTo-Json
    
    try {
        $response = Invoke-RestMethod -Uri "$API_URL/status/video" `
            -Method POST `
            -Headers @{ "Authorization" = "Bearer $TOKEN"; "Content-Type" = "application/json" } `
            -Body $body
        
        Write-Host "✅ Sucesso: $($response | ConvertTo-Json)" -ForegroundColor Green
    }
    catch {
        Write-Host "❌ Erro: $($_.Exception.Message)" -ForegroundColor Red
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $responseBody = $reader.ReadToEnd()
            Write-Host "Detalhes: $responseBody" -ForegroundColor Red
        }
    }
    Write-Host ""
}

# Verificar se o servidor está rodando
try {
    $healthCheck = Invoke-RestMethod -Uri "$API_URL/health" -Method GET
    Write-Host "✅ Servidor está rodando" -ForegroundColor Green
}
catch {
    Write-Host "❌ Servidor não está rodando. Inicie o servidor primeiro." -ForegroundColor Red
    exit 1
}

Write-Host ""

# Test 1: Teste com base64 válido (pequeno video de exemplo)
Write-Host "=== Teste 1: Base64 pequeno ===" -ForegroundColor Cyan
$smallVideoBase64 = "data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDEAAAAIZnJlZQAAAWJtZGF0AAACrAYF//+p3EXpvebZSLeWLNgg2SPu73gyNjQgLSBjb3JlIDE2NCByMzEzNSBhM2E1NmY5IC0gSC4yNjQvTVBFRy00IEFWQyBjb2RlYyAtIENvcHlsZWZ0IDIwMDMtMjAyMyAtIGh0dHA6Ly93d3cudmlkZW9sYW4ub3JnL3gyNjQuaHRtbCAtIG9wdGlvbnM6IGNhYmFjPTEgcmVmPTMgZGVibG9jaz0xOjA6MCBhbmFseXNlPTB4MzoweDExMyBtZT1oZXggc3VibWU9NyBwc3k9MSBwc3lfcmQ9MS4wMDowLjAwIG1peGVkX3JlZj0xIG1lX3JhbmdlPTE2IGNocm9tYV9tZT0xIHRyZWxsaXM9MSA4eDhkY3Q9MSBjcW09MCBkZWFkem9uZT0yMSwxMSBmYXN0X3Bza2lwPTEgY2hyb21hX3FwX29mZnNldD0tMiBzbGljZXM9NCBucF9yZWZzPTEgcmNfbG9va2FoZWFkPTQwIHJlZj0yIGtleWludD0yNTAga2V5aW50X21pbj0yNSBzY2VuZWN1dD00MCBpbnRyYV9yZWZyZXNoPTAgcmNfcmVzZW49MSBibHVyYXlfY29tcGF0PTAgYnJhbWVzPTMgYl9weXJhbWlkPTIgYl9hZGFwdD0xIGJfYmlhcz0wIGRpcmVjdD0xIHdlaWdodGI9MSBvcGVuX2dvcD0wIHdlaWdodHA9MiBrZXlpbnQ9MjUgZnJhbWVfcmVmPTEgcmM9YWJyIGJpdHJhdGU9NTEyIHJhdGV0b2w9MS4wIHFjb21wPTAuNjAgcXBtaW49MCBxcG1heD02OSBxcHN0ZXA9NCBpbml0X3FwPTAgYXE9MToxLjAwAAAAABWAZ2H//u2Xg=="
Test-VideoStatus -Video $smallVideoBase64 -Source "base64" -Caption "Teste de vídeo base64" -TestName "Base64 válido"

# Test 2: URL de vídeo (exemplo)
Write-Host "=== Teste 2: URL de vídeo ===" -ForegroundColor Cyan
$videoUrl = "https://sample-videos.com/zip/10/mp4/SampleVideo_360x240_1mb.mp4"
Test-VideoStatus -Video $videoUrl -Source "url" -Caption "Teste de vídeo URL" -TestName "URL de vídeo"

# Test 3: Base64 inválido
Write-Host "=== Teste 3: Base64 inválido ===" -ForegroundColor Cyan
Test-VideoStatus -Video "base64inválido" -Source "base64" -Caption "Teste erro" -TestName "Base64 inválido"

# Test 4: URL inválida
Write-Host "=== Teste 4: URL inválida ===" -ForegroundColor Cyan
Test-VideoStatus -Video "http://url-inexistente.com/video.mp4" -Source "url" -Caption "Teste erro" -TestName "URL inválida"

# Test 5: Arquivo inexistente
Write-Host "=== Teste 5: Arquivo inexistente ===" -ForegroundColor Cyan
Test-VideoStatus -Video "C:\arquivo_inexistente.mp4" -Source "file" -Caption "Teste erro" -TestName "Arquivo inexistente"

Write-Host "=== Fim dos testes ===" -ForegroundColor Green
Write-Host ""
Write-Host "Verifique os logs do servidor para ver as mensagens de debug detalhadas!" -ForegroundColor Yellow
