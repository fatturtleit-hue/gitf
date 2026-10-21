export interface Issue {
  id: number
  repo_id: number
  number: number
  title: string
  body: string
  author_id: number
  author_username: string
  assignee_id: number | null
  assignee_username?: string
  status: 'open' | 'closed'
  is_pr: boolean
  pr_id?: number
  milestone_id?: number
  closed_at?: string
  created_at: string
  updated_at: string
}

export interface Comment {
  id: number
  issue_id: number
  author_id: number
  author_username: string
  body: string
  path?: string
  line?: number
  commit_id?: string
  created_at: string
}

export interface Label {
  id: number
  repo_id: number
  name: string
  color: string
}

export interface Milestone {
  id: number
  repo_id: number
  title: string
  description: string
  due_date?: string
  status: 'open' | 'closed'
  created_at: string
}

export interface CreateIssueRequest {
  title: string
  body?: string
  assignee_id?: number
  milestone_id?: number
  label_ids?: number[]
}
