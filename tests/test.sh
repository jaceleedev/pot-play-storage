#\!/bin/bash

# Test script for pot-play-storage MVP

echo "🚀 Testing pot-play-storage API..."

# Base URL
BASE_URL="http://localhost:8090/api/v1/files"

# 1. Test upload
echo -e "\n📤 Testing file upload..."
# Create a test image
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" | base64 -d > test.png

UPLOAD_RESPONSE=$(curl -s -X POST $BASE_URL -F "file=@test.png")
echo "Response: $UPLOAD_RESPONSE"

# Extract ID from response
FILE_ID=$(echo $UPLOAD_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "File ID: $FILE_ID"

# 2. Test list
echo -e "\n📋 Testing file list..."
curl -s $BASE_URL | jq .

# 3. Test download
echo -e "\n📥 Testing file download..."
curl -s -o downloaded.png "$BASE_URL/$FILE_ID"
if [ -f downloaded.png ]; then
    echo "✅ Download successful"
else
    echo "❌ Download failed"
fi

# 4. Test delete
echo -e "\n🗑️ Testing file delete..."
curl -s -X DELETE "$BASE_URL/$FILE_ID"
echo "✅ Delete completed"

# 5. Verify deletion
echo -e "\n📋 Verifying deletion..."
curl -s $BASE_URL | jq .

# Cleanup
rm -f test.png downloaded.png

echo -e "\n✅ All tests completed\!"
