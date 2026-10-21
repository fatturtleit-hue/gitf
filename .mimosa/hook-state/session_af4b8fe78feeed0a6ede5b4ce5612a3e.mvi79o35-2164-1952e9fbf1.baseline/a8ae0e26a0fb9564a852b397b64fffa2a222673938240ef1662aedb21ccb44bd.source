import { get, post, put, patch, del, getText } from './request'
import type {
  Repository,
  CreateRepoRequest,
  UpdateRepoRequest,
  CreateForkRequest,
  Collaborator,
  FileDiff,
  TagInfo,
  TreeEntry,
  CommitInfo,
  BlobInfo,
  RepoStats,
  PaginatedResponse
} from '@/api/types'

function repoPath(owner: string, name: string): string {
  return `/repos/${encodeURIComponent(owner)}/${encodeURIComponent(name)}`
}

export const repoApi = {
  list(search?: string, page?: number, perPage?: number): Promise<PaginatedResponse<Repository>> {
    const params: Record<string, string | number> = {}
    if (search) params.search = search
    if (page != null && page > 0) params.page = page
    if (perPage != null && perPage > 0) params.per_page = perPage
    return get<PaginatedResponse<Repository>>('/repos', { params })
  },

  create(data: CreateRepoRequest): Promise<Repository> {
    return post<Repository>('/repos', data)
  },

  get(owner: string, name: string): Promise<Repository> {
    return get<Repository>(repoPath(owner, name))
  },

  del(owner: string, name: string): Promise<unknown> {
    return del<unknown>(repoPath(owner, name))
  },

  stats(owner: string, name: string, branch: string): Promise<RepoStats> {
    return get<RepoStats>(`${repoPath(owner, name)}/stats`, { params: { branch } })
  },

  branches(owner: string, name: string): Promise<{ branches: string[]; default: string }> {
    return get<{ branches: string[]; default: string }>(`${repoPath(owner, name)}/branches`)
  },

  commits(owner: string, name: string, branch: string, limit?: number): Promise<{ commits: CommitInfo[] }> {
    const params: Record<string, string | number> = { branch }
    if (limit != null && limit > 0) params.limit = limit
    return get<{ commits: CommitInfo[] }>(`${repoPath(owner, name)}/commits`, { params })
  },

  getCommit(owner: string, name: string, hash: string): Promise<CommitInfo> {
    return get<CommitInfo>(`${repoPath(owner, name)}/commits/${encodeURIComponent(hash)}`)
  },

  tree(owner: string, name: string, branch: string, path: string): Promise<{ entries: TreeEntry[] }> {
    return get<{ entries: TreeEntry[] }>(`${repoPath(owner, name)}/tree`, { params: { branch, path } })
  },

  blob(owner: string, name: string, branch: string, path: string): Promise<BlobInfo> {
    return get<BlobInfo>(`${repoPath(owner, name)}/blob`, { params: { branch, path } })
  },

  raw(owner: string, name: string, branch: string, path: string): Promise<string> {
    return getText(`${repoPath(owner, name)}/raw`, { params: { branch, path } })
  },

  listCollaborators(owner: string, name: string): Promise<{ items: Collaborator[] }> {
    return get<{ items: Collaborator[] }>(`${repoPath(owner, name)}/collaborators`)
  },

  addCollaborator(
    owner: string,
    name: string,
    username: string,
    permission: string,
    quotaBytes: number
  ): Promise<Collaborator> {
    return post<Collaborator>(`${repoPath(owner, name)}/collaborators`, {
      username,
      permission,
      quota_bytes: quotaBytes
    })
  },

  updateCollaborator(
    owner: string,
    name: string,
    collabId: number,
    permission: string,
    quotaBytes: number
  ): Promise<Collaborator> {
    return put<Collaborator>(`${repoPath(owner, name)}/collaborators/${collabId}`, {
      permission,
      quota_bytes: quotaBytes
    })
  },

  removeCollaborator(owner: string, name: string, userId: number): Promise<unknown> {
    return del<unknown>(`${repoPath(owner, name)}/collaborators/${userId}`)
  },

  tags(owner: string, name: string): Promise<{ tags: TagInfo[] }> {
    return get<{ tags: TagInfo[] }>(`${repoPath(owner, name)}/tags`)
  },

  compare(
    owner: string,
    name: string,
    base: string,
    head: string
  ): Promise<{ diffs: FileDiff[]; stats: Record<string, string> }> {
    return get<{ diffs: FileDiff[]; stats: Record<string, string> }>(
      `${repoPath(owner, name)}/compare/${encodeURIComponent(base)}...${encodeURIComponent(head)}`
    )
  },

  archiveUrl(owner: string, name: string, ref: string, format: 'tar.gz' | 'zip' = 'tar.gz'): string {
    return `/api/v1${repoPath(owner, name)}/archive/${encodeURIComponent(ref)}?format=${format}`
  },

  fork(owner: string, name: string, req: CreateForkRequest): Promise<Repository> {
    return post<Repository>(`${repoPath(owner, name)}/fork`, req)
  },

  update(owner: string, name: string, req: UpdateRepoRequest): Promise<Repository> {
    return patch<Repository>(repoPath(owner, name), req)
  },

  adminStats(): Promise<{
    disk_total: number
    disk_used: number
    disk_free: number
    repo_count: number
    user_count: number
  }> {
    return get('/admin/stats')
  }
}
