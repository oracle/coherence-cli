/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oracle/coherence-cli/pkg/fetcher"
	"github.com/spf13/viper"
)

// migrationFile permits failures in each persistence phase to be tested without
// changing process-wide filesystem operations or touching a user's configuration.
type migrationFile interface {
	io.WriteCloser
	Name() string
	Chmod(os.FileMode) error
	Sync() error
}

type configMigrationOps struct {
	readFile   func(string) ([]byte, error)
	createTemp func(string, string) (migrationFile, error)
	restrict   func(migrationFile, os.FileMode) error
	replace    func(string, string) error
	syncDir    func(string) error
}

func defaultConfigMigrationOps() configMigrationOps {
	return configMigrationOps{
		readFile:   os.ReadFile,
		createTemp: func(dir, pattern string) (migrationFile, error) { return os.CreateTemp(dir, pattern) },
		restrict:   restrictMigrationFile,
		replace:    replaceMigrationFile,
		syncDir:    syncMigrationDirectory,
	}
}

type configMigrationResult struct {
	contents []byte
	backup   string
	names    []string
}

// migrateLegacyTLSConfig is file-only: neither the caller's Viper overrides nor
// AutomaticEnv can affect eligibility. It returns nil when no migration is due.
func migrateLegacyTLSConfig(path string, ops configMigrationOps) (*configMigrationResult, error) {
	original, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read legacy TLS configuration %q: %w", path, err)
	}
	format := strings.TrimPrefix(filepath.Ext(path), ".")
	// Use the raw codec, not a typed struct or AllSettings: preserve unknown keys,
	// including connection extensions, null values, literal dots, and key spelling.
	registry := viper.NewCodecRegistry()
	data := make(map[string]any)
	if err = decodeConfigForUpdate(format, original, data); err != nil {
		return nil, fmt.Errorf("decode legacy TLS configuration %q: %w", path, err)
	}
	if _, err = migrationKey(data, ignoreCertsContextKey); err != nil {
		return nil, fmt.Errorf("read legacy TLS configuration %q: %w", path, err)
	}
	reader := viper.New()
	reader.SetConfigType(format)
	if err = reader.ReadConfig(bytes.NewReader(original)); err != nil {
		return nil, fmt.Errorf("decode legacy TLS configuration %q: %w", path, err)
	}
	var persisted CoherenceCLIConfig
	if err = reader.Unmarshal(&persisted); err != nil {
		return nil, fmt.Errorf("load legacy TLS configuration %q: %w", path, err)
	}
	if !persisted.IgnoreInvalidCerts {
		return nil, nil
	}

	names, err := prepareLegacyTLSMigration(data)
	if err != nil {
		return nil, fmt.Errorf("prepare legacy TLS configuration %q: %w", path, err)
	}
	encoder, err := registry.Encoder(format)
	if err != nil {
		return nil, err
	}
	migrated, err := encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("encode migrated TLS configuration %q: %w", path, err)
	}
	// Validate the complete replacement before creating recovery files.
	if err = reader.ReadConfig(bytes.NewReader(migrated)); err == nil {
		persisted = CoherenceCLIConfig{}
		err = reader.Unmarshal(&persisted)
	}
	if err != nil {
		return nil, fmt.Errorf("validate migrated TLS configuration %q: %w", path, err)
	}

	// Follow a selected configuration symlink without replacing the link itself.
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve configuration %q: %w", path, err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("configuration %q must be a regular file", path)
	}
	dir, base := filepath.Dir(target), filepath.Base(target)
	stamp := time.Now().UTC().Format("2006-01-02_15-04-05.000000000_UTC")
	backup, err := writeMigrationFile(ops, dir, base+".tls-backup-"+stamp+"-*.bak", original, info.Mode())
	if err != nil {
		return nil, fmt.Errorf("back up configuration %q (incomplete backup %q); original not replaced: %w", path, backup, err)
	}
	staged, err := writeMigrationFile(ops, dir, "."+base+".tls-stage-*", migrated, info.Mode())
	if err == nil {
		err = ops.syncDir(dir)
	}
	if err != nil {
		return nil, fmt.Errorf("stage configuration %q; original not replaced; backup %q, staging file %q retained: %w", path, backup, staged, err)
	}
	// Do not knowingly overwrite a configuration changed while preparing it.
	currentInfo, statErr := os.Stat(target)
	current, readErr := os.ReadFile(target)
	if statErr != nil || readErr != nil || !os.SameFile(info, currentInfo) || !bytes.Equal(original, current) {
		return nil, fmt.Errorf("configuration %q changed during migration; retry after reviewing it; backup %q and staging file %q retained", path, backup, staged)
	}
	// Never remove/truncate the original as a fallback for a failed replacement.
	if err = ops.replace(staged, target); err != nil {
		return nil, fmt.Errorf("replace configuration %q; backup %q and staging file %q retained: %w", path, backup, staged, err)
	}
	if err = ops.syncDir(dir); err != nil {
		return nil, fmt.Errorf("sync replaced configuration %q; replacement may already be installed; backup %q retained: %w", path, backup, err)
	}
	return &configMigrationResult{contents: migrated, backup: backup, names: names}, nil
}

func prepareLegacyTLSMigration(data map[string]any) ([]string, error) {
	globalKey, err := migrationKey(data, ignoreCertsContextKey)
	if err != nil {
		return nil, err
	}
	clustersKey, err := migrationKey(data, clusterKey)
	if err != nil {
		return nil, err
	}
	var names []string
	if entries, ok := data[clustersKey].([]any); ok {
		for _, entry := range entries {
			connection, ok := entry.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid saved connection: expected an object")
			}
			choiceKey, err := migrationKey(connection, ignoreCertsContextKey)
			if err != nil {
				return nil, err
			}
			// Null, like a missing optional *bool, has no explicit choice.
			if connection[choiceKey] != nil {
				continue
			}
			urlKey, err := migrationKey(connection, "connectionurl")
			if err != nil {
				return nil, err
			}
			address, _ := connection[urlKey].(string)
			if !fetcher.IsValidHTTPSURL(address) {
				continue
			}
			nameKey, err := migrationKey(connection, "name")
			if err != nil {
				return nil, err
			}
			name, _ := connection[nameKey].(string)
			connection[choiceKey] = true
			names = append(names, name)
		}
	} else if data[clustersKey] != nil {
		return nil, fmt.Errorf("invalid saved connections: expected a list")
	}
	data[globalKey] = false
	return names, nil
}

// Match Viper's case-insensitive keys, refusing ambiguous spellings rather than
// allowing map iteration order to select a security setting.
func migrationKey(data map[string]any, wanted string) (string, error) {
	key := ""
	for candidate := range data {
		if strings.EqualFold(candidate, wanted) {
			if key != "" {
				return "", fmt.Errorf("ambiguous configuration key %q", wanted)
			}
			key = candidate
		}
	}
	if key == "" {
		key = strings.ToLower(wanted)
	}
	return key, nil
}

// Preserve JSON number tokens when updating raw configuration. Viper's default
// JSON decoder rounds unknown integers/decimals through float64. Other formats
// keep their existing codecs; ordinary typed runtime loading remains unchanged.
func decodeConfigForUpdate(format string, contents []byte, data map[string]any) error {
	if format == "json" {
		decoder := json.NewDecoder(bytes.NewReader(contents))
		decoder.UseNumber()
		if err := decoder.Decode(&data); err != nil {
			return err
		}
		// Match Unmarshal's refusal of trailing data, including a second value.
		if err := decoder.Decode(new(any)); err != io.EOF {
			return errors.New("unexpected trailing data in JSON configuration")
		}
		return nil
	}
	decoder, err := viper.NewCodecRegistry().Decoder(format)
	if err != nil {
		return err
	}
	return decoder.Decode(contents, data)
}

func writeMigrationFile(ops configMigrationOps, dir, pattern string, contents []byte, mode os.FileMode) (name string, err error) {
	file, err := ops.createTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name = file.Name()
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	// Restrict the empty file before any possibly sensitive bytes are written.
	if err = ops.restrict(file, mode); err != nil {
		return name, err
	}
	var n int
	if n, err = file.Write(contents); err == nil && n != len(contents) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = file.Sync()
	}
	return name, err
}

// writeStartupConfigValue changes only the intended startup field. Viper's
// AllSettings/WriteConfig drops null extensions and normalizes unknown keys;
// using it immediately after migration would undo the preservation above.
func writeStartupConfigValue(live *viper.Viper, key string, value any) error {
	return updateConfigFile(live.ConfigFileUsed(), key, func(data map[string]any) error {
		storedKey, err := migrationKey(data, key)
		if err != nil {
			return err
		}
		data[storedKey] = value
		return nil
	}, defaultConfigMigrationOps())
}

// updateConfigFile owns one original read/decode for a disk-only update. setting
// labels diagnostics; it is not a mutation scope or a Viper key path. update runs
// synchronously once to validate and mutate that raw map; it must not reread disk,
// write files, issue requests or mutate live state. Callers may publish captured
// map aliases or proposed live values only after a nil error. The final read only
// compares against the original bytes; it never supplies a new mutation snapshot.
// This conflict check is best effort: another writer can still change the file
// between the check and replacement.
func updateConfigFile(path, setting string, update func(map[string]any) error, ops configMigrationOps) error {
	original, err := ops.readFile(path)
	if err != nil {
		return err
	}
	format := strings.TrimPrefix(filepath.Ext(path), ".")
	registry := viper.NewCodecRegistry()
	data := make(map[string]any)
	if err = decodeConfigForUpdate(format, original, data); err != nil {
		return err
	}
	if err = update(data); err != nil {
		return err
	}
	encoder, err := registry.Encoder(format)
	if err != nil {
		return err
	}
	contents, err := encoder.Encode(data)
	if err != nil {
		return err
	}
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	staged, err := writeMigrationFile(ops, dir, "."+filepath.Base(target)+".config-stage-*", contents, info.Mode())
	if err == nil {
		err = ops.syncDir(dir)
	}
	if err == nil {
		current, readErr := ops.readFile(target)
		if readErr != nil || !bytes.Equal(current, original) {
			err = fmt.Errorf("configuration changed during update; retry after reviewing it")
		}
	}
	if err == nil {
		err = ops.replace(staged, target)
	}
	if err == nil {
		err = ops.syncDir(dir)
	}
	if err != nil {
		return fmt.Errorf("write setting %q in %q; staging file %q may be retained: %w", setting, path, staged, err)
	}
	return nil
}

// loadExistingConfig commits migration to disk before synchronizing live state.
func loadExistingConfig(live *viper.Viper, output io.Writer, ops configMigrationOps) (CoherenceCLIConfig, error) {
	var config CoherenceCLIConfig
	result, err := migrateLegacyTLSConfig(live.ConfigFileUsed(), ops)
	if err != nil {
		return config, err
	}
	if result != nil {
		if err = live.ReadConfig(bytes.NewReader(result.contents)); err != nil {
			return config, fmt.Errorf("reload migrated configuration; backup %q retained: %w", result.backup, err)
		}
		// A prior command in the same process may have installed a cluster override.
		live.Set(clusterKey, nil)
	}
	// Disable only the deprecated global environment override. Other AutomaticEnv
	// settings retain their existing behavior, including after later startup writes.
	live.Set(ignoreCertsContextKey, false)
	if err = live.Unmarshal(&config); err != nil {
		return config, err
	}
	if result != nil {
		if len(result.names) == 0 {
			_, _ = fmt.Fprintf(output, "Cleared legacy global ignoreInvalidCerts; no eligible HTTPS connections. Backup: %q.\n", result.backup)
		} else {
			_, _ = fmt.Fprintf(output, "Migrated legacy ignoreInvalidCerts to saved HTTPS connections %q and cleared the global setting. Backup: %q. Review the configuration and set unnecessary connection exceptions to false.\n", result.names, result.backup)
		}
	}
	return config, nil
}
