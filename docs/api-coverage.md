# Notion CLI API Coverage

This document tracks the CLI's coverage of the Notion API.

## Coverage Matrix

| API Resource | Endpoint | CLI Command | Status |
|-------------|----------|-------------|--------|
| **Blocks** | GET /blocks/{id} | `block get` | ✅ |
| | GET /blocks/{id}/children | `block children` | ✅ |
| | PATCH /blocks/{id}/children | `block append` | ✅ |
| | PATCH /blocks/{id} | `block update` | ✅ |
| | DELETE /blocks/{id} | `block delete` | ✅ |
| **Pages** | POST /pages | `page create` | ✅ |
| | GET /pages/{id} | `page get` | ✅ |
| | GET /pages/{id}/properties/{prop} | `page property` | ✅ |
| | PATCH /pages/{id} | `page update` | ✅ |
| | POST /pages/{id}/move | `page move` | ✅ |
| **Databases** | POST /databases | `db create` | ✅ |
| | GET /databases/{id} | `db get` | ✅ |
| | PATCH /databases/{id} | `db update` | ✅ |
| | POST /databases/{id}/query | `db query` | ✅ |
| **Data Sources** | POST /data_sources | `datasource create` | ✅ |
| | GET /data_sources/{id} | `datasource get` | ✅ |
| | PATCH /data_sources/{id} | `datasource update` | ✅ |
| | POST /data_sources/{id}/query | `datasource query` | ✅ |
| | GET /data_sources/templates | `datasource templates` | ✅ |
| | (via search) | `datasource list` | ✅ |
| **Comments** | POST /comments | `comment add` | ✅ |
| | GET /comments | `comment list` | ✅ |
| | GET /comments/{id} | `comment get` | ✅ |
| **File Uploads** | POST /files | `file upload` | ✅ |
| | GET /files/{id} | `file get` | ✅ |
| | GET /files | `file list` | ✅ |
| **Users** | GET /users | `user list` | ✅ |
| | GET /users/{id} | `user get` | ✅ |
| | GET /users/me | `user me` | ✅ |
| **Search** | POST /search | `search` | ✅ |

## Flag Consistency

Commands that return lists support these flags:

| Command | `--all` | `--results-only` | `--page-size` |
|---------|---------|------------------|---------------|
| `user list` | ✅ | ✅ | ✅ |
| `search` | ✅ | ✅ | ✅ |
| `db query` | ✅ | ✅ | ✅ |
| `datasource query` | ✅ | ✅ | ✅ |
| `datasource list` | ✅ | ✅ | ✅ |
| `comment list` | ✅ | ✅ | ✅ |
| `block children` | ✅ | ❌ | ✅ |
| `file list` | ❌ | ❌ | ✅ |

### Notes

- `--all`: Fetches all pages of results automatically (handles pagination)
- `--results-only`: Outputs only the results array without pagination metadata
- `--page-size`: Controls number of results per page (max 100)

## Remaining Gaps

1. **`block children`**: Missing `--results-only` flag
2. **`file list`**: Missing `--all` and `--results-only` flags

## Last Updated

2026-01-10
