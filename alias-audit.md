# Alias & Shortcode Audit

Date: 2026-02-14  
Repo: `/Users/sadimir/code/experiments/notion-cli`

## What Changed In This Pass
- Fixed highest-risk overlap by removing root `--force` alias for `--yes`.
- Added generated aliases for commands that previously had none.
- Added safe shorthand aliases (`-x`) where no collisions were introduced.

## Current Summary
- Total commands: **130**
- Commands with aliases: **130**
- Commands without aliases: **0**
- Unique command alias tokens: **75**
- Total command-defined shorthand flags: **239**
- Visible shorthand flags: **19**
- Hidden shorthand aliases: **220**
- Sibling command-token collisions: **0**
- Flag-name shadowing vs ancestor persistent flags: **4**
- Per-command shorthand conflicts: **0**

## Highest-Risk Overlap Status
Resolved: root `--force` no longer exists, so it no longer collides semantically with `ntn page sync --force`.

## Remaining Name Shadowing
| Command Path | Local Flag | Ancestor Owner | Risk |
|---|---|---|---|
| `ntn bulk archive` | `--limit` | `ntn` | Low |
| `ntn bulk update` | `--limit` | `ntn` | Low |
| `ntn page export` | `--format` | `ntn` | Medium |
| `ntn page sync` | `--output` | `ntn` | Medium |

## Newly Added Aliases (Previously Missing)
| Command Path | Alias Added |
|---|---|
| `ntn` | `notion` |
| `ntn api` | `a` |
| `ntn api request` | `r` |
| `ntn api status` | `s` |
| `ntn auth` | `au` |
| `ntn auth add-token` | `at` |
| `ntn auth login` | `l` |
| `ntn auth logout` | `lo` |
| `ntn auth status` | `s` |
| `ntn block add-breadcrumb` | `ab` |
| `ntn block add-columns` | `ac` |
| `ntn block add-divider` | `ad` |
| `ntn block add-toc` | `at` |
| `ntn bulk` | `bu` |
| `ntn bulk archive` | `a` |
| `ntn completion` | `co` |
| `ntn completion bash` | `b` |
| `ntn completion fish` | `f` |
| `ntn completion powershell` | `p` |
| `ntn completion zsh` | `z` |
| `ntn config path` | `p` |
| `ntn config set` | `s` |
| `ntn fetch` | `fe` |
| `ntn login` | `l` |
| `ntn logout` | `lo` |
| `ntn mcp` | `m` |
| `ntn mcp db` | `d` |
| `ntn mcp login` | `l` |
| `ntn mcp logout` | `lo` |
| `ntn mcp status` | `st` |
| `ntn mcp tools` | `t` |
| `ntn skill init` | `i` |
| `ntn skill path` | `p` |
| `ntn skill sync` | `s` |
| `ntn user me` | `m` |
| `ntn webhook parse` | `p` |
| `ntn webhook verify` | `v` |
| `ntn whoami` | `w` |
| `ntn workspace add` | `a` |
| `ntn workspace use` | `u` |

## Full Command Alias Inventory
| Command Path | Aliases |
|---|---|
| `ntn` | `notion` |
| `ntn api` | `a` |
| `ntn api request` | `r` |
| `ntn api status` | `s` |
| `ntn auth` | `au` |
| `ntn auth add-token` | `at` |
| `ntn auth login` | `l` |
| `ntn auth logout` | `lo` |
| `ntn auth status` | `s` |
| `ntn block` | `b`, `blocks` |
| `ntn block add` | `a` |
| `ntn block add bullet` | `bl` |
| `ntn block add callout` | `co` |
| `ntn block add code` | `cd` |
| `ntn block add file` | `f` |
| `ntn block add heading` | `h` |
| `ntn block add image` | `i`, `img` |
| `ntn block add number` | `nl`, `num` |
| `ntn block add paragraph` | `p`, `para` |
| `ntn block add quote` | `qt` |
| `ntn block add todo` | `td` |
| `ntn block add toggle` | `tg` |
| `ntn block add-breadcrumb` | `ab` |
| `ntn block add-columns` | `ac` |
| `ntn block add-divider` | `ad` |
| `ntn block add-toc` | `at` |
| `ntn block append` | `ap` |
| `ntn block children` | `list`, `ls` |
| `ntn block delete` | `d`, `rm` |
| `ntn block get` | `g` |
| `ntn block update` | `u`, `up` |
| `ntn bulk` | `bu` |
| `ntn bulk archive` | `a` |
| `ntn bulk update` | `up` |
| `ntn comment` | `c`, `comments` |
| `ntn comment add` | `a`, `create`, `mk` |
| `ntn comment get` | `g` |
| `ntn comment list` | `ls` |
| `ntn completion` | `co` |
| `ntn completion bash` | `b` |
| `ntn completion fish` | `f` |
| `ntn completion powershell` | `p` |
| `ntn completion zsh` | `z` |
| `ntn config` | `cfg` |
| `ntn config path` | `p` |
| `ntn config set` | `s` |
| `ntn config show` | `g` |
| `ntn create` | `mk` |
| `ntn datasource` | `ds` |
| `ntn datasource create` | `c`, `mk` |
| `ntn datasource get` | `g` |
| `ntn datasource list` | `ls` |
| `ntn datasource query` | `q` |
| `ntn datasource templates` | `t` |
| `ntn datasource update` | `u`, `up` |
| `ntn db` | `database`, `databases` |
| `ntn db backup` | `bak` |
| `ntn db create` | `c`, `mk` |
| `ntn db get` | `g` |
| `ntn db list` | `ls` |
| `ntn db query` | `q` |
| `ntn db update` | `u`, `up` |
| `ntn delete` | `d`, `rm` |
| `ntn fetch` | `fe` |
| `ntn file` | `f`, `files` |
| `ntn file get` | `g` |
| `ntn file list` | `ls` |
| `ntn file upload` | `up` |
| `ntn get` | `g` |
| `ntn import` | `im` |
| `ntn import csv` | `c` |
| `ntn list` | `ls` |
| `ntn login` | `l` |
| `ntn logout` | `lo` |
| `ntn mcp` | `m` |
| `ntn mcp comment` | `cm` |
| `ntn mcp comment add` | `a` |
| `ntn mcp comment list` | `ls` |
| `ntn mcp create` | `c`, `mk` |
| `ntn mcp db` | `d` |
| `ntn mcp db create` | `c`, `mk` |
| `ntn mcp db update` | `u`, `up` |
| `ntn mcp duplicate` | `dup` |
| `ntn mcp edit` | `e`, `up` |
| `ntn mcp fetch` | `f` |
| `ntn mcp login` | `l` |
| `ntn mcp logout` | `lo` |
| `ntn mcp move` | `mv` |
| `ntn mcp query` | `q` |
| `ntn mcp search` | `s` |
| `ntn mcp status` | `st` |
| `ntn mcp teams` | `tm` |
| `ntn mcp tools` | `t` |
| `ntn mcp users` | `u` |
| `ntn open` | `o` |
| `ntn page` | `p`, `pages` |
| `ntn page create` | `c`, `mk` |
| `ntn page create-batch` | `cb` |
| `ntn page delete` | `d`, `rm` |
| `ntn page duplicate` | `dup` |
| `ntn page export` | `ex` |
| `ntn page get` | `g` |
| `ntn page list` | `ls` |
| `ntn page move` | `mv` |
| `ntn page properties` | `props` |
| `ntn page property` | `prop` |
| `ntn page sync` | `sy` |
| `ntn page update` | `u`, `up` |
| `ntn page update-batch` | `ub` |
| `ntn resolve` | `r`, `res` |
| `ntn search` | `find`, `q`, `s` |
| `ntn skill` | `sk` |
| `ntn skill edit` | `up` |
| `ntn skill init` | `i` |
| `ntn skill path` | `p` |
| `ntn skill sync` | `s` |
| `ntn user` | `u`, `users` |
| `ntn user get` | `g` |
| `ntn user list` | `ls` |
| `ntn user me` | `m` |
| `ntn webhook` | `wh` |
| `ntn webhook parse` | `p` |
| `ntn webhook verify` | `v` |
| `ntn whoami` | `w` |
| `ntn workspace` | `workspaces`, `ws` |
| `ntn workspace add` | `a` |
| `ntn workspace list` | `ls` |
| `ntn workspace remove` | `rm` |
| `ntn workspace show` | `g` |
| `ntn workspace use` | `u` |

## Visible Short Flag Shorthands
| Command Path | Flag | Shorthand |
|---|---|---|
| `ntn db backup` | `--output-dir` | `-d` |
| `ntn file upload` | `--page` | `-p` |
| `ntn mcp create` | `--file` | `-f` |
| `ntn mcp create` | `--parent` | `-p` |
| `ntn mcp create` | `--title` | `-t` |
| `ntn mcp db create` | `--parent` | `-p` |
| `ntn mcp db create` | `--title` | `-t` |
| `ntn mcp db update` | `--title` | `-t` |
| `ntn mcp move` | `--parent` | `-p` |
| `ntn mcp query` | `--params` | `-P` |
| `ntn mcp query` | `--view` | `-v` |
| `ntn mcp search` | `--ai` | `-a` |
| `ntn page sync` | `--force` | `-f` |
| `ntn page sync` | `--output` | `-o` |
| `ntn search` | `--text` | `-t` |
| `ntn` | `--output` | `-o` |
| `ntn` | `--query` | `-q` |
| `ntn` | `--workspace` | `-w` |
| `ntn` | `--yes` | `-y` |

## Hidden Shorthand Aliases (Generated + Existing)
| Command Path | Hidden Flag | Shorthand |
|---|---|---|
| `ntn api request` | `--b` | `-b` |
| `ntn api request` | `--e` | `-e` |
| `ntn api request` | `--i` | `-i` |
| `ntn api request` | `--n` | `-n` |
| `ntn api request` | `--p` | `-p` |
| `ntn api request` | `--r` | `-r` |
| `ntn api status` | `--r` | `-r` |
| `ntn block add callout` | `--e` | `-e` |
| `ntn block add code` | `--l` | `-l` |
| `ntn block add file` | `--c` | `-c` |
| `ntn block add file` | `--f` | `-f` |
| `ntn block add heading` | `--l` | `-l` |
| `ntn block add image` | `--c` | `-c` |
| `ntn block add image` | `--f` | `-f` |
| `ntn block add todo` | `--c` | `-c` |
| `ntn block add-columns` | `--c` | `-c` |
| `ntn block add-toc` | `--c` | `-c` |
| `ntn block append` | `--a` | `-a` |
| `ntn block append` | `--c` | `-c` |
| `ntn block append` | `--f` | `-f` |
| `ntn block append` | `--m` | `-m` |
| `ntn block append` | `--n` | `-n` |
| `ntn block append` | `--t` | `-t` |
| `ntn block children` | `--a` | `-a` |
| `ntn block children` | `--d` | `-d` |
| `ntn block children` | `--l` | `-l` |
| `ntn block children` | `--p` | `-p` |
| `ntn block children` | `--s` | `-s` |
| `ntn block delete` | `--d` | `-d` |
| `ntn block update` | `--a` | `-a` |
| `ntn block update` | `--c` | `-c` |
| `ntn block update` | `--d` | `-d` |
| `ntn bulk archive` | `--d` | `-d` |
| `ntn bulk archive` | `--e` | `-e` |
| `ntn bulk archive` | `--l` | `-l` |
| `ntn bulk update` | `--d` | `-d` |
| `ntn bulk update` | `--e` | `-e` |
| `ntn bulk update` | `--l` | `-l` |
| `ntn bulk update` | `--s` | `-s` |
| `ntn comment add` | `--a` | `-a` |
| `ntn comment add` | `--d` | `-d` |
| `ntn comment add` | `--e` | `-e` |
| `ntn comment add` | `--p` | `-p` |
| `ntn comment add` | `--v` | `-v` |
| `ntn comment add` | `--x` | `-x` |
| `ntn comment list` | `--a` | `-a` |
| `ntn comment list` | `--p` | `-p` |
| `ntn comment list` | `--s` | `-s` |
| `ntn datasource create` | `--f` | `-f` |
| `ntn datasource create` | `--p` | `-p` |
| `ntn datasource create` | `--r` | `-r` |
| `ntn datasource list` | `--a` | `-a` |
| `ntn datasource list` | `--p` | `-p` |
| `ntn datasource list` | `--s` | `-s` |
| `ntn datasource query` | `--a` | `-a` |
| `ntn datasource query` | `--c` | `-c` |
| `ntn datasource query` | `--e` | `-e` |
| `ntn datasource query` | `--f` | `-f` |
| `ntn datasource query` | `--g` | `-g` |
| `ntn datasource query` | `--i` | `-i` |
| `ntn datasource query` | `--l` | `-l` |
| `ntn datasource query` | `--m` | `-m` |
| `ntn datasource query` | `--n` | `-n` |
| `ntn datasource query` | `--p` | `-p` |
| `ntn datasource query` | `--r` | `-r` |
| `ntn datasource query` | `--s` | `-s` |
| `ntn datasource query` | `--t` | `-t` |
| `ntn datasource query` | `--u` | `-u` |
| `ntn datasource update` | `--f` | `-f` |
| `ntn datasource update` | `--p` | `-p` |
| `ntn db backup` | `--c` | `-c` |
| `ntn db backup` | `--e` | `-e` |
| `ntn db backup` | `--i` | `-i` |
| `ntn db create` | `--c` | `-c` |
| `ntn db create` | `--d` | `-d` |
| `ntn db create` | `--e` | `-e` |
| `ntn db create` | `--f` | `-f` |
| `ntn db create` | `--i` | `-i` |
| `ntn db create` | `--n` | `-n` |
| `ntn db create` | `--p` | `-p` |
| `ntn db create` | `--r` | `-r` |
| `ntn db create` | `--t` | `-t` |
| `ntn db list` | `--a` | `-a` |
| `ntn db list` | `--p` | `-p` |
| `ntn db list` | `--s` | `-s` |
| `ntn db list` | `--t` | `-t` |
| `ntn db query` | `--a` | `-a` |
| `ntn db query` | `--c` | `-c` |
| `ntn db query` | `--d` | `-d` |
| `ntn db query` | `--e` | `-e` |
| `ntn db query` | `--f` | `-f` |
| `ntn db query` | `--g` | `-g` |
| `ntn db query` | `--i` | `-i` |
| `ntn db query` | `--l` | `-l` |
| `ntn db query` | `--m` | `-m` |
| `ntn db query` | `--n` | `-n` |
| `ntn db query` | `--p` | `-p` |
| `ntn db query` | `--r` | `-r` |
| `ntn db query` | `--s` | `-s` |
| `ntn db query` | `--t` | `-t` |
| `ntn db query` | `--u` | `-u` |
| `ntn db update` | `--a` | `-a` |
| `ntn db update` | `--c` | `-c` |
| `ntn db update` | `--d` | `-d` |
| `ntn db update` | `--e` | `-e` |
| `ntn db update` | `--f` | `-f` |
| `ntn db update` | `--i` | `-i` |
| `ntn db update` | `--p` | `-p` |
| `ntn db update` | `--r` | `-r` |
| `ntn db update` | `--t` | `-t` |
| `ntn fetch` | `--t` | `-t` |
| `ntn file list` | `--p` | `-p` |
| `ntn file list` | `--s` | `-s` |
| `ntn file upload` | `--r` | `-r` |
| `ntn import csv` | `--b` | `-b` |
| `ntn import csv` | `--c` | `-c` |
| `ntn import csv` | `--d` | `-d` |
| `ntn import csv` | `--f` | `-f` |
| `ntn import csv` | `--s` | `-s` |
| `ntn import` | `--b` | `-b` |
| `ntn import` | `--d` | `-d` |
| `ntn import` | `--f` | `-f` |
| `ntn mcp create` | `--c` | `-c` |
| `ntn mcp create` | `--d` | `-d` |
| `ntn mcp create` | `--r` | `-r` |
| `ntn mcp db create` | `--r` | `-r` |
| `ntn mcp db update` | `--i` | `-i` |
| `ntn mcp db update` | `--p` | `-p` |
| `ntn mcp db update` | `--r` | `-r` |
| `ntn mcp edit` | `--e` | `-e` |
| `ntn mcp edit` | `--i` | `-i` |
| `ntn mcp edit` | `--n` | `-n` |
| `ntn mcp edit` | `--p` | `-p` |
| `ntn mcp edit` | `--r` | `-r` |
| `ntn mcp users` | `--c` | `-c` |
| `ntn mcp users` | `--p` | `-p` |
| `ntn mcp users` | `--u` | `-u` |
| `ntn page create-batch` | `--a` | `-a` |
| `ntn page create-batch` | `--c` | `-c` |
| `ntn page create-batch` | `--d` | `-d` |
| `ntn page create-batch` | `--f` | `-f` |
| `ntn page create-batch` | `--p` | `-p` |
| `ntn page create-batch` | `--t` | `-t` |
| `ntn page create` | `--a` | `-a` |
| `ntn page create` | `--d` | `-d` |
| `ntn page create` | `--e` | `-e` |
| `ntn page create` | `--f` | `-f` |
| `ntn page create` | `--i` | `-i` |
| `ntn page create` | `--p` | `-p` |
| `ntn page create` | `--r` | `-r` |
| `ntn page create` | `--s` | `-s` |
| `ntn page create` | `--t` | `-t` |
| `ntn page duplicate` | `--d` | `-d` |
| `ntn page duplicate` | `--i` | `-i` |
| `ntn page duplicate` | `--n` | `-n` |
| `ntn page duplicate` | `--p` | `-p` |
| `ntn page duplicate` | `--t` | `-t` |
| `ntn page export` | `--f` | `-f` |
| `ntn page get` | `--e` | `-e` |
| `ntn page get` | `--n` | `-n` |
| `ntn page move` | `--a` | `-a` |
| `ntn page move` | `--p` | `-p` |
| `ntn page move` | `--t` | `-t` |
| `ntn page properties` | `--i` | `-i` |
| `ntn page properties` | `--s` | `-s` |
| `ntn page properties` | `--t` | `-t` |
| `ntn page properties` | `--v` | `-v` |
| `ntn page sync` | `--d` | `-d` |
| `ntn page sync` | `--p` | `-p` |
| `ntn page sync` | `--s` | `-s` |
| `ntn page sync` | `--t` | `-t` |
| `ntn page sync` | `--u` | `-u` |
| `ntn page update-batch` | `--c` | `-c` |
| `ntn page update-batch` | `--f` | `-f` |
| `ntn page update-batch` | `--p` | `-p` |
| `ntn page update` | `--a` | `-a` |
| `ntn page update` | `--d` | `-d` |
| `ntn page update` | `--f` | `-f` |
| `ntn page update` | `--i` | `-i` |
| `ntn page update` | `--m` | `-m` |
| `ntn page update` | `--p` | `-p` |
| `ntn page update` | `--r` | `-r` |
| `ntn page update` | `--s` | `-s` |
| `ntn page update` | `--t` | `-t` |
| `ntn page update` | `--u` | `-u` |
| `ntn page update` | `--v` | `-v` |
| `ntn resolve` | `--a` | `-a` |
| `ntn resolve` | `--e` | `-e` |
| `ntn resolve` | `--p` | `-p` |
| `ntn resolve` | `--s` | `-s` |
| `ntn resolve` | `--t` | `-t` |
| `ntn search` | `--a` | `-a` |
| `ntn search` | `--c` | `-c` |
| `ntn search` | `--f` | `-f` |
| `ntn search` | `--p` | `-p` |
| `ntn search` | `--s` | `-s` |
| `ntn skill sync` | `--a` | `-a` |
| `ntn user list` | `--a` | `-a` |
| `ntn user list` | `--p` | `-p` |
| `ntn user list` | `--s` | `-s` |
| `ntn webhook parse` | `--p` | `-p` |
| `ntn webhook verify` | `--i` | `-i` |
| `ntn webhook verify` | `--p` | `-p` |
| `ntn webhook verify` | `--s` | `-s` |
| `ntn workspace add` | `--a` | `-a` |
| `ntn workspace add` | `--d` | `-d` |
| `ntn workspace add` | `--t` | `-t` |
| `ntn` | `--a` | `-a` |
| `ntn` | `--b` | `-b` |
| `ntn` | `--d` | `-d` |
| `ntn` | `--e` | `-e` |
| `ntn` | `--f` | `-f` |
| `ntn` | `--i` | `-i` |
| `ntn` | `--json` | `-j` |
| `ntn` | `--l` | `-l` |
| `ntn` | `--m` | `-m` |
| `ntn` | `--r` | `-r` |
| `ntn` | `--s` | `-s` |
| `ntn` | `--t` | `-t` |
| `ntn` | `--u` | `-u` |

## Full Flag Alias Groups (Shared Backing Value)
| Command Path | Linked Flags | Hidden Count |
|---|---|---:|
| `ntn` | `a (-a)`, `fail-empty`, `fe` | 2 |
| `ntn` | `b (-b)`, `sb`, `sort-by` | 2 |
| `ntn` | `d (-d)`, `debug` | 1 |
| `ntn` | `desc`, `e (-e)` | 1 |
| `ntn` | `error-format`, `f (-f)` | 1 |
| `ntn` | `fds`, `fields`, `i (-i)` | 2 |
| `ntn` | `io`, `items-only`, `results-only`, `ro`, `t (-t)` | 4 |
| `ntn` | `j`, `json (-j)` | 2 |
| `ntn` | `jsonpath`, `s (-s)` | 1 |
| `ntn` | `l (-l)`, `latest` | 1 |
| `ntn` | `limit`, `m (-m)` | 1 |
| `ntn` | `no-input`, `yes (-y)` | 1 |
| `ntn` | `out`, `output (-o)` | 1 |
| `ntn` | `qf`, `query-file`, `u (-u)` | 2 |
| `ntn` | `qr`, `query (-q)` | 1 |
| `ntn` | `r (-r)`, `recent` | 1 |
| `ntn api request` | `b (-b)`, `body` | 1 |
| `ntn api request` | `e (-e)`, `header` | 1 |
| `ntn api request` | `i (-i)`, `include-headers` | 1 |
| `ntn api request` | `n (-n)`, `no-auth` | 1 |
| `ntn api request` | `p (-p)`, `paginate` | 1 |
| `ntn api request` | `r (-r)`, `raw` | 1 |
| `ntn api status` | `r (-r)`, `refresh` | 1 |
| `ntn block add callout` | `e (-e)`, `emoji` | 1 |
| `ntn block add code` | `l (-l)`, `language` | 1 |
| `ntn block add file` | `c (-c)`, `caption` | 1 |
| `ntn block add file` | `f (-f)`, `file` | 1 |
| `ntn block add heading` | `l (-l)`, `level` | 1 |
| `ntn block add image` | `c (-c)`, `caption` | 1 |
| `ntn block add image` | `f (-f)`, `file` | 1 |
| `ntn block add todo` | `c (-c)`, `checked` | 1 |
| `ntn block add-columns` | `c (-c)`, `columns` | 1 |
| `ntn block add-toc` | `c (-c)`, `color` | 1 |
| `ntn block append` | `a (-a)`, `after` | 1 |
| `ntn block append` | `c (-c)`, `ch`, `children` | 2 |
| `ntn block append` | `chf`, `children-file`, `f (-f)` | 2 |
| `ntn block append` | `content`, `n (-n)` | 1 |
| `ntn block append` | `m (-m)`, `md` | 1 |
| `ntn block append` | `t (-t)`, `type` | 1 |
| `ntn block children` | `a (-a)`, `all` | 1 |
| `ntn block children` | `d (-d)`, `depth` | 1 |
| `ntn block children` | `l (-l)`, `plain` | 1 |
| `ntn block children` | `p (-p)`, `page-size` | 1 |
| `ntn block children` | `s (-s)`, `start-cursor` | 1 |
| `ntn block delete` | `d (-d)`, `dry-run` | 1 |
| `ntn block update` | `a (-a)`, `archived` | 1 |
| `ntn block update` | `c (-c)`, `content` | 1 |
| `ntn block update` | `d (-d)`, `dr`, `dry-run` | 2 |
| `ntn bulk archive` | `d (-d)`, `dr`, `dry-run` | 2 |
| `ntn bulk archive` | `e (-e)`, `where` | 1 |
| `ntn bulk archive` | `l (-l)`, `limit` | 1 |
| `ntn bulk update` | `d (-d)`, `dr`, `dry-run` | 2 |
| `ntn bulk update` | `e (-e)`, `where` | 1 |
| `ntn bulk update` | `l (-l)`, `limit` | 1 |
| `ntn bulk update` | `s (-s)`, `set` | 1 |
| `ntn comment add` | `a (-a)`, `pa`, `page`, `parent` | 3 |
| `ntn comment add` | `d (-d)`, `discussion-id` | 1 |
| `ntn comment add` | `e (-e)`, `m`, `mention` | 2 |
| `ntn comment add` | `p (-p)`, `page-mention`, `pm` | 2 |
| `ntn comment add` | `t`, `text`, `x (-x)` | 2 |
| `ntn comment add` | `v (-v)`, `verbose` | 1 |
| `ntn comment list` | `a (-a)`, `all` | 1 |
| `ntn comment list` | `p (-p)`, `page-size` | 1 |
| `ntn comment list` | `s (-s)`, `start-cursor` | 1 |
| `ntn datasource create` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn datasource create` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn datasource create` | `properties`, `props`, `r (-r)` | 2 |
| `ntn datasource list` | `a (-a)`, `all` | 1 |
| `ntn datasource list` | `p (-p)`, `page-size` | 1 |
| `ntn datasource list` | `s (-s)`, `start-cursor` | 1 |
| `ntn datasource query` | `a (-a)`, `all` | 1 |
| `ntn datasource query` | `assigned-to`, `assignee`, `s (-s)` | 2 |
| `ntn datasource query` | `assignee-prop`, `p (-p)` | 1 |
| `ntn datasource query` | `c (-c)`, `start-cursor` | 1 |
| `ntn datasource query` | `e (-e)`, `select-equals` | 1 |
| `ntn datasource query` | `f (-f)`, `fi`, `filter` | 2 |
| `ntn datasource query` | `ff`, `filter-file`, `i (-i)` | 2 |
| `ntn datasource query` | `g (-g)`, `page-size` | 1 |
| `ntn datasource query` | `l (-l)`, `select-property` | 1 |
| `ntn datasource query` | `m (-m)`, `select-match` | 1 |
| `ntn datasource query` | `n (-n)`, `select-not` | 1 |
| `ntn datasource query` | `priority`, `r (-r)` | 1 |
| `ntn datasource query` | `priority-prop`, `t (-t)` | 1 |
| `ntn datasource query` | `sp`, `status-prop` | 1 |
| `ntn datasource query` | `status`, `u (-u)` | 1 |
| `ntn datasource update` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn datasource update` | `p (-p)`, `properties`, `props` | 2 |
| `ntn db backup` | `c (-c)`, `content` | 1 |
| `ntn db backup` | `e (-e)`, `export-format` | 1 |
| `ntn db backup` | `i (-i)`, `incremental` | 1 |
| `ntn db create` | `c (-c)`, `cover` | 1 |
| `ntn db create` | `d (-d)`, `datasource-title` | 1 |
| `ntn db create` | `description`, `e (-e)` | 1 |
| `ntn db create` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn db create` | `i (-i)`, `icon` | 1 |
| `ntn db create` | `inline`, `n (-n)` | 1 |
| `ntn db create` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn db create` | `properties`, `props`, `r (-r)` | 2 |
| `ntn db create` | `t (-t)`, `title` | 1 |
| `ntn db list` | `a (-a)`, `all` | 1 |
| `ntn db list` | `p (-p)`, `page-size` | 1 |
| `ntn db list` | `s (-s)`, `start-cursor` | 1 |
| `ntn db list` | `t (-t)`, `title-match` | 1 |
| `ntn db query` | `a (-a)`, `all` | 1 |
| `ntn db query` | `assigned-to`, `assignee`, `s (-s)` | 2 |
| `ntn db query` | `assignee-prop`, `p (-p)` | 1 |
| `ntn db query` | `c (-c)`, `start-cursor` | 1 |
| `ntn db query` | `d (-d)`, `datasource`, `ds` | 2 |
| `ntn db query` | `e (-e)`, `select-equals` | 1 |
| `ntn db query` | `f (-f)`, `fi`, `filter` | 2 |
| `ntn db query` | `ff`, `filter-file`, `i (-i)` | 2 |
| `ntn db query` | `g (-g)`, `page-size` | 1 |
| `ntn db query` | `l (-l)`, `select-property` | 1 |
| `ntn db query` | `m (-m)`, `select-match` | 1 |
| `ntn db query` | `n (-n)`, `select-not` | 1 |
| `ntn db query` | `priority`, `r (-r)` | 1 |
| `ntn db query` | `priority-prop`, `t (-t)` | 1 |
| `ntn db query` | `sp`, `status-prop` | 1 |
| `ntn db query` | `status`, `u (-u)` | 1 |
| `ntn db update` | `a (-a)`, `archived` | 1 |
| `ntn db update` | `c (-c)`, `cover` | 1 |
| `ntn db update` | `d (-d)`, `datasource`, `ds` | 2 |
| `ntn db update` | `description`, `e (-e)` | 1 |
| `ntn db update` | `dr`, `dry-run`, `r (-r)` | 2 |
| `ntn db update` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn db update` | `i (-i)`, `icon` | 1 |
| `ntn db update` | `p (-p)`, `properties`, `props` | 2 |
| `ntn db update` | `t (-t)`, `title` | 1 |
| `ntn fetch` | `t (-t)`, `type` | 1 |
| `ntn file list` | `p (-p)`, `page-size` | 1 |
| `ntn file list` | `s (-s)`, `start-cursor` | 1 |
| `ntn file upload` | `page (-p)`, `pg` | 1 |
| `ntn file upload` | `prop`, `property`, `r (-r)` | 2 |
| `ntn import` | `b (-b)`, `batch-size` | 1 |
| `ntn import` | `d (-d)`, `dry-run` | 1 |
| `ntn import` | `f (-f)`, `file` | 1 |
| `ntn import csv` | `b (-b)`, `batch-size` | 1 |
| `ntn import csv` | `c (-c)`, `column-map` | 1 |
| `ntn import csv` | `d (-d)`, `dry-run` | 1 |
| `ntn import csv` | `f (-f)`, `file` | 1 |
| `ntn import csv` | `s (-s)`, `skip-rows` | 1 |
| `ntn mcp create` | `c (-c)`, `content` | 1 |
| `ntn mcp create` | `d (-d)`, `data-source` | 1 |
| `ntn mcp create` | `properties`, `r (-r)` | 1 |
| `ntn mcp db create` | `properties`, `r (-r)` | 1 |
| `ntn mcp db update` | `i (-i)`, `id` | 1 |
| `ntn mcp db update` | `p (-p)`, `properties` | 1 |
| `ntn mcp db update` | `r (-r)`, `trash` | 1 |
| `ntn mcp edit` | `e (-e)`, `replace-range` | 1 |
| `ntn mcp edit` | `i (-i)`, `insert-after` | 1 |
| `ntn mcp edit` | `n (-n)`, `new` | 1 |
| `ntn mcp edit` | `p (-p)`, `properties` | 1 |
| `ntn mcp edit` | `r (-r)`, `replace` | 1 |
| `ntn mcp users` | `c (-c)`, `cursor` | 1 |
| `ntn mcp users` | `p (-p)`, `page-size` | 1 |
| `ntn mcp users` | `u (-u)`, `user-id` | 1 |
| `ntn page create` | `a (-a)`, `assignee` | 1 |
| `ntn page create` | `d (-d)`, `datasource`, `ds` | 2 |
| `ntn page create` | `e (-e)`, `properties`, `props` | 2 |
| `ntn page create` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn page create` | `i (-i)`, `title` | 1 |
| `ntn page create` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn page create` | `parent-type`, `t (-t)` | 1 |
| `ntn page create` | `priority`, `r (-r)` | 1 |
| `ntn page create` | `s (-s)`, `status` | 1 |
| `ntn page create-batch` | `a (-a)`, `pa`, `parent` | 2 |
| `ntn page create-batch` | `c (-c)`, `continue-on-error` | 1 |
| `ntn page create-batch` | `d (-d)`, `datasource`, `ds` | 2 |
| `ntn page create-batch` | `f (-f)`, `file` | 1 |
| `ntn page create-batch` | `p (-p)`, `pages` | 1 |
| `ntn page create-batch` | `parent-type`, `t (-t)` | 1 |
| `ntn page duplicate` | `d (-d)`, `datasource`, `ds` | 2 |
| `ntn page duplicate` | `i (-i)`, `title` | 1 |
| `ntn page duplicate` | `n (-n)`, `no-children` | 1 |
| `ntn page duplicate` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn page duplicate` | `parent-type`, `t (-t)` | 1 |
| `ntn page export` | `f (-f)`, `format` | 1 |
| `ntn page get` | `e (-e)`, `editable` | 1 |
| `ntn page get` | `enrich`, `n (-n)` | 1 |
| `ntn page move` | `a (-a)`, `after` | 1 |
| `ntn page move` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn page move` | `parent-type`, `t (-t)` | 1 |
| `ntn page properties` | `i (-i)`, `simple` | 1 |
| `ntn page properties` | `only-set`, `s (-s)` | 1 |
| `ntn page properties` | `t (-t)`, `types` | 1 |
| `ntn page properties` | `v (-v)`, `with-values` | 1 |
| `ntn page sync` | `d (-d)`, `dr`, `dry-run` | 2 |
| `ntn page sync` | `p (-p)`, `pa`, `parent` | 2 |
| `ntn page sync` | `parent-type`, `t (-t)` | 1 |
| `ntn page sync` | `pull`, `u (-u)` | 1 |
| `ntn page sync` | `push`, `s (-s)` | 1 |
| `ntn page update` | `a (-a)`, `archived` | 1 |
| `ntn page update` | `assignee`, `s (-s)` | 1 |
| `ntn page update` | `d (-d)`, `dr`, `dry-run` | 2 |
| `ntn page update` | `f (-f)`, `properties-file`, `props-file` | 2 |
| `ntn page update` | `i (-i)`, `title` | 1 |
| `ntn page update` | `m (-m)`, `mention` | 1 |
| `ntn page update` | `p (-p)`, `priority` | 1 |
| `ntn page update` | `properties`, `props`, `r (-r)` | 2 |
| `ntn page update` | `rich-text`, `t (-t)` | 1 |
| `ntn page update` | `status`, `u (-u)` | 1 |
| `ntn page update` | `v (-v)`, `verbose` | 1 |
| `ntn page update-batch` | `c (-c)`, `continue-on-error` | 1 |
| `ntn page update-batch` | `f (-f)`, `file` | 1 |
| `ntn page update-batch` | `p (-p)`, `pages` | 1 |
| `ntn resolve` | `a (-a)`, `all` | 1 |
| `ntn resolve` | `e (-e)`, `exact` | 1 |
| `ntn resolve` | `p (-p)`, `page-size` | 1 |
| `ntn resolve` | `s (-s)`, `start-cursor` | 1 |
| `ntn resolve` | `t (-t)`, `type` | 1 |
| `ntn search` | `a (-a)`, `all` | 1 |
| `ntn search` | `c (-c)`, `start-cursor` | 1 |
| `ntn search` | `f (-f)`, `fi`, `filter` | 2 |
| `ntn search` | `p (-p)`, `page-size` | 1 |
| `ntn search` | `s (-s)`, `sort` | 1 |
| `ntn skill sync` | `a (-a)`, `add-new` | 1 |
| `ntn user list` | `a (-a)`, `all` | 1 |
| `ntn user list` | `p (-p)`, `page-size` | 1 |
| `ntn user list` | `s (-s)`, `start-cursor` | 1 |
| `ntn webhook parse` | `p (-p)`, `payload` | 1 |
| `ntn webhook verify` | `i (-i)`, `signature` | 1 |
| `ntn webhook verify` | `p (-p)`, `payload` | 1 |
| `ntn webhook verify` | `s (-s)`, `secret` | 1 |
| `ntn workspace add` | `a (-a)`, `api-url` | 1 |
| `ntn workspace add` | `d (-d)`, `default` | 1 |
| `ntn workspace add` | `t (-t)`, `token-source` | 1 |

## Regeneration Note
This file is generated from the live Cobra command tree. Re-run the audit after command/flag changes and compare:
1. sibling token conflicts
2. name shadowing
3. shorthand conflicts
