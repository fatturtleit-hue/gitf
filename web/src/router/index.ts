import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/home/HomeView.vue')
    },
    {
      path: '/',
      component: () => import('@/views/auth/AuthLayout.vue'),
      children: [
        {
          path: 'login',
          name: 'login',
          component: () => import('@/views/auth/LoginView.vue')
        },
        {
          path: 'register',
          name: 'register',
          component: () => import('@/views/auth/RegisterView.vue'),
          beforeEnter: () => {
            const auth = useAuthStore()
            return auth.registrationEnabled
          }
        }
      ]
    },
    {
      path: '/repos',
      name: 'repos',
      component: () => import('@/views/repos/RepoListView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/repos/new',
      name: 'repo-new',
      component: () => import('@/views/repos/RepoCreateView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/orgs',
      name: 'orgs',
      component: () => import('@/views/orgs/OrgListView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/orgs/new',
      name: 'org-new',
      component: () => import('@/views/orgs/OrgCreateView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/orgs/:org',
      name: 'org-shell',
      component: () => import('@/views/orgs/OrgShellView.vue'),
      meta: { requiresAuth: true },
      redirect: { name: 'org-members' },
      children: [
        {
          path: 'members',
          name: 'org-members',
          component: () => import('@/views/orgs/OrgMembersView.vue')
        },
        {
          path: 'teams',
          name: 'org-teams',
          component: () => import('@/views/orgs/OrgTeamsView.vue')
        },
        {
          path: 'repos',
          name: 'org-repos',
          component: () => import('@/views/orgs/OrgReposView.vue')
        },
        {
          path: 'settings',
          name: 'org-settings',
          component: () => import('@/views/orgs/OrgSettingsView.vue')
        }
      ]
    },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('@/views/admin/AdminView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/tokens',
      name: 'tokens',
      component: () => import('@/views/admin/PatTokensView.vue'),
      meta: { requiresAuth: true }
    },
    {
      path: '/:owner/:name',
      name: 'repo-shell',
      component: () => import('@/views/repo/RepoShellView.vue'),
      redirect: { name: 'repo-tree' },
      children: [
        {
          path: 'tree',
          name: 'repo-tree',
          component: () => import('@/views/repo/FileBrowserView.vue')
        },
        {
          path: 'blob',
          name: 'repo-blob',
          component: () => import('@/views/repo/BlobViewerView.vue')
        },
        {
          path: 'commits',
          name: 'repo-commits',
          component: () => import('@/views/repo/CommitListView.vue')
        },
        {
          path: 'commit/:hash',
          name: 'repo-commit',
          component: () => import('@/views/repo/CommitDetailView.vue')
        },
        {
          path: 'issues',
          name: 'repo-issues',
          component: () => import('@/views/repo/IssueListView.vue')
        },
        {
          path: 'issues/new',
          name: 'repo-issue-new',
          component: () => import('@/views/repo/IssueCreateView.vue')
        },
        {
          path: 'issues/:number',
          name: 'repo-issue-detail',
          component: () => import('@/views/repo/IssueDetailView.vue')
        },
        {
          path: 'board',
          name: 'repo-board',
          component: () => import('@/views/repo/KanbanBoardView.vue')
        },
        {
          path: 'tags',
          name: 'repo-tags',
          component: () => import('@/views/repo/TagListView.vue')
        },
        {
          path: 'releases',
          name: 'repo-releases',
          component: () => import('@/views/repo/ReleaseListView.vue')
        },
        {
          path: 'compare/:base...:head',
          name: 'repo-compare',
          component: () => import('@/views/repo/CompareView.vue')
        },
        {
          path: 'settings',
          name: 'repo-settings',
          component: () => import('@/views/repo/RepoSettingsView.vue')
        },
        {
          path: 'pulls',
          name: 'repo-pulls',
          component: () => import('@/views/repo/PullRequestListView.vue')
        },
        {
          path: 'pulls/new',
          name: 'repo-pull-new',
          component: () => import('@/views/repo/PullRequestCreateView.vue')
        },
        {
          path: 'pulls/:number',
          name: 'repo-pull-detail',
          component: () => import('@/views/repo/PullRequestDetailView.vue')
        },
        {
          path: 'hooks',
          name: 'repo-hooks',
          component: () => import('@/views/repo/WebhookListView.vue')
        },
        {
          path: 'labels',
          name: 'repo-labels',
          component: () => import('@/views/repo/LabelsView.vue')
        },
        {
          path: 'milestones',
          name: 'repo-milestones',
          component: () => import('@/views/repo/MilestonesView.vue')
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/not-found/NotFoundView.vue')
    }
  ]
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    next({ name: 'login', query: { returnUrl: to.fullPath } })
  } else {
    next()
  }
})

export default router
