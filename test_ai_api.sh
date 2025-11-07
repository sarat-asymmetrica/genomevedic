#!/bin/bash

# GenomeVedic AI Variant Interpreter - API Test Script
# Usage: ./test_ai_api.sh [API_KEY]

echo "=== GenomeVedic AI Variant Interpreter - API Test ==="
echo ""

# Check if API key is provided
API_KEY="${1:-$OPENAI_API_KEY}"

if [ -z "$API_KEY" ]; then
    echo "ERROR: OpenAI API key not provided"
    echo "Usage: ./test_ai_api.sh sk-your-api-key-here"
    echo "   or: export OPENAI_API_KEY=sk-your-key && ./test_ai_api.sh"
    exit 1
fi

# Set API endpoint
API_URL="http://localhost:8080"

echo "Testing API endpoint: $API_URL"
echo ""

# Test 1: Health check
echo "=== Test 1: Health Check ==="
curl -s -X GET "$API_URL/api/v1/health" | jq '.' || echo "Server not running?"
echo ""

# Test 2: Explain TP53 R175H (cancer hotspot)
echo "=== Test 2: TP53 R175H (Cancer Hotspot) ==="
curl -s -X POST "$API_URL/api/v1/variants/explain" \
  -H "Content-Type: application/json" \
  -d '{
    "gene": "TP53",
    "variant": "R175H",
    "chromosome": "17",
    "position": 7577538,
    "ref_allele": "C",
    "alt_allele": "A",
    "include_references": true
  }' | jq '.'
echo ""

# Test 3: Explain BRCA1 185delAG
echo "=== Test 3: BRCA1 185delAG (Pathogenic Variant) ==="
curl -s -X POST "$API_URL/api/v1/variants/explain" \
  -H "Content-Type: application/json" \
  -d '{
    "gene": "BRCA1",
    "variant": "185delAG",
    "chromosome": "17",
    "position": 43094464,
    "ref_allele": "AG",
    "alt_allele": "-",
    "include_references": true
  }' | jq '.'
echo ""

# Test 4: Batch processing
echo "=== Test 4: Batch Explain (Multiple Variants) ==="
curl -s -X POST "$API_URL/api/v1/variants/batch-explain" \
  -H "Content-Type: application/json" \
  -d '[
    {
      "gene": "TP53",
      "variant": "R175H",
      "chromosome": "17",
      "position": 7577538,
      "ref_allele": "C",
      "alt_allele": "A"
    },
    {
      "gene": "KRAS",
      "variant": "G12D",
      "chromosome": "12",
      "position": 25398284,
      "ref_allele": "C",
      "alt_allele": "T"
    }
  ]' | jq '.'
echo ""

# Test 5: Cache statistics
echo "=== Test 5: Cache Statistics ==="
curl -s -X GET "$API_URL/api/v1/cache/stats" | jq '.'
echo ""

echo "=== All Tests Complete ==="
echo ""
echo "Notes:"
echo "- First calls will be uncached (slower, ~3-5s)"
echo "- Subsequent calls should be cached (faster, <100ms)"
echo "- Check 'cached' field in response"
echo "- Monitor 'cost_usd' to track expenses"
echo ""
