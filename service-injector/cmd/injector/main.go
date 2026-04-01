package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"service-injector/db"
	"service-injector/runner"
	"service-injector/seeder"
	"service-injector/templategen"
	"service-injector/validator"
)

// ── Paths (relative to service-injector/) ────────────────────────────────────
const (
	dbPath          = "./mf_implementations.db"
	project1Dir     = "../project1-mf-backend"
	project2Dir     = "../project2-mf-implementations"
	templatesDir    = "../templates"
	bootstrapOutput = "../project1-mf-backend/internal/bootstrap/bootstrap.go"
)

// ── MF implementations to seed from project2 ────────────────────────────────
var seedEntries = []seeder.Entry{
	{
		Module:      "mfcreate",
		PackageName: "mfcreate",
		ImportPath:  "project2-mf-implementations/mfcreate",
		StructName:  "MFCreateImpl",
		Interface:   "MFCreate",
		SourceFile:  "mfcreate/mf_create.go",
	},
	{
		Module:      "mftransfer",
		PackageName: "mftransfer",
		ImportPath:  "project2-mf-implementations/mftransfer",
		StructName:  "MFTransferImpl",
		Interface:   "MFTransfer",
		SourceFile:  "mftransfer/mf_transfer.go",
	},
	{
		Module:      "mfupdate",
		PackageName: "mfupdate",
		ImportPath:  "project2-mf-implementations/mfupdate",
		StructName:  "MFUpdateImpl",
		Interface:   "MFUpdate",
		SourceFile:  "mfupdate/mf_update.go",
	},
	{
		Module:      "mfdelete",
		PackageName: "mfdelete",
		ImportPath:  "project2-mf-implementations/mfdelete",
		StructName:  "MFDeleteImpl",
		Interface:   "MFDelete",
		SourceFile:  "mfdelete/mf_delete.go",
	},
}

func main() {
	banner("SEBI MUTUAL FUND — INJECTION PIPELINE")

	// ── STEP 1: Project1 structure ────────────────────────────────────────────
	step(1, "Project1 — MF backend skeleton (stubs only)")
	fmt.Println(`
  project1-mf-backend/
  ├── cmd/server/main.go              Echo server, calls bootstrap.RegisterAll()
  ├── internal/contracts/interfaces.go  MFCreateService, MFTransferService, etc.
  ├── internal/registry/registry.go   Register*/Get* injection points
  ├── internal/service/mf_service.go  Stubs — delegate to registry (NO impl)
  ├── internal/handler/mf_handler.go  Echo handlers for all 4 routes
  └── internal/bootstrap/bootstrap.go PLACEHOLDER — will be overwritten below

  Routes exposed by project1:
    POST   /mutualfund           →  handler → service → MFCreate()
    POST   /mutualfund/transfer  →  handler → service → MFTransfer()
    PUT    /mutualfund           →  handler → service → MFUpdate()
    DELETE /mutualfund           →  handler → service → MFDelete()

  Without injection: every route returns "implementation not injected"
`)

	// ── STEP 2: Project2 implementations ─────────────────────────────────────
	step(2, "Project2 — Concrete MF implementations (separate module)")
	fmt.Println(`
  project2-mf-implementations/
  ├── mfcreate/mf_create.go    MFCreateImpl  → KYC check, PAN validate, unit allotment
  ├── mftransfer/mf_transfer.go MFTransferImpl → exit load, STCG/LTCG, unit switch
  ├── mfupdate/mf_update.go    MFUpdateImpl  → OTP verify, SIP/bank/nominee update
  └── mfdelete/mf_delete.go    MFDeleteImpl  → lock-in check, TDS calc, redemption
`)

	// ── STEP 3: Open DB ───────────────────────────────────────────────────────
	step(3, "Opening SQLite — storing implementations from project2")

	store, err := db.NewStore(dbPath)
	if err != nil {
		log.Fatalf("DB open: %v", err)
	}
	defer store.Close()
	fmt.Printf("  SQLite ready: %s\n\n", dbPath)

	// Seed all 4 implementations into DB
	if err := seeder.SeedFromProject2(store, project2Dir, seedEntries); err != nil {
		log.Fatalf("Seed: %v", err)
	}

	// ── STEP 4: Yaegi smoke-test ──────────────────────────────────────────────
	step(4, "Yaegi smoke-test — validating each implementation")

	records, err := store.LoadValidated()
	if err != nil {
		log.Fatalf("Load validated: %v", err)
	}

	for _, r := range records {
		res := validator.Validate(r.SourceCode, r.PackageName, r.StructName)
		if res.Valid {
			fmt.Printf("  [YAEGI] %-15s ✓ symbol %s.%s OK\n", r.Module, r.PackageName, r.StructName)
		} else {
			// Yaegi has known limitations with struct types — not a blocker.
			// go build is the authoritative gate (Step 6).
			fmt.Printf("  [YAEGI] %-15s ⚠  %s  (go build is the real gate)\n", r.Module, res.Error)
		}
	}

	// ── STEP 5: Gonja renders bootstrap.go ───────────────────────────────────
	step(5, "Gonja (Jinja2) renders bootstrap.go → injects into project1")

	fmt.Println(templategen.DebugContext(records))

	tplPath := filepath.Join(templatesDir, "bootstrap.go.gonja")
	fmt.Printf("  Template : %s\n", tplPath)
	fmt.Printf("  Output   : %s\n\n", bootstrapOutput)

	rendered, err := templategen.RenderToString(tplPath, records)
	if err != nil {
		log.Fatalf("Gonja render: %v", err)
	}

	if err := templategen.WriteToFile(bootstrapOutput, rendered); err != nil {
		log.Fatalf("Write bootstrap.go: %v", err)
	}

	fmt.Println("  ── Generated bootstrap.go (preview) ──────────────────────")
	preview := rendered
	if len(preview) > 900 {
		preview = preview[:900] + "\n  ... (truncated)"
	}
	fmt.Println(preview)
	fmt.Println("  ──────────────────────────────────────────────────────────\n")

	// Mark all records injected in DB
	for _, r := range records {
		_ = store.MarkInjected(r.ID)
	}
	fmt.Printf("  %d implementations marked as injected in DB.\n", len(records))

	// ── STEP 6: go build project1 ────────────────────────────────────────────
	step(6, "Building project1 — verifying injection compiles")

	abs1, _ := filepath.Abs(project1Dir)

	fmt.Println("  Running: gofmt -w .")
	if err := runner.GoFmt(abs1); err != nil {
		fmt.Printf("  [WARN] %v\n", err)
	} else {
		fmt.Println("  gofmt ✓")
	}

	fmt.Println("  Running: go build ./...")
	build := runner.GoBuild(abs1)
	if !build.Success {
		fmt.Printf("\n  [BUILD FAILED]\n%s\n", build.Error)
		fmt.Println()
		printWorkspaceHelp()
		os.Exit(1)
	}
	fmt.Println("  go build ✓  — project1 compiles with all 4 implementations wired in")

	// ── STEP 7: Final instructions ────────────────────────────────────────────
	step(7, "Start the SEBI MF API server")
	fmt.Println(`
  cd ../project1-mf-backend
  go run ./cmd/server

  Then test with curl:

  # CREATE — POST /mutualfund
  curl -s -X POST http://localhost:8080/mutualfund \
    -H "Content-Type: application/json" \
    -d '{
      "investor_id":     "INV001",
      "investor_name":   "Abhijeet Wankhade",
      "pan":             "ABCDE1234F",
      "kyc_status":      "VERIFIED",
      "demat_account_no":"1234567890123456",
      "scheme_code":     "INF204K01I28",
      "scheme_name":     "Axis Bluechip Fund - Growth",
      "amc":             "Axis AMC",
      "fund_category":   "EQUITY",
      "fund_type":       "GROWTH",
      "investment_mode": "LUMPSUM",
      "amount":          10000,
      "nav":             45.50,
      "units":           219.780,
      "risk_profile":    "MODERATE",
      "bank_account_no": "9876543210",
      "ifsc":            "HDFC0001234",
      "platform":        "API"
    }' | jq .

  # TRANSFER — POST /mutualfund/transfer
  curl -s -X POST http://localhost:8080/mutualfund/transfer \
    -H "Content-Type: application/json" \
    -d '{
      "investor_id":       "INV001",
      "pan":               "ABCDE1234F",
      "folio_number":      "INV-INV001-INF2-123456",
      "from_scheme_code":  "INF204K01I28",
      "from_scheme_name":  "Axis Bluechip Fund",
      "from_amc":          "Axis AMC",
      "redemption_units":  100,
      "from_nav":          45.50,
      "to_scheme_code":    "INF204K01T51",
      "to_scheme_name":    "Axis Midcap Fund",
      "to_amc":            "Axis AMC",
      "to_nav":            32.75,
      "transfer_type":     "SWITCH",
      "exit_load_applicable": false,
      "exit_load_percent": 0,
      "stcg_applicable":   true,
      "ltcg_applicable":   false
    }' | jq .

  # UPDATE — PUT /mutualfund
  curl -s -X PUT http://localhost:8080/mutualfund \
    -H "Content-Type: application/json" \
    -d '{
      "investor_id":   "INV001",
      "pan":           "ABCDE1234F",
      "folio_number":  "INV-INV001-INF2-123456",
      "scheme_code":   "INF204K01I28",
      "update_type":   "SIP_AMOUNT",
      "new_sip_amount": 2000,
      "reason":        "Increase monthly SIP",
      "auth_otp":      "123456"
    }' | jq .

  # DELETE (Redeem) — DELETE /mutualfund
  curl -s -X DELETE http://localhost:8080/mutualfund \
    -H "Content-Type: application/json" \
    -d '{
      "investor_id":       "INV001",
      "pan":               "ABCDE1234F",
      "folio_number":      "INV-INV001-INF2-123456",
      "scheme_code":       "INF204K01I28",
      "redemption_type":   "PARTIAL",
      "units":             50,
      "redemption_mode":   "UNITS",
      "bank_account_no":   "9876543210",
      "ifsc":              "HDFC0001234",
      "exit_load_applicable": false,
      "lock_in_period_over": true,
      "tds_percent":       10,
      "auth_otp":          "123456"
    }' | jq .
`)

	banner("PIPELINE COMPLETE")
}

func banner(msg string) {
	line := "════════════════════════════════════════════════════════════════"
	fmt.Printf("\n%s\n  %s\n%s\n\n", line, msg, line)
}

func step(n int, title string) {
	fmt.Printf("\n──────────────────────────────────────────────────────────────\n")
	fmt.Printf("  STEP %d: %s\n", n, title)
	fmt.Printf("──────────────────────────────────────────────────────────────\n")
}

func printWorkspaceHelp() {
	fmt.Println(`  ── Setup needed ──────────────────────────────────────────────
  Run these once from the mf-sebi/ root directory:

    go work init
    go work use ./project1-mf-backend
    go work use ./project2-mf-implementations
    go work use ./service-injector

    cd service-injector
    go get github.com/nikolalohinski/gonja/v2
    go get github.com/traefik/yaegi@latest
    go get modernc.org/sqlite
    go mod tidy

    cd ../project1-mf-backend
    go get github.com/labstack/echo/v4
    go mod tidy

  Then re-run:
    cd service-injector && go run ./cmd/injector
  ─────────────────────────────────────────────────────────────`)
}
