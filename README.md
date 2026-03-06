# grlx-lsp

Language Server Protocol (LSP) server for [grlx](https://github.com/gogrlx/grlx) recipe files (`.grlx`).

## Features

- **Completion** — ingredient.method names, property keys, requisite types, step ID references
- **Diagnostics** — unknown ingredients/methods, missing required properties, unknown properties, invalid requisite types
- **Hover** — documentation for ingredients, methods, properties, and requisite types

## Installation

```bash
go install github.com/gogrlx/grlx-lsp/cmd/grlx-lsp@latest
```

## Editor Setup

### Neovim (nvim-lspconfig)

```lua
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "*.grlx",
  callback = function()
    vim.bo.filetype = "grlx"
  end,
})

local lspconfig = require("lspconfig")
local configs = require("lspconfig.configs")

configs.grlx_lsp = {
  default_config = {
    cmd = { "grlx-lsp" },
    filetypes = { "grlx" },
    root_dir = lspconfig.util.find_git_ancestor,
    settings = {},
  },
}

lspconfig.grlx_lsp.setup({})
```

### VS Code

Create `.vscode/settings.json`:

```json
{
  "files.associations": {
    "*.grlx": "yaml"
  }
}
```

Then configure a generic LSP client extension to run `grlx-lsp` for the `grlx` file type.

## Supported Ingredients

| Ingredient | Methods |
|-----------|---------|
| `cmd` | run |
| `file` | absent, append, cached, contains, content, directory, exists, managed, missing, prepend, symlink, touch |
| `group` | absent, exists, present |
| `pkg` | cleaned, group_installed, held, installed, key_managed, latest, purged, removed, repo_managed |
| `service` | disabled, enabled, masked, restarted, running, stopped, unmasked |
| `user` | absent, exists, present |

## Recipe Format

grlx recipes are YAML files with Go template support:

```yaml
include:
  - apache
  - .dev

steps:
  install nginx:
    pkg.installed:
      - name: nginx
  start nginx:
    service.running:
      - name: nginx
      - requisites:
        - require: install nginx
```

## License

0BSD
