package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry manages all available commands
type Registry struct {
	commands map[string]Command
	aliases  map[string]string
	mutex    sync.RWMutex
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
	}
}

// Register registers a command
func (r *Registry) Register(cmd Command) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := strings.ToLower(cmd.Name())

	// Check if command already exists
	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}

	// Register command
	r.commands[name] = cmd

	// Register aliases
	for _, alias := range cmd.Aliases() {
		alias = strings.ToLower(alias)
		if _, exists := r.aliases[alias]; exists {
			return fmt.Errorf("alias %s already registered", alias)
		}
		r.aliases[alias] = name
	}

	return nil
}

// Unregister removes a command
func (r *Registry) Unregister(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name = strings.ToLower(name)

	// Get command to remove its aliases
	if cmd, exists := r.commands[name]; exists {
		// Remove aliases
		for _, alias := range cmd.Aliases() {
			delete(r.aliases, strings.ToLower(alias))
		}
		// Remove command
		delete(r.commands, name)
	}
}

// Get retrieves a command by name or alias
func (r *Registry) Get(name string) (Command, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	name = strings.ToLower(name)

	// Try direct lookup
	if cmd, exists := r.commands[name]; exists {
		return cmd, true
	}

	// Try alias lookup
	if realName, exists := r.aliases[name]; exists {
		if cmd, exists := r.commands[realName]; exists {
			return cmd, true
		}
	}

	return nil, false
}

// List returns all registered commands
func (r *Registry) List() []Command {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	commands := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		commands = append(commands, cmd)
	}

	// Sort by name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name() < commands[j].Name()
	})

	return commands
}

// ListNames returns all command names
func (r *Registry) ListNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetHelp returns help text for a command
func (r *Registry) GetHelp(name string) string {
	cmd, exists := r.Get(name)
	if !exists {
		return fmt.Sprintf("Command '%s' not found", name)
	}

	var help strings.Builder
	help.WriteString(fmt.Sprintf("Command: /%s\n", cmd.Name()))

	if len(cmd.Aliases()) > 0 {
		help.WriteString(fmt.Sprintf("Aliases: /%s\n", strings.Join(cmd.Aliases(), ", /")))
	}

	help.WriteString(fmt.Sprintf("Description: %s\n", cmd.Description()))
	help.WriteString(fmt.Sprintf("Usage: %s\n", cmd.Usage()))

	return help.String()
}

// GetAllHelp returns help text for all commands
func (r *Registry) GetAllHelp() string {
	commands := r.List()

	var help strings.Builder
	help.WriteString("Available Commands:\n\n")

	for _, cmd := range commands {
		help.WriteString(fmt.Sprintf("/%s", cmd.Name()))
		if len(cmd.Aliases()) > 0 {
			help.WriteString(fmt.Sprintf(" (/%s)", strings.Join(cmd.Aliases(), ", /")))
		}
		help.WriteString(fmt.Sprintf("\n  %s\n\n", cmd.Description()))
	}

	return help.String()
}

// Count returns the number of registered commands
func (r *Registry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.commands)
}
