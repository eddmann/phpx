#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PASS=0
FAIL=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXAMPLES_DIR="$SCRIPT_DIR"

echo "========================================"
echo "  phpx Examples Test Suite"
echo "========================================"
echo ""

# Helper function to run a test
run_test() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    printf "%-45s" "$name"

    # Run command and capture output
    if output=$(eval "$cmd" 2>&1); then
        # Check if expected string is in output (if provided)
        if [ -n "$expected" ]; then
            if echo "$output" | grep -qF "$expected"; then
                echo -e "${GREEN}PASS${NC}"
                ((PASS++))
                return 0
            else
                echo -e "${RED}FAIL${NC}"
                echo "  Expected: $expected"
                echo "  Got: $output"
                ((FAIL++))
                return 1
            fi
        else
            echo -e "${GREEN}PASS${NC}"
            ((PASS++))
            return 0
        fi
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Error: $output"
        ((FAIL++))
        return 1
    fi
}

# Helper for tests that should show specific blocked behavior
run_test_contains() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    printf "%-45s" "$name"

    # Run command and capture output (allow non-zero exit)
    output=$(eval "$cmd" 2>&1) || true

    if echo "$output" | grep -qF "$expected"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASS++))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Expected to contain: $expected"
        echo "  Got: $output"
        ((FAIL++))
        return 1
    fi
}

echo "Basic Examples"
echo "----------------------------------------"

# 01 - Hello World
run_test "01-hello-world.php" \
    "phpx '$EXAMPLES_DIR/01-hello-world.php'" \
    "Hello from phpx!"

# 02 - PHP Version
run_test "02-php-version.php" \
    "phpx '$EXAMPLES_DIR/02-php-version.php'" \
    "PHP Version:"

# 03 - CLI Arguments
run_test "03-cli-arguments.php" \
    "phpx '$EXAMPLES_DIR/03-cli-arguments.php' hello world" \
    "Total: 2 arguments"

echo ""
echo "Package Management"
echo "----------------------------------------"

# 04 - Single Package (Carbon)
run_test "04-single-package.php" \
    "phpx '$EXAMPLES_DIR/04-single-package.php'" \
    "Current time:"

# 05 - Multiple Packages
run_test "05-multiple-packages.php" \
    "phpx '$EXAMPLES_DIR/05-multiple-packages.php'" \
    "timestamp"

echo ""
echo "PHP Constraints & Extensions"
echo "----------------------------------------"

# 06 - PHP Constraint
run_test "06-php-constraint.php" \
    "phpx '$EXAMPLES_DIR/06-php-constraint.php'" \
    "Running on PHP"

# 07 - Common Extensions
run_test "07-common-extensions.php" \
    "phpx '$EXAMPLES_DIR/07-common-extensions.php'" \
    "Extension status:"

# 08 - Bulk Extensions (intl)
run_test "08-bulk-extensions.php" \
    "phpx '$EXAMPLES_DIR/08-bulk-extensions.php'" \
    "US Currency:"

echo ""
echo "HTTP & Data Processing"
echo "----------------------------------------"

# 09 - HTTP Client
run_test "09-http-client.php" \
    "phpx '$EXAMPLES_DIR/09-http-client.php'" \
    "Name:"

# 10 - JSON Processing
run_test "10-json-processing.php" \
    "echo '{\"test\": 123}' | phpx '$EXAMPLES_DIR/10-json-processing.php'" \
    "Parsed JSON:"

# 11 - CLI App
run_test "11-cli-app.php" \
    "phpx '$EXAMPLES_DIR/11-cli-app.php' -- --name TestUser --shout" \
    "HELLO, TESTUSER!"

echo ""
echo "Sandbox Features"
echo "----------------------------------------"

# 12 - Sandbox Basic
run_test "12-sandbox-basic.php" \
    "phpx '$EXAMPLES_DIR/12-sandbox-basic.php' --sandbox --memory 64 --timeout 10 --cpu 5" \
    "Memory limit: 64M"

# 13 - Sandbox Offline
run_test_contains "13-sandbox-offline.php" \
    "phpx '$EXAMPLES_DIR/13-sandbox-offline.php' --offline" \
    "Network blocked"

# 14 - Sandbox Allow Host
run_test_contains "14-sandbox-allow-host.php" \
    "phpx '$EXAMPLES_DIR/14-sandbox-allow-host.php' --allow-host httpbin.org" \
    "[allowed] httpbin.org"

# 15 - Sandbox Filesystem
echo "hello" > /tmp/phpx-test.txt
run_test "15-sandbox-filesystem.php" \
    "phpx '$EXAMPLES_DIR/15-sandbox-filesystem.php' --sandbox --allow-read /tmp --allow-write /tmp" \
    "Write: OK"
rm -f /tmp/phpx-test.txt /tmp/phpx-output.txt

# 16 - Sandbox Env
run_test_contains "16-sandbox-env.php" \
    "API_KEY=secret123 DEBUG=1 phpx '$EXAMPLES_DIR/16-sandbox-env.php' --sandbox --allow-env API_KEY,DEBUG" \
    "API_KEY: set"

echo ""
echo "========================================"
echo -e "  Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
echo "========================================"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
