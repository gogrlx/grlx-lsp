// Package schema defines the grlx recipe schema: ingredients, methods,
// properties, requisite types, and top-level recipe keys.
package schema

// Ingredient describes a grlx ingredient and its available methods.
type Ingredient struct {
	Name        string
	Description string
	Methods     []Method
}

// Method describes a single method on an ingredient.
type Method struct {
	Name        string
	Description string
	Properties  []Property
}

// Property describes a configurable property for a method.
type Property struct {
	Key         string
	Type        string // "string", "bool", "[]string"
	Required    bool
	Description string
}

// RequisiteType describes a valid requisite condition.
type RequisiteType struct {
	Name        string
	Description string
}

// Registry holds the complete grlx schema for lookup.
type Registry struct {
	Ingredients    []Ingredient
	RequisiteTypes []RequisiteType
}

// TopLevelKeys are the valid top-level keys in a .grlx recipe file.
var TopLevelKeys = []string{"include", "steps"}

// AllRequisiteTypes returns the known requisite conditions.
var AllRequisiteTypes = []RequisiteType{
	{Name: "require", Description: "Run this step only after the required step succeeds"},
	{Name: "require_any", Description: "Run this step if any of the listed steps succeed"},
	{Name: "onchanges", Description: "Run this step only if all listed steps made changes"},
	{Name: "onchanges_any", Description: "Run this step if any of the listed steps made changes"},
	{Name: "onfail", Description: "Run this step only if all listed steps failed"},
	{Name: "onfail_any", Description: "Run this step if any of the listed steps failed"},
}

// DefaultRegistry returns the built-in grlx ingredient registry,
// mirroring the ingredients defined in gogrlx/grlx.
func DefaultRegistry() *Registry {
	return &Registry{
		Ingredients:    allIngredients(),
		RequisiteTypes: AllRequisiteTypes,
	}
}

// FindIngredient returns the ingredient with the given name, or nil.
func (r *Registry) FindIngredient(name string) *Ingredient {
	for i := range r.Ingredients {
		if r.Ingredients[i].Name == name {
			return &r.Ingredients[i]
		}
	}
	return nil
}

// FindMethod returns the method on the given ingredient, or nil.
func (r *Registry) FindMethod(ingredientName, methodName string) *Method {
	ing := r.FindIngredient(ingredientName)
	if ing == nil {
		return nil
	}
	for i := range ing.Methods {
		if ing.Methods[i].Name == methodName {
			return &ing.Methods[i]
		}
	}
	return nil
}

// AllDottedNames returns all "ingredient.method" strings.
func (r *Registry) AllDottedNames() []string {
	var names []string
	for _, ing := range r.Ingredients {
		for _, m := range ing.Methods {
			names = append(names, ing.Name+"."+m.Name)
		}
	}
	return names
}

func allIngredients() []Ingredient {
	return []Ingredient{
		cmdIngredient(),
		fileIngredient(),
		groupIngredient(),
		pkgIngredient(),
		serviceIngredient(),
		userIngredient(),
	}
}

func cmdIngredient() Ingredient {
	return Ingredient{
		Name:        "cmd",
		Description: "Execute shell commands",
		Methods: []Method{
			{
				Name:        "run",
				Description: "Run a shell command",
				Properties: []Property{
					{Key: "name", Type: "string", Required: true, Description: "The command to run"},
					{Key: "runas", Type: "string", Required: false, Description: "User to run the command as"},
					{Key: "cwd", Type: "string", Required: false, Description: "Working directory"},
					{Key: "env", Type: "[]string", Required: false, Description: "Environment variables"},
					{Key: "shell", Type: "string", Required: false, Description: "Shell to use"},
					{Key: "creates", Type: "string", Required: false, Description: "Only run if this file does not exist"},
					{Key: "unless", Type: "string", Required: false, Description: "Only run if this command fails"},
					{Key: "onlyif", Type: "string", Required: false, Description: "Only run if this command succeeds"},
				},
			},
		},
	}
}

func fileIngredient() Ingredient {
	return Ingredient{
		Name:        "file",
		Description: "Manage files and directories",
		Methods: []Method{
			{Name: "absent", Description: "Ensure a file is absent", Properties: []Property{
				{Key: "name", Type: "string", Required: true, Description: "The name/path of the file to delete"},
			}},
			{Name: "append", Description: "Append text to a file", Properties: []Property{
				{Key: "name", Type: "string", Required: true, Description: "The name/path of the file to append to"},
				{Key: "makedirs", Type: "bool", Required: false, Description: "Create parent directories if they do not exist"},
				{Key: "source", Type: "string", Required: false, Description: "Append lines from a file sourced from this path/URL"},
				{Key: "source_hash", Type: "string", Required: false, Description: "Hash to verify the file specified by source"},
				{Key: "source_hashes", Type: "[]string", Required: false, Description: "Corresponding hashes for sources"},
				{Key: "sources", Type: "[]string", Required: false, Description: "Source, but in list format"},
				{Key: "template", Type: "bool", Required: false, Description: "Render the file as a template before appending"},
				{Key: "text", Type: "[]string", Required: false, Description: "The text to append to the file"},
			}},
			{Name: "cached", Description: "Cache a remote file locally", Properties: []Property{
				{Key: "name", Type: "string", Required: true, Description: "Local path for the cached file"},
				{Key: "source", Type: "string", Required: true, Description: "URL or path to cache from"},
				{Key: "hash", Type: "string", Required: false, Description: "Expected hash of the file"},
				{Key: "skip_verify", Type: "bool", Required: false, Description: "Skip hash verification"},
			}},
			{Name: "contains", Description: "Ensure a file contains specific content", Properties: []Property{
				{Key: "name", Type: "string", Required: true, Description: "Path of the file"},
				{Key: "source", Type: "string", Required: true, Description: "Source file to check against"},
				{Key: "source_hash", Type: "string", Required: false},
				{Key: "source_hashes", Type: "[]string", Required: false},
				{Key: "sources", Type: "[]string", Required: false},
				{Key: "template", Type: "bool", Required: false},
				{Key: "text", Type: "[]string", Required: false, Description: "Text that must be present"},
			}},
			{Name: "content", Description: "Manage the entire content of a file", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "text", Type: "[]string", Required: false},
				{Key: "makedirs", Type: "bool", Required: false},
				{Key: "source", Type: "string", Required: false},
				{Key: "source_hash", Type: "string", Required: false},
				{Key: "template", Type: "bool", Required: false},
				{Key: "sources", Type: "[]string", Required: false},
				{Key: "source_hashes", Type: "[]string", Required: false},
			}},
			{Name: "directory", Description: "Ensure a directory exists with given permissions", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "user", Type: "string", Required: false},
				{Key: "group", Type: "string", Required: false},
				{Key: "recurse", Type: "bool", Required: false},
				{Key: "dir_mode", Type: "string", Required: false},
				{Key: "file_mode", Type: "string", Required: false},
				{Key: "makedirs", Type: "bool", Required: false},
			}},
			{Name: "exists", Description: "Ensure a file exists (touch if needed)", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "managed", Description: "Download and manage a file from a source", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "source", Type: "string", Required: true},
				{Key: "source_hash", Type: "string", Required: false},
				{Key: "user", Type: "string", Required: false},
				{Key: "group", Type: "string", Required: false},
				{Key: "mode", Type: "string", Required: false},
				{Key: "template", Type: "bool", Required: false},
				{Key: "makedirs", Type: "bool", Required: false},
				{Key: "dir_mode", Type: "string", Required: false},
				{Key: "sources", Type: "[]string", Required: false},
				{Key: "source_hashes", Type: "[]string", Required: false},
			}},
			{Name: "missing", Description: "Verify a file does not exist (no-op check)", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "prepend", Description: "Prepend text to a file", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "text", Type: "[]string", Required: false},
				{Key: "makedirs", Type: "bool", Required: false},
				{Key: "source", Type: "string", Required: false},
				{Key: "source_hash", Type: "string", Required: false},
				{Key: "template", Type: "bool", Required: false},
				{Key: "sources", Type: "[]string", Required: false},
				{Key: "source_hashes", Type: "[]string", Required: false},
			}},
			{Name: "symlink", Description: "Manage a symbolic link", Properties: []Property{
				{Key: "name", Type: "string", Required: true, Description: "Path of the symlink"},
				{Key: "target", Type: "string", Required: true, Description: "Target the symlink points to"},
				{Key: "makedirs", Type: "bool", Required: false},
				{Key: "user", Type: "string", Required: false},
				{Key: "group", Type: "string", Required: false},
				{Key: "mode", Type: "string", Required: false},
			}},
			{Name: "touch", Description: "Touch a file (update mtime, create if missing)", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
		},
	}
}

func groupIngredient() Ingredient {
	return Ingredient{
		Name:        "group",
		Description: "Manage system groups",
		Methods: []Method{
			{Name: "absent", Description: "Ensure a group is absent", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "exists", Description: "Check if a group exists", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "present", Description: "Ensure a group is present", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "gid", Type: "string", Required: false},
			}},
		},
	}
}

func pkgIngredient() Ingredient {
	return Ingredient{
		Name:        "pkg",
		Description: "Manage system packages",
		Methods: []Method{
			{Name: "cleaned", Description: "Clean package cache"},
			{Name: "group_installed", Description: "Install a package group", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "held", Description: "Hold a package at current version", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "installed", Description: "Ensure a package is installed", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "version", Type: "string", Required: false},
			}},
			{Name: "key_managed", Description: "Manage a package signing key", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "source", Type: "string", Required: false},
			}},
			{Name: "latest", Description: "Ensure a package is at the latest version", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "purged", Description: "Purge a package (remove with config)", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "removed", Description: "Remove a package", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "repo_managed", Description: "Manage a package repository", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "source", Type: "string", Required: false},
			}},
		},
	}
}

func serviceIngredient() Ingredient {
	return Ingredient{
		Name:        "service",
		Description: "Manage system services",
		Methods: []Method{
			{Name: "disabled", Description: "Ensure a service is disabled", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "enabled", Description: "Ensure a service is enabled", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "masked", Description: "Mask a service", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "restarted", Description: "Restart a service", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "running", Description: "Ensure a service is running", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "enable", Type: "bool", Required: false, Description: "Also enable the service"},
			}},
			{Name: "stopped", Description: "Ensure a service is stopped", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "unmasked", Description: "Unmask a service", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
		},
	}
}

func userIngredient() Ingredient {
	return Ingredient{
		Name:        "user",
		Description: "Manage system users",
		Methods: []Method{
			{Name: "absent", Description: "Ensure a user is absent", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "exists", Description: "Check if a user exists", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
			}},
			{Name: "present", Description: "Ensure a user is present", Properties: []Property{
				{Key: "name", Type: "string", Required: true},
				{Key: "uid", Type: "string", Required: false},
				{Key: "gid", Type: "string", Required: false},
				{Key: "home", Type: "string", Required: false},
				{Key: "shell", Type: "string", Required: false},
				{Key: "groups", Type: "[]string", Required: false},
			}},
		},
	}
}
