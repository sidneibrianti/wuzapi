# Script de teste para endpoints de Status do WuzAPI
# Autor: Implementação de Status - WuzAPI  
# Data: 2025-09-07
# Versão: Windows PowerShell

param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$Token = $env:WUZAPI_TOKEN
)

# Configurações
$ContentType = "application/json"

# Função para logging colorido
function Write-Log {
    param([string]$Message, [string]$Type = "Info")
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    switch ($Type) {
        "Success" { Write-Host "[$timestamp] ✅ $Message" -ForegroundColor Green }
        "Warning" { Write-Host "[$timestamp] ⚠️  $Message" -ForegroundColor Yellow }
        "Error"   { Write-Host "[$timestamp] ❌ $Message" -ForegroundColor Red }
        default   { Write-Host "[$timestamp] 🔍 $Message" -ForegroundColor Blue }
    }
}

# Função para fazer requisições
function Invoke-ApiTest {
    param(
        [string]$Method,
        [string]$Endpoint,
        [string]$Body = $null,
        [string]$Description
    )
    
    Write-Log "Testando: $Description"
    Write-Host "Endpoint: $Method $Endpoint" -ForegroundColor Cyan
    
    $headers = @{
        "Authorization" = "Bearer $Token"
        "Content-Type" = $ContentType
    }
    
    try {
        $uri = "$BaseUrl$Endpoint"
        
        if ($Body) {
            Write-Host "Payload: $Body" -ForegroundColor Gray
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers -Body $Body
        } else {
            $response = Invoke-RestMethod -Uri $uri -Method $Method -Headers $headers
        }
        
        Write-Log "Status: 200 OK" "Success"
        Write-Host "Response: $($response | ConvertTo-Json -Depth 3)" -ForegroundColor Green
        
    } catch {
        $statusCode = $_.Exception.Response.StatusCode.Value__
        Write-Log "Status: $statusCode" "Error"
        Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
        
        if ($_.ErrorDetails.Message) {
            Write-Host "Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
        }
    }
    
    Write-Host "----------------------------------------" -ForegroundColor DarkGray
    Write-Host ""
}

# Banner
Write-Host ""
Write-Host "🚀 WuzAPI - Teste dos Endpoints de Status" -ForegroundColor Magenta
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host ""

# Validar configurações
if (-not $Token -or $Token -eq "") {
    Write-Log "Token não configurado. Configure a variável de ambiente WUZAPI_TOKEN" "Warning"
    Write-Host "Exemplo: `$env:WUZAPI_TOKEN = 'seu_token_real'" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "Configurações:" -ForegroundColor Cyan
Write-Host "  Base URL: $BaseUrl" -ForegroundColor Gray
Write-Host "  Token: $($Token.Substring(0, [Math]::Min(10, $Token.Length)))..." -ForegroundColor Gray
Write-Host ""

# Teste 1: Status de texto simples
$textPayload = @{
    text = "🧪 Status de teste via API WuzAPI!"
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/text" -Body $textPayload -Description "Status de texto simples"

# Teste 2: Status de texto com formatação
$colorTextPayload = @{
    text = "Status colorido! 🎨"
    background_color = 4294901760
    text_color = 4294967295
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/text" -Body $colorTextPayload -Description "Status de texto com cores"

# Teste 3: Status de imagem via URL
$imageUrlPayload = @{
    image = "https://picsum.photos/800/600"
    source = "url"
    caption = "📸 Imagem de teste via URL"
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/image" -Body $imageUrlPayload -Description "Status de imagem via URL"

# Teste 4: Status de imagem base64 (exemplo pequeno - pixel transparente)
$imageBase64Payload = @{
    image = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
    source = "base64"
    caption = "🖼️ Imagem base64 de teste"
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/image" -Body $imageBase64Payload -Description "Status de imagem base64"

# Teste 5: Status de vídeo (comentado)
Write-Log "Teste de vídeo comentado - necessário URL válida de vídeo MP4" "Warning"

# Teste 6: Status de áudio (comentado)
Write-Log "Teste de áudio comentado - necessário arquivo de áudio válido" "Warning"

# Teste 7: Configurações de privacidade
Invoke-ApiTest -Method "GET" -Endpoint "/status/privacy" -Description "Configurações de privacidade"

# Teste 8: Validação de erro - texto muito longo
$longText = "A" * 700
$longTextPayload = @{
    text = $longText
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/text" -Body $longTextPayload -Description "Teste de validação - texto longo"

# Teste 9: Validação de erro - formato inválido
$invalidPayload = @{
    image = "dados_invalidos"
    source = "base64"
} | ConvertTo-Json

Invoke-ApiTest -Method "POST" -Endpoint "/status/send/image" -Body $invalidPayload -Description "Teste de validação - base64 inválido"

# Resumo final
Write-Host ""
Write-Host "🏁 Testes concluídos!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Próximos passos:" -ForegroundColor Cyan
Write-Host "1. Verificar se todos os testes retornaram status 200"
Write-Host "2. Confirmar se os status aparecem no WhatsApp"
Write-Host "3. Testar com diferentes tipos de mídia"
Write-Host "4. Validar autenticação com tokens inválidos"
Write-Host ""

Write-Host "💡 Instruções de uso:" -ForegroundColor Cyan
Write-Host "1. Configure seu token: `$env:WUZAPI_TOKEN = 'seu_token'"
Write-Host "2. Execute: .\test_status_endpoints.ps1"
Write-Host "3. Ou com parâmetros: .\test_status_endpoints.ps1 -Token 'seu_token' -BaseUrl 'http://localhost:8080'"
Write-Host "4. Verifique os resultados no terminal e no WhatsApp"
Write-Host ""
