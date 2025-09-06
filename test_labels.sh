#!/bin/bash

# Script de teste para verificar se os endpoints de labels estão funcionando
# Uso: ./test_labels.sh

echo "=== Teste Completo dos Endpoints de Labels WuzAPI ==="
echo ""

BASE_URL="http://localhost:3000"
API_TOKEN="seu_token_aqui"
CHAT_JID="5519992278626@s.whatsapp.net"

echo "🔍 1. Listando labels existentes..."
curl -s -X GET "$BASE_URL/labels/list" \
  -H "Authorization: Bearer $API_TOKEN" | jq .

echo ""
echo "🔍 2. Criando nova label..."
LABEL_RESPONSE=$(curl -s -X POST "$BASE_URL/labels/create" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Teste AppState",
    "color": 1
  }')

echo $LABEL_RESPONSE | jq .
LABEL_ID=$(echo $LABEL_RESPONSE | jq -r '.data.label_id // .label_id')

echo ""
echo "🔍 3. Associando chat à label..."
curl -s -X POST "$BASE_URL/labels/associate" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"chat_jid\": \"$CHAT_JID\",
    \"label_id\": \"$LABEL_ID\"
  }" | jq .

echo ""
echo "🔍 4. Editando a label..."
curl -s -X POST "$BASE_URL/labels/edit" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"label_id\": \"$LABEL_ID\",
    \"name\": \"Teste AppState Editado\",
    \"color\": 2
  }" | jq .

echo ""
echo "🔍 5. Solicitando sincronização..."
curl -s -X POST "$BASE_URL/labels/sync" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" | jq .

echo ""
echo "🔍 6. Removendo associação..."
curl -s -X POST "$BASE_URL/labels/disassociate" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"chat_jid\": \"$CHAT_JID\",
    \"label_id\": \"$LABEL_ID\"
  }" | jq .

echo ""
echo "🔍 7. Deletando a label..."
curl -s -X POST "$BASE_URL/labels/delete" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"label_id\": \"$LABEL_ID\"
  }" | jq .

echo ""
echo "✅ Testes concluídos!"
echo "📝 Label ID criado: $LABEL_ID"
echo "📝 Chat JID testado: $CHAT_JID"
echo ""
echo "📋 O que foi testado:"
echo "   ✅ Listagem de labels"
echo "   ✅ Criação de label via AppState"
echo "   ✅ Associação chat-label via AppState"
echo "   ✅ Edição de label via AppState"
echo "   ✅ Sincronização de labels"
echo "   ✅ Remoção de associação via AppState"
echo "   ✅ Deleção de label via AppState"
echo ""
echo "� Para usar este script:"
echo "   1. Substitua 'seu_token_aqui' pelo token real do usuário"
echo "   2. Substitua o CHAT_JID por um JID válido"
echo "   3. Execute: chmod +x test_labels.sh && ./test_labels.sh"
