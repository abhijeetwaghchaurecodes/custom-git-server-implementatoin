package seeder

import (
	"crypto/sha256"
	"fmt"
	"os"
	"service-injector/db"
)

// Entry describes one implementation file to seed from project2.
type Entry struct {
	Module      string
	PackageName string
	ImportPath  string
	StructName  string
	Interface   string // must match registry.RegisterMF* suffix e.g. "MFCreate"
	SourceFile  string // relative path inside project2 root
}

// SeedFromProject2 reads each implementation source from project2,
// computes sha256, and stores as a validated record in SQLite.
func SeedFromProject2(store *db.Store, project2Root string, entries []Entry) error {
	for _, e := range entries {
		srcPath := project2Root + "/" + e.SourceFile
		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", srcPath, err)
		}

		checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(srcBytes))

		id, err := store.Save(db.ImplementationRecord{
			Module:      e.Module,
			PackageName: e.PackageName,
			ImportPath:  e.ImportPath,
			StructName:  e.StructName,
			Interface:   e.Interface,
			SourceCode:  string(srcBytes),
			Checksum:    checksum,
			Status:      "validated",
			GoVersion:   "1.21",
		})
		if err != nil {
			return fmt.Errorf("store %s: %w", e.Module, err)
		}
		fmt.Printf("  [DB] %-15s stored  id=%-3d  checksum=%s...\n",
			e.Module, id, checksum[:26])
	}
	return nil
}
