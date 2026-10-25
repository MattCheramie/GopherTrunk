package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattCheramie/GopherTrunk/internal/config"
	"github.com/MattCheramie/GopherTrunk/internal/ekc"
	"github.com/MattCheramie/GopherTrunk/internal/radio/p25"
)

// Writing and editing KFDtool containers (issue #1305): `gophertrunk ekc`
// grew create / add / remove / passwd / import so GopherTrunk can be the
// place keys are managed, not just read. Every GopherTrunk key type has a
// TIA-102 ALGID a container can hold (the RC4 family as ADP 0xAA, which is
// also how DMR Enhanced Privacy keys are keyed), so the native keystore IS
// the .ekc format: a container written here opens in KFDtool and the other
// way round, and the daemon loads either through encryption_key_files.
//
// Key material never travels on the command line (a flag value lands in
// shell history and ps): it is typed at a no-echo prompt or read from an
// environment variable, and nothing here prints it.

// ekcMinPassword is the shortest password a new container accepts.
const ekcMinPassword = 8

// ekcAlgorithmID is the TIA-102 ALGID a container stores an encryption_keys
// algorithm under, picked by key length where the algorithm has variants:
// des 0x81; tdes 0x83 (16-byte two-key) / 0x86 (24-byte three-key); aes 0x85
// (AES-128) / 0x84 (AES-256); the RC4 family (rc4, arc4, adp) 0xAA. It is the
// inverse of ekcAlgorithm, so a key round-trips to the algorithm it came in
// as.
func ekcAlgorithmID(alg string, keyLen int) (int, error) {
	switch norm := (config.EncryptionKeyConfig{Algorithm: alg}).NormalizedAlgorithm(); norm {
	case "des":
		return 0x81, nil
	case "tdes":
		if keyLen == 16 {
			return 0x83, nil
		}
		return 0x86, nil
	case "aes":
		if keyLen == 16 {
			return 0x85, nil
		}
		return 0x84, nil
	case "rc4":
		return 0xAA, nil
	default:
		return 0, fmt.Errorf("unknown algorithm %q (use rc4, adp, des, tdes or aes)", alg)
	}
}

// ekcKeyFromConfig validates an encryption_keys entry and turns it into a
// container key: a traffic key, or a KEK when kek is set.
func ekcKeyFromConfig(k config.EncryptionKeyConfig, name string, keyset int, kek bool) (ekc.Key, error) {
	if err := k.Validate(); err != nil {
		return ekc.Key{}, err
	}
	b, err := k.KeyBytes()
	if err != nil {
		return ekc.Key{}, err
	}
	alg, err := ekcAlgorithmID(k.Algorithm, len(b))
	if err != nil {
		return ekc.Key{}, err
	}
	return ekc.Key{
		Name: name, KeysetID: keyset, ActiveKeyset: true, KeyID: int(k.KeyID),
		AlgorithmID: alg, TEK: !kek, KEK: kek, Material: b,
	}, nil
}

// writeContainerFile encrypts c under password and replaces path with it
// atomically (a temp file in the same directory, renamed over it), mode 0600:
// a failed write never leaves half a container where the old one was.
func writeContainerFile(path string, c *ekc.Container, password string) error {
	data, err := ekc.Encode(c, password)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after the rename
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// ekcNewPassword is the password for a new (or re-keyed) container: from
// envName when given, else typed twice at the terminal.
func ekcNewPassword(envName, what string) (string, error) {
	var pw string
	if envName != "" {
		v, ok := os.LookupEnv(envName)
		if !ok {
			return "", fmt.Errorf("%s is not set", envName)
		}
		pw = v
	} else {
		a, ok, err := ekcPasswordPrompt("New password for " + what + ": ")
		if err != nil {
			return "", err
		}
		if !ok {
			return "", errors.New("no terminal to ask for a password: set it in an environment variable and pass its name")
		}
		b, _, err := ekcPasswordPrompt("Repeat the password: ")
		if err != nil {
			return "", err
		}
		if a != b {
			return "", errors.New("the two passwords differ")
		}
		pw = a
	}
	if len(pw) < ekcMinPassword {
		return "", fmt.Errorf("the password must be at least %d characters", ekcMinPassword)
	}
	return pw, nil
}

// ekcKeyMaterial is a key's hex: from envName when given, else typed at a
// no-echo prompt.
func ekcKeyMaterial(envName string, keyID int) (string, error) {
	if envName != "" {
		v, ok := os.LookupEnv(envName)
		if !ok {
			return "", fmt.Errorf("%s is not set", envName)
		}
		return v, nil
	}
	v, ok, err := ekcPasswordPrompt(fmt.Sprintf("Key for key ID %d (hex, not shown): ", keyID))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("no terminal to type the key at: put it in an environment variable and pass -key-env")
	}
	return v, nil
}

// ekcMain runs one `gophertrunk ekc` subcommand, writing its report to out.
// The bare form (`ekc -in file`, no subcommand) lists, as it always did.
func ekcMain(args []string, out io.Writer) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return ekcList(args, out)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		return ekcList(rest, out)
	case "create":
		return ekcCreate(rest, out)
	case "add":
		return ekcAdd(rest, out)
	case "remove":
		return ekcRemove(rest, out)
	case "passwd":
		return ekcPasswd(rest, out)
	case "import":
		return ekcImport(rest, out)
	}
	return fmt.Errorf("unknown subcommand %q (list, create, add, remove, passwd, import)", sub)
}

func ekcFlags(name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet("ekc "+name, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), usage)
		fs.PrintDefaults()
	}
	return fs
}

func ekcList(args []string, out io.Writer) error {
	fs := ekcFlags("list", `gophertrunk ekc [list] -in keys.ekc [-password-env VAR] [-keyset N]

Lists every key in the container (item, name, keyset, SLN, key ID,
algorithm, TEK or KEK) and whether a system with this file under
encryption_key_files would load it. Key material is never printed.`)
	in := fs.String("in", "", "the .ekc file")
	passwordEnv := fs.String("password-env", "", "read the password from this environment variable instead of asking")
	keyset := fs.Int("keyset", 0, "show what encryption_key_files would load with this keyset (0 = all keysets)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-in is required")
	}
	c, err := openKeyFile("", config.EncryptionKeyFileConfig{Path: *in, PasswordEnv: *passwordEnv})
	if err != nil {
		return err
	}
	writeEKCListing(out, c, *keyset)
	return nil
}

func ekcCreate(args []string, out io.Writer) error {
	fs := ekcFlags("create", `gophertrunk ekc create -out keys.ekc [-password-env VAR]

Creates an empty container. Asks for its password twice (or reads it from
-password-env); at least 8 characters.`)
	path := fs.String("out", "", "the .ekc file to create (must not exist)")
	passwordEnv := fs.String("password-env", "", "read the new password from this environment variable instead of asking")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-out is required")
	}
	if _, err := os.Stat(*path); err == nil {
		return fmt.Errorf("%s already exists; add keys to it with `gophertrunk ekc add`", *path)
	}
	pw, err := ekcNewPassword(*passwordEnv, *path)
	if err != nil {
		return err
	}
	if err := writeContainerFile(*path, ekc.New(), pw); err != nil {
		return err
	}
	fmt.Fprintf(out, "created %s (empty); add keys with `gophertrunk ekc add -in %s ...`\n", *path, *path)
	return nil
}

func ekcAdd(args []string, out io.Writer) error {
	fs := ekcFlags("add", `gophertrunk ekc add -in keys.ekc -key-id N -algorithm ALG [-name LABEL] [-keyset N] [-kek]
                   [-key-env VAR] [-password-env VAR]

Adds one key. ALG is rc4 (or adp; DMR Enhanced Privacy and P25 ADP), des,
tdes or aes; the key length picks the TDES / AES variant. The key is typed
at a prompt that does not show it, or read from -key-env; it is never a
command-line argument.`)
	path := fs.String("in", "", "the .ekc file")
	keyID := fs.Int("key-id", -1, "the key ID radios carry in their encryption sync / PI header (0..65535)")
	alg := fs.String("algorithm", "", "rc4 / adp, des, tdes or aes")
	name := fs.String("name", "", "a label for the key (default: \"key <id>\")")
	keyset := fs.Int("keyset", 1, "the KFD keyset to put it in")
	kek := fs.Bool("kek", false, "store it as a key-encryption key (GopherTrunk decrypts no voice with it)")
	keyEnv := fs.String("key-env", "", "read the key's hex from this environment variable instead of asking")
	passwordEnv := fs.String("password-env", "", "read the container password from this environment variable instead of asking")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *keyID < 0 || *keyID > 0xFFFF || *alg == "" || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-in, -key-id (0..65535) and -algorithm are required")
	}
	if *keyset < 1 {
		return fmt.Errorf("-keyset %d must be 1 or more", *keyset)
	}
	c, pw, err := openKeyFileWithPassword("", config.EncryptionKeyFileConfig{Path: *path, PasswordEnv: *passwordEnv})
	if err != nil {
		return err
	}
	material, err := ekcKeyMaterial(*keyEnv, *keyID)
	if err != nil {
		return err
	}
	label := *name
	if label == "" {
		label = fmt.Sprintf("key %d", *keyID)
	}
	k, err := ekcKeyFromConfig(config.EncryptionKeyConfig{KeyID: uint16(*keyID), Algorithm: *alg, Key: material}, label, *keyset, *kek)
	if err != nil {
		return fmt.Errorf("key ID %d: %w", *keyID, err)
	}
	item, err := c.Add(k)
	if err != nil {
		return err
	}
	if err := writeContainerFile(*path, c, pw); err != nil {
		return err
	}
	fmt.Fprintf(out, "added item %d to %s: %q, key ID %d, %s, keyset %d\n", item, *path, label, *keyID,
		ekcAlgorithmLabel(k.AlgorithmID), k.KeysetID)
	return nil
}

func ekcRemove(args []string, out io.Writer) error {
	fs := ekcFlags("remove", `gophertrunk ekc remove -in keys.ekc -item N [-password-env VAR]

Removes one key by its ITEM number (the first column of `+"`gophertrunk ekc list`"+`),
and drops it from any key group.`)
	path := fs.String("in", "", "the .ekc file")
	item := fs.Int("item", 0, "the item number to remove")
	passwordEnv := fs.String("password-env", "", "read the container password from this environment variable instead of asking")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || *item < 1 || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-in and -item are required")
	}
	c, pw, err := openKeyFileWithPassword("", config.EncryptionKeyFileConfig{Path: *path, PasswordEnv: *passwordEnv})
	if err != nil {
		return err
	}
	var gone ekc.Key
	for _, k := range c.Keys {
		if k.ID == *item {
			gone = k
		}
	}
	if !c.Remove(*item) {
		return fmt.Errorf("%s has no item %d (see `gophertrunk ekc list -in %s`)", *path, *item, *path)
	}
	if err := writeContainerFile(*path, c, pw); err != nil {
		return err
	}
	fmt.Fprintf(out, "removed item %d from %s: %q, key ID %d\n", *item, *path, gone.Name, gone.KeyID)
	return nil
}

func ekcPasswd(args []string, out io.Writer) error {
	fs := ekcFlags("passwd", `gophertrunk ekc passwd -in keys.ekc [-password-env VAR] [-new-password-env VAR]

Re-encrypts the container under a new password.`)
	path := fs.String("in", "", "the .ekc file")
	passwordEnv := fs.String("password-env", "", "read the current password from this environment variable instead of asking")
	newEnv := fs.String("new-password-env", "", "read the new password from this environment variable instead of asking")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-in is required")
	}
	c, _, err := openKeyFileWithPassword("", config.EncryptionKeyFileConfig{Path: *path, PasswordEnv: *passwordEnv})
	if err != nil {
		return err
	}
	pw, err := ekcNewPassword(*newEnv, *path)
	if err != nil {
		return err
	}
	if err := writeContainerFile(*path, c, pw); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s re-encrypted under the new password\n", *path)
	return nil
}

func ekcImport(args []string, out io.Writer) error {
	fs := ekcFlags("import", `gophertrunk ekc import -config gophertrunk.yaml -system NAME -out keys.ekc
                      [-password-env VAR]

Moves a system's plain-text encryption_keys into a container: creates -out
(asking for a new password) or adds to it (asking for its password), then
prints the encryption_key_files entry to put in the config in their place.
The config file itself is not changed.`)
	cfgPath := fs.String("config", "gophertrunk.yaml", "the config file")
	system := fs.String("system", "", "the trunking system whose encryption_keys to move")
	path := fs.String("out", "", "the .ekc file to write (created if missing)")
	passwordEnv := fs.String("password-env", "", "read the container password from this environment variable instead of asking")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *system == "" || *path == "" || fs.NArg() > 0 {
		fs.Usage()
		return errors.New("-system and -out are required")
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	var sys *config.SystemConfig
	for i := range cfg.Trunking.Systems {
		if cfg.Trunking.Systems[i].Name == *system {
			sys = &cfg.Trunking.Systems[i]
		}
	}
	if sys == nil {
		return fmt.Errorf("%s has no trunking system %q", *cfgPath, *system)
	}
	if len(sys.EncryptionKeys) == 0 {
		return fmt.Errorf("system %q has no encryption_keys to import", *system)
	}
	var c *ekc.Container
	var pw string
	if _, err := os.Stat(*path); err == nil {
		c, pw, err = openKeyFileWithPassword(*system, config.EncryptionKeyFileConfig{Path: *path, PasswordEnv: *passwordEnv})
		if err != nil {
			return err
		}
	} else {
		c = ekc.New()
		if pw, err = ekcNewPassword(*passwordEnv, *path); err != nil {
			return err
		}
	}
	added, n := 0, len(sys.EncryptionKeys)
	for i, k := range sys.EncryptionKeys {
		key, err := ekcKeyFromConfig(k, fmt.Sprintf("%s key %d", *system, k.KeyID), 1, false)
		if err != nil {
			return fmt.Errorf("encryption_keys[%d] (key ID %d): %w", i, k.KeyID, err)
		}
		if _, err := c.Add(key); err != nil {
			fmt.Fprintf(out, "skipped key ID %d: %v\n", k.KeyID, err)
			continue
		}
		added++
	}
	if err := writeContainerFile(*path, c, pw); err != nil {
		return err
	}
	env := *passwordEnv
	if env == "" {
		env = "GT_EKC_PASSWORD"
	}
	fmt.Fprintf(out, `wrote %d of %d key(s) from system %q to %s.

Now, in %s, replace that system's encryption_keys with:

      encryption_key_files:
        - path: %s
          password_env: %s   # or leave it out to be asked at startup

then delete the plain-text keys. Copies of the old config (backups, version
control) still hold them.
`, added, n, *system, *path, *cfgPath, *path, env)
	return nil
}

// ekcAlgorithmLabel renders an ALGID for a report line: "0x84 (AES-256)".
func ekcAlgorithmLabel(id int) string { return p25.FormatAlgorithm(uint8(id)) }
