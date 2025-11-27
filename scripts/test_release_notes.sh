#!/bin/bash
# Test script to generate release notes locally
# Run from anywhere: ./scripts/test_release_notes.sh

set -e

# Get the repository root (one level up from this script)
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "======================================"
echo "Testing AI Release Notes Generation"
echo "======================================"
echo ""

# Set environment variables
export CURRENT_TAG="v0.9.1"
export PREV_TAG="v0.9.0"
export GITHUB_REPOSITORY="niklas-heer/tdx"
export AI_MODEL="anthropic/claude-haiku-4.5"

# Check if .env exists in repo root
if [ ! -f "$REPO_ROOT/.env" ]; then
    echo "Error: .env file not found in repository root!"
    echo "Create a .env file with your OPENROUTER_API_KEY"
    exit 1
fi

echo "Configuration:"
echo "  Current Tag: $CURRENT_TAG"
echo "  Previous Tag: $PREV_TAG"
echo "  Repository: $GITHUB_REPOSITORY"
echo "  AI Model: $AI_MODEL"
echo ""

# Check if python-dotenv is installed
if ! python3 -c "import dotenv" 2>/dev/null; then
    echo "Installing python-dotenv..."
    pip3 install python-dotenv > /dev/null
fi

# Check if requests is installed
if ! python3 -c "import requests" 2>/dev/null; then
    echo "Installing requests..."
    pip3 install requests > /dev/null
fi

echo "======================================"
echo "Generating release notes..."
echo "======================================"
echo ""

# Run the script from repo root
python3 "$REPO_ROOT/.github/scripts/generate_release_notes.py"

echo ""
echo "======================================"
echo "Done!"
echo "======================================"
