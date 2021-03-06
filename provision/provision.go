// Copyright 2014 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package provision provides interfaces that need to be satisfied in order to
// implement a new provisioner on tsuru.
package provision

import (
	"fmt"
	"github.com/tsuru/tsuru/action"
	"github.com/tsuru/tsuru/app/bind"
	"io"
)

type Status string

func (s Status) String() string {
	return string(s)
}

const (
	// building - is while the unit is being provisioned,
	// it occurs during a deploy.
	StatusBuilding = Status("building")
	// error - when an error occurs caused by the application code.
	StatusError = Status("error")
	// is when an error occurs caused by tsuru internal problems.
	StatusDown = Status("down")
	// is when the app process is up but it is not bound to the
	// right host ("0.0.0.0") and right port ($PORT).
	// If your process is a worker its state will be unreachable.
	StatusUnreachable = Status("unreachable")
	// Is when the app process is up and bound to the right
	// host ("0.0.0.0") and right port ($PORT).
	StatusStarted = Status("started")
	// stopped - is when the Docker container is stopped
	StatusStopped = Status("stopped")
)

// Unit represents a provision unit. Can be a machine, container or anything
// IP-addressable.
type Unit struct {
	Name    string
	AppName string
	Type    string
	Ip      string
	Status  Status
}

//  GetIp returns the Unit.IP.
func (u *Unit) GetIp() string {
	return u.Ip
}

// Available returns true if the unit status is started or unreachable.
func (u *Unit) Available() bool {
	return u.Status == StatusStarted || u.Status == StatusUnreachable
}

// Named is something that has a name, providing the GetName method.
type Named interface {
	GetName() string
}

// App represents a tsuru app.
//
// It contains only relevant information for provisioning.
type App interface {
	Named
	// Log should be used to log messages in the app.
	Log(message, source, unit string) error

	// GetPlatform returns the platform (type) of the app. It is equivalent
	// to the Unit `Type` field.
	GetPlatform() string

	// GetDeploy returns the deploys that an app has.
	GetDeploys() uint

	Units() []Unit
	RemoveUnit(id string) error

	// Run executes the command in app units. Commands executed with this
	// method should have access to environment variables defined in the
	// app.
	Run(cmd string, w io.Writer, once bool) error

	Restart(io.Writer) error

	SerializeEnvVars() error

	Envs() map[string]bind.EnvVar

	// Ready marks the app as ready for deployment.
	Ready() error

	GetMemory() int
	GetSwap() int
	GetUpdatePlatform() bool
}

type CNameManager interface {
	SetCName(app App, cname string) error
	UnsetCName(app App, cname string) error
}

// ArchiveDeployer is a provisioner that can deploy archives.
type ArchiveDeployer interface {
	ArchiveDeploy(app App, archiveURL string, w io.Writer) error
}

// GitDeployer is a provisioner that can deploy the application from a Git
// repository.
type GitDeployer interface {
	GitDeploy(app App, version string, w io.Writer) error
}

// Provisioner is the basic interface of this package.
//
// Any tsuru provisioner must implement this interface in order to provision
// tsuru apps.
type Provisioner interface {
	// Provision is called when tsuru is creating the app.
	Provision(App) error

	// Destroy is called when tsuru is destroying the app.
	Destroy(App) error

	// AddUnits adds units to an app. The first parameter is the app, the
	// second is the number of units to be added.
	//
	// It returns a slice containing all added units
	AddUnits(App, uint) ([]Unit, error)

	// RemoveUnit removes a unit from the app. It receives the app and the name
	// of the unit to be removed.
	RemoveUnit(App, string) error

	// ExecuteCommand runs a command in all units of the app.
	ExecuteCommand(stdout, stderr io.Writer, app App, cmd string, args ...string) error

	// ExecuteCommandOnce runs a command in one unit of the app.
	ExecuteCommandOnce(stdout, stderr io.Writer, app App, cmd string, args ...string) error

	Restart(App) error
	Stop(App) error

	// Start start the app units.
	Start(App) error

	// CollectStatus returns information about all provisioned units. It's used
	// by tsuru collector when updating the status of apps in the database.
	CollectStatus() error

	// Addr returns the address for an app.
	//
	// tsuru will use this method to get the IP (although it might not be
	// an actual IP, collector calls it "IP") of the app from the
	// provisioner.
	Addr(App) (string, error)

	// Swap change the router between two apps.
	Swap(App, App) error

	// Units returns information about units by App.
	Units(App) []Unit
}

// CustomizedDeployPipelineProvisioner is a provisioner with a customized
// deploy pipeline.
type CustomizedDeployPipelineProvisioner interface {
	DeployPipeline() *action.Pipeline
}

// ExtensibleProvisioner is a provisioner where administrators can manage
// platforms (automatically adding, removing and updating platforms).
type ExtensibleProvisioner interface {
	PlatformAdd(name string, args map[string]string, w io.Writer) error
	PlatformUpdate(name string, args map[string]string, w io.Writer) error
}

var provisioners = make(map[string]Provisioner)

// Register registers a new provisioner in the Provisioner registry.
func Register(name string, p Provisioner) {
	provisioners[name] = p
}

// Get gets the named provisioner from the registry.
func Get(name string) (Provisioner, error) {
	p, ok := provisioners[name]
	if !ok {
		return nil, fmt.Errorf("Unknown provisioner: %q.", name)
	}
	return p, nil
}

// Registry returns the list of registered provisioners.
func Registry() []Provisioner {
	registry := make([]Provisioner, 0, len(provisioners))
	for _, p := range provisioners {
		registry = append(registry, p)
	}
	return registry
}

type Error struct {
	Reason string
	Err    error
}

func (e *Error) Error() string {
	var err string
	if e.Err != nil {
		err = e.Err.Error() + ": " + e.Reason
	} else {
		err = e.Reason
	}
	return err
}
