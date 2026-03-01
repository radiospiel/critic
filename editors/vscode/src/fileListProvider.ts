/**
 * fileListProvider.ts
 *
 * TreeDataProvider for the Critic Activity Bar view.
 * Shows files grouped under category nodes: Conversations, All Files, then
 * dynamic categories from project.critic (e.g. Tests, Hidden Files).
 * "All Files" shows a nested directory tree; other categories show flat lists.
 */

import * as vscode from 'vscode'
import { CriticClient, FileSummary, FileConversationSummary } from './criticClient'
import { matchesAnyPattern } from './globMatch'

// ---- Types ---------------------------------------------------------------- //

type TreeNode = CategoryNode | DirectoryNode | FileTreeItem

interface FileCategory {
  name: string
  patterns: string[]
}

// ---- Category node -------------------------------------------------------- //

/** Fixed categories always shown (in order). */
const FIXED_CATEGORIES: { key: string; label: string; icon: string }[] = [
  { key: 'conversations', label: 'Conversations', icon: 'comment-discussion' },
  { key: 'files',         label: 'All Files',      icon: 'file' },
]

/** Map well-known category names to display labels and icons. */
const KNOWN_CATEGORY_STYLE: Record<string, { label: string; icon: string }> = {
  test:   { label: 'Tests',        icon: 'beaker' },
  hidden: { label: 'Hidden Files', icon: 'eye-closed' },
}

function categoryDisplay(name: string): { label: string; icon: string } {
  const known = KNOWN_CATEGORY_STYLE[name]
  if (known) return known
  // Capitalize first letter, e.g. "docs" → "Docs"
  const label = name.charAt(0).toUpperCase() + name.slice(1)
  return { label, icon: 'folder' }
}

export class CategoryNode extends vscode.TreeItem {
  readonly categoryKey: string
  children: (DirectoryNode | FileTreeItem)[] = []

  constructor(key: string, label: string, icon: string, count: number) {
    super(label, vscode.TreeItemCollapsibleState.Collapsed)
    this.categoryKey = key
    this.iconPath = new vscode.ThemeIcon(icon)
    this.description = `${count}`
    this.contextValue = 'category'
  }
}

// ---- Directory node (for tree structure under "All Files") ---------------- //

export class DirectoryNode extends vscode.TreeItem {
  children: (DirectoryNode | FileTreeItem)[] = []

  constructor(public readonly dirName: string) {
    super(dirName, vscode.TreeItemCollapsibleState.Expanded)
    this.iconPath = vscode.ThemeIcon.Folder
    this.contextValue = 'directory'
  }
}

// ---- File tree item ------------------------------------------------------- //

export class FileTreeItem extends vscode.TreeItem {
  constructor(
    public readonly filePath: string,
    public readonly fileStatus: string,
    conversationSummary: FileConversationSummary | undefined,
    gitRoot: string,
    /** When true, label shows only the basename (used in tree views). */
    basenameOnly: boolean = false,
    /** Git ref for the base version (used to open diff view). */
    private readonly baseRef: string = '',
  ) {
    super(
      basenameOnly ? filePath.split('/').pop()! : filePath,
      vscode.TreeItemCollapsibleState.None,
    )

    this.description = ''

    this.resourceUri = vscode.Uri.file(gitRoot ? `${gitRoot}/${filePath}` : filePath)

    const badge = statusBadge(fileStatus)
    if (badge) {
      this.iconPath = new vscode.ThemeIcon(badge.icon, badge.color)
    }

    if (conversationSummary && conversationSummary.totalCount > 0) {
      const { unresolvedCount, totalCount, hasUnreadAiMessages } = conversationSummary
      const parts: string[] = []
      if (unresolvedCount > 0) parts.push(`${unresolvedCount} open`)
      if (totalCount - unresolvedCount > 0) parts.push(`${totalCount - unresolvedCount} resolved`)
      if (hasUnreadAiMessages) parts.push('unread')

      this.tooltip = `${filePath} [${fileStatus}] — ${parts.join(', ')}`
      this.description = `💬 ${totalCount}`
    } else {
      this.tooltip = `${filePath} [${fileStatus}]`
    }

    this.command = {
      command: 'critic.openFile',
      title: 'Open File',
      arguments: [this.resourceUri, filePath, fileStatus, this.baseRef],
    }
  }
}

function statusBadge(status: string): { icon: string; color: vscode.ThemeColor } | undefined {
  switch (status.toUpperCase()) {
    case 'MODIFIED':
    case 'M':
      return { icon: 'diff-modified', color: new vscode.ThemeColor('gitDecoration.modifiedResourceForeground') }
    case 'ADDED':
    case 'A':
      return { icon: 'diff-added', color: new vscode.ThemeColor('gitDecoration.addedResourceForeground') }
    case 'DELETED':
    case 'D':
      return { icon: 'diff-removed', color: new vscode.ThemeColor('gitDecoration.deletedResourceForeground') }
    case 'RENAMED':
    case 'R':
      return { icon: 'diff-renamed', color: new vscode.ThemeColor('gitDecoration.renamedResourceForeground') }
    default:
      return undefined
  }
}

// ---- Directory tree builder ----------------------------------------------- //

function buildDirectoryTree(
  items: FileTreeItem[],
  gitRoot: string,
  summaryMap: Map<string, FileConversationSummary>,
  baseRef: string,
): (DirectoryNode | FileTreeItem)[] {
  interface DirEntry {
    dirs: Map<string, DirEntry>
    files: FileTreeItem[]
  }

  const root: DirEntry = { dirs: new Map(), files: [] }

  for (const item of items) {
    const parts = item.filePath.split('/')
    let current = root
    for (let i = 0; i < parts.length - 1; i++) {
      if (!current.dirs.has(parts[i])) {
        current.dirs.set(parts[i], { dirs: new Map(), files: [] })
      }
      current = current.dirs.get(parts[i])!
    }
    // Re-create item with basename-only label for tree display
    const treeItem = new FileTreeItem(
      item.filePath,
      item.fileStatus,
      summaryMap.get(item.filePath),
      gitRoot,
      true,
      baseRef,
    )
    current.files.push(treeItem)
  }

  function toNodes(entry: DirEntry): (DirectoryNode | FileTreeItem)[] {
    const result: (DirectoryNode | FileTreeItem)[] = []

    // Sort directories alphabetically
    const sortedDirs = [...entry.dirs.entries()].sort(([a], [b]) => a.localeCompare(b))
    for (const [name, child] of sortedDirs) {
      const dirNode = new DirectoryNode(name)
      dirNode.children = toNodes(child)
      result.push(dirNode)
    }

    // Sort files alphabetically
    entry.files.sort((a, b) => a.filePath.localeCompare(b.filePath))
    result.push(...entry.files)

    return result
  }

  return toNodes(root)
}

// ---- Categorization ------------------------------------------------------- //

function categorizeFile(path: string, categories: FileCategory[]): string {
  for (const category of categories) {
    if (matchesAnyPattern(path, category.patterns)) {
      return category.name
    }
  }
  return 'source'
}

// ---- Provider ------------------------------------------------------------- //

export class FileListProvider implements vscode.TreeDataProvider<TreeNode> {
  private allItems: FileTreeItem[] = []
  private categoryNodes: CategoryNode[] = []
  private gitRoot = ''
  private baseRef = ''
  private categories: FileCategory[] = []
  private summaryMap = new Map<string, FileConversationSummary>()
  private expandedKey: string = 'conversations'
  private suppressCollapseHandler = false

  private _onDidChangeTreeData = new vscode.EventEmitter<TreeNode | undefined | void>()
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event

  setGitRoot(root: string): void {
    this.gitRoot = root
  }

  setBaseRef(ref: string): void {
    this.baseRef = ref
  }

  /** Call when a category is expanded — collapses all others. */
  onDidExpand(node: CategoryNode): void {
    if (node.categoryKey === this.expandedKey) return
    this.expandedKey = node.categoryKey
    // Re-fire so collapsed nodes get their new collapsibleState
    this.suppressCollapseHandler = true
    this._onDidChangeTreeData.fire()
  }

  /** Call when a category is collapsed — re-expand it (always one open). */
  onDidCollapse(node: CategoryNode): void {
    if (this.suppressCollapseHandler) {
      this.suppressCollapseHandler = false
      return
    }
    // Keep it expanded: re-fire to restore Expanded state
    this._onDidChangeTreeData.fire()
  }

  async refresh(client: CriticClient): Promise<void> {
    try {
      const [diffResult, summaries, projectConfig] = await Promise.all([
        client.getDiffSummary(),
        client.getConversationsSummary(),
        client.getProjectConfig(),
      ])

      this.categories = projectConfig.categories

      this.summaryMap = new Map<string, FileConversationSummary>()
      for (const s of summaries) {
        this.summaryMap.set(s.filePath, s)
      }

      this.allItems = diffResult.files.map((f: FileSummary) => {
        const path = f.newPath || f.oldPath
        return new FileTreeItem(path, f.status, this.summaryMap.get(path), this.gitRoot, false, this.baseRef)
      })

      this.recompute()
    } catch {
      this.allItems = []
      this.categoryNodes = []
      this._onDidChangeTreeData.fire()
    }
  }

  private recompute(): void {
    const sortByPath = (a: FileTreeItem, b: FileTreeItem) =>
      a.filePath.localeCompare(b.filePath)

    // Dynamic buckets: one per category from config, plus "source" for uncategorized
    const categoryBuckets = new Map<string, FileTreeItem[]>()
    for (const cat of this.categories) {
      categoryBuckets.set(cat.name, [])
    }

    const sourceFiles: FileTreeItem[] = []

    for (const item of this.allItems) {
      const cat = categorizeFile(item.filePath, this.categories)
      if (cat === 'source') {
        sourceFiles.push(item)
      } else {
        categoryBuckets.get(cat)!.push(item)
      }
    }

    sourceFiles.sort(sortByPath)

    // Conversations: all non-hidden files that have conversations
    const hiddenBucket = categoryBuckets.get('hidden') ?? []
    const hiddenPaths = new Set(hiddenBucket.map((f) => f.filePath))
    const conversationFiles = this.allItems
      .filter((item) => {
        if (hiddenPaths.has(item.filePath)) return false
        const summary = this.summaryMap.get(item.filePath)
        return summary && summary.totalCount > 0
      })
      .sort(sortByPath)

    // Build ordered list of buckets: conversations, files (source), then each config category
    const buckets: { key: string; label: string; icon: string; files: FileTreeItem[]; isTree: boolean }[] = []

    buckets.push({
      key: 'conversations',
      label: 'Conversations',
      icon: 'comment-discussion',
      files: conversationFiles,
      isTree: false,
    })

    buckets.push({
      key: 'files',
      label: 'All Files',
      icon: 'file',
      files: sourceFiles,
      isTree: true,
    })

    for (const cat of this.categories) {
      const display = categoryDisplay(cat.name)
      const files = categoryBuckets.get(cat.name) ?? []
      buckets.push({
        key: cat.name,
        label: display.label,
        icon: display.icon,
        files,
        isTree: false,
      })
    }

    // If the currently expanded key has no files, fall back to first non-empty
    const hasExpanded = buckets.some((b) => b.key === this.expandedKey && b.files.length > 0)
    if (!hasExpanded) {
      const first = buckets.find((b) => b.files.length > 0)
      if (first) this.expandedKey = first.key
    }

    this.categoryNodes = []
    for (const bucket of buckets) {
      if (bucket.files.length === 0) continue
      const node = new CategoryNode(bucket.key, bucket.label, bucket.icon, bucket.files.length)

      if (bucket.isTree) {
        node.children = buildDirectoryTree(bucket.files, this.gitRoot, this.summaryMap, this.baseRef)
      } else {
        node.children = bucket.files
      }

      node.collapsibleState =
        bucket.key === this.expandedKey
          ? vscode.TreeItemCollapsibleState.Expanded
          : vscode.TreeItemCollapsibleState.Collapsed
      this.categoryNodes.push(node)
    }

    this._onDidChangeTreeData.fire()
  }

  clear(): void {
    this.allItems = []
    this.categoryNodes = []
    this._onDidChangeTreeData.fire()
  }

  getTreeItem(element: TreeNode): vscode.TreeItem {
    return element
  }

  getChildren(element?: TreeNode): TreeNode[] {
    if (!element) {
      return this.categoryNodes
    }
    if (element instanceof CategoryNode) {
      return element.children
    }
    if (element instanceof DirectoryNode) {
      return element.children
    }
    return []
  }

  dispose(): void {
    this._onDidChangeTreeData.dispose()
  }
}
