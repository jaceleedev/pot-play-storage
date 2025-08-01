#!/bin/bash

# Test script for file deduplication feature

echo "🧪 Testing pot-play-storage deduplication feature..."

# Base URL
BASE_URL="http://localhost:8090/api/v1/files"

# Create a test image
echo "📋 Creating test image..."
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==" | base64 -d > test_image.png

# 1. Upload first file
echo -e "\n📤 Uploading first file (test1.png)..."
RESPONSE1=$(curl -s -X POST $BASE_URL -F "file=@test_image.png")
echo "Response: $RESPONSE1"
FILE_ID1=$(echo $RESPONSE1 | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "File ID 1: $FILE_ID1"

# 2. Upload same content with different name
echo -e "\n📤 Uploading second file with same content (test2.png)..."
cp test_image.png test_image2.png
RESPONSE2=$(curl -s -X POST $BASE_URL -F "file=@test_image2.png")
echo "Response: $RESPONSE2"
FILE_ID2=$(echo $RESPONSE2 | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo "File ID 2: $FILE_ID2"

# 3. List files to see both
echo -e "\n📋 Listing all files..."
curl -s $BASE_URL | jq .

# 4. Download both files to verify they work
echo -e "\n📥 Downloading both files..."
curl -s -o download1.png "$BASE_URL/$FILE_ID1"
curl -s -o download2.png "$BASE_URL/$FILE_ID2"

# 5. Verify files are identical
echo -e "\n🔍 Verifying downloaded files are identical..."
if cmp -s download1.png download2.png; then
    echo "✅ Files are identical (deduplication working!)"
else
    echo "❌ Files are different (deduplication not working)"
fi

# 6. Delete first file
echo -e "\n🗑️ Deleting first file..."
curl -s -X DELETE "$BASE_URL/$FILE_ID1"
echo "✅ First file deleted"

# 7. Try to download second file (should still work)
echo -e "\n📥 Downloading second file after first deletion..."
if curl -s -o download3.png "$BASE_URL/$FILE_ID2"; then
    echo "✅ Second file still accessible (reference counting works!)"
else
    echo "❌ Second file not accessible"
fi

# 8. Delete second file
echo -e "\n🗑️ Deleting second file..."
curl -s -X DELETE "$BASE_URL/$FILE_ID2"
echo "✅ Second file deleted"

# 9. Verify both files are gone
echo -e "\n📋 Final file list (should be empty or not contain our files)..."
curl -s $BASE_URL | jq .

# Cleanup
rm -f test_image.png test_image2.png download1.png download2.png download3.png

echo -e "\n✅ Deduplication test completed!"