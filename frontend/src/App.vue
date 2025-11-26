<template>
  <div class="min-h-screen bg-base-200">
    <!-- Navbar -->
    <div class="navbar bg-primary text-primary-content shadow-lg sticky top-0 z-50">
      <div class="flex-1">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 ml-4 mr-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
        <h1 class="text-2xl font-bold">{{ t('title') }}</h1>
      </div>
      <div class="flex-none gap-3 mr-4">
        <div class="dropdown dropdown-end">
          <label tabindex="0" class="btn btn-ghost btn-circle">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129" />
            </svg>
          </label>
          <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
            <li><a @click="$i18n.locale = 'en'" :class="{'active': $i18n.locale === 'en'}">🇬🇧 English</a></li>
            <li><a @click="$i18n.locale = 'de'" :class="{'active': $i18n.locale === 'de'}">🇩🇪 Deutsch</a></li>
          </ul>
        </div>
        
        <label class="swap swap-rotate btn btn-ghost btn-circle">
          <input type="checkbox" class="theme-controller" @change="toggleTheme" />
          <svg class="swap-off h-6 w-6 fill-current" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
            <path d="M5.64,17l-.71.71a1,1,0,0,0,0,1.41,1,1,0,0,0,1.41,0l.71-.71A1,1,0,0,0,5.64,17ZM5,12a1,1,0,0,0-1-1H3a1,1,0,0,0,0,2H4A1,1,0,0,0,5,12Zm7-7a1,1,0,0,0,1-1V3a1,1,0,0,0-2,0V4A1,1,0,0,0,12,5ZM5.64,7.05a1,1,0,0,0,.7.29,1,1,0,0,0,.71-.29,1,1,0,0,0,0-1.41l-.71-.71A1,1,0,0,0,4.93,6.34Zm12,.29a1,1,0,0,0,.7-.29l.71-.71a1,1,0,1,0-1.41-1.41L17,5.64a1,1,0,0,0,0,1.41A1,1,0,0,0,17.66,7.34ZM21,11H20a1,1,0,0,0,0,2h1a1,1,0,0,0,0-2Zm-9,8a1,1,0,0,0-1,1v1a1,1,0,0,0,2,0V20A1,1,0,0,0,12,19ZM18.36,17A1,1,0,0,0,17,18.36l.71.71a1,1,0,0,0,1.41,0,1,1,0,0,0,0-1.41ZM12,6.5A5.5,5.5,0,1,0,17.5,12,5.51,5.51,0,0,0,12,6.5Zm0,9A3.5,3.5,0,1,1,15.5,12,3.5,3.5,0,0,1,12,15.5Z"/>
          </svg>
          <svg class="swap-on h-6 w-6 fill-current" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
            <path d="M21.64,13a1,1,0,0,0-1.05-.14,8.05,8.05,0,0,1-3.37.73A8.15,8.15,0,0,1,9.08,5.49a8.59,8.59,0,0,1,.25-2A1,1,0,0,0,8,2.36,10.14,10.14,0,1,0,22,14.05,1,1,0,0,0,21.64,13Zm-9.5,6.69A8.14,8.14,0,0,1,7.08,5.22v.27A10.15,10.15,0,0,0,17.22,15.63a9.79,9.79,0,0,0,2.1-.22A8.11,8.11,0,0,1,12.14,19.73Z"/>
          </svg>
        </label>
        
        <button @click="fetchBackups" class="btn btn-accent gap-2" :disabled="loading">
          <svg v-if="!loading" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span v-if="loading" class="loading loading-spinner loading-sm"></span>
          {{ t('refresh') }}
        </button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="container mx-auto p-4 md:p-8 max-w-7xl">
      <!-- Global Actions -->
      <div v-if="!loading && backups.length > 0" class="mb-4 flex justify-between items-center">
        <label class="label cursor-pointer gap-2">
          <input type="checkbox" v-model="showDisabled" class="checkbox checkbox-sm" />
          <span class="label-text">{{ t('showDisabled') }}</span>
        </label>
        <button 
          @click="pruneAll" 
          :disabled="pruning['all']"
          class="btn btn-outline btn-warning btn-sm gap-2">
          <svg v-if="!pruning['all']" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          <span v-if="pruning['all']" class="loading loading-spinner loading-xs"></span>
          {{ pruning['all'] ? t('pruning') : t('pruneAll') }}
        </button>
      </div>

      <!-- Stats Overview -->
      <div v-if="!loading && backups.length > 0" class="stats stats-vertical lg:stats-horizontal shadow w-full mb-8 bg-base-100">
        <div class="stat">
          <div class="stat-figure text-primary">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
            </svg>
          </div>
          <div class="stat-title">{{ t('totalTargets') }}</div>
          <div class="stat-value text-primary">{{ backups.length }}</div>
        </div>
        
        <div class="stat">
          <div class="stat-figure text-success">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="stat-title">{{ t('healthy') }}</div>
          <div class="stat-value text-success">{{ healthyCount }}</div>
        </div>
        
        <div class="stat">
          <div class="stat-figure text-error">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="stat-title">{{ t('issues') }}</div>
          <div class="stat-value text-error">{{ issuesCount }}</div>
        </div>

        <div class="stat">
          <div class="stat-figure text-warning">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
          </div>
          <div class="stat-title">{{ t('locked') }}</div>
          <div class="stat-value text-warning">{{ lockedCount }}</div>
        </div>
      </div>

      <!-- Agents Section -->
      <div v-if="agents.length > 0" class="mb-8">
        <h2 class="text-2xl font-bold mb-4 flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
          </svg>
          Backup Agents
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div v-for="agent in agents" :key="agent.id" class="card bg-base-100 shadow-lg border border-base-300">
            <div class="card-body p-4">
              <div class="flex items-start justify-between mb-2">
                <h3 class="font-bold text-lg">{{ agent.hostname }}</h3>
                <div class="badge" :class="agentStatusClass(agent)">{{ agent.status }}</div>
              </div>
              <div class="text-sm space-y-1 text-base-content/70">
                <div class="flex justify-between">
                  <span>Version:</span>
                  <span class="font-mono">{{ agent.version }}</span>
                </div>
                <div class="flex justify-between">
                  <span>OS:</span>
                  <span class="font-mono">{{ agent.os }}/{{ agent.arch }}</span>
                </div>
                <div class="flex justify-between">
                  <span>Uptime:</span>
                  <span>{{ formatUptime(agent.uptime_seconds) }}</span>
                </div>
                <div class="flex justify-between">
                  <span>Last seen:</span>
                  <span>{{ formatTime(agent.last_seen_at) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Policies Section -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-2xl font-bold flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            Backup Policies
          </h2>
          <button @click="openCreatePolicyModal" class="btn btn-primary btn-sm gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Create Policy
          </button>
        </div>
        
        <!-- Empty State -->
        <div v-if="policies.length === 0" class="card bg-base-100 shadow-lg border border-base-300">
          <div class="card-body items-center text-center py-12">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-base-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <h3 class="text-xl font-bold mb-2">No Backup Policies</h3>
            <p class="text-base-content/60 mb-4">Create your first backup policy to get started</p>
            <button @click="openCreatePolicyModal" class="btn btn-primary gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              Create Your First Policy
            </button>
          </div>
        </div>
        
        <!-- Policies Grid -->
        <div v-else class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div v-for="policy in policies" :key="policy.id" class="card bg-base-100 shadow-lg border border-base-300">
            <div class="card-body p-4">
              <div class="flex items-start justify-between mb-3">
                <div class="flex-1">
                  <h3 class="font-bold text-lg">{{ policy.name }}</h3>
                  <p v-if="policy.description" class="text-sm text-base-content/60 mt-1">{{ policy.description }}</p>
                </div>
                <div class="badge badge-lg" :class="getPolicyStatusClass(policy)">
                  {{ policy.enabled ? '✓ Enabled' : 'Disabled' }}
                </div>
              </div>
              
              <div class="divider my-2"></div>
              
              <div class="space-y-2 text-sm">
                <!-- Schedule -->
                <div class="flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span class="opacity-70">Schedule:</span>
                  <span class="font-semibold">{{ formatCron(policy.schedule) }}</span>
                </div>
                
                <!-- Repository -->
                <div class="flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-secondary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
                  </svg>
                  <span class="opacity-70">Repository:</span>
                  <span class="font-mono text-xs">{{ policy.repository_type }}</span>
                </div>
                
                <!-- Retention -->
                <div class="flex items-center gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
                  </svg>
                  <span class="opacity-70">Retention:</span>
                  <span class="font-semibold">
                    {{ policy.retention_rules?.keep_last || 0 }} last, 
                    {{ policy.retention_rules?.keep_daily || 0 }} daily
                  </span>
                </div>
                
                <!-- Paths -->
                <div class="flex items-start gap-2">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-info mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                  <span class="opacity-70">Paths:</span>
                  <span class="font-mono text-xs">{{ policy.include_paths?.paths?.length || 0 }} included</span>
                </div>
              </div>
              
              <div class="card-actions justify-end mt-4 gap-2">
                <button @click="openAssignModal(policy)" class="btn btn-sm btn-ghost gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                  Assign
                </button>
                <button @click="openDetachModal(policy)" class="btn btn-sm btn-ghost gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                  </svg>
                  Detach
                </button>
                <button @click="openEditPolicyModal(policy)" class="btn btn-sm btn-ghost gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  Edit
                </button>
                <button @click="confirmDeletePolicy(policy)" class="btn btn-sm btn-error btn-ghost gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Backup Runs Section -->
      <div class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-2xl font-bold flex items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            Recent Backup Runs
          </h2>
          <div class="flex gap-2">
            <select v-model="runsFilter.status" class="select select-bordered select-sm">
              <option value="">All Statuses</option>
              <option value="success">Success</option>
              <option value="failed">Failed</option>
              <option value="running">Running</option>
            </select>
            <select v-model="runsFilter.agentId" class="select select-bordered select-sm">
              <option value="">All Agents</option>
              <option v-for="agent in agents" :key="agent.id" :value="agent.id">{{ agent.hostname }}</option>
            </select>
          </div>
        </div>
        
        <div v-if="loadingRuns" class="flex justify-center py-12">
          <span class="loading loading-spinner loading-lg"></span>
        </div>
        
        <div v-else-if="backupRuns.length === 0" class="card bg-base-100 shadow-lg border border-base-300">
          <div class="card-body items-center text-center py-12">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-base-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <h3 class="text-xl font-bold mb-2">No Backup Runs</h3>
            <p class="text-base-content/60">Backup runs will appear here once agents start executing policies</p>
          </div>
        </div>
        
        <div v-else class="overflow-x-auto">
          <table class="table table-zebra">
            <thead>
              <tr>
                <th>Status</th>
                <th>Agent</th>
                <th>Policy</th>
                <th>Start Time</th>
                <th>Duration</th>
                <th>Snapshot ID</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="run in backupRuns" :key="run.id" class="hover">
                <td>
                  <div class="badge" :class="getRunStatusClass(run.status)">
                    {{ run.status }}
                  </div>
                </td>
                <td>
                  <span class="font-mono text-sm">{{ getAgentName(run.agent_id) }}</span>
                </td>
                <td>
                  <span class="font-mono text-sm">{{ getPolicyName(run.policy_id) }}</span>
                </td>
                <td>{{ formatTime(run.start_time) }}</td>
                <td>
                  <span v-if="run.duration_seconds">{{ formatDuration(run.duration_seconds) }}</span>
                  <span v-else class="text-base-content/40">-</span>
                </td>
                <td>
                  <code v-if="run.snapshot_id" class="text-xs bg-base-200 px-2 py-1 rounded">{{ run.snapshot_id.substring(0, 8) }}</code>
                  <span v-else class="text-base-content/40">-</span>
                </td>
                <td>
                  <button @click="viewRunDetails(run)" class="btn btn-ghost btn-xs">
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                    Details
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex flex-col justify-center items-center h-96">
        <span class="loading loading-bars loading-lg text-primary"></span>
        <p class="mt-4 text-base-content/60">{{ t('loading') }}</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="alert alert-error shadow-lg">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0 stroke-current" fill="none" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div>
          <h3 class="font-bold">{{ t('error') }}</h3>
          <div class="text-xs">{{ error }}</div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="backups.length === 0" class="hero min-h-96 bg-base-100 rounded-lg shadow-xl">
        <div class="hero-content text-center">
          <div class="max-w-md">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-24 w-24 mx-auto text-base-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <h1 class="text-3xl font-bold mt-6">{{ t('noBackups') }}</h1>
            <p class="py-6 text-base-content/60">{{ t('noBackupsDesc') }}</p>
          </div>
        </div>
      </div>

      <!-- Backup Cards Grid -->
      <TransitionGroup 
        v-if="filteredBackups.length > 0"
        name="backup-list" 
        tag="div" 
        class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
        <div v-for="backup in filteredBackups" :key="backup.name" 
             class="card bg-base-100 shadow-xl hover:shadow-2xl transition-all duration-300 border border-base-300">
          <div class="card-body">
            <!-- Header -->
            <div class="flex items-start justify-between mb-4">
              <h2 class="card-title text-2xl flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
                </svg>
                {{ backup.name }}
              </h2>
              <div class="badge badge-lg gap-2" :class="getHealthBadge(backup)">
                <span v-if="backup.health && !isLocked(backup)">✓</span>
                <span v-else-if="isLocked(backup)">🔒</span>
                <span v-else>✗</span>
                {{ getHealthText(backup) }}
              </div>
            </div>

            <div class="divider my-0"></div>

            <!-- Stats -->
            <div class="grid grid-cols-2 gap-4 my-4">
              <div class="stat bg-base-200 rounded-lg p-4 cursor-pointer hover:bg-base-300 transition-colors" @click="showSnapshots(backup.name)">
                <div class="stat-title text-xs">{{ t('snapshots') }}</div>
                <div class="stat-value text-2xl text-primary">{{ backup.snapshotCount }}</div>
                <div class="stat-desc text-xs">{{ t('clickToView') }}</div>
              </div>
              <div class="stat bg-base-200 rounded-lg p-4 cursor-pointer hover:bg-base-300 transition-colors" @click="showFiles(backup.name, backup.latestSnapshotID)">
                <div class="stat-title text-xs">{{ t('files') }}</div>
                <div class="stat-value text-2xl text-secondary">{{ backup.fileCount || 0 }}</div>
                <div class="stat-desc text-xs">{{ t('clickToView') }}</div>
              </div>
            </div>

            <!-- Latest Backup -->
            <div class="flex items-center gap-3 p-3 bg-base-200 rounded-lg">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div class="flex-1">
                <div class="text-xs opacity-70">{{ t('latestBackup') }}</div>
                <div class="font-semibold text-sm">{{ formatDate(backup.latestBackup) }}</div>
              </div>
            </div>

            <!-- Last Checked -->
            <div class="flex items-center gap-3 p-3 bg-base-200 rounded-lg mt-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-info" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              </svg>
              <div class="flex-1">
                <div class="text-xs opacity-70">{{ t('lastChecked') }}</div>
                <div class="font-semibold text-sm">{{ formatDate(backup.checkedAt) }}</div>
              </div>
            </div>

            <!-- Status Message -->
            <div v-if="backup.statusMessage && !isLocked(backup)" class="alert alert-warning mt-4 py-2 px-3">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span class="text-xs">{{ backup.statusMessage }}</span>
            </div>

            <!-- Unlock Button -->
            <div v-if="isLocked(backup)" class="card-actions justify-end mt-4">
              <button 
                @click="unlockRepo(backup.name)" 
                :disabled="unlocking[backup.name]"
                class="btn btn-warning gap-2 w-full">
                <svg v-if="!unlocking[backup.name]" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 11V7a4 4 0 118 0m-4 8v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2z" />
                </svg>
                <span v-if="unlocking[backup.name]" class="loading loading-spinner loading-sm"></span>
                {{ unlocking[backup.name] ? t('unlocking') : t('unlock') }}
              </button>
            </div>

            <!-- Action Buttons -->
            <div v-if="!isLocked(backup)" class="card-actions justify-end mt-4 gap-2">
              <button 
                @click="pruneRepo(backup.name)" 
                :disabled="pruning[backup.name]"
                class="btn btn-outline btn-error btn-sm gap-2 flex-1">
                <svg v-if="!pruning[backup.name]" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <span v-if="pruning[backup.name]" class="loading loading-spinner loading-xs"></span>
                {{ pruning[backup.name] ? t('pruning') : t('prune') }}
              </button>
              <button 
                @click="toggleDisabled(backup.name)" 
                :disabled="toggling[backup.name]"
                class="btn btn-outline btn-sm gap-2 flex-1"
                :class="backup.disabled ? 'btn-success' : 'btn-info'">
                <svg v-if="!toggling[backup.name]" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path v-if="backup.disabled" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path v-if="backup.disabled" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  <path v-if="!backup.disabled" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
                <span v-if="toggling[backup.name]" class="loading loading-spinner loading-xs"></span>
                {{ toggling[backup.name] 
                    ? (backup.disabled ? t('enabling') : t('disabling'))
                    : (backup.disabled ? t('enable') : t('disable')) }}
              </button>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Footer -->
    <footer class="footer footer-center p-10 bg-base-100 text-base-content mt-16">
      <aside>
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
        <p class="font-bold">{{ t('title') }}</p>
        <p class="text-base-content/60">{{ t('footerText') }}</p>
      </aside>
    </footer>

    <!-- Snapshot Details Modal -->
    <dialog ref="snapshotModal" class="modal">
      <div class="modal-box max-w-5xl">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-2xl mb-4 flex items-center gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          {{ t('snapshotsFor') }} {{ selectedBackupName }}
        </h3>

        <div v-if="loadingSnapshots" class="flex justify-center items-center py-20">
          <span class="loading loading-bars loading-lg text-primary"></span>
        </div>

        <div v-else-if="snapshotError" class="alert alert-error">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>{{ snapshotError }}</span>
        </div>

        <div v-else-if="snapshots.length === 0" class="text-center py-12 text-base-content/60">
          {{ t('noSnapshots') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="table table-zebra table-pin-rows">
            <thead>
              <tr>
                <th>{{ t('snapshotId') }}</th>
                <th>{{ t('time') }}</th>
                <th>{{ t('hostname') }}</th>
                <th>{{ t('paths') }}</th>
                <th>{{ t('tags') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="snapshot in snapshots" :key="snapshot.id" class="hover">
                <td>
                  <code class="text-xs bg-base-200 px-2 py-1 rounded">{{ snapshot.short_id || snapshot.id?.substring(0, 8) }}</code>
                </td>
                <td>
                  <div class="flex flex-col">
                    <span class="font-semibold">{{ formatSnapshotDate(snapshot.time) }}</span>
                    <span class="text-xs opacity-70">{{ formatRelativeDate(snapshot.time) }}</span>
                  </div>
                </td>
                <td>
                  <div class="badge badge-ghost badge-sm">{{ snapshot.hostname }}</div>
                </td>
                <td>
                  <div class="flex flex-wrap gap-1">
                    <span v-for="(path, idx) in snapshot.paths" :key="idx" class="badge badge-outline badge-xs">
                      {{ path }}
                    </span>
                  </div>
                </td>
                <td>
                  <div class="flex flex-wrap gap-1">
                    <span v-for="(tag, idx) in snapshot.tags" :key="idx" class="badge badge-primary badge-xs">
                      {{ tag }}
                    </span>
                    <span v-if="!snapshot.tags || snapshot.tags.length === 0" class="text-xs opacity-50">-</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="modal-action">
          <form method="dialog">
            <button class="btn btn-primary">{{ t('close') }}</button>
          </form>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button>close</button>
      </form>
    </dialog>

    <!-- File List Modal -->
    <dialog ref="fileModal" class="modal">
      <div class="modal-box max-w-5xl">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-2xl mb-4 flex items-center gap-3">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
          </svg>
          {{ t('filesFor') }} {{ selectedBackupName }}
        </h3>

        <div v-if="files.length === 0" class="text-center py-12 text-base-content/60">
          {{ t('noFiles') }}
        </div>

        <div v-else>
          <div class="mb-4 flex items-center gap-4">
            <div class="form-control flex-1">
              <input 
                v-model="fileFilter" 
                type="text" 
                :placeholder="t('filterFiles')" 
                class="input input-bordered w-full" />
            </div>
            <div class="stats shadow">
              <div class="stat py-2 px-4">
                <div class="stat-title text-xs">{{ t('totalFiles') }}</div>
                <div class="stat-value text-lg">{{ filteredFiles.length }}</div>
              </div>
            </div>
          </div>

          <div class="overflow-x-auto max-h-96">
            <table class="table table-zebra table-pin-rows table-xs">
              <thead>
                <tr>
                  <th>{{ t('path') }}</th>
                  <th class="text-right">{{ t('size') }}</th>
                  <th>{{ t('modified') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(file, idx) in filteredFiles" :key="idx" class="hover">
                  <td>
                    <div class="flex items-center gap-2">
                      <svg v-if="file.type === 'dir'" xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-warning" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                      </svg>
                      <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-info" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                      </svg>
                      <code class="text-xs">{{ file.path }}</code>
                    </div>
                  </td>
                  <td class="text-right">
                    <span class="badge badge-ghost badge-sm">{{ formatFileSize(file.size) }}</span>
                  </td>
                  <td>
                    <span class="text-xs">{{ formatSnapshotDate(file.mtime) }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="modal-action">
          <form method="dialog">
            <button class="btn btn-primary">{{ t('close') }}</button>
          </form>
        </div>
      </div>
      <form method="dialog" class="modal-backdrop">
        <button>close</button>
      </form>
    </dialog>

    <!-- Create Policy Modal -->
    <dialog ref="createPolicyModal" class="modal">
      <div class="modal-box max-w-4xl">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-lg mb-4">Create Backup Policy</h3>
        
        <div class="space-y-4">
          <!-- Name & Description -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Policy Name *</span>
              </label>
              <input 
                v-model="newPolicy.name" 
                type="text" 
                placeholder="e.g., daily-backup" 
                class="input input-bordered" 
                required
              />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Schedule (Cron) *</span>
              </label>
              <input 
                v-model="newPolicy.schedule" 
                type="text" 
                placeholder="0 2 * * * or every 6h" 
                class="input input-bordered" 
                required
              />
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Description</span>
            </label>
            <textarea 
              v-model="newPolicy.description" 
              class="textarea textarea-bordered" 
              placeholder="Optional description"
              rows="2"
            ></textarea>
          </div>

          <!-- Repository -->
          <div class="divider">Repository Configuration</div>
          
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Repository Type *</span>
              </label>
              <select v-model="newPolicy.repository_type" class="select select-bordered">
                <option value="s3">S3 (AWS/MinIO/Wasabi)</option>
                <option value="rest-server">REST Server</option>
                <option value="fs">Local Filesystem</option>
                <option value="sftp">SFTP</option>
              </select>
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Repository URL *</span>
              </label>
              <input 
                v-model="newPolicy.repository_url" 
                type="text" 
                placeholder="s3:s3.amazonaws.com/bucket" 
                class="input input-bordered" 
                required
              />
            </div>
          </div>

          <!-- Paths -->
          <div class="divider">Backup Paths</div>
          
          <div class="form-control">
            <label class="label">
              <span class="label-text">Include Paths (one per line) *</span>
            </label>
            <textarea 
              v-model="includePaths" 
              class="textarea textarea-bordered font-mono text-sm" 
              placeholder="/home&#10;/etc&#10;/var/www"
              rows="3"
            ></textarea>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Exclude Patterns (one per line)</span>
            </label>
            <textarea 
              v-model="excludePaths" 
              class="textarea textarea-bordered font-mono text-sm" 
              placeholder="*.log&#10;.cache&#10;node_modules"
              rows="3"
            ></textarea>
          </div>

          <!-- Retention -->
          <div class="divider">Retention Rules</div>
          
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Last</span>
              </label>
              <input 
                v-model.number="newPolicy.retention_rules.keep_last" 
                type="number" 
                min="0"
                class="input input-bordered" 
              />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Daily</span>
              </label>
              <input 
                v-model.number="newPolicy.retention_rules.keep_daily" 
                type="number" 
                min="0"
                class="input input-bordered" 
              />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Weekly</span>
              </label>
              <input 
                v-model.number="newPolicy.retention_rules.keep_weekly" 
                type="number" 
                min="0"
                class="input input-bordered" 
              />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Monthly</span>
              </label>
              <input 
                v-model.number="newPolicy.retention_rules.keep_monthly" 
                type="number" 
                min="0"
                class="input input-bordered" 
              />
            </div>
          </div>

          <!-- Options -->
          <div class="form-control">
            <label class="label cursor-pointer justify-start gap-2">
              <input v-model="newPolicy.enabled" type="checkbox" class="checkbox" />
              <span class="label-text">Enable policy immediately</span>
            </label>
          </div>
        </div>

        <div class="modal-action">
          <button @click="createPolicy" :disabled="creatingPolicy" class="btn btn-primary">
            <span v-if="creatingPolicy" class="loading loading-spinner loading-sm"></span>
            {{ creatingPolicy ? 'Creating...' : 'Create Policy' }}
          </button>
          <form method="dialog">
            <button class="btn">Cancel</button>
          </form>
        </div>
      </div>
    </dialog>

    <!-- Edit Policy Modal -->
    <dialog ref="editPolicyModal" class="modal">
      <div class="modal-box max-w-3xl" v-if="editingPolicy">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-2xl mb-4">Edit Policy</h3>
        
        <div class="space-y-4">
          <!-- Same form fields as create, but bound to editingPolicy -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Policy Name *</span>
              </label>
              <input 
                v-model="editingPolicy.name" 
                type="text" 
                placeholder="e.g., daily-backup" 
                class="input input-bordered" 
                required />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Schedule (Cron) *</span>
              </label>
              <input 
                v-model="editingPolicy.schedule" 
                type="text" 
                placeholder="0 2 * * *" 
                class="input input-bordered" 
                required />
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">Description</span>
            </label>
            <textarea 
              v-model="editingPolicy.description" 
              class="textarea textarea-bordered" 
              rows="2" 
              placeholder="Describe this backup policy"></textarea>
          </div>

          <div class="divider">Repository</div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Repository Type *</span>
              </label>
              <select v-model="editingPolicy.repository_type" class="select select-bordered">
                <option value="s3">S3 (AWS/MinIO/Wasabi)</option>
                <option value="rest-server">REST Server</option>
                <option value="fs">Local Filesystem</option>
                <option value="sftp">SFTP</option>
              </select>
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Repository URL *</span>
              </label>
              <input 
                v-model="editingPolicy.repository_url" 
                type="text" 
                placeholder="/backup/path or s3:bucket/prefix" 
                class="input input-bordered" 
                required />
            </div>
          </div>

          <div class="divider">Paths</div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Include Paths * (one per line)</span>
              </label>
              <textarea 
                v-model="editIncludePaths" 
                class="textarea textarea-bordered font-mono text-sm" 
                rows="4" 
                placeholder="/home/user&#10;/var/www"
                required></textarea>
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Exclude Patterns (one per line)</span>
              </label>
              <textarea 
                v-model="editExcludePaths" 
                class="textarea textarea-bordered font-mono text-sm" 
                rows="4" 
                placeholder="*.tmp&#10;*.log&#10;node_modules"></textarea>
            </div>
          </div>

          <div class="divider">Retention Rules</div>

          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Last</span>
              </label>
              <input v-model.number="editingPolicy.retention_rules.keep_last" type="number" min="0" class="input input-bordered" />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Daily</span>
              </label>
              <input v-model.number="editingPolicy.retention_rules.keep_daily" type="number" min="0" class="input input-bordered" />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Weekly</span>
              </label>
              <input v-model.number="editingPolicy.retention_rules.keep_weekly" type="number" min="0" class="input input-bordered" />
            </div>
            <div class="form-control">
              <label class="label">
                <span class="label-text">Keep Monthly</span>
              </label>
              <input v-model.number="editingPolicy.retention_rules.keep_monthly" type="number" min="0" class="input input-bordered" />
            </div>
          </div>

          <div class="form-control">
            <label class="label cursor-pointer justify-start gap-2">
              <input v-model="editingPolicy.enabled" type="checkbox" class="checkbox" />
              <span class="label-text">Policy enabled</span>
            </label>
          </div>
        </div>

        <div class="modal-action">
          <button @click="updatePolicy" :disabled="updatingPolicy" class="btn btn-primary">
            <span v-if="updatingPolicy" class="loading loading-spinner loading-sm"></span>
            {{ updatingPolicy ? 'Updating...' : 'Update Policy' }}
          </button>
          <form method="dialog">
            <button class="btn">Cancel</button>
          </form>
        </div>
      </div>
    </dialog>

    <!-- Delete Confirmation Modal -->
    <dialog ref="deleteConfirmModal" class="modal">
      <div class="modal-box">
        <h3 class="font-bold text-lg">Delete Policy</h3>
        <p class="py-4">Are you sure you want to delete the policy <strong>{{ policyToDelete?.name }}</strong>? This action cannot be undone.</p>
        <div class="modal-action">
          <button @click="deletePolicy" :disabled="deletingPolicy" class="btn btn-error">
            <span v-if="deletingPolicy" class="loading loading-spinner loading-sm"></span>
            {{ deletingPolicy ? 'Deleting...' : 'Delete' }}
          </button>
          <form method="dialog">
            <button class="btn">Cancel</button>
          </form>
        </div>
      </div>
    </dialog>

    <!-- Assign Policy Modal -->
    <dialog ref="assignModal" class="modal">
      <div class="modal-box">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-lg mb-4">Assign Policy to Agent</h3>
        <p class="mb-4">Select an agent to assign <strong>{{ policyToAssign?.name }}</strong></p>
        
        <div class="space-y-2">
          <div 
            v-for="agent in agents" 
            :key="agent.id"
            class="flex items-center justify-between p-3 border rounded hover:bg-base-200 cursor-pointer"
            @click="assignPolicyToAgent(agent.id)">
            <div>
              <div class="font-semibold">{{ agent.hostname }}</div>
              <div class="text-sm text-base-content/60">{{ agent.os }} / {{ agent.arch }}</div>
            </div>
            <div class="badge" :class="agent.status === 'online' ? 'badge-success' : 'badge-ghost'">
              {{ agent.status }}
            </div>
          </div>
          <div v-if="!agents || agents.length === 0" class="text-center py-8 text-base-content/60">
            No agents available
          </div>
        </div>
      </div>
    </dialog>

    <!-- Detach Policy Modal -->
    <dialog ref="detachModal" class="modal">
      <div class="modal-box">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-lg mb-4">Detach Policy from Agents</h3>
        <p class="mb-4">Select an agent to detach <strong>{{ policyToDetach?.name }}</strong></p>
        
        <div v-if="loadingAssignedAgents" class="flex justify-center py-8">
          <span class="loading loading-spinner loading-lg"></span>
        </div>
        
        <div v-else class="space-y-2">
          <div 
            v-for="agent in assignedAgents" 
            :key="agent.id"
            class="flex items-center justify-between p-3 border rounded hover:bg-base-200">
            <div>
              <div class="font-semibold">{{ agent.hostname }}</div>
              <div class="text-sm text-base-content/60">{{ agent.status }}</div>
            </div>
            <button 
              @click="detachPolicyFromAgent(agent.id)" 
              :disabled="detachingAgent === agent.id"
              class="btn btn-sm btn-error btn-outline">
              <span v-if="detachingAgent === agent.id" class="loading loading-spinner loading-xs"></span>
              {{ detachingAgent === agent.id ? 'Detaching...' : 'Detach' }}
            </button>
          </div>
          <div v-if="!assignedAgents || assignedAgents.length === 0" class="text-center py-8 text-base-content/60">
            No agents assigned to this policy
          </div>
        </div>
      </div>
    </dialog>

    <!-- Run Details Modal -->
    <dialog ref="runDetailsModal" class="modal">
      <div class="modal-box max-w-4xl" v-if="selectedRun">
        <form method="dialog">
          <button class="btn btn-sm btn-circle btn-ghost absolute right-2 top-2">✕</button>
        </form>
        <h3 class="font-bold text-2xl mb-4">Backup Run Details</h3>
        
        <div v-if="loadingRunDetails" class="flex justify-center py-12">
          <span class="loading loading-spinner loading-lg"></span>
        </div>
        
        <div v-else class="space-y-4">
          <!-- Run Info -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="label"><span class="label-text font-bold">Status</span></label>
              <div class="badge badge-lg" :class="getRunStatusClass(selectedRun.status)">
                {{ selectedRun.status }}
              </div>
            </div>
            <div>
              <label class="label"><span class="label-text font-bold">Duration</span></label>
              <p>{{ formatDuration(selectedRun.duration_seconds) }}</p>
            </div>
            <div>
              <label class="label"><span class="label-text font-bold">Agent</span></label>
              <p class="font-mono text-sm">{{ getAgentName(selectedRun.agent_id) }}</p>
            </div>
            <div>
              <label class="label"><span class="label-text font-bold">Policy</span></label>
              <p class="font-mono text-sm">{{ getPolicyName(selectedRun.policy_id) }}</p>
            </div>
            <div>
              <label class="label"><span class="label-text font-bold">Start Time</span></label>
              <p>{{ new Date(selectedRun.start_time).toLocaleString() }}</p>
            </div>
            <div v-if="selectedRun.end_time">
              <label class="label"><span class="label-text font-bold">End Time</span></label>
              <p>{{ new Date(selectedRun.end_time).toLocaleString() }}</p>
            </div>
          </div>
          
          <div v-if="selectedRun.snapshot_id">
            <label class="label"><span class="label-text font-bold">Snapshot ID</span></label>
            <code class="bg-base-200 px-3 py-2 rounded block font-mono text-sm">{{ selectedRun.snapshot_id }}</code>
          </div>
          
          <div v-if="selectedRun.error_message">
            <label class="label"><span class="label-text font-bold">Error Message</span></label>
            <div class="alert alert-error">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{{ selectedRun.error_message }}</span>
            </div>
          </div>
          
          <div v-if="selectedRun.log">
            <label class="label"><span class="label-text font-bold">Logs</span></label>
            <div class="mockup-code max-h-96 overflow-y-auto">
              <pre class="px-4"><code>{{ selectedRun.log }}</code></pre>
            </div>
          </div>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const backups = ref([])
const agents = ref([])
const policies = ref([])
const loading = ref(false)
const error = ref(null)
const unlocking = ref({})
const pruning = ref({})
const toggling = ref({})
const showDisabled = ref(true)
const snapshots = ref([])
const loadingSnapshots = ref(false)
const snapshotError = ref(null)
const selectedBackupName = ref('')
const snapshotModal = ref(null)
const files = ref([])
const fileFilter = ref('')
const fileModal = ref(null)
const createPolicyModal = ref(null)
const creatingPolicy = ref(false)
const includePaths = ref('')
const excludePaths = ref('')
const editPolicyModal = ref(null)
const updatingPolicy = ref(false)
const editingPolicy = ref(null)
const editIncludePaths = ref('')
const editExcludePaths = ref('')
const deleteConfirmModal = ref(null)
const deletingPolicy = ref(false)
const policyToDelete = ref(null)
const assignModal = ref(null)
const policyToAssign = ref(null)
const detachModal = ref(null)
const policyToDetach = ref(null)
const assignedAgents = ref([])
const loadingAssignedAgents = ref(false)
const detachingAgent = ref(null)
const backupRuns = ref([])
const loadingRuns = ref(false)
const runsFilter = ref({ status: '', agentId: '' })
const runDetailsModal = ref(null)
const selectedRun = ref(null)
const loadingRunDetails = ref(false)
const newPolicy = ref({
  name: '',
  description: '',
  schedule: '0 2 * * *',
  repository_type: 's3',
  repository_url: '',
  repository_config: {},
  include_paths: { paths: [] },
  exclude_paths: { patterns: [] },
  retention_rules: {
    keep_last: 7,
    keep_daily: 30,
    keep_weekly: 4,
    keep_monthly: 12
  },
  enabled: true
})

const API_BASE = '/api/v1'

// Get auth credentials from localStorage or prompt user
const getAuthHeaders = () => {
  const username = localStorage.getItem('auth_username')
  const password = localStorage.getItem('auth_password')
  
  if (username && password) {
    const credentials = btoa(`${username}:${password}`)
    return { 'Authorization': `Basic ${credentials}` }
  }
  return {}
}

const promptForAuth = () => {
  const username = prompt('Username:')
  const password = prompt('Password:')
  
  if (username && password) {
    localStorage.setItem('auth_username', username)
    localStorage.setItem('auth_password', password)
    return true
  }
  return false
}

const clearAuth = () => {
  localStorage.removeItem('auth_username')
  localStorage.removeItem('auth_password')
}

const healthyCount = computed(() => backups.value.filter(b => b.health && !isLocked(b)).length)
const issuesCount = computed(() => backups.value.filter(b => !b.health && !isLocked(b)).length)
const lockedCount = computed(() => backups.value.filter(b => isLocked(b)).length)

const filteredBackups = computed(() => {
  if (showDisabled.value) return backups.value
  return backups.value.filter(b => !b.disabled)
})

const filteredFiles = computed(() => {
  if (!fileFilter.value) return files.value
  const filter = fileFilter.value.toLowerCase()
  return files.value.filter(f => f.path?.toLowerCase().includes(filter))
})

const fetchBackups = async () => {
  // Only show loading spinner on first load, not on refresh
  const isFirstLoad = backups.value.length === 0
  if (isFirstLoad) {
    loading.value = true
  }
  error.value = null
  try {
    const response = await fetch(`${API_BASE}/status`, {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      loading.value = false // Stop loading spinner before prompting
      if (promptForAuth()) {
        return await fetchBackups() // Retry with new credentials
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    const newBackups = Array.isArray(data) ? data : [data]
    
    // Sort by name to maintain consistent order
    newBackups.sort((a, b) => a.name.localeCompare(b.name))
    
    // Update existing backups in place instead of replacing the array
    if (backups.value.length === 0) {
      // First load - just assign
      backups.value = newBackups
    } else {
      // Create a map of existing backups for quick lookup
      const existingMap = new Map(backups.value.map(b => [b.name, b]))
      const newMap = new Map(newBackups.map(b => [b.name, b]))
      
      // Update existing entries
      backups.value.forEach((backup, index) => {
        const updated = newMap.get(backup.name)
        if (updated) {
          // Update properties individually to maintain reactivity
          Object.assign(backups.value[index], updated)
        }
      })
      
      // Remove backups that no longer exist
      backups.value = backups.value.filter(b => newMap.has(b.name))
      
      // Add new backups at the end (sorted position will be handled by computed)
      newBackups.forEach(newBackup => {
        if (!existingMap.has(newBackup.name)) {
          backups.value.push(newBackup)
        }
      })
      
      // Re-sort to maintain order
      backups.value.sort((a, b) => a.name.localeCompare(b.name))
    }
  } catch (err) {
    error.value = err.message
  } finally {
    if (isFirstLoad) {
      loading.value = false
    }
  }
}

const unlockRepo = async (name) => {
  unlocking.value[name] = true
  try {
    const response = await fetch(`${API_BASE}/unlock/${name}`, {
      method: 'POST',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      unlocking.value[name] = false // Stop loading spinner before prompting
      if (promptForAuth()) {
        return await unlockRepo(name) // Retry with new credentials
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    // Wait a brief moment for the backend to re-check the repository
    await new Promise(resolve => setTimeout(resolve, 1500))
    await fetchBackups()
  } catch (err) {
    error.value = err.message
  } finally {
    unlocking.value[name] = false
  }
}

const pruneRepo = async (name) => {
  pruning.value[name] = true
  try {
    const response = await fetch(`${API_BASE}/prune/${name}`, {
      method: 'POST',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      pruning.value[name] = false
      if (promptForAuth()) {
        return await pruneRepo(name)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    // Wait a brief moment for the backend to re-check the repository
    await new Promise(resolve => setTimeout(resolve, 1500))
    await fetchBackups()
  } catch (err) {
    error.value = err.message
  } finally {
    pruning.value[name] = false
  }
}

const pruneAll = async () => {
  pruning.value['all'] = true
  try {
    const response = await fetch(`${API_BASE}/prune/all`, {
      method: 'POST',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      pruning.value['all'] = false
      if (promptForAuth()) {
        return await pruneAll()
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    // Wait a bit longer for all repositories to be re-checked
    await new Promise(resolve => setTimeout(resolve, 3000))
    await fetchBackups()
  } catch (err) {
    error.value = err.message
  } finally {
    pruning.value['all'] = false
  }
}

const toggleDisabled = async (name) => {
  toggling.value[name] = true
  try {
    const response = await fetch(`${API_BASE}/toggle/${name}`, {
      method: 'POST',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      toggling.value[name] = false
      if (promptForAuth()) {
        return await toggleDisabled(name)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    await fetchBackups()
  } catch (err) {
    error.value = err.message
  } finally {
    toggling.value[name] = false
  }
}

const getHealthBadge = (backup) => {
  if (isLocked(backup)) return 'badge-warning'
  return backup.health ? 'badge-success' : 'badge-error'
}

const getHealthText = (backup) => {
  if (isLocked(backup)) return t('locked')
  return backup.health ? t('healthy') : t('unhealthy')
}

const isLocked = (backup) => {
  return backup.statusMessage?.includes('locked')
}

const formatDate = (dateString) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  const now = new Date()
  const diff = now - date
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(hours / 24)
  
  if (days > 0) return `${days}d ${hours % 24}h ago`
  if (hours > 0) return `${hours}h ago`
  return 'Just now'
}

const formatSnapshotDate = (dateString) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  return date.toLocaleString()
}

const formatRelativeDate = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  const now = new Date()
  const diff = now - date
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(hours / 24)
  const months = Math.floor(days / 30)
  
  if (months > 0) return `${months} month${months > 1 ? 's' : ''} ago`
  if (days > 0) return `${days} day${days > 1 ? 's' : ''} ago`
  if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`
  return 'Just now'
}

const showSnapshots = async (name) => {
  selectedBackupName.value = name
  loadingSnapshots.value = true
  snapshotError.value = null
  snapshots.value = []
  
  snapshotModal.value?.showModal()
  
  try {
    const response = await fetch(`${API_BASE}/snapshots/${name}`, {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      snapshotModal.value?.close()
      if (promptForAuth()) {
        return await showSnapshots(name)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    snapshots.value = Array.isArray(data) ? data : []
  } catch (err) {
    snapshotError.value = err.message
  } finally {
    loadingSnapshots.value = false
  }
}

const showFiles = async (name, snapshotID) => {
  if (!snapshotID) {
    error.value = 'No snapshot available'
    return
  }
  
  selectedBackupName.value = name
  files.value = []
  fileFilter.value = ''
  fileModal.value?.showModal()
  
  try {
    const response = await fetch(`${API_BASE}/snapshot/${snapshotID}`, {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      fileModal.value?.close()
      if (promptForAuth()) {
        return await showFiles(name, snapshotID)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    
    const data = await response.json()
    files.value = Array.isArray(data) ? data : []
  } catch (err) {
    error.value = err.message
  }
}

const formatFileSize = (bytes) => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
}

const formatTime = (timestamp) => {
  if (!timestamp) return 'Never'
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now - date
  const diffMins = Math.floor(diffMs / 60000)
  
  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 7) return `${diffDays}d ago`
  
  return date.toLocaleDateString()
}

const toggleTheme = (e) => {
  document.documentElement.setAttribute('data-theme', e.target.checked ? 'dark' : 'light')
}

const fetchAgents = async () => {
  try {
    const response = await fetch('/agents', {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      if (promptForAuth()) {
        return await fetchAgents()
      }
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    agents.value = data.agents || []
  } catch (err) {
    console.error('Failed to fetch agents:', err)
  }
}

const agentStatusClass = (agent) => {
  if (agent.status === 'online') return 'badge-success'
  if (agent.status === 'offline') return 'badge-error'
  return 'badge-warning'
}

const formatUptime = (seconds) => {
  if (!seconds) return 'N/A'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

const fetchPolicies = async () => {
  try {
    const response = await fetch('/policies', {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      if (promptForAuth()) {
        return await fetchPolicies()
      }
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    policies.value = Array.isArray(data) ? data : []
    console.log('Fetched policies:', policies.value)
  } catch (err) {
    console.error('Failed to fetch policies:', err)
  }
}

const getPolicyStatusClass = (policy) => {
  return policy.enabled ? 'badge-success' : 'badge-ghost'
}

const formatCron = (schedule) => {
  // Simple cron formatter - could be enhanced
  if (schedule === '0 2 * * *') return 'Daily at 2 AM'
  if (schedule === '0 */6 * * *') return 'Every 6 hours'
  if (schedule.startsWith('every ')) return schedule
  return schedule
}

const openCreatePolicyModal = () => {
  // Reset form
  newPolicy.value = {
    name: '',
    description: '',
    schedule: '0 2 * * *',
    repository_type: 's3',
    repository_url: '',
    repository_config: {},
    include_paths: { paths: [] },
    exclude_paths: { patterns: [] },
    retention_rules: {
      keep_last: 7,
      keep_daily: 30,
      keep_weekly: 4,
      keep_monthly: 12
    },
    enabled: true
  }
  includePaths.value = ''
  excludePaths.value = ''
  createPolicyModal.value?.showModal()
}

const createPolicy = async () => {
  creatingPolicy.value = true
  
  try {
    // Parse paths from textarea
    const includePathsArray = includePaths.value
      .split('\n')
      .map(p => p.trim())
      .filter(p => p.length > 0)
    
    const excludePathsArray = excludePaths.value
      .split('\n')
      .map(p => p.trim())
      .filter(p => p.length > 0)
    
    if (includePathsArray.length === 0) {
      error.value = 'At least one include path is required'
      creatingPolicy.value = false
      return
    }
    
    // Prepare policy data with camelCase field names as expected by API
    const repository = {
      type: newPolicy.value.repository_type,
      ...newPolicy.value.repository_config
    }
    
    // Add type-specific fields
    if (newPolicy.value.repository_type === 'fs') {
      repository.path = newPolicy.value.repository_url
    } else {
      repository.url = newPolicy.value.repository_url
    }
    
    const policyData = {
      name: newPolicy.value.name,
      description: newPolicy.value.description || null,
      schedule: newPolicy.value.schedule,
      includePaths: includePathsArray,
      excludePaths: excludePathsArray.length > 0 ? excludePathsArray : undefined,
      repository,
      retentionRules: newPolicy.value.retention_rules,
      enabled: newPolicy.value.enabled
    }
    
    console.log('Creating policy:', JSON.stringify(policyData, null, 2))
    
    const response = await fetch('/policies', {
      method: 'POST',
      headers: {
        ...getAuthHeaders(),
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(policyData)
    })
    
    if (response.status === 401) {
      clearAuth()
      creatingPolicy.value = false
      if (promptForAuth()) {
        return await createPolicy()
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }
    
    // Success - close modal and refresh
    createPolicyModal.value?.close()
    await fetchPolicies()
    error.value = null
  } catch (err) {
    error.value = err.message
  } finally {
    creatingPolicy.value = false
  }
}

const openEditPolicyModal = (policy) => {
  // Deep copy the policy to avoid modifying the original
  editingPolicy.value = {
    id: policy.id,
    name: policy.name,
    description: policy.description || '',
    schedule: policy.schedule,
    repository_type: policy.repository?.type || 's3',
    repository_url: policy.repository?.url || policy.repository?.path || '',
    repository_config: {},
    retention_rules: {
      keep_last: policy.retentionRules?.keep_last || 7,
      keep_daily: policy.retentionRules?.keep_daily || 30,
      keep_weekly: policy.retentionRules?.keep_weekly || 4,
      keep_monthly: policy.retentionRules?.keep_monthly || 12
    },
    enabled: policy.enabled
  }
  
  // Populate path textareas
  editIncludePaths.value = (policy.includePaths || []).join('\n')
  editExcludePaths.value = (policy.excludePaths || []).join('\n')
  
  editPolicyModal.value?.showModal()
}

const updatePolicy = async () => {
  updatingPolicy.value = true
  
  try {
    // Parse paths from textarea
    const includePathsArray = editIncludePaths.value
      .split('\n')
      .map(p => p.trim())
      .filter(p => p.length > 0)
    
    const excludePathsArray = editExcludePaths.value
      .split('\n')
      .map(p => p.trim())
      .filter(p => p.length > 0)
    
    if (includePathsArray.length === 0) {
      error.value = 'At least one include path is required'
      updatingPolicy.value = false
      return
    }
    
    // Prepare repository object
    const repository = {
      type: editingPolicy.value.repository_type,
      ...editingPolicy.value.repository_config
    }
    
    if (editingPolicy.value.repository_type === 'fs') {
      repository.path = editingPolicy.value.repository_url
    } else {
      repository.url = editingPolicy.value.repository_url
    }
    
    const policyData = {
      name: editingPolicy.value.name,
      description: editingPolicy.value.description || null,
      schedule: editingPolicy.value.schedule,
      includePaths: includePathsArray,
      excludePaths: excludePathsArray.length > 0 ? excludePathsArray : undefined,
      repository,
      retentionRules: editingPolicy.value.retention_rules,
      enabled: editingPolicy.value.enabled
    }
    
    const response = await fetch(`/policies/${editingPolicy.value.id}`, {
      method: 'PUT',
      headers: {
        ...getAuthHeaders(),
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(policyData)
    })
    
    if (response.status === 401) {
      clearAuth()
      updatingPolicy.value = false
      if (promptForAuth()) {
        return await updatePolicy()
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }
    
    // Success - close modal and refresh
    editPolicyModal.value?.close()
    await fetchPolicies()
    error.value = null
  } catch (err) {
    error.value = err.message
  } finally {
    updatingPolicy.value = false
  }
}

const confirmDeletePolicy = (policy) => {
  policyToDelete.value = policy
  deleteConfirmModal.value?.showModal()
}

const deletePolicy = async () => {
  if (!policyToDelete.value) return
  
  deletingPolicy.value = true
  
  try {
    const response = await fetch(`/policies/${policyToDelete.value.id}`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      deletingPolicy.value = false
      if (promptForAuth()) {
        return await deletePolicy()
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }
    
    // Success - close modal and refresh
    deleteConfirmModal.value?.close()
    await fetchPolicies()
    policyToDelete.value = null
    error.value = null
  } catch (err) {
    error.value = err.message
  } finally {
    deletingPolicy.value = false
  }
}

const openAssignModal = (policy) => {
  policyToAssign.value = policy
  assignModal.value?.showModal()
}

const assignPolicyToAgent = async (agentId) => {
  if (!policyToAssign.value) return
  
  try {
    const response = await fetch(`/agents/${agentId}/policies/${policyToAssign.value.id}`, {
      method: 'POST',
      headers: {
        ...getAuthHeaders(),
        'Content-Type': 'application/json'
      }
    })
    
    if (response.status === 401) {
      clearAuth()
      if (promptForAuth()) {
        return await assignPolicyToAgent(agentId)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }
    
    // Success - close modal and show success message
    assignModal.value?.close()
    policyToAssign.value = null
    error.value = null
    alert('Policy assigned successfully!')
  } catch (err) {
    error.value = err.message
    alert(`Failed to assign policy: ${err.message}`)
  }
}

const openDetachModal = async (policy) => {
  policyToDetach.value = policy
  loadingAssignedAgents.value = true
  detachModal.value?.showModal()
  
  try {
    const response = await fetch(`/policies/${policy.id}/agents`, {
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      if (promptForAuth()) {
        return await openDetachModal(policy)
      }
      return
    }
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    assignedAgents.value = data.agents || []
  } catch (err) {
    console.error('Failed to fetch assigned agents:', err)
    assignedAgents.value = []
  } finally {
    loadingAssignedAgents.value = false
  }
}

const detachPolicyFromAgent = async (agentId) => {
  if (!policyToDetach.value) return
  
  detachingAgent.value = agentId
  
  try {
    const response = await fetch(`/agents/${agentId}/policies/${policyToDetach.value.id}`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    })
    
    if (response.status === 401) {
      clearAuth()
      detachingAgent.value = null
      if (promptForAuth()) {
        return await detachPolicyFromAgent(agentId)
      }
      error.value = 'Authentication cancelled'
      return
    }
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }
    
    // Success - remove from list
    assignedAgents.value = assignedAgents.value.filter(a => a.id !== agentId)
    error.value = null
    alert('Policy detached successfully!')
  } catch (err) {
    error.value = err.message
    alert(`Failed to detach policy: ${err.message}`)
  } finally {
    detachingAgent.value = null
  }
}

const fetchBackupRuns = async () => {
  if (!runsFilter.value.agentId && agents.value.length === 0) return
  
  loadingRuns.value = true
  try {
    // Fetch runs for all agents or filtered agent
    const agentIds = runsFilter.value.agentId ? [runsFilter.value.agentId] : agents.value.map(a => a.id)
    
    const allRuns = []
    for (const agentId of agentIds) {
      const url = new URL(`/agents/${agentId}/backup-runs`, window.location.origin)
      if (runsFilter.value.status) {
        url.searchParams.set('status', runsFilter.value.status)
      }
      url.searchParams.set('limit', '50')
      
      const response = await fetch(url.pathname + url.search, {
        headers: getAuthHeaders()
      })
      
      if (response.ok) {
        const data = await response.json()
        allRuns.push(...(data.runs || []))
      }
    }
    
    // Sort by start time descending
    allRuns.sort((a, b) => new Date(b.start_time) - new Date(a.start_time))
    backupRuns.value = allRuns.slice(0, 50)
  } catch (err) {
    console.error('Failed to fetch backup runs:', err)
    backupRuns.value = []
  } finally {
    loadingRuns.value = false
  }
}

const getRunStatusClass = (status) => {
  if (status === 'success') return 'badge-success'
  if (status === 'failed') return 'badge-error'
  if (status === 'running') return 'badge-warning'
  return 'badge-ghost'
}

const getAgentName = (agentId) => {
  const agent = agents.value.find(a => a.id === agentId)
  return agent ? agent.hostname : agentId.substring(0, 8)
}

const getPolicyName = (policyId) => {
  const policy = policies.value.find(p => p.id === policyId)
  return policy ? policy.name : policyId.substring(0, 8)
}

const formatDuration = (seconds) => {
  if (!seconds) return '-'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  if (mins > 0) return `${mins}m ${secs}s`
  return `${secs}s`
}

const viewRunDetails = async (run) => {
  selectedRun.value = run
  loadingRunDetails.value = true
  runDetailsModal.value?.showModal()
  
  try {
    const response = await fetch(`/agents/${run.agent_id}/backup-runs/${run.id}`, {
      headers: getAuthHeaders()
    })
    
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = await response.json()
    selectedRun.value = data
  } catch (err) {
    console.error('Failed to fetch run details:', err)
  } finally {
    loadingRunDetails.value = false
  }
}

// Watch for filter changes
watch([() => runsFilter.value.status, () => runsFilter.value.agentId], () => {
  fetchBackupRuns()
})

// Watch for agents being loaded
watch(agents, (newAgents) => {
  if (newAgents.length > 0 && backupRuns.value.length === 0) {
    fetchBackupRuns()
  }
})

onMounted(() => {
  fetchBackups()
  fetchAgents()
  fetchPolicies()
  setInterval(fetchBackups, 30000)
  setInterval(fetchAgents, 30000)
  setInterval(fetchPolicies, 30000)
  setInterval(fetchBackupRuns, 30000)
})
</script>

<style scoped>
/* Transition for backup cards */
.backup-list-enter-active {
  transition: all 0.5s ease;
}

.backup-list-leave-active {
  transition: all 0.5s ease;
  position: absolute;
  width: calc(100% - 1.5rem);
}

.backup-list-enter-from {
  opacity: 0;
  transform: translateY(-30px) scale(0.95);
}

.backup-list-leave-to {
  opacity: 0;
  transform: translateY(30px) scale(0.95);
}

.backup-list-move {
  transition: transform 0.5s ease;
}
</style>
