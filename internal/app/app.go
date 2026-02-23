package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jasonchiu/envlock-com/internal/config"
	"github.com/jasonchiu/envlock-com/internal/keys"
	"github.com/jasonchiu/envlock-com/internal/recipients"
)

func Run(args []string) error {
	if len(args) == 0 {
		printRootUsage()
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "status":
		return runStatus(args[1:])
	case "project":
		return runProject(args[1:])
	case "recipients":
		return runRecipients(args[1:])
	case "help", "--help", "-h":
		printRootUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printRootUsage() {
	fmt.Println("envlock - encrypted .env sharing over Tigris")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  envlock <command> [args]")
	fmt.Println()
	fmt.Println("Core commands (implemented):")
	fmt.Println("  init                  Generate local device keypair")
	fmt.Println("  status                Show local/project setup status")
	fmt.Println("  project init          Initialize project config")
	fmt.Println("  project show          Show project config")
	fmt.Println("  recipients list       List project recipients")
	fmt.Println("  recipients add        Add recipient (manual fallback)")
	fmt.Println("  recipients remove     Remove recipient")
	fmt.Println()
	fmt.Println("Planned commands:")
	fmt.Println("  push, pull, rekey, enroll")
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	name := fs.String("name", "", "device name (defaults to hostname)")
	keyName := fs.String("key-name", "default", "local key profile name")
	force := fs.Bool("force", false, "overwrite existing key if present")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("init does not accept positional arguments")
	}

	deviceName := strings.TrimSpace(*name)
	if deviceName == "" {
		host, err := os.Hostname()
		if err != nil || strings.TrimSpace(host) == "" {
			deviceName = "device"
		} else {
			deviceName = host
		}
	}

	generated, err := keys.Generate(deviceName)
	if err != nil {
		return err
	}

	path, err := keys.DefaultKeyPath(*keyName)
	if err != nil {
		return err
	}
	if err := keys.WriteIdentity(path, generated, *force); err != nil {
		return err
	}

	fmt.Printf("Created local device key: %s\n", path)
	fmt.Printf("Device name: %s\n", generated.DeviceName)
	fmt.Printf("Public key: %s\n", generated.Recipient.String())
	fmt.Printf("Fingerprint: %s\n", keys.Fingerprint(generated.Recipient.String()))
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	keyName := fs.String("key-name", "default", "local key profile name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("status does not accept positional arguments")
	}

	keyPath, err := keys.DefaultKeyPath(*keyName)
	if err != nil {
		return err
	}
	fmt.Printf("Key path: %s\n", keyPath)
	if st, err := os.Stat(keyPath); err == nil {
		fmt.Printf("Local key: present (%d bytes)\n", st.Size())
		id, meta, err := keys.LoadIdentity(keyPath)
		if err == nil {
			fmt.Printf("Device name: %s\n", meta.DeviceName)
			fmt.Printf("Public key: %s\n", id.Recipient().String())
			fmt.Printf("Fingerprint: %s\n", keys.Fingerprint(id.Recipient().String()))
		}
	} else if os.IsNotExist(err) {
		fmt.Println("Local key: missing")
	} else {
		return err
	}

	proj, projPath, err := config.LoadProjectFromCWD()
	if err == nil {
		fmt.Printf("Project config: %s\n", projPath)
		fmt.Printf("App: %s\n", proj.AppName)
		fmt.Printf("Bucket: %s\n", proj.Bucket)
		fmt.Printf("Prefix: %s\n", proj.Prefix)
		rPath := filepath.Join(filepath.Dir(projPath), "recipients.json")
		if r, err := recipients.Load(rPath); err == nil {
			fmt.Printf("Recipients: %d active / %d total\n", r.ActiveCount(), len(r.Recipients))
		} else if os.IsNotExist(err) {
			fmt.Println("Recipients: none (missing .envlock/recipients.json)")
		} else {
			return err
		}
		return nil
	}
	if errors.Is(err, config.ErrProjectNotFound) {
		fmt.Println("Project config: not found in current directory")
		return nil
	}
	return err
}

func runProject(args []string) error {
	if len(args) == 0 {
		printProjectUsage()
		return nil
	}
	switch args[0] {
	case "init":
		return runProjectInit(args[1:])
	case "show":
		return runProjectShow(args[1:])
	case "help", "--help", "-h":
		printProjectUsage()
		return nil
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func printProjectUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock project init --app <name> --bucket <bucket>")
	fmt.Println("  envlock project show")
}

func runProjectInit(args []string) error {
	fs := flag.NewFlagSet("project init", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	appName := fs.String("app", "", "application name (defaults to current folder name)")
	bucket := fs.String("bucket", "", "Tigris bucket name (required)")
	prefix := fs.String("prefix", "", "object prefix (defaults to <app>)")
	endpoint := fs.String("endpoint", "", "optional S3 endpoint override")
	keyName := fs.String("key-name", "default", "local key profile used for auto-adding this device")
	deviceName := fs.String("name", "", "recipient device name override")
	force := fs.Bool("force", false, "overwrite existing project config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("project init does not accept positional arguments")
	}
	if strings.TrimSpace(*bucket) == "" {
		return errors.New("--bucket is required")
	}
	app := strings.TrimSpace(*appName)
	if app == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		app = filepath.Base(cwd)
	}
	if strings.TrimSpace(app) == "" || app == "." || app == string(filepath.Separator) {
		return errors.New("could not infer app name from current directory; pass --app")
	}

	idPath, err := keys.DefaultKeyPath(*keyName)
	if err != nil {
		return err
	}
	id, meta, err := keys.LoadIdentity(idPath)
	if err != nil {
		return fmt.Errorf("load local key (%s): %w (run `envlock init` first)", idPath, err)
	}

	projectDir := config.ProjectDirPath(".")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}
	projPath := config.ProjectFilePath(".")
	if _, err := os.Stat(projPath); err == nil && !*force {
		return fmt.Errorf("project config already exists at %s (use --force to overwrite)", projPath)
	}

	pfx := strings.TrimSpace(*prefix)
	if pfx == "" {
		pfx = config.DefaultPrefix(app)
	}
	proj := config.Project{
		Version:  1,
		AppName:  app,
		Bucket:   strings.TrimSpace(*bucket),
		Prefix:   pfx,
		Endpoint: strings.TrimSpace(*endpoint),
	}
	if err := config.WriteProject(projPath, proj); err != nil {
		return err
	}

	rPath := filepath.Join(projectDir, "recipients.json")
	store, err := recipients.LoadOrInit(rPath)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(*deviceName)
	if name == "" {
		name = meta.DeviceName
	}
	if err := store.Add(recipients.Recipient{
		Name:        name,
		PublicKey:   id.Recipient().String(),
		Fingerprint: keys.Fingerprint(id.Recipient().String()),
		CreatedAt:   time.Now().UTC(),
		Status:      recipients.StatusActive,
		Source:      "local-init",
		Note:        "Added during project init",
	}); err != nil {
		if !errors.Is(err, recipients.ErrDuplicateRecipient) {
			return err
		}
	}
	if err := recipients.Write(rPath, store); err != nil {
		return err
	}

	fmt.Printf("Project initialized: %s\n", projPath)
	fmt.Printf("Recipients file: %s\n", rPath)
	fmt.Printf("Added local device recipient: %s (%s)\n", name, keys.Fingerprint(id.Recipient().String()))
	return nil
}

func runProjectShow(args []string) error {
	fs := flag.NewFlagSet("project show", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("project show does not accept positional arguments")
	}
	proj, projPath, err := config.LoadProjectFromCWD()
	if err != nil {
		return err
	}
	fmt.Printf("Project file: %s\n", projPath)
	fmt.Printf("Version: %d\n", proj.Version)
	fmt.Printf("App: %s\n", proj.AppName)
	fmt.Printf("Bucket: %s\n", proj.Bucket)
	fmt.Printf("Prefix: %s\n", proj.Prefix)
	if proj.Endpoint != "" {
		fmt.Printf("Endpoint: %s\n", proj.Endpoint)
	}
	return nil
}

func runRecipients(args []string) error {
	if len(args) == 0 {
		printRecipientsUsage()
		return nil
	}
	switch args[0] {
	case "list":
		return runRecipientsList(args[1:])
	case "add":
		return runRecipientsAdd(args[1:])
	case "remove":
		return runRecipientsRemove(args[1:])
	case "help", "--help", "-h":
		printRecipientsUsage()
		return nil
	default:
		return fmt.Errorf("unknown recipients command %q", args[0])
	}
}

func printRecipientsUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock recipients list")
	fmt.Println("  envlock recipients add <name> <age-public-key> [--note <text>]")
	fmt.Println("  envlock recipients remove <name|fingerprint>")
}

func recipientsFileInCWD() (string, error) {
	_, projPath, err := config.LoadProjectFromCWD()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(projPath), "recipients.json"), nil
}

func runRecipientsList(args []string) error {
	fs := flag.NewFlagSet("recipients list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	all := fs.Bool("all", false, "include revoked recipients")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("recipients list does not accept positional arguments")
	}
	rPath, err := recipientsFileInCWD()
	if err != nil {
		return err
	}
	store, err := recipients.Load(rPath)
	if err != nil {
		return err
	}

	items := make([]recipients.Recipient, 0, len(store.Recipients))
	for _, r := range store.Recipients {
		if !*all && r.Status != recipients.StatusActive {
			continue
		}
		items = append(items, r)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	if len(items) == 0 {
		fmt.Println("No recipients")
		return nil
	}
	for _, r := range items {
		fmt.Printf("- %s\n", r.Name)
		fmt.Printf("  status: %s\n", r.Status)
		fmt.Printf("  fingerprint: %s\n", r.Fingerprint)
		fmt.Printf("  source: %s\n", r.Source)
		fmt.Printf("  created_at: %s\n", r.CreatedAt.UTC().Format(time.RFC3339))
		if r.Note != "" {
			fmt.Printf("  note: %s\n", r.Note)
		}
	}
	return nil
}

func runRecipientsAdd(args []string) error {
	fs := flag.NewFlagSet("recipients add", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	note := fs.String("note", "", "optional note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return errors.New("usage: envlock recipients add <name> <age-public-key> [--note <text>]")
	}
	name := fs.Arg(0)
	pub := fs.Arg(1)
	if err := keys.ValidateRecipientString(pub); err != nil {
		return fmt.Errorf("invalid recipient public key: %w", err)
	}

	rPath, err := recipientsFileInCWD()
	if err != nil {
		return err
	}
	store, err := recipients.LoadOrInit(rPath)
	if err != nil {
		return err
	}
	if err := store.Add(recipients.Recipient{
		Name:        name,
		PublicKey:   pub,
		Fingerprint: keys.Fingerprint(pub),
		CreatedAt:   time.Now().UTC(),
		Status:      recipients.StatusActive,
		Source:      "manual",
		Note:        strings.TrimSpace(*note),
	}); err != nil {
		return err
	}
	if err := recipients.Write(rPath, store); err != nil {
		return err
	}
	fmt.Printf("Added recipient %q (%s)\n", name, keys.Fingerprint(pub))
	return nil
}

func runRecipientsRemove(args []string) error {
	fs := flag.NewFlagSet("recipients remove", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	hard := fs.Bool("hard", false, "delete recipient instead of marking revoked")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: envlock recipients remove <name|fingerprint>")
	}
	query := fs.Arg(0)
	rPath, err := recipientsFileInCWD()
	if err != nil {
		return err
	}
	store, err := recipients.Load(rPath)
	if err != nil {
		return err
	}
	if *hard {
		removed, err := store.Delete(query)
		if err != nil {
			return err
		}
		if err := recipients.Write(rPath, store); err != nil {
			return err
		}
		fmt.Printf("Deleted recipient %q (%s)\n", removed.Name, removed.Fingerprint)
		return nil
	}
	revoked, err := store.Revoke(query)
	if err != nil {
		return err
	}
	if err := recipients.Write(rPath, store); err != nil {
		return err
	}
	fmt.Printf("Revoked recipient %q (%s)\n", revoked.Name, revoked.Fingerprint)
	fmt.Println("Note: existing encrypted blobs remain decryptable until rekeyed.")
	return nil
}
