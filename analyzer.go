package cfg

import (
	"fmt"
	"io/ioutil"
)

// analyzer contains base data for analyzing all supported types of config files.
//
// The working file is considered to be the current local or active config file
// driving the local application.

// The master file is considered to be the 'compare to' file which could either
// be a local example file or an active remote config file on a server.
type analyzer struct {
	working   []byte
	master    []byte
	bash      *bash
	missing   []string
	different []string
}

// newAnalyzer returns a new analyzer
func newAnalyzer(c Config) (*analyzer, error) {
	a := analyzer{}

	// attempt to connect if a hostAlias is provided
	if len(c.HostAlias) > 0 {
		if err := a.connect(c.HostAlias); err != nil {
			return nil, err
		}
	}

	if err := a.read(c.WorkingPath, c.MasterPath); err != nil {
		return nil, err
	}

	return &a, nil
}

// ScanJson will scan two .json configuration files returning a slice
// of keys that exist in the master file and are missing in the working file
func ScanJson(c Config) ([]string, error) {
	analyzer, err := newJsonAnalyzer(c)
	if err != nil {
		return nil, err
	}

	analyzer.scan()

	return analyzer.missing, nil
}

// PrintJson uses ScanJson to retrieve a slice of missing keys and will then
// print out the difference / discrepencies between the master and working files
func PrintJson(c Config) error {
	analyzer, err := newJsonAnalyzer(c)
	if err != nil {
		return err
	}

	analyzer.scan()

	if len(analyzer.missing) > 0 {
		fmt.Printf("(!) found missing keys in %s: %+v\n", c.WorkingPath, analyzer.missing)
		return nil
	}

	equal, err := analyzer.equality()
	if err != nil {
		return err
	}

	if !equal {
		fmt.Printf("(!) %s and %s are different. Ignore if this is intentional\n", c.WorkingPath, c.MasterPath)
	}

	return nil
}

// ScanEnv will scan two .env configuration files returning a slice
// of keys that exist in the master file and are missing in the working file
func ScanEnv(c Config) ([]string, error) {
	analyzer, err := newEnvAnalyzer(c)
	if err != nil {
		return nil, err
	}

	analyzer.scan()

	return analyzer.missing, nil
}

// PrintEnv uses ScanEnv to retrieve a slice of missing keys and will then
// print out the difference / discrepencies between the master and working files
func PrintEnv(c Config) error {
	analyzer, err := newEnvAnalyzer(c)
	if err != nil {
		return err
	}

	analyzer.scan()

	if len(analyzer.missing) > 0 {
		fmt.Printf("(!) found missing keys in %s: %+v\n", c.WorkingPath, analyzer.missing)
		return nil
	}

	if len(analyzer.different) > 0 {
		fmt.Printf("(!) %s and %s are different. Ignore if this is intentional\n", c.WorkingPath, c.MasterPath)
		fmt.Printf("%+v\n", analyzer.different)
		return nil
	}

	return nil
}

// connect will attempt to connect to an external host via SSH. The idea is to
// return with an error if the connection fails, otherwise carry on until the
// connection is made again by reading in the contents of the remote config.
// currently this only supports connection via bash/ssh
func (a *analyzer) connect(hostAlias string) error {

	a.bash = newBash(hostAlias)

	if err := a.bash.ssh(); err != nil {
		fmt.Errorf("could not connect to host %s. %s", hostAlias, err)
	}

	return nil
}

// read will read a config file to []byte
func (a *analyzer) read(workingPath, masterPath string) error {

	var err error

	a.working, err = ioutil.ReadFile(workingPath)
	if err != nil {
		return fmt.Errorf("could not open %s. %s", workingPath, err)
	}

	// we have a remote file. read in the contents via scp
	if a.bash != nil {
		a.master, err = a.bash.scp(masterPath)
		if err != nil {
			return fmt.Errorf("could not open %s. %s", masterPath, err)
		}

		return nil
	}

	a.master, err = ioutil.ReadFile(masterPath)
	if err != nil {
		return fmt.Errorf("could not open %s. %s", masterPath, err)
	}

	return nil
}
