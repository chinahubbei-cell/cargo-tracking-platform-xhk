#!/bin/bash
set -e

# Build Main Frontend
echo ">>> Building Main Frontend..."
cd trackcard-frontend
# Ensure dependencies (optional, but good)
if [ ! -d "node_modules" ]; then
    npm install
fi
VITE_API_BASE_URL=/api npm run build
cd ..

# Build Admin Frontend
echo ">>> Building Admin Frontend..."
cd trackcard-admin/admin-frontend
if [ ! -d "node_modules" ]; then
    npm install
fi
VITE_API_BASE_URL=/api npm run build
cd ../..

# Upload
echo ">>> Uploading to server..."
chmod +x upload_frontend.exp
expect upload_frontend.exp
