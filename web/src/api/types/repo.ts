export interface Repository {
  id: number
  owner_id: number
  owner_type: string
  owner_username: string
  name: string
  description: string
  is_private: boolean
  default_branch: string
  quota_bytes: number
  forked_from_id: number | null
  created_at: string
  updated_at: string
}

export interface CreateRepoRequest {
  name: string
  description?: string
  is_private?: boolean
  default_branch?: string
}

export interface UpdateRepoRequest {
  name?: string
  description?: string
  is_private?: boolean
  default_branch?: string
}

export interface CreateForkRequest {
  name: string
  description: string
}

export interface Collaborator {
  id: number
  user_id: number
  username: string
  email: string
  permission: string
  quota_bytes: number
  created_at: string
}

export interface FileDiff {
  path: string
  old_path?: string
  status: 'added' | 'modified' | 'deleted' | 'renamed'
  patch: string
}

export interface TagInfo {
  name: string
  hash: string
  message?: string
  tagger?: string
  date?: string
  is_annotated: boolean
}

export interface TreeEntry {
  name: string
  path: string
  type: 'blob' | 'tree'
  mode: string
  size: number
}

export interface CommitInfo {
  hash: string
  short_hash: string
  message: string
  author: string
  email: string
  date: string
  parents: string[]
}

export interface BlobInfo {
  content: string
  size: number
  path: string
}

export interface RepoStats {
  size: number
  file_count: number
  dir_count: number
  commit_count: number
  branch_count: number
}
