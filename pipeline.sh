#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# SEBI MF Backend — Full CI/CD Pipeline
# ═══════════════════════════════════════════════════════════════════════════════
#
# This script orchestrates the complete pipeline:
#   1. Validate prerequisites (git, docker)
#   2. Run the service injector (wire implementations into project1)
#   3. Verify the build compiles locally
#   4. Push code to GitHub
#   5. Build Docker image (3-stage: inject → build → runtime)
#   6. Push Docker image to GHCR
#   7. Pull and run the image (simulating a different machine)
#   8. Smoke test all 4 SEBI MF APIs
#
# Usage:
#   ./pipeline.sh                          # Interactive — prompts for values
#   ./pipeline.sh --auto                   # Uses env vars, no prompts
#
# Required environment variables (for --auto mode):
#   GITHUB_USERNAME   — Your GitHub username
#   GITHUB_TOKEN      — GitHub PAT with repo + packages:write scope
#   GITHUB_REPO_URL   — Full repo URL (e.g. https://github.com/user/sebi-mf-backend.git)
#
# Optional:
#   IMAGE_NAME        — Docker image name (default: sebi-mf-backend)
#   IMAGE_TAG         — Docker image tag (default: latest)
#   HOST_PORT         — Port to expose on host (default: 8080)
#
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

banner() {
    echo -e "\n${CYAN}════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}\n"
}

step() {
    echo -e "\n${BLUE}──────────────────────────────────────────────────────────────${NC}"
    echo -e "${BOLD}  STEP $1: $2${NC}"
    echo -e "${BLUE}──────────────────────────────────────────────────────────────${NC}\n"
}

ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
fail() { echo -e "  ${RED}✗${NC} $1"; }
warn() { echo -e "  ${YELLOW}⚠${NC} $1"; }
info() { echo -e "  ${CYAN}ℹ${NC} $1"; }

# ── Configuration ─────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}"

IMAGE_NAME="${IMAGE_NAME:-sebi-mf-backend}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
HOST_PORT="${HOST_PORT:-8080}"
CONTAINER_NAME="${IMAGE_NAME}-container"
REGISTRY="ghcr.io"
AUTO_MODE=false

if [[ "${1:-}" == "--auto" ]]; then
    AUTO_MODE=true
fi

banner "SEBI MUTUAL FUND — DOCKER CI/CD PIPELINE"

# ── Step 1: Prerequisites ────────────────────────────────────────────────────
step 1 "Checking prerequisites"

check_cmd() {
    if command -v "$1" &>/dev/null; then
        local ver
        ver=$($1 --version 2>&1 | head -1)
        ok "$1 installed: $ver"
        return 0
    else
        fail "$1 not found"
        return 1
    fi
}

PREREQS_OK=true
check_cmd git    || PREREQS_OK=false
check_cmd docker || PREREQS_OK=false

if ! $PREREQS_OK; then
    echo -e "\n${RED}Missing prerequisites. Install them and retry.${NC}"
    exit 1
fi

# Docker daemon check
if docker info &>/dev/null; then
    ok "Docker daemon is running"
else
    fail "Docker daemon is not running. Start Docker Desktop or dockerd."
    exit 1
fi

# ── Step 2: Collect configuration ─────────────────────────────────────────────
step 2 "Pipeline configuration"

if $AUTO_MODE; then
    : "${GITHUB_USERNAME:?GITHUB_USERNAME required in --auto mode}"
    : "${GITHUB_TOKEN:?GITHUB_TOKEN required in --auto mode}"
    : "${GITHUB_REPO_URL:?GITHUB_REPO_URL required in --auto mode}"
else
    if [[ -z "${GITHUB_USERNAME:-}" ]]; then
        read -rp "  GitHub username: " GITHUB_USERNAME
    fi
    if [[ -z "${GITHUB_TOKEN:-}" ]]; then
        read -rsp "  GitHub PAT (hidden): " GITHUB_TOKEN
        echo
    fi
    if [[ -z "${GITHUB_REPO_URL:-}" ]]; then
        read -rp "  GitHub repo URL (e.g. https://github.com/$GITHUB_USERNAME/custom-git-server-implementatoin.git): " GITHUB_REPO_URL
    fi
fi

info "Image:     ${REGISTRY}/${GITHUB_USERNAME}/${IMAGE_NAME}:${IMAGE_TAG}"
info "Repo:      ${GITHUB_REPO_URL}"
info "Host port: ${HOST_PORT}"

# ── Step 3: Verify project structure ──────────────────────────────────────────
step 3 "Verifying project structure"

REQUIRED_FILES=(
    "go.work"
    "Dockerfile"
    "templates/bootstrap.go.gonja"
    "project1-mf-backend/go.mod"
    "project1-mf-backend/cmd/server/main.go"
    "project1-mf-backend/internal/contracts/interfaces.go"
    "project1-mf-backend/internal/registry/registry.go"
    "project1-mf-backend/internal/service/mf_service.go"
    "project1-mf-backend/internal/handler/mf_handler.go"
    "project1-mf-backend/internal/bootstrap/bootstrap.go"
    "project2-mf-implementations/go.mod"
    "project2-mf-implementations/mfcreate/mf_create.go"
    "project2-mf-implementations/mftransfer/mf_transfer.go"
    "project2-mf-implementations/mfupdate/mf_update.go"
    "project2-mf-implementations/mfdelete/mf_delete.go"
    "service-injector/go.mod"
    "service-injector/cmd/injector/main.go"
)

ALL_PRESENT=true
for f in "${REQUIRED_FILES[@]}"; do
    if [[ -f "${PROJECT_ROOT}/${f}" ]]; then
        ok "$f"
    else
        fail "Missing: $f"
        ALL_PRESENT=false
    fi
done

if ! $ALL_PRESENT; then
    echo -e "\n${RED}Project structure incomplete. Ensure all files exist.${NC}"
    exit 1
fi

# ── Step 4: Push code to GitHub ───────────────────────────────────────────────
step 4 "Pushing code to GitHub"

cd "${PROJECT_ROOT}"

# Inject credentials into the remote URL for non-interactive push
AUTH_REPO_URL=$(echo "${GITHUB_REPO_URL}" | sed "s|https://|https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@|")

# Initialize git if needed
if [[ ! -d .git ]]; then
    git init
    git branch -M main
    info "Initialized new git repository"
fi

# Configure git user
git config user.name "${GITHUB_USERNAME}"
git config user.email "${GITHUB_USERNAME}@users.noreply.github.com"

# Disable credential helper to avoid interactive prompts
git config --local credential.helper ""

# Set remote
EXISTING_REMOTE=$(git remote get-url origin 2>/dev/null || echo "")
if [[ -z "$EXISTING_REMOTE" ]]; then
    git remote add origin "${AUTH_REPO_URL}"
elif [[ "$EXISTING_REMOTE" != "$AUTH_REPO_URL" ]]; then
    git remote set-url origin "${AUTH_REPO_URL}"
fi

# Stage all files
git add -A

# Commit (skip if nothing changed)
if git diff --cached --quiet 2>/dev/null; then
    warn "No changes to commit — pushing existing commits"
else
    COMMIT_MSG="feat: SEBI MF backend with Docker pipeline — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    git commit -m "$COMMIT_MSG"
    ok "Committed: $COMMIT_MSG"
fi

# Fetch remote and handle divergence
git fetch origin main 2>/dev/null || true
if git rev-parse origin/main &>/dev/null; then
    git update-ref refs/remotes/origin/main "$(git rev-parse FETCH_HEAD 2>/dev/null || echo "")" 2>/dev/null || true
fi

# Push (force if needed for first push or diverged state)
if git push -u origin main 2>/dev/null; then
    ok "Pushed to origin/main"
else
    warn "Normal push failed, trying force push..."
    git push --force -u origin main
    ok "Force-pushed to origin/main"
fi

COMMIT_HASH=$(git rev-parse --short HEAD)
ok "Latest commit: ${COMMIT_HASH}"

# Reset remote URL to strip credentials from .git/config
git remote set-url origin "${GITHUB_REPO_URL}"

# ── Step 5: Build Docker image ───────────────────────────────────────────────
step 5 "Building Docker image (3-stage: inject → build → runtime)"

info "This runs the injector inside Docker — no local Go toolchain needed"
info "Build context: ${PROJECT_ROOT}"

FULL_IMAGE_REF="${IMAGE_NAME}:${IMAGE_TAG}"

docker build \
    --no-cache \
    -t "${FULL_IMAGE_REF}" \
    -f "${PROJECT_ROOT}/Dockerfile" \
    "${PROJECT_ROOT}" 2>&1 | while IFS= read -r line; do
    echo "  │ $line"
done

if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
    ok "Docker image built: ${FULL_IMAGE_REF}"
    IMAGE_SIZE=$(docker image inspect "${FULL_IMAGE_REF}" --format='{{.Size}}' | awk '{printf "%.1fMB", $1/1048576}')
    info "Image size: ${IMAGE_SIZE}"
else
    fail "Docker build failed"
    exit 1
fi

# ── Step 6: Login and push to GHCR ───────────────────────────────────────────
step 6 "Pushing Docker image to GitHub Container Registry"

echo "${GITHUB_TOKEN}" | docker login "${REGISTRY}" -u "${GITHUB_USERNAME}" --password-stdin 2>&1 | while IFS= read -r line; do
    echo "  │ $line"
done
ok "Logged in to ${REGISTRY}"

REMOTE_IMAGE_REF="${REGISTRY}/${GITHUB_USERNAME}/${IMAGE_NAME}:${IMAGE_TAG}"
docker tag "${FULL_IMAGE_REF}" "${REMOTE_IMAGE_REF}"
ok "Tagged: ${REMOTE_IMAGE_REF}"

docker push "${REMOTE_IMAGE_REF}" 2>&1 | while IFS= read -r line; do
    echo "  │ $line"
done
ok "Pushed to ${REMOTE_IMAGE_REF}"

# ── Step 7: Pull and run (simulates a different machine) ─────────────────────
step 7 "Pulling and running container (simulating remote deploy)"

# Remove any existing container with the same name
docker rm -f "${CONTAINER_NAME}" 2>/dev/null || true

# Force a fresh pull (to prove it works from the registry)
docker rmi "${REMOTE_IMAGE_REF}" 2>/dev/null || true

info "Pulling from registry..."
docker pull "${REMOTE_IMAGE_REF}" 2>&1 | while IFS= read -r line; do
    echo "  │ $line"
done

docker run -d \
    --name "${CONTAINER_NAME}" \
    -p "${HOST_PORT}:8080" \
    "${REMOTE_IMAGE_REF}"

ok "Container '${CONTAINER_NAME}' running on port ${HOST_PORT}"

# Wait for the server to be ready
info "Waiting for server to start..."
for i in {1..15}; do
    if curl -sf "http://localhost:${HOST_PORT}/health" >/dev/null 2>&1; then
        ok "Server is healthy!"
        break
    fi
    sleep 1
    if [[ $i -eq 15 ]]; then
        fail "Server did not start within 15 seconds"
        echo "  Container logs:"
        docker logs "${CONTAINER_NAME}" 2>&1 | tail -20 | while IFS= read -r line; do
            echo "    │ $line"
        done
        exit 1
    fi
done

# ── Step 8: Smoke test all 4 SEBI MF APIs ────────────────────────────────────
step 8 "Smoke testing all 4 SEBI MF API endpoints"

BASE_URL="http://localhost:${HOST_PORT}"
PASS=0
FAIL_COUNT=0

test_api() {
    local method="$1"
    local endpoint="$2"
    local payload="$3"
    local label="$4"

    echo -e "\n  ${BOLD}Testing: ${label}${NC}"
    echo -e "  ${method} ${endpoint}"

    RESPONSE=$(curl -sf -X "${method}" "${BASE_URL}${endpoint}" \
        -H "Content-Type: application/json" \
        -d "${payload}" 2>&1) || {
        fail "${label} — request failed"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        return
    }

    # Check if response contains "success": true
    if echo "$RESPONSE" | grep -q '"success":true'; then
        ok "${label} — SUCCESS"
        echo "$RESPONSE" | python3 -m json.tool 2>/dev/null | head -20 | while IFS= read -r line; do
            echo "    $line"
        done
        PASS=$((PASS + 1))
    else
        fail "${label} — returned error"
        echo "$RESPONSE" | python3 -m json.tool 2>/dev/null | while IFS= read -r line; do
            echo "    $line"
        done
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# Test 1: CREATE — POST /mutualfund
test_api POST "/mutualfund" '{
    "investor_id": "INV001",
    "investor_name": "Abhijeet Wankhade",
    "pan": "ABCDE1234F",
    "kyc_status": "VERIFIED",
    "demat_account_no": "1234567890123456",
    "scheme_code": "INF204K01I28",
    "scheme_name": "Axis Bluechip Fund - Growth",
    "amc": "Axis AMC",
    "fund_category": "EQUITY",
    "fund_type": "GROWTH",
    "investment_mode": "LUMPSUM",
    "amount": 10000,
    "nav": 45.50,
    "units": 219.780,
    "risk_profile": "MODERATE",
    "bank_account_no": "9876543210",
    "ifsc": "HDFC0001234",
    "platform": "API"
}' "MFCreate — New investment"

# Test 2: TRANSFER — POST /mutualfund/transfer
test_api POST "/mutualfund/transfer" '{
    "investor_id": "INV001",
    "pan": "ABCDE1234F",
    "folio_number": "INV-INV001-INF2-123456",
    "from_scheme_code": "INF204K01I28",
    "from_scheme_name": "Axis Bluechip Fund",
    "from_amc": "Axis AMC",
    "redemption_units": 100,
    "from_nav": 45.50,
    "to_scheme_code": "INF204K01T51",
    "to_scheme_name": "Axis Midcap Fund",
    "to_amc": "Axis AMC",
    "to_nav": 32.75,
    "transfer_type": "SWITCH",
    "exit_load_applicable": false,
    "exit_load_percent": 0,
    "stcg_applicable": true,
    "ltcg_applicable": false
}' "MFTransfer — Scheme switch"

# Test 3: UPDATE — PUT /mutualfund
test_api PUT "/mutualfund" '{
    "investor_id": "INV001",
    "pan": "ABCDE1234F",
    "folio_number": "INV-INV001-INF2-123456",
    "scheme_code": "INF204K01I28",
    "update_type": "SIP_AMOUNT",
    "new_sip_amount": 2000,
    "reason": "Increase monthly SIP",
    "auth_otp": "123456"
}' "MFUpdate — SIP amount change"

# Test 4: DELETE — DELETE /mutualfund
test_api DELETE "/mutualfund" '{
    "investor_id": "INV001",
    "pan": "ABCDE1234F",
    "folio_number": "INV-INV001-INF2-123456",
    "scheme_code": "INF204K01I28",
    "redemption_type": "PARTIAL",
    "units": 50,
    "redemption_mode": "UNITS",
    "bank_account_no": "9876543210",
    "ifsc": "HDFC0001234",
    "exit_load_applicable": false,
    "lock_in_period_over": true,
    "tds_percent": 10,
    "auth_otp": "123456"
}' "MFDelete — Partial redemption"

# ── Summary ───────────────────────────────────────────────────────────────────
banner "PIPELINE COMPLETE"

echo -e "  ${BOLD}Results:${NC}"
echo -e "  ${GREEN}✓ Passed:${NC} ${PASS}/4"
if [[ $FAIL_COUNT -gt 0 ]]; then
    echo -e "  ${RED}✗ Failed:${NC} ${FAIL_COUNT}/4"
fi

echo -e "\n  ${BOLD}Artifacts:${NC}"
echo -e "  Repository:  ${GITHUB_REPO_URL}"
echo -e "  Docker image: ${REMOTE_IMAGE_REF}"
echo -e "  Container:    ${CONTAINER_NAME} → http://localhost:${HOST_PORT}"
echo -e "  Commit:       ${COMMIT_HASH}"

echo -e "\n  ${BOLD}To deploy on another machine:${NC}"
echo -e "  ${CYAN}docker pull ${REMOTE_IMAGE_REF}${NC}"
echo -e "  ${CYAN}docker run -d -p 8080:8080 ${REMOTE_IMAGE_REF}${NC}"

echo -e "\n  ${BOLD}To stop:${NC}"
echo -e "  ${CYAN}docker rm -f ${CONTAINER_NAME}${NC}"

if [[ $FAIL_COUNT -eq 0 ]]; then
    echo -e "\n  ${GREEN}${BOLD}All 4 SEBI MF APIs are live and responding from the Docker container!${NC}"
else
    echo -e "\n  ${YELLOW}${BOLD}Some tests failed — check the output above.${NC}"
    exit 1
fi
