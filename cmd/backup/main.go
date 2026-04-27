// backup is a small CLI for creating, validating and restoring kita-springer
// SQLite backups offline. Useful when the server isn't running, or when you
// need to script periodic snapshots from cron/launchd.
//
// Usage:
//
//	kita-springer-backup export  [-db PATH] [-out PATH]
//	kita-springer-backup verify   -in PATH
//	kita-springer-backup restore  -in PATH [-db PATH] [-y] [-reset-password]
//
// Restore is conceptually identical to the web UI's POST /api/restore — it
// validates, atomically replaces the target file and (optionally) clears the
// password so the next server start lands in setup mode. By default the local
// CLI keeps the password (the assumption is that you have shell access and
// know what you're doing); pass -reset-password for parity with the web flow.
//
// IMPORTANT: do NOT run `restore` while the server is running. The server
// holds the file open; replacing it underneath causes undefined behaviour.
// Stop the service first.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pak/kita-springer-manager/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "export":
		cmdExport(os.Args[2:])
	case "verify":
		cmdVerify(os.Args[2:])
	case "restore":
		cmdRestore(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `kita-springer-backup — offline DB backup/restore

Commands:
  export   Create a clean single-file SQLite backup of the live DB.
  verify   Sanity-check that a file is a usable kita-springer backup.
  restore  Replace the live DB with a backup file. STOP THE SERVER FIRST.

Run "kita-springer-backup <command> -h" for command-specific flags.
`)
}

// defaultDBPath returns the DB the CLI should use unless -db is given:
// honours $DB_PATH (parity with the server), otherwise falls back to
// data/app.db relative to the current working directory — which is what
// you want when running the CLI from the project root or from a service
// install dir like /usr/local/kita-springer/.
func defaultDBPath() string {
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	return "data/app.db"
}

func cmdExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dbPath := fs.String("db", defaultDBPath(), "Source SQLite DB to back up")
	outPath := fs.String("out", "", "Output file (default: kita-springer-YYYY-MM-DD.db in CWD)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if *outPath == "" {
		*outPath = fmt.Sprintf("kita-springer-%s.db", time.Now().Format("2006-01-02"))
	}

	absDB, _ := filepath.Abs(*dbPath)
	fmt.Printf("Quell-DB: %s\n", absDB)

	if _, err := os.Stat(*dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "DB nicht gefunden. Pfad explizit angeben mit -db oder $DB_PATH setzen.\n")
		os.Exit(1)
	}
	if _, err := os.Stat(*outPath); err == nil {
		fmt.Fprintf(os.Stderr, "Zieldatei existiert bereits: %s — bitte umbenennen oder entfernen.\n", *outPath)
		os.Exit(1)
	}

	// Open the source DB read-only to avoid running migrations on a backup
	// target. WAL mode + busy_timeout still apply via the Open helper, so a
	// running server's writes don't block us.
	conn, err := db.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB öffnen: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	abs, _ := filepath.Abs(*outPath)
	if _, err := conn.Exec("VACUUM INTO ?", abs); err != nil {
		fmt.Fprintf(os.Stderr, "VACUUM INTO: %v\n", err)
		os.Exit(1)
	}

	stat, _ := os.Stat(*outPath)
	fmt.Printf("✓ Backup erstellt: %s (%d bytes)\n", *outPath, stat.Size())
}

func cmdVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	in := fs.String("in", "", "Backup-Datei zum Prüfen")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *in == "" {
		fmt.Fprintln(os.Stderr, "-in fehlt")
		fs.Usage()
		os.Exit(2)
	}

	if err := db.ValidateBackup(*in); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Ungültig: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Gültiger kita-springer-Backup: %s\n", *in)
}

func cmdRestore(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	in := fs.String("in", "", "Backup-Datei zum Einspielen")
	dbPath := fs.String("db", defaultDBPath(), "Ziel-DB (wird ersetzt!)")
	yes := fs.Bool("y", false, "Sicherheitsabfrage überspringen")
	reset := fs.Bool("reset-password", false, "Passwort nach Restore wipen (wie das Web-UI)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	if *in == "" {
		fmt.Fprintln(os.Stderr, "-in fehlt")
		fs.Usage()
		os.Exit(2)
	}

	if err := db.ValidateBackup(*in); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Backup ungültig: %v\n", err)
		os.Exit(1)
	}

	target, _ := filepath.Abs(*dbPath)
	source, _ := filepath.Abs(*in)
	fmt.Printf("Backup-Datei: %s\n", source)
	fmt.Printf("Ziel-DB:      %s\n", target)
	if *reset {
		fmt.Println("Passwort wird nach Restore zurückgesetzt (-reset-password).")
	}

	if !*yes {
		if !confirm("Aktuelle Datenbank wird unwiederbringlich überschrieben. Fortfahren? [y/N] ") {
			fmt.Println("Abgebrochen.")
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Zielverzeichnis anlegen: %v\n", err)
		os.Exit(1)
	}

	// Stage to a temp file on the same filesystem as the target so the final
	// rename is atomic. We *copy* the input rather than rename it, so the
	// caller keeps the original backup file.
	staged, err := os.CreateTemp(filepath.Dir(target), "restore-*.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Temp-Datei: %v\n", err)
		os.Exit(1)
	}
	stagedPath := staged.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(stagedPath)
		}
	}()
	if err := copyFile(*in, staged); err != nil {
		_ = staged.Close()
		fmt.Fprintf(os.Stderr, "Kopieren: %v\n", err)
		os.Exit(1)
	}
	if err := staged.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Stage close: %v\n", err)
		os.Exit(1)
	}

	// Sidecars from the previous DB belong to the old inode — clean up.
	_ = os.Remove(target + "-wal")
	_ = os.Remove(target + "-shm")

	if err := os.Rename(stagedPath, target); err != nil {
		fmt.Fprintf(os.Stderr, "Ersetzen: %v\n", err)
		os.Exit(1)
	}
	cleanup = false

	if *reset {
		conn, err := db.Open(target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reopen: %v\n", err)
			os.Exit(1)
		}
		if _, err := conn.Exec(`UPDATE settings SET value='' WHERE key='auth_password_hash'`); err != nil {
			fmt.Fprintf(os.Stderr, "Passwort wipe: %v\n", err)
			conn.Close()
			os.Exit(1)
		}
		conn.Close()
		fmt.Println("Passwort gewiped — beim nächsten Server-Start läuft der Setup-Flow.")
	}

	fmt.Printf("✓ Restore abgeschlossen: %s\n", target)
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes" || line == "j" || line == "ja"
}

func copyFile(src string, dst *os.File) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if _, err := dst.ReadFrom(in); err != nil {
		return err
	}
	return nil
}
