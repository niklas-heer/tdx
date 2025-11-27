#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Pull latest changes first to avoid diverged branches
echo "Pulling latest changes from remote..."
git pull --rebase || {
    echo -e "${RED}Failed to pull changes. Please resolve any conflicts first.${NC}"
    exit 1
}
echo ""

# Get current version from tdx.toml
CURRENT_VERSION=$(grep '^version' tdx.toml | cut -d'"' -f2)

echo -e "${GREEN}Current version: v${CURRENT_VERSION}${NC}"
echo ""

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Calculate next versions
NEXT_MAJOR="$((MAJOR + 1)).0.0"
NEXT_MINOR="${MAJOR}.$((MINOR + 1)).0"
NEXT_PATCH="${MAJOR}.${MINOR}.$((PATCH + 1))"

# Prompt for release type
echo "Select release type:"
echo "  1) Major  (${NEXT_MAJOR}) - Breaking changes"
echo "  2) Minor  (${NEXT_MINOR}) - New features"
echo "  3) Patch  (${NEXT_PATCH}) - Bug fixes"
echo ""
read -p "Enter choice [1-3]: " choice

case $choice in
    1) NEW_VERSION=$NEXT_MAJOR ;;
    2) NEW_VERSION=$NEXT_MINOR ;;
    3) NEW_VERSION=$NEXT_PATCH ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${YELLOW}New version will be: v${NEW_VERSION}${NC}"
read -p "Continue? [y/N] " confirm

if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
fi

# Update version in tdx.toml
echo "Updating tdx.toml..."
sed -i.bak "s/^version = \"${CURRENT_VERSION}\"/version = \"${NEW_VERSION}\"/" tdx.toml
rm -f tdx.toml.bak

# Commit the version bump
echo "Committing version bump..."
git add tdx.toml
git commit -m "chore: bump version to v${NEW_VERSION}"

# Create and push tag
echo "Creating tag v${NEW_VERSION}..."
git tag "v${NEW_VERSION}"

echo "Pushing to remote..."
git push
git push --tags

echo ""
echo -e "${GREEN}Release v${NEW_VERSION} created and pushed!${NC}"
echo "GitHub Actions will now build and publish the release."
