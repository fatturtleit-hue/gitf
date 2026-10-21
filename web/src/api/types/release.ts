export interface Release {
  id: number
  repo_id: number
  tag_name: string
  target_commitish: string
  name: string
  body: string
  is_draft: boolean
  is_prerelease: boolean
  author_id: number
  author_username: string
  created_at: string
  updated_at: string
}

export interface ReleaseAsset {
  id: number
  release_id: number
  name: string
  size: number
  content_type: string
  created_at: string
}

export interface CreateReleaseRequest {
  tag_name: string
  name: string
  body: string
  is_draft?: boolean
  is_prerelease?: boolean
}
