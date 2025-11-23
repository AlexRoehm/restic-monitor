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
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const backups = ref([])
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

const toggleTheme = (e) => {
  document.documentElement.setAttribute('data-theme', e.target.checked ? 'dark' : 'light')
}

onMounted(() => {
  fetchBackups()
  setInterval(fetchBackups, 30000)
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
