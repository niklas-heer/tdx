#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

FAILED=0
PASSED=0

# Build tdx
echo "Building tdx..."
go build -o tdx ./cmd/tdx

test_case() {
    local name="$1"
    local input_file="$2"
    local keystrokes="$3"
    local expected_file="$4"

    echo -e "\n${YELLOW}Testing: $name${NC}"

    # Create temp file
    cp "$input_file" /tmp/test_input.md

    # Run tdx with keystrokes
    printf "$keystrokes" | ./tdx --show-headings /tmp/test_input.md > /dev/null 2>&1 || true

    # Compare result
    if diff -q /tmp/test_input.md "$expected_file" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC}: $name"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $name"
        echo "Expected:"
        cat "$expected_file"
        echo ""
        echo "Got:"
        cat /tmp/test_input.md
        ((FAILED++))
    fi
}

# Test 1: Move within section - move up once
cat > /tmp/input1.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected1.md << 'EOF'
## Section A

- [ ] Task A2
- [ ] Task A1
- [ ] Task A3
EOF

test_case "Move A2 up one position within section" "/tmp/input1.md" "jmk\r\x1b" "/tmp/expected1.md"

# Test 2: Move within section - move down once
cat > /tmp/input2.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected2.md << 'EOF'
## Section A

- [ ] Task A2
- [ ] Task A1
- [ ] Task A3
EOF

test_case "Move A1 down one position within section" "/tmp/input2.md" "mj\r\x1b" "/tmp/expected2.md"

# Test 3: Chained moves - move same item twice
cat > /tmp/input3.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected3.md << 'EOF'
## Section A

- [ ] Task A2
- [ ] Task A3
- [ ] Task A1
EOF

test_case "Move A1 down twice (chained)" "/tmp/input3.md" "mjmj\r\x1b" "/tmp/expected3.md"

# Test 4: Cross-section move up
cat > /tmp/input4.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2

## Section B
- [ ] Task B1
- [ ] Task B2
EOF

cat > /tmp/expected4.md << 'EOF'
## Section A

- [ ] Task A1
- [ ] Task A2
- [ ] Task B1

## Section B

- [ ] Task B2
EOF

test_case "Move B1 up across section boundary" "/tmp/input4.md" "jjmk\r\x1b" "/tmp/expected4.md"

# Test 5: Cross-section move down
cat > /tmp/input5.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2

## Section B
- [ ] Task B1
- [ ] Task B2
EOF

cat > /tmp/expected5.md << 'EOF'
## Section A

- [ ] Task A1

## Section B

- [ ] Task A2
- [ ] Task B1
- [ ] Task B2
EOF

test_case "Move A2 down across section boundary" "/tmp/input5.md" "jmj\r\x1b" "/tmp/expected5.md"

# Test 6: Move to very top
cat > /tmp/input6.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected6.md << 'EOF'
## Section A

- [ ] Task A3
- [ ] Task A1
- [ ] Task A2
EOF

test_case "Move A3 to top (multiple moves)" "/tmp/input6.md" "jjmkmk\r\x1b" "/tmp/expected6.md"

# Test 7: Move to very bottom
cat > /tmp/input7.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected7.md << 'EOF'
## Section A

- [ ] Task A2
- [ ] Task A3
- [ ] Task A1
EOF

test_case "Move A1 to bottom (multiple moves)" "/tmp/input7.md" "mjmj\r\x1b" "/tmp/expected7.md"

# Test 8: Cancel move - cursor should return to original position
cat > /tmp/input8.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

cat > /tmp/expected8.md << 'EOF'
## Section A
- [ ] Task A1
- [ ] Task A2
- [ ] Task A3
EOF

test_case "Cancel move returns cursor to original position" "/tmp/input8.md" "jmk\x1b" "/tmp/expected8.md"

# Summary
echo ""
echo "================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo "================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi
