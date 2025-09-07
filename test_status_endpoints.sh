#!/bin/bash
# Script de teste para endpoints de Status do WuzAPI
# Autor: Implementação de Status - WuzAPI
# Data: 2025-09-07

set -e

# Configurações
BASE_URL="http://localhost:8080"
TOKEN="${WUZAPI_TOKEN:-seu_token_aqui}"
CONTENT_TYPE="application/json"

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para logging
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# Função para fazer requisições
make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    log "Testando: $description"
    echo "Endpoint: $method $endpoint"
    
    if [ -n "$data" ]; then
        echo "Payload: $data"
        response=$(curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -H "Content-Type: $CONTENT_TYPE" \
            -d "$data" \
            -w "HTTP_STATUS:%{http_code}")
    else
        response=$(curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $TOKEN" \
            -w "HTTP_STATUS:%{http_code}")
    fi
    
    http_status=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
    response_body=$(echo "$response" | sed 's/HTTP_STATUS:[0-9]*$//')
    
    if [ "$http_status" -eq 200 ]; then
        success "Status: $http_status"
        echo "Response: $response_body"
    else
        error "Status: $http_status"
        echo "Response: $response_body"
    fi
    
    echo "----------------------------------------"
    echo
}

# Banner
echo
echo "🚀 WuzAPI - Teste dos Endpoints de Status"
echo "=========================================="
echo

# Validar se o token foi fornecido
if [ "$TOKEN" = "seu_token_aqui" ]; then
    warning "Token não configurado. Defina a variável WUZAPI_TOKEN ou edite o script."
    echo "Exemplo: export WUZAPI_TOKEN=seu_token_real"
    echo
fi

# Teste 1: Status de texto simples
make_request "POST" "/status/send/text" '{
    "text": "🧪 Status de teste via API WuzAPI!"
}' "Status de texto simples"

# Teste 2: Status de texto com formatação
make_request "POST" "/status/send/text" '{
    "text": "Status colorido! 🎨",
    "background_color": 4294901760,
    "text_color": 4294967295
}' "Status de texto com cores"

# Teste 3: Status de imagem via URL
make_request "POST" "/status/send/image" '{
    "image": "https://picsum.photos/800/600",
    "source": "url",
    "caption": "📸 Imagem de teste via URL"
}' "Status de imagem via URL"

# Teste 4: Status de imagem base64 (exemplo pequeno)
make_request "POST" "/status/send/image" '{
    "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
    "source": "base64",
    "caption": "🖼️ Imagem base64 de teste"
}' "Status de imagem base64"

# Teste 5: Status de vídeo via URL (usando um MP4 pequeno de exemplo)
warning "Teste de vídeo comentado - necessário URL válida de vídeo MP4"
# make_request "POST" "/status/send/video" '{
#     "video": "https://sample-videos.com/zip/10/mp4/SampleVideo_128kb_mp4.mp4",
#     "source": "url",
#     "caption": "🎥 Vídeo de teste"
# }' "Status de vídeo via URL"

# Teste 6: Status de áudio (comentado - necessário arquivo de áudio válido)
warning "Teste de áudio comentado - necessário arquivo de áudio válido"
# make_request "POST" "/status/send/audio" '{
#     "audio": "/path/to/audio.mp3",
#     "source": "file",
#     "ptt": true
# }' "Status de áudio PTT"

# Teste 7: Configurações de privacidade
make_request "GET" "/status/privacy" "" "Configurações de privacidade"

# Teste 8: Validação de erro - texto muito longo
make_request "POST" "/status/send/text" '{
    "text": "' $(printf 'A%.0s' {1..700}) '"
}' "Teste de validação - texto longo"

# Teste 9: Validação de erro - formato inválido
make_request "POST" "/status/send/image" '{
    "image": "dados_invalidos",
    "source": "base64"
}' "Teste de validação - base64 inválido"

echo
echo "🏁 Testes concluídos!"
echo
echo "📋 Próximos passos:"
echo "1. Verificar se todos os testes retornaram status 200"
echo "2. Confirmar se os status aparecem no WhatsApp"
echo "3. Testar com diferentes tipos de mídia"
echo "4. Validar autenticação com tokens inválidos"
echo

# Instruções de uso
echo "💡 Instruções de uso:"
echo "1. Configure seu token: export WUZAPI_TOKEN=seu_token"
echo "2. Execute: bash test_status_endpoints.sh"
echo "3. Verifique os resultados no terminal e no WhatsApp"
echo
