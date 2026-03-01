/**
 * baseFileProvider.ts
 *
 * TextDocumentContentProvider for the "critic-base" URI scheme.
 * Serves file contents from a git ref via `git show <ref>:<path>`.
 *
 * URI format: critic-base:/<filePath>?ref=<gitRef>
 */

import * as vscode from 'vscode'
import { execFile } from 'child_process'

export class BaseFileProvider implements vscode.TextDocumentContentProvider {
  private gitRoot = ''

  setGitRoot(root: string): void {
    this.gitRoot = root
  }

  provideTextDocumentContent(uri: vscode.Uri): Promise<string> {
    const filePath = uri.path.startsWith('/') ? uri.path.slice(1) : uri.path
    const ref = new URLSearchParams(uri.query).get('ref') ?? 'HEAD'

    return new Promise<string>((resolve, reject) => {
      execFile(
        'git',
        ['show', `${ref}:${filePath}`],
        { cwd: this.gitRoot || undefined, maxBuffer: 10 * 1024 * 1024 },
        (err, stdout) => {
          if (err) {
            // File doesn't exist at this ref (e.g. newly added file)
            resolve('')
            return
          }
          resolve(stdout)
        }
      )
    })
  }
}

export function makeBaseUri(filePath: string, ref: string): vscode.Uri {
  return vscode.Uri.parse(`critic-base:/${filePath}?ref=${encodeURIComponent(ref)}`)
}
