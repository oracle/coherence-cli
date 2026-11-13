/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Managed saves serialize effective Viper settings. Ordinary configuration writes
// and F4's raw disk-only migration transaction retain their separate semantics.
type managedConfigFile interface {
	io.Writer
	Name() string
	Sync() error
	Close() error
}

// File-operation seams keep persistence failure tests isolated from user files.
var createManagedConfigTemp = func(dir, pattern string) (managedConfigFile, error) {
	return os.CreateTemp(dir, pattern)
}

var renameManagedConfig = os.Rename

// Write a complete replacement beside the destination and rename only after success.
// A failed serialization/write must leave the prior on-disk settings intact.
func writeManagedConfigAtomically() error {
	filename := viper.ConfigFileUsed()
	if filename == "" {
		return fmt.Errorf("no configuration file selected")
	}
	destination := filename
	info, err := os.Lstat(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		destination, err = filepath.EvalSymlinks(filename)
		if err != nil {
			return fmt.Errorf("unable to resolve configuration symlink %s: %w", filename, err)
		}
	}
	f, err := createManagedConfigTemp(filepath.Dir(destination), ".cohctl-management-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	// Viper uses the selected config type (or selected file's extension), just
	// as ReadInConfig does. WriteConfigAs would select the temporary extension.
	if err = viper.WriteConfigTo(f); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return renameManagedConfig(name, destination)
}

var writeManagedConfig = writeManagedConfigAtomically

func persistManagedConnection(connection ClusterConnection, create bool) error {
	oldClusters, oldContext := Config.Clusters, viper.Get(currentContextKey)
	oldViperClusters := viper.Get(clusterKey)
	updated := append([]ClusterConnection(nil), oldClusters...)
	if create {
		updated = append(updated, connection)
	} else {
		found := false
		for i := range updated {
			if updated[i].Name == connection.Name {
				updated[i] = connection
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unable to persist unknown cluster %s", connection.Name)
		}
	}
	// Use Viper's configuration field names for every codec. In particular,
	// ClusterConnection's public JSON output tags (url, clusterVersionParam)
	// differ from the field names used by Viper's configuration decoder.
	settings, err := configurationClusters(updated)
	if err != nil {
		return fmt.Errorf("unable to encode management settings: %w", err)
	}
	viper.Set(clusterKey, settings)
	if create {
		viper.Set(currentContextKey, connection.Name)
	}
	if err := writeManagedConfig(); err != nil {
		viper.Set(clusterKey, oldViperClusters)
		viper.Set(currentContextKey, oldContext)
		return fmt.Errorf("unable to save management settings; previous configuration retained: %w", err)
	}
	Config.Clusters = updated
	if create {
		Config.CurrentContext = connection.Name
	}
	return nil
}
