package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jasonchiu/envlock/internal/backend"
	"github.com/jasonchiu/envlock/internal/config"
	"github.com/jasonchiu/envlock/internal/enroll"
	"github.com/jasonchiu/envlock/internal/keys"
	"github.com/jasonchiu/envlock/internal/recipients"
	"github.com/jasonchiu/envlock/internal/remote"
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
	case "login":
		return runLogin(args[1:])
	case "whoami":
		return runWhoami(args[1:])
	case "secrets":
		return runSecrets(args[1:])
	case "invite":
		return runInvite(args[1:])
	case "devices":
		return runDevices(args[1:])
	case "requests":
		return runRequests(args[1:])
	case "recipients":
		return runRecipients(args[1:])
	case "enroll":
		return runEnroll(args[1:])
	case "help", "--help", "-h":
		printRootUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printRootUsage() {
	fmt.Println("envlock - encrypted .env sharing")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  envlock <command> [args]")
	fmt.Println()
	fmt.Println("Core commands (implemented):")
	fmt.Println("  init                  Generate local device keypair")
	fmt.Println("  status                Show local/project setup status")
	fmt.Println("  project init          Initialize project config")
	fmt.Println("  project show          Show project config")
	fmt.Println("  invite create         Create invite token (alias of enroll invite)")
	fmt.Println("  invite join           Join using invite token/url (alias of enroll join)")
	fmt.Println("  devices ls            List devices (alias of recipients list)")
	fmt.Println("  devices revoke        Revoke device (alias of recipients remove)")
	fmt.Println("  requests ls           List enrollment requests (alias of enroll list)")
	fmt.Println("  requests approve      Approve enrollment request (alias of enroll approve)")
	fmt.Println("  requests reject       Reject enrollment request (alias of enroll reject)")
	fmt.Println("  recipients list       List project recipients")
	fmt.Println("  recipients add        Add recipient (manual fallback)")
	fmt.Println("  recipients remove     Remove recipient")
	fmt.Println("  enroll invite         Create invite token for a new machine")
	fmt.Println("  enroll join           Submit join request using invite token")
	fmt.Println("  enroll list           List enrollment requests")
	fmt.Println("  enroll approve        Approve enrollment request")
	fmt.Println("  enroll reject         Reject enrollment request")
	fmt.Println()
	fmt.Println("Scaffolded (server-backed flow planned):")
	fmt.Println("  login                 Browser login (not implemented yet)")
	fmt.Println("  whoami                Show authenticated user (not implemented yet)")
	fmt.Println("  secrets               push/pull/ls/status/rekey command family (not implemented yet)")
}

func runLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	server := fs.String("server", "", "envlock server URL (planned)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("login does not accept positional arguments")
	}
	_ = server
	return errors.New("login is not implemented yet (planned: browser localhost callback with copy/paste fallback)")
}

func runWhoami(args []string) error {
	fs := flag.NewFlagSet("whoami", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("whoami does not accept positional arguments")
	}
	return errors.New("whoami is not implemented yet (server-backed auth is planned)")
}

func runSecrets(args []string) error {
	if len(args) == 0 {
		printSecretsUsage()
		return nil
	}
	switch args[0] {
	case "push", "pull", "ls", "status", "rekey":
		return fmt.Errorf("secrets %s is not implemented yet (server-backed secrets workflow planned)", args[0])
	case "help", "--help", "-h":
		printSecretsUsage()
		return nil
	default:
		return fmt.Errorf("unknown secrets command %q", args[0])
	}
}

func printSecretsUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock secrets push <path>")
	fmt.Println("  envlock secrets pull <name> [--out <path>] [--force]")
	fmt.Println("  envlock secrets ls")
	fmt.Println("  envlock secrets status")
	fmt.Println("  envlock secrets rekey <name>")
	fmt.Println("  envlock secrets rekey --all")
}

func runInvite(args []string) error {
	if len(args) == 0 {
		printInviteUsage()
		return nil
	}
	switch args[0] {
	case "create":
		return runEnrollInvite(args[1:])
	case "join":
		return runInviteJoin(args[1:])
	case "help", "--help", "-h":
		printInviteUsage()
		return nil
	default:
		return fmt.Errorf("unknown invite command %q", args[0])
	}
}

func printInviteUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock invite create [--ttl 15m]")
	fmt.Println("  envlock invite join <invite-token-or-url>")
	fmt.Println("  envlock invite join --token <invite-token-or-url>")
}

func runInviteJoin(args []string) error {
	return runEnrollJoin(args)
}

func runDevices(args []string) error {
	if len(args) == 0 {
		printDevicesUsage()
		return nil
	}
	switch args[0] {
	case "ls", "list":
		return runRecipientsList(args[1:])
	case "revoke", "remove":
		return runRecipientsRemove(args[1:])
	case "add":
		return runRecipientsAdd(args[1:])
	case "help", "--help", "-h":
		printDevicesUsage()
		return nil
	default:
		return fmt.Errorf("unknown devices command %q", args[0])
	}
}

func printDevicesUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock devices ls [--all]")
	fmt.Println("  envlock devices revoke <name|fingerprint>")
	fmt.Println("  envlock devices add <name> <age-public-key> [--note <text>]  # manual fallback")
}

func runRequests(args []string) error {
	if len(args) == 0 {
		printRequestsUsage()
		return nil
	}
	switch args[0] {
	case "ls", "list":
		return runEnrollList(args[1:])
	case "approve":
		return runEnrollApprove(args[1:])
	case "reject":
		return runEnrollReject(args[1:])
	case "help", "--help", "-h":
		printRequestsUsage()
		return nil
	default:
		return fmt.Errorf("unknown requests command %q", args[0])
	}
}

func printRequestsUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock requests ls [--all]")
	fmt.Println("  envlock requests approve <request-id> [--note <text>]")
	fmt.Println("  envlock requests reject <request-id> [--reason <text>]")
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
		rs, err := remote.New(context.Background(), proj)
		if err != nil {
			fmt.Printf("Recipients: unavailable (%v)\n", err)
		} else if r, err := rs.LoadRecipients(context.Background()); err == nil {
			fmt.Printf("Recipients (Tigris): %d active / %d total\n", r.ActiveCount(), len(r.Recipients))
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
	case "create":
		return runProjectInit(args[1:])
	case "use":
		return runProjectUse(args[1:])
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
	fmt.Println("  envlock project create --app <name> --bucket <bucket>   # alias (current Tigris path)")
	fmt.Println("  envlock project use <name>                               # planned server-backed flow")
	fmt.Println("  envlock project show")
}

func runProjectUse(args []string) error {
	fs := flag.NewFlagSet("project use", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: envlock project use <name>")
	}
	return errors.New("project use is not implemented yet (planned: select server-backed project)")
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
	rs, err := remote.New(context.Background(), proj)
	if err != nil {
		return fmt.Errorf("initialize remote metadata store: %w", err)
	}
	store, err := rs.LoadRecipients(context.Background())
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
	if err := rs.WriteRecipients(context.Background(), store); err != nil {
		return err
	}
	if err := config.WriteProject(projPath, proj); err != nil {
		return err
	}

	fmt.Printf("Project initialized: %s\n", projPath)
	fmt.Printf("Remote recipients object initialized in bucket %q under prefix %q\n", proj.Bucket, proj.Prefix)
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

func remoteStoreFromCWD(ctx context.Context) (backend.Store, config.Project, error) {
	proj, _, err := config.LoadProjectFromCWD()
	if err != nil {
		return nil, config.Project{}, err
	}
	rs, err := remote.New(ctx, proj)
	if err != nil {
		return nil, config.Project{}, err
	}
	return rs, proj, nil
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
	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}
	store, err := rs.LoadRecipients(context.Background())
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

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}
	store, err := rs.LoadRecipients(context.Background())
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
	if err := rs.WriteRecipients(context.Background(), store); err != nil {
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
	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}
	store, err := rs.LoadRecipients(context.Background())
	if err != nil {
		return err
	}
	if *hard {
		removed, err := store.Delete(query)
		if err != nil {
			return err
		}
		if err := rs.WriteRecipients(context.Background(), store); err != nil {
			return err
		}
		fmt.Printf("Deleted recipient %q (%s)\n", removed.Name, removed.Fingerprint)
		return nil
	}
	revoked, err := store.Revoke(query)
	if err != nil {
		return err
	}
	if err := rs.WriteRecipients(context.Background(), store); err != nil {
		return err
	}
	fmt.Printf("Revoked recipient %q (%s)\n", revoked.Name, revoked.Fingerprint)
	fmt.Println("Note: existing encrypted blobs remain decryptable until rekeyed.")
	return nil
}

func runEnroll(args []string) error {
	if len(args) == 0 {
		printEnrollUsage()
		return nil
	}
	switch args[0] {
	case "invite":
		return runEnrollInvite(args[1:])
	case "join":
		return runEnrollJoin(args[1:])
	case "list":
		return runEnrollList(args[1:])
	case "approve":
		return runEnrollApprove(args[1:])
	case "reject":
		return runEnrollReject(args[1:])
	case "help", "--help", "-h":
		printEnrollUsage()
		return nil
	default:
		return fmt.Errorf("unknown enroll command %q", args[0])
	}
}

func printEnrollUsage() {
	fmt.Println("Usage:")
	fmt.Println("  envlock enroll invite [--ttl 15m]")
	fmt.Println("  envlock enroll join <invite-token-or-url> [--name <device-name>]")
	fmt.Println("  envlock enroll join --token <invite-token-or-url> [--name <device-name>]")
	fmt.Println("  envlock enroll list [--all]")
	fmt.Println("  envlock enroll approve <request-id>")
	fmt.Println("  envlock enroll reject <request-id> [--reason <text>]")
}

func runEnrollInvite(args []string) error {
	fs := flag.NewFlagSet("enroll invite", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	ttl := fs.Duration("ttl", 15*time.Minute, "invite token time-to-live")
	keyName := fs.String("key-name", "default", "local key profile name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("enroll invite does not accept positional arguments")
	}
	if *ttl <= 0 {
		return errors.New("--ttl must be > 0")
	}

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}

	var createdBy string
	if keyPath, err := keys.DefaultKeyPath(*keyName); err == nil {
		if _, meta, err := keys.LoadIdentity(keyPath); err == nil {
			createdBy = meta.DeviceName
		}
	}

	invite, token, err := enroll.NewInvite(*ttl, createdBy)
	if err != nil {
		return err
	}
	if err := rs.SaveInvite(context.Background(), invite); err != nil {
		return err
	}

	fmt.Printf("Created invite: %s\n", invite.ID)
	fmt.Printf("Expires at: %s\n", invite.ExpiresAt.Format(time.RFC3339))
	fmt.Println("Invite storage: Tigris (project metadata)")
	fmt.Printf("Invite token (share with new machine): %s\n", token)
	return nil
}

func runEnrollJoin(args []string) error {
	fs := flag.NewFlagSet("enroll join", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	token := fs.String("token", "", "invite token from trusted machine")
	keyName := fs.String("key-name", "default", "local key profile name")
	deviceName := fs.String("name", "", "override device name for enrollment request")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return errors.New("usage: envlock enroll join <invite-token-or-url> [--name <device-name>]")
	}
	resolvedToken := strings.TrimSpace(*token)
	if resolvedToken == "" && fs.NArg() == 1 {
		resolvedToken = strings.TrimSpace(fs.Arg(0))
	}
	resolvedToken = extractInviteToken(resolvedToken)
	if strings.TrimSpace(resolvedToken) == "" {
		return errors.New("invite token is required (pass <token-or-url> or --token)")
	}

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}

	inviteID, _, err := enroll.ParseToken(resolvedToken)
	if err != nil {
		return err
	}
	invite, err := rs.LoadInvite(context.Background(), inviteID)
	if err != nil {
		return err
	}
	if err := enroll.VerifyToken(invite, resolvedToken); err != nil {
		return err
	}
	if err := enroll.ValidateInviteForJoin(invite, time.Now().UTC()); err != nil {
		return err
	}

	keyPath, err := keys.DefaultKeyPath(*keyName)
	if err != nil {
		return err
	}
	id, meta, err := keys.LoadIdentity(keyPath)
	if err != nil {
		return fmt.Errorf("load local key (%s): %w (run `envlock init` first)", keyPath, err)
	}
	name := strings.TrimSpace(*deviceName)
	if name == "" {
		name = meta.DeviceName
	}
	if name == "" {
		name = "device"
	}

	existing, err := rs.ListRequests(context.Background())
	if err != nil {
		return err
	}
	req, err := enroll.NewJoinRequest(existing, invite, name, id.Recipient().String(), keys.Fingerprint(id.Recipient().String()))
	if err != nil {
		return err
	}
	if err := rs.SaveRequest(context.Background(), req); err != nil {
		return err
	}

	fmt.Printf("Created enrollment request: %s\n", req.ID)
	fmt.Println("Request storage: Tigris (project metadata)")
	fmt.Printf("Device: %s (%s)\n", req.DeviceName, req.Fingerprint)
	return nil
}

func extractInviteToken(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "envlock-invite-") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if tok := strings.TrimSpace(u.Query().Get("token")); tok != "" {
		return tok
	}
	return s
}

func runEnrollList(args []string) error {
	fs := flag.NewFlagSet("enroll list", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	all := fs.Bool("all", false, "include non-pending requests")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("enroll list does not accept positional arguments")
	}

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}
	requests, err := rs.ListRequests(context.Background())
	if err != nil {
		return err
	}
	printed := 0
	for _, r := range requests {
		if !*all && r.Status != enroll.RequestStatusPending {
			continue
		}
		printed++
		fmt.Printf("- %s\n", r.ID)
		fmt.Printf("  status: %s\n", r.Status)
		fmt.Printf("  device: %s\n", r.DeviceName)
		fmt.Printf("  fingerprint: %s\n", r.Fingerprint)
		fmt.Printf("  invite_id: %s\n", r.InviteID)
		fmt.Printf("  created_at: %s\n", r.CreatedAt.UTC().Format(time.RFC3339))
		if !r.DecisionAt.IsZero() {
			fmt.Printf("  decision_at: %s\n", r.DecisionAt.UTC().Format(time.RFC3339))
		}
		if r.DecisionNote != "" {
			fmt.Printf("  note: %s\n", r.DecisionNote)
		}
	}
	if printed == 0 {
		if *all {
			fmt.Println("No enrollment requests")
		} else {
			fmt.Println("No pending enrollment requests")
		}
	}
	return nil
}

func runEnrollApprove(args []string) error {
	fs := flag.NewFlagSet("enroll approve", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	note := fs.String("note", "", "optional approval note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: envlock enroll approve <request-id> [--note <text>]")
	}
	reqID := strings.TrimSpace(fs.Arg(0))

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}

	req, err := rs.LoadRequest(context.Background(), reqID)
	if err != nil {
		return err
	}
	if req.Status != enroll.RequestStatusPending {
		return fmt.Errorf("request %s is %s (expected pending)", req.ID, req.Status)
	}

	invite, err := rs.LoadInvite(context.Background(), req.InviteID)
	if err != nil {
		return err
	}
	if err := enroll.ValidateInviteForApproval(invite); err != nil {
		return err
	}

	store, err := rs.LoadRecipients(context.Background())
	if err != nil {
		return err
	}
	addErr := store.Add(recipients.Recipient{
		Name:        req.DeviceName,
		PublicKey:   req.PublicKey,
		Fingerprint: req.Fingerprint,
		CreatedAt:   time.Now().UTC(),
		Status:      recipients.StatusActive,
		Source:      "enroll-approve",
		Note:        "Added via enrollment request " + req.ID,
	})
	if addErr != nil && !errors.Is(addErr, recipients.ErrDuplicateRecipient) {
		return addErr
	}
	if err := rs.WriteRecipients(context.Background(), store); err != nil {
		return err
	}

	now := time.Now().UTC()
	req.Status = enroll.RequestStatusApproved
	req.DecisionAt = now
	req.DecisionNote = strings.TrimSpace(*note)
	if err := rs.SaveRequest(context.Background(), req); err != nil {
		return err
	}

	invite.Status = enroll.InviteStatusUsed
	invite.UsedByRequestID = req.ID
	invite.UsedAt = now
	if err := rs.SaveInvite(context.Background(), invite); err != nil {
		return err
	}

	if addErr != nil && errors.Is(addErr, recipients.ErrDuplicateRecipient) {
		fmt.Printf("Approved request %s (recipient already existed): %s (%s)\n", req.ID, req.DeviceName, req.Fingerprint)
	} else {
		fmt.Printf("Approved request %s and added recipient: %s (%s)\n", req.ID, req.DeviceName, req.Fingerprint)
	}
	return nil
}

func runEnrollReject(args []string) error {
	fs := flag.NewFlagSet("enroll reject", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	reason := fs.String("reason", "", "optional rejection reason")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: envlock enroll reject <request-id> [--reason <text>]")
	}
	reqID := strings.TrimSpace(fs.Arg(0))

	rs, _, err := remoteStoreFromCWD(context.Background())
	if err != nil {
		return err
	}
	req, err := rs.LoadRequest(context.Background(), reqID)
	if err != nil {
		return err
	}
	if req.Status != enroll.RequestStatusPending {
		return fmt.Errorf("request %s is %s (expected pending)", req.ID, req.Status)
	}
	req.Status = enroll.RequestStatusRejected
	req.DecisionAt = time.Now().UTC()
	req.DecisionNote = strings.TrimSpace(*reason)
	if err := rs.SaveRequest(context.Background(), req); err != nil {
		return err
	}
	fmt.Printf("Rejected request %s for %s (%s)\n", req.ID, req.DeviceName, req.Fingerprint)
	return nil
}
