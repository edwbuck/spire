# Reconfiguration in SPIRE

## Problem Statement

When reconfiguring the k8s_psat plugin, the only current means
to effect the reconfiguration is to restart the SPIRE Servers,
which will retrigger parsing of the server configuration file
and launch new k8s_psat plugins configured with the new settings.

## Basic Goals

The support for reconfiguration aims to permit an already running
SPIRE Server to load a new configuration and use it.  As there
are many smaller details involved, a few basic goals are outlined:

- A successful reconfiguration should launch in the latest
  successful reconfiguration upon shutdown and relaunch.
- Bad re-configurations should be ignored with feedback as to why
  they cannot be used.
- Changes made should either be accepted or rejected and the effect
  of the change should be felt immediately after acceptance.

## Undecided items

A few items in this write-up are still undecided.  Like most undecided
items, there are usually a few obvious choices:

### Server State and Configuration State

Many configurations apply to items that have a lifecycle.  Changes in
the configuration will imply that a configuration has a lifecycle.  To
clarify what is meant by lifecycle in this case, it is any item managed
by SPIRE that is created, maintained, and then destroyed.  Thus X.509
SVIDs may be deemed items that have a lifecycle, as are the SPIRE
entries that define them.

When dealing with items that have a lifecycle, there are differences
in opinion as to how to manage the lifecycle.  The differences roughly
break down as follows.  Note that each option below is mutually
exclusive:

-  The item should be checked against the new configuration, and if it
   cannot exist under the new configuraiton, it should be (within the
   limits of the software) recalled and destroyed.

-  The configuration should make no attempt to alter the as-launched
   lifecycle of an item, even if it couldn't be created under the
   current configuration, but should not support the creation of
   additional items of the old configuration when the new configuration
   can't support their creation.

Clearly the latter approach might require more detail as to what are
permissible items to remain in service.

### Means of Storing the New Configuration File

The means of loading a new configuration file is currently under debate.
As currently there is only one configuration file, this question has
never had to deal with the transition of one file to another. Here are
a few alternatives, and the list is non-exhaustive:

- No support for dual storage.  Backups of the old config would manually
  be created prior to putting the new configuration in place.

- Placing the new configuration in a non-conflicting file name / path,
  such that as the server accepts it, it overwrites the old config
  file.

- Extending the configuration subsystem with a plugin that permits
  configuration management based on differing policies and approaches
  defined by external configuration plugins.

Perhaps the intitial efforts might use a simple approach of having a
new file location for proposed new configuration files, and that
approach pivots to a plugin based approach in the future.

## Current Design Considerations

When adding new features to a product, integration of the new features
is deemed elegant and easy to use based on the concept that it doesn't
drastically alter how the product is perceived to already work.  SPIRE
lacks a single location where all design decisions are documented, so
this list attempts to detail the design decisions that are apparent
from extensive use of the product and readings of the source code.

### Single Config File per Server

Each SPIRE server has a single config file, which only differs during
the necessary config file upgrades performed between major releases.

### Plugins and Server Combined in Configuration File

The current configuration file combines both the plugin and server
configurations as one.  This is intentional, as the server launches
the plugins, configuring them as they are launched.

## Functional Points and Expected Points of Change

For the plugins to be reconfigured under the current design
considerations, the current handling of configuration items within
SPIRE must change.  Some of these points of change are new additions to
the current source code of SPIRE, while others are alterations to the
existing source code of SPIRE.  The overall functional points include:

- Config file change detection
- Config file loading
- Config file acceptance including
  - Presentation of the new config to the existing plugins
- Config file cutover including
  - Plugins acting on the new configuration

We will discuss each of the major points below, adding additional points
as necessary to support the major functional points.

### Config file change detection

This is one of the easier points to manage, and as such will likely be
the point where most people will have differening opinions on how the
change is detected.  There are basically two major approaches:

- The SPIRE server actively polls an item, either through periodic
  checking or a notification system like (inotify) to detect a new
  configuration file.

- The SPIRE server reacts to an command that informs the SPIRE server
  that a new configuration file is in place and should be considered.

The polling and command approaches both include a large degree of
flexibilty in implementation details.  For the moment, we will assume
that future decisions will weigh the options and select the best option
for SPIRE.

### Config file loading

Once the config file is detected, the new new config file will be loaded
creating a new Config struct.  The mechanisim of the loading will be
similar to the current loading, with the exception that it will not
depend or update any of the settings under the active configuration.

Some aspects of the config file loading may need cleanup work.  

SPIRE currently imports foreign structs which cleanly load sub-sections
of various aspects of the server config.  Probably we will not need to
alter this behavior.

SPIRE also loads some sections of the config as Abstract Syntax Tree
nodes under the configuration framework.  It then inspects these nodes
and performs AST node to struct replacement.  Ideally, such areas of
the config loading will have this logic centeralized into the loading
logic, and removed from the validation logic.

### Config File Acceptance

Acceptance will likely involve two distinct phases, which may be
combined into one overall process, provided that they are implemented
in order.  The first phase would ensure that values are valid in the
sense that they could launch a SPIRE server.  The second phase would
ensure that values are valid in the sense that a SPIRE server that is
already running with the currently applied values could transition the
configuration to the new value(s).  For the purposes of differentiating
the two kinds of acceptance, we will call configurations passing the
first set of tests "valid" and configurations passing both sets of tests
"valid" and "usable".

Under this approach, the overall process looks like:

1. Check if the config is valid
2. Report a configuration failure if any part of the config is not
   valid.
3. Check if the config is usable (in the narrow sense of being
   reconfigured)
4. Report a reconfiguration failure if any part of the config is not
   usable when considering the prior configuration.

Checks for validity and usability can be performed in the same API call.
The proposed API is to have the entire Config struct available, 
permitting any kind of between module / plugin agreement, should such
agreement become necessary in the future.

Each module would be responsible for checking the configuration elements
that concern the module, reporting errors as necessary.  This means that
reported errors would be localized to the module, as would validation
logic, and if a value impacts two different modules, each module could
report different sets of errors, based on their own individual needs.

Note: No service is required to support reconfiguration.  If a service
does not support reconfiguration, the same startup validation is applied
to new configurations, but usability errors are thrown for any element
that differs from the running configuration.

The "catalog" module currently launches the plugins, but we will call
it a "pluginmanager" to align with its role and recommended rename.  For
the intial implementation, the recommendation is that this module
respond to any reconfiguration request that alters the plugin name,
type, or count with an usability error.  Future implementations can
decide if certain plugins can be altered live.

### Config File Cutover

After the config is deemed usable, the main Config struct would be
replaced by the now-validated new Config struct.  This would then
trigger each user of the Config struct to check the values they can
react to for changes and to apply the necessary changes internally to
effect the new configuration.

Note: At this point in time, it is unclear if the cutover should be
automatic, there may be interest in awaiting a second command to
cutover after holding a validated new configuration.  For the purposes
here, we assume that the cutover is automatic.

Each module that supports one or more reconfigurations will have to
impement the logic to shift to the new configuration here.  

The "catalog" module currently launches the plugins, but we again
promote renaming it to "pluginmanager".  This module will be responsible
for shutting down any plugin that needs to be terminated, and launching
any new plugin that needs to be initialized, passing in the new
configuration.  If the plugin type, name, sha256sum, and count don't
indicate a change, then this plugin will only inform the already running
plugin of the new configuration, leaving the plugin to react
accordingly.

### Functional Points Recap

1. The system will act upon a config file change.
2. The system will parse the config file structurally.
3. The system will marshall the config file elements into a golang
   struct, raising errors if such marshalling cannot be performed.
4. The system will validate the config file values are appropriately
   typed (no string in int fields, etc.)
5. The system will validate the config file as a whole is valid, meaning
   that the values within the config file are consistent and can launch
   a spire server with all of its plugins.
6. The system will validate the config file is usable, meaning that the
   SPIRE server can trasition from the currently running configuration
   to the proposed new configuration.
7. Upon finding the config file usable, the system will either roll on
   to the new config file as the current config file automatically or
   in response to an additional command.
8. Once the active config file changes, the modules and plugins will
   be notified of the config file change and will alter their behavior
   to match the new configuration.

While there is some support in the past for some levels of validation,
the link between the SERVER and the plugins often blocks a full
validation of the new configuration, especially when using custom
plugins.  Additonally some aspects of expermiental plugin configuration
is currently under-policed for validation, and some sections of the
configuration exhibit near-polymoriphic behavior, by treading an entire
sub-section of the configuration as a tree of nodes, to be validated
outside of the actual configuration model's loading of the config data.

Finally, the usability requirement is new.  SPIRE has sometimes mixed
the validation concerns with the usability concerns in the past; but,
SPIRE has never formally attempted to detail if a running server can
transition from one setting to another.  To clarify between the two
categories, in the event a setting would not permit a SPIRE server to
launch, it would be deemed invalid.  The new category of usable is
narrowly defined as having a value which is otherwise valid, but cannot
be transitioned to from the currently functioning configuration.

#### Plan and Recommendations

For this to move forward, a plan to improve the validation of the SPIRE
configuration.  This plan consists of three main goals:

1. Each element of the configuration will be defined fully within
   SPIRE's configuration loading system (currently we default a large
   amount of configuration loading to external modules).
2. The validation of the configuration be requested independent of the
   running configuration, using a minimum of two Config structs, one
   holding the running configuration and one holding the proposed new
   configuration.
3. For maintainability, the modules that use the configuration validate
   each element they use independently within the module.  This will
   hopefully reduce the need for "updates at a distance" where logic
   that uses the configuration is subject to validation that occurs
   outside the module, say at startup-time.
4. For readability, the catalog module is renamed to the pluginloader.
5. For consistent application of policy, the sections of the
   configuration that impact plugins are handled by the pluginloader
   which defers validation to logic within the plugin.  This limits
   the extension of the pattern to only the need to marshal the
   configuration across the plugin sockets.
6. A means of determining validation failures and usability failures
   distinctly is employed.  Ideally thorugh the use of distinct
   validation tests that when all passing, would be followed by 
   usability tests.

While other approaches are possible, the creation of this policy would
permit easier management of value validation values as well as report
if the new values could be used.

#### Default implementations

As all plugins and modules would require updates, it is important to
guide the updates to have minimal impacts and speed the use of the
development changes.  The default behaviors of all users of the
configuration should be:

-  Validate whatever elements are already validated.  Add in comments
   and Issues for elements that undergo insufficient validation under
   the current code base.

-  Only report as usable new configurations that match the current
   configuration elements.  This is a tautology, and will always
   report as true, as a new instance of the same configuration is
   already running and switching to it is a NOOP.
