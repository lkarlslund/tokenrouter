function adminApp() {
  return {
    activeTab: 'status',
    stats: {},
    appVersion: '',
    providers: [],
    conversationThreads: [],
    conversationRecords: [],
    selectedConversationKey: '',
    conversationsSettings: {enabled:true, max_items:5000, max_age_days:30},
    conversationsSaveInProgress: false,
    showConversationsSettingsModal: false,
    showConversationDetailModal: false,
    showConversationRawModal: false,
    showLogSettingsModal: false,
    conversationsSearch: '',
    conversationsListHtml: '',
    conversationsPagerHtml: '',
    conversationDetailHtml: '<div class="small text-body-secondary">Select a conversation to inspect chat flow.</div>',
    conversationRawHtml: '<div class="small text-body-secondary">Select a conversation message to inspect raw payloads.</div>',
    conversationTitle: '',
    conversationDetailMessageCount: 0,
    conversationThinkVisible: {},
    conversationSystemVisible: {},
    conversationIncludeInternal: false,
    conversationsCaptureEnabled: true,
    conversationsDebounceTimer: null,
    logEntries: [],
    logEntriesHtml: '',
    logPagerHtml: '',
    logEntriesShownCount: 0,
    logEntriesTotalCount: 0,
    logLevelFilter: 'trace',
    logSearch: '',
    logDebounceTimer: null,
    logMaxLines: 5000,
    logSaveInProgress: false,
    popularProviders: [],
    modelsCatalog: [],
    allowLocalhostNoAuth: false,
    allowHostDockerInternalNoAuth: false,
    allowLocalhostNoAuthEffective: false,
    allowHostDockerInternalNoAuthEffective: false,
    autoEnablePublicFreeModels: false,
    autoDetectLocalServers: true,
    autoRemoveExpiredTokens: true,
    autoRemoveEmptyQuotaTokens: false,
    securitySaveInProgress: false,
    networkBindMode: 'localhost',
    networkBindSelection: 'localhost',
    networkCustomBind: '',
    networkPort: 7050,
    networkHTTPMode: 'enabled',
    networkActiveAddrs: [],
    networkDetectedAddrs: [],
    networkPending: null,
    networkApplyInProgress: false,
    tlsBindMode: 'localhost',
    tlsBindSelection: 'localhost',
    tlsCustomBind: '',
    tlsPort: 443,
    tlsSettings: {enabled:false, mode:'letsencrypt', domain:'', email:'', cache_dir:'', cert_pem:'', key_pem:'', cert_configured:false, key_configured:false},
    tlsSaveInProgress: false,
    tlsActionInProgress: false,
    statsSummaryHtml: '',
    quotaSummaryHtml: '',
    lastGoodQuotaByProvider: {},
    quotaSearch: '',
    statsRangeHours: '8',
    performanceRangeHours: '8',
    statusUpdateSpeed: 'realtime',
    usageChartGroupBy: 'model',
    usageChart: null,
    providersTableHtml: '',
    accessTokensTableHtml: '',
    accessTokensPagerHtml: '',
    modelsTableHtml: '',
    performanceTableHtml: '',
    benchmarkHtml: '<div class="small text-body-secondary">No benchmark running.</div>',
    modelsFreshnessHtml: '',
    modalStatusHtml: '',
    deviceCodeFetchInProgress: false,
    deviceTokenPollInProgress: false,
    oauthLoginInProgress: false,
    oauthPollState: '',
    showAddProviderModal: false,
    showProvidersSettingsModal: false,
    showAccessSettingsModal: false,
    addProviderStep: 'pick_provider',
    selectedPreset: '',
    presetInfoHtml: '',
    overrideProviderSettings: false,
    authMode: 'api_key',
    providerSearch: '',
    modelsSearch: '',
    performanceSearch: '',
    modelsSortBy: 'provider',
    modelsSortAsc: true,
    performanceSortBy: 'provider',
    performanceSortAsc: true,
    modelsFreeOnly: false,
    providersPage: 1,
    providersPageSize: 25,
    accessPage: 1,
    accessPageSize: 25,
    modelsPage: 1,
    modelsPageSize: 25,
    performancePage: 1,
    performancePageSize: 25,
    conversationsPage: 1,
    conversationsPageSize: 25,
    logsPage: 1,
    logsPageSize: 25,
    modelsRefreshInProgress: false,
    modelsInitialized: false,
    modelsInitialLoadInProgress: false,
    providersRefreshInProgress: false,
    showBenchmarkModal: false,
    benchmarkRunning: false,
    benchmarkRun: null,
    benchmarkDraft: {chats_per_model: 3, messages_per_chat: 3, stop_after_tokens: 0},
    quotaRefreshInProgress: false,
    statsLoading: false,
    statsRenderToken: 0,
    editingProviderName: '',
    runtimeInstanceID: '',
    reloadingForRuntimeUpdate: false,
    reloadIdleThresholdMs: 30000,
    lastUserInteractionAt: 0,
    userActivityTrackingInstalled: false,
    pendingReloadTimer: null,
    pendingReloadKind: '',
    realtimePausedForReload: false,
    runtimePatchInProgress: false,
    runtimeScriptFingerprints: null,
    wsFailureCount: 0,
    themeMode: 'auto',
    tabBarCanScrollLeft: false,
    tabBarCanScrollRight: false,
    tabContentLoaded: {},
    tabContentLoading: {},
    activeTabCacheKey: 'opp_admin_active_tab_v1',
    modelsCacheKey: 'opp_models_catalog_cache_v1',
    modelsFreeOnlyCacheKey: 'opp_models_free_only_v1',
    themeCacheKey: 'opp_theme_mode_v1',
    usageChartGroupCacheKey: 'opp_usage_chart_group_v1',
    statsRangeCacheKey: 'opp_stats_range_hours_v1',
    performanceRangeCacheKey: 'opp_performance_range_hours_v1',
    statusUpdateSpeedCacheKey: 'opp_status_update_speed_v1',
    logLevelFilterCacheKey: 'opp_log_level_filter_v1',
    accessTokens: [],
    requiresInitialTokenSetup: false,
    initialSetupDialogDismissed: false,
    initialSetupDialogDismissCacheKey: 'opp_initial_setup_dialog_dismissed_v1',
    showAddAccessTokenModal: false,
    showConfirmModal: false,
    confirmModalTitle: 'Confirm Action',
    confirmModalMessage: '',
    confirmModalConfirmLabel: 'Confirm',
    confirmModalBusy: false,
    confirmModalAction: null,
    accessTokenDraft: {id:'', name:'', key:'', role:'inferrer', expiry_preset:'never', expires_at:'', quota_enabled:false, quota_requests_limit:'', quota_requests_interval_seconds:'0', quota_tokens_limit:'', quota_tokens_interval_seconds:'0', disable_localhost_no_auth:false},
    ws: null,
    wsReconnectTimer: null,
    wsBackoffMs: 1000,
    fallbackPoller: null,
    intervalPoller: null,
    draft: {name:'',provider_type:'',base_url:'',api_key:'',auth_token:'',refresh_token:'',token_expires_at:'',account_id:'',device_auth_url:'',device_code:'',device_auth_id:'',device_code_url:'',device_token_url:'',device_client_id:'',device_scope:'',device_grant_type:'',oauth_authorize_url:'',oauth_token_url:'',oauth_client_id:'',oauth_client_secret:'',oauth_scope:'',enabled:true,timeout_seconds:60},
    oauthAdvanced: false,
    init() {
      this.lastUserInteractionAt = Date.now();
      this.installUserActivityTracking();
      this.restoreLocalStorageFromHandoff();
      this.captureRuntimeScriptFingerprints();
      window.__adminSortModels = (col) => this.sortModelsBy(col);
      window.__adminSortPerformance = (col) => this.sortPerformanceBy(col);
      window.__adminProvidersFirstPage = () => this.setProvidersPage(1);
      window.__adminProvidersPrevPage = () => this.setProvidersPage(this.providersPage - 1);
      window.__adminProvidersNextPage = () => this.setProvidersPage(this.providersPage + 1);
      window.__adminProvidersLastPage = () => this.setProvidersPage(Number.MAX_SAFE_INTEGER);
      window.__adminSetProvidersPage = (v) => this.setProvidersPage(v);
      window.__adminSetProvidersPageSize = (v) => this.setProvidersPageSize(v);
      window.__adminAccessFirstPage = () => this.setAccessPage(1);
      window.__adminAccessPrevPage = () => this.setAccessPage(this.accessPage - 1);
      window.__adminAccessNextPage = () => this.setAccessPage(this.accessPage + 1);
      window.__adminAccessLastPage = () => this.setAccessPage(Number.MAX_SAFE_INTEGER);
      window.__adminSetAccessPage = (v) => this.setAccessPage(v);
      window.__adminSetAccessPageSize = (v) => this.setAccessPageSize(v);
      window.__adminModelsFirstPage = () => this.setModelsPage(1);
      window.__adminModelsPrevPage = () => this.setModelsPage(this.modelsPage - 1);
      window.__adminModelsNextPage = () => this.setModelsPage(this.modelsPage + 1);
      window.__adminModelsLastPage = () => this.setModelsPage(Number.MAX_SAFE_INTEGER);
      window.__adminSetModelsPage = (v) => this.setModelsPage(v);
      window.__adminSetModelsPageSize = (v) => this.setModelsPageSize(v);
      window.__adminPerformanceFirstPage = () => this.setPerformancePage(1);
      window.__adminPerformancePrevPage = () => this.setPerformancePage(this.performancePage - 1);
      window.__adminPerformanceNextPage = () => this.setPerformancePage(this.performancePage + 1);
      window.__adminPerformanceLastPage = () => this.setPerformancePage(Number.MAX_SAFE_INTEGER);
      window.__adminSetPerformancePage = (v) => this.setPerformancePage(v);
      window.__adminSetPerformancePageSize = (v) => this.setPerformancePageSize(v);
      window.__adminConversationsFirstPage = () => this.setConversationsPage(1);
      window.__adminConversationsPrevPage = () => this.setConversationsPage(this.conversationsPage - 1);
      window.__adminConversationsNextPage = () => this.setConversationsPage(this.conversationsPage + 1);
      window.__adminConversationsLastPage = () => this.setConversationsPage(Number.MAX_SAFE_INTEGER);
      window.__adminSetConversationsPage = (v) => this.setConversationsPage(v);
      window.__adminSetConversationsPageSize = (v) => this.setConversationsPageSize(v);
      window.__adminLogsFirstPage = () => this.setLogsPage(1);
      window.__adminLogsPrevPage = () => this.setLogsPage(this.logsPage - 1);
      window.__adminLogsNextPage = () => this.setLogsPage(this.logsPage + 1);
      window.__adminLogsLastPage = () => this.setLogsPage(Number.MAX_SAFE_INTEGER);
      window.__adminSetLogsPage = (v) => this.setLogsPage(v);
      window.__adminSetLogsPageSize = (v) => this.setLogsPageSize(v);
      window.__adminSetUsageChartGroup = (v) => this.setUsageChartGroup(v);
      this.hydrateModelsFromCache();
      this.restoreModelsFreeOnly();
      this.restoreThemeMode();
      this.restoreUsageChartGroup();
      this.restoreStatsRangeHours();
      this.restorePerformanceRangeHours();
      this.restoreStatusUpdateSpeed();
      this.restoreLogLevelFilter();
      this.restoreActiveTab();
      this.restoreInitialSetupDialogDismissed();
      this.loadStats(false);
      this.loadProviders();
      this.loadAccessTokens();
      this.loadPopularProviders();
      this.loadSecuritySettings();
      this.refreshNetworkTabFromConfig();
      this.loadVersion();
      this.loadBenchmarkStatus();
      this.startRealtimeUpdates();
      if (window.matchMedia) {
        const media = window.matchMedia('(prefers-color-scheme: dark)');
        if (media && media.addEventListener) {
          media.addEventListener('change', () => {
            if (this.themeMode === 'auto') this.applyThemeMode();
          });
        }
      }
      this.$nextTick(() => this.initTabScroll());
      window.addEventListener('load', () => this.initTabScroll(), {once: true});
      window.addEventListener('resize', () => this.updateTabScrollButtons(), {passive: true});
    },
    installUserActivityTracking() {
      if (this.userActivityTrackingInstalled) return;
      this.userActivityTrackingInstalled = true;
      const markActive = () => this.noteUserInteraction();
      const opts = {capture: true, passive: true};
      window.addEventListener('pointerdown', markActive, opts);
      window.addEventListener('keydown', markActive, opts);
      window.addEventListener('input', markActive, opts);
      window.addEventListener('wheel', markActive, opts);
      window.addEventListener('touchstart', markActive, opts);
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') markActive();
      });
    },
    noteUserInteraction() {
      this.lastUserInteractionAt = Date.now();
      if (this.pendingReloadKind) this.schedulePendingReloadCheck(this.reloadIdleThresholdMs);
    },
    idleForMs() {
      return Math.max(0, Date.now() - Number(this.lastUserInteractionAt || 0));
    },
    clearPendingReload() {
      if (this.pendingReloadTimer) {
        clearTimeout(this.pendingReloadTimer);
        this.pendingReloadTimer = null;
      }
      this.pendingReloadKind = '';
    },
    schedulePendingReloadCheck(delayMs) {
      if (this.pendingReloadTimer) clearTimeout(this.pendingReloadTimer);
      const delay = Math.max(25, Math.trunc(Number(delayMs || 0)));
      this.pendingReloadTimer = setTimeout(() => {
        this.pendingReloadTimer = null;
        this.tryRunPendingReload();
      }, delay);
    },
    tryRunPendingReload() {
      if (!this.pendingReloadKind) return;
      const idle = this.idleForMs();
      if (idle >= this.reloadIdleThresholdMs) {
        const kind = this.pendingReloadKind;
        this.clearPendingReload();
        this.performReload(kind);
        return;
      }
      this.schedulePendingReloadCheck(this.reloadIdleThresholdMs - idle);
    },
    reinitAlpineTreeWhenReady(rootEl) {
      const root = rootEl || (document.body && document.body.firstElementChild);
      if (!root) return;
      const run = () => {
        if (!(window.Alpine && typeof window.Alpine.initTree === 'function')) return;
        try {
          if (typeof window.Alpine.mutateDom === 'function') {
            window.Alpine.mutateDom(() => window.Alpine.initTree(root));
          } else {
            window.Alpine.initTree(root);
          }
        } catch (_) {}
      };
      if (document.readyState === 'complete') {
        if (typeof window.requestAnimationFrame === 'function') {
          window.requestAnimationFrame(() => run());
        } else {
          setTimeout(run, 0);
        }
        return;
      }
      window.addEventListener('load', run, {once: true});
    },
    requestUIReload(kind) {
      const reloadKind = String(kind || 'default').trim() || 'default';
      if (reloadKind === 'runtime_update') {
        this.realtimePausedForReload = true;
        if (typeof this.stopRealtimeUpdates === 'function') {
          try { this.stopRealtimeUpdates(); } catch (_) {}
        }
      }
      if (this.pendingReloadKind !== 'runtime_update' || reloadKind === 'runtime_update') {
        this.pendingReloadKind = reloadKind;
      }
      const idle = this.idleForMs();
      if (idle >= this.reloadIdleThresholdMs) {
        this.clearPendingReload();
        this.performReload(reloadKind);
        return;
      }
      this.schedulePendingReloadCheck(this.reloadIdleThresholdMs - idle);
    },
    async handleRuntimeUpdate() {
      if (this.runtimePatchInProgress) return;
      this.runtimePatchInProgress = true;
      try {
        const patched = await this.tryHotPatchRuntimeHTMLOnly();
        if (patched) {
          this.reloadingForRuntimeUpdate = false;
          return;
        }
        this.requestUIReload('runtime_update');
      } finally {
        this.runtimePatchInProgress = false;
      }
    },
    runtimeManagedScriptURLsFromDoc(doc) {
      const d = doc || document;
      const out = [];
      const nodes = d.querySelectorAll('script[src]');
      for (let i = 0; i < nodes.length; i++) {
        const raw = String(nodes[i].getAttribute('src') || '').trim();
        if (!raw) continue;
        let u = '';
        try {
          u = new URL(raw, window.location.href).toString();
        } catch (_) {
          continue;
        }
        if (!u.includes('/admin/static/') || !u.endsWith('.js')) continue;
        if (!out.includes(u)) out.push(u);
      }
      return out.sort();
    },
    hashTextFNV1a(input) {
      const s = String(input || '');
      let h = 2166136261;
      for (let i = 0; i < s.length; i++) {
        h ^= s.charCodeAt(i);
        h += (h << 1) + (h << 4) + (h << 7) + (h << 8) + (h << 24);
      }
      return (h >>> 0).toString(16).padStart(8, '0');
    },
    async fetchScriptFingerprintMap(urls) {
      const list = Array.isArray(urls) ? urls.slice() : [];
      const out = {};
      await Promise.all(list.map(async (u) => {
        try {
          const r = await fetch(u + (u.includes('?') ? '&' : '?') + '_rt_probe=' + String(Date.now()), {
            method: 'GET',
            cache: 'no-store',
            credentials: 'same-origin'
          });
          if (!r.ok) {
            out[u] = 'status:' + String(r.status || 0);
            return;
          }
          const txt = await r.text().catch(() => '');
          out[u] = this.hashTextFNV1a(txt);
        } catch (_) {
          out[u] = 'error';
        }
      }));
      return out;
    },
    async captureRuntimeScriptFingerprints() {
      const urls = this.runtimeManagedScriptURLsFromDoc(document);
      if (!urls.length) {
        this.runtimeScriptFingerprints = {};
        return;
      }
      this.runtimeScriptFingerprints = await this.fetchScriptFingerprintMap(urls);
    },
    runtimeScriptMapsEqual(a, b) {
      const left = a || {};
      const right = b || {};
      const keysA = Object.keys(left).sort();
      const keysB = Object.keys(right).sort();
      if (keysA.length !== keysB.length) return false;
      for (let i = 0; i < keysA.length; i++) {
        if (keysA[i] !== keysB[i]) return false;
        if (String(left[keysA[i]] || '') !== String(right[keysA[i]] || '')) return false;
      }
      return true;
    },
    syncHeadFromDocument(nextDoc) {
      try {
        const nextTitle = String((nextDoc && nextDoc.title) || '').trim();
        if (nextTitle) document.title = nextTitle;
      } catch (_) {}
      try {
        const curStyles = Array.from(document.head.querySelectorAll('style'));
        const nextStyles = Array.from((nextDoc && nextDoc.head ? nextDoc.head : document.head).querySelectorAll('style'));
        const max = Math.max(curStyles.length, nextStyles.length);
        for (let i = 0; i < max; i++) {
          const cur = curStyles[i] || null;
          const nxt = nextStyles[i] || null;
          if (cur && !nxt) {
            cur.remove();
            continue;
          }
          if (!cur && nxt) {
            document.head.appendChild(nxt.cloneNode(true));
            continue;
          }
          if (cur && nxt) {
            const nextText = String(nxt.textContent || '');
            if (String(cur.textContent || '') !== nextText) cur.textContent = nextText;
          }
        }
      } catch (_) {}
    },
    morphDOMNode(curNode, nextNode) {
      if (!curNode || !nextNode) return;
      if (curNode.nodeType !== nextNode.nodeType) {
        curNode.replaceWith(nextNode.cloneNode(true));
        return;
      }
      if (curNode.nodeType === Node.TEXT_NODE) {
        const nextText = String(nextNode.nodeValue || '');
        if (String(curNode.nodeValue || '') !== nextText) curNode.nodeValue = nextText;
        return;
      }
      if (curNode.nodeType !== Node.ELEMENT_NODE) return;
      const curTag = String(curNode.tagName || '').toLowerCase();
      const nextTag = String(nextNode.tagName || '').toLowerCase();
      if (curTag !== nextTag) {
        curNode.replaceWith(nextNode.cloneNode(true));
        return;
      }
      try {
        const curAttrs = curNode.attributes;
        for (let i = curAttrs.length - 1; i >= 0; i--) {
          const name = curAttrs[i].name;
          if (!nextNode.hasAttribute(name)) curNode.removeAttribute(name);
        }
        const nextAttrs = nextNode.attributes;
        for (let i = 0; i < nextAttrs.length; i++) {
          const name = nextAttrs[i].name;
          const value = String(nextAttrs[i].value || '');
          if (curNode.getAttribute(name) !== value) curNode.setAttribute(name, value);
        }
      } catch (_) {
        // Some attribute names used by template frameworks can fail set/removeAttribute.
        curNode.replaceWith(nextNode.cloneNode(true));
        return;
      }
      const curChildren = Array.from(curNode.childNodes);
      const nextChildren = Array.from(nextNode.childNodes);
      const max = Math.max(curChildren.length, nextChildren.length);
      for (let i = 0; i < max; i++) {
        const curChild = curChildren[i] || null;
        const nextChild = nextChildren[i] || null;
        if (curChild && !nextChild) {
          curChild.remove();
          continue;
        }
        if (!curChild && nextChild) {
          curNode.appendChild(nextChild.cloneNode(true));
          continue;
        }
        this.morphDOMNode(curChild, nextChild);
      }
    },
    async tryHotPatchRuntimeHTMLOnly() {
      const urls = this.runtimeManagedScriptURLsFromDoc(document);
      const currentMap = this.runtimeScriptFingerprints || {};
      if (!Object.keys(currentMap).length) {
        await this.captureRuntimeScriptFingerprints();
      }
      const baseline = this.runtimeScriptFingerprints || {};
      const nextMap = await this.fetchScriptFingerprintMap(urls);
      if (!this.runtimeScriptMapsEqual(baseline, nextMap)) return false;
      let nextDoc = null;
      try {
        const resp = await fetch('/admin?_rt_probe=' + String(Date.now()), {method: 'GET', cache: 'no-store', credentials: 'same-origin'});
        if (!resp.ok) return false;
        const html = await resp.text().catch(() => '');
        if (!String(html || '').trim()) return false;
        nextDoc = new DOMParser().parseFromString(html, 'text/html');
      } catch (_) {
        return false;
      }
      if (!nextDoc || !nextDoc.body) return false;
      const nextDocURLs = this.runtimeManagedScriptURLsFromDoc(nextDoc);
      if (urls.length !== nextDocURLs.length) return false;
      for (let i = 0; i < urls.length; i++) {
        if (urls[i] !== nextDocURLs[i]) return false;
      }
      const currentRoot = document.body && document.body.firstElementChild;
      const nextRoot = nextDoc.body.firstElementChild;
      if (!currentRoot || !nextRoot) return false;
      this.syncHeadFromDocument(nextDoc);
      this.morphDOMNode(currentRoot, nextRoot);
      this.reinitAlpineTreeWhenReady(currentRoot);
      this.runtimeScriptFingerprints = nextMap;
      return true;
    },
    async waitForServerReadyBeforeReload(timeoutMs) {
      const timeout = Math.max(1000, Math.trunc(Number(timeoutMs || 0) || 30000));
      const startedAt = Date.now();
      let delayMs = 250;
      while ((Date.now() - startedAt) < timeout) {
        try {
          const probeURL = '/admin/static/admin.js?_rt_ready_probe=' + String(Date.now());
          const resp = await fetch(probeURL, {
            method: 'GET',
            cache: 'no-store',
            credentials: 'same-origin'
          });
          if (resp && resp.ok) return true;
        } catch (_) {}
        await new Promise((resolve) => setTimeout(resolve, delayMs));
        delayMs = Math.min(1000, delayMs + 100);
      }
      return false;
    },
    async performReload(kind) {
      const reloadKind = String(kind || 'default').trim() || 'default';
      if (typeof this.stopRealtimeUpdates === 'function') {
        try { this.stopRealtimeUpdates(); } catch (_) {}
      }
      await this.waitForServerReadyBeforeReload(45000);
      if (reloadKind === 'runtime_update') {
        try {
          const u = new URL(window.location.href);
          u.searchParams.set('_rt_reload', String(Date.now()));
          window.location.replace(u.toString());
          return;
        } catch (_) {}
      }
      window.location.reload();
    },
    startRealtimeUpdates() { return window.AdminRealtime.startRealtimeUpdates.call(this); },
    stopRealtimeUpdates() { return window.AdminRealtime.stopRealtimeUpdates.call(this); },
    isStatusRealtimeMode() { return window.AdminRealtime.isStatusRealtimeMode.call(this); },
    statusUpdateIntervalMs() { return window.AdminRealtime.statusUpdateIntervalMs.call(this); },
    configureStatusUpdates() { return window.AdminRealtime.configureStatusUpdates.call(this); },
    ensureFallbackPolling() { return window.AdminRealtime.ensureFallbackPolling.call(this); },
    clearFallbackPolling() { return window.AdminRealtime.clearFallbackPolling.call(this); },
    scheduleRealtimeReconnect() { return window.AdminRealtime.scheduleRealtimeReconnect.call(this); },
    connectRealtimeWebSocket() { return window.AdminRealtime.connectRealtimeWebSocket.call(this); },
    sendWSSubscription() { return window.AdminRealtime.sendWSSubscription.call(this); },
    noteWSFailure() { return window.AdminRealtime.noteWSFailure.call(this); },
    handleRealtimeRefresh(forceStats) { return window.AdminRealtime.handleRealtimeRefresh.call(this, forceStats); },
    handleRealtimeChange(scope) { return window.AdminRealtime.handleRealtimeChange.call(this, scope); },
    headers() { return {'Content-Type':'application/json'}; },
    async readAPIError(response, fallbackMessage) {
      const fallback = String(fallbackMessage || 'Request failed').trim() || 'Request failed';
      if (!response) {
        return {body: {}, text: '', message: fallback};
      }
      let body = null;
      let text = '';
      const ct = String((response.headers && response.headers.get && response.headers.get('content-type')) || '').toLowerCase();
      if (ct.includes('application/json')) {
        body = await response.json().catch(() => null);
      } else {
        text = await response.text().catch(() => '');
        const trimmed = String(text || '').trim();
        if (trimmed && (trimmed.startsWith('{') || trimmed.startsWith('['))) {
          try { body = JSON.parse(trimmed); } catch (_) {}
        }
      }
      if (!text && body && typeof body === 'object') {
        const jsonMsg =
          (typeof body.error === 'string' && body.error) ||
          (typeof body.message === 'string' && body.message) ||
          (typeof body.detail === 'string' && body.detail) ||
          '';
        text = String(jsonMsg || '').trim();
      }
      const msg = String(text || '').trim() || (fallback + ' (HTTP ' + Number(response.status || 0) + ')');
      return {body: (body && typeof body === 'object') ? body : {}, text: String(text || '').trim(), message: msg};
    },
    toast(type, message, duration) {
      const msg = String(message || '').trim();
      if (!msg) return;
      if (window.AdminToast && typeof window.AdminToast.show === 'function') {
        window.AdminToast.show({type, message: msg, duration});
      }
    },
    toastSuccess(message, duration) { this.toast('success', message, duration); },
    toastError(message, duration) { this.toast('error', message, duration); },
    toastInfo(message, duration) { this.toast('info', message, duration); },
    toastWarning(message, duration) { this.toast('warning', message, duration); },
    openConfirmModal(opts, onConfirm) {
      const o = opts || {};
      this.confirmModalTitle = String(o.title || 'Confirm Action').trim();
      this.confirmModalMessage = String(o.message || '').trim();
      this.confirmModalConfirmLabel = String(o.confirmLabel || 'Confirm').trim();
      this.confirmModalAction = (typeof onConfirm === 'function') ? onConfirm : null;
      this.confirmModalBusy = false;
      this.showConfirmModal = true;
    },
    closeConfirmModal() {
      if (this.confirmModalBusy) return;
      this.showConfirmModal = false;
      this.confirmModalAction = null;
    },
    async confirmModalProceed() {
      if (this.confirmModalBusy) return;
      const fn = this.confirmModalAction;
      this.confirmModalBusy = true;
      try {
        if (typeof fn === 'function') await fn();
      } finally {
        this.confirmModalBusy = false;
        this.showConfirmModal = false;
        this.confirmModalAction = null;
      }
    },
    async apiFetch(url, opts) { return window.AdminApi.apiFetch.call(this, url, opts); },
    observeRuntimeInstance(response) { return window.AdminApi.observeRuntimeInstance.call(this, response); },
    reloadForRuntimeUpdate() { return window.AdminApi.reloadForRuntimeUpdate.call(this); },
    tabMountElement(tab) {
      return document.getElementById('admin-tab-' + String(tab || '').trim());
    },
    async ensureTabContentLoaded(tab, forceRefresh) {
      const t = String(tab || '').trim();
      if (!t) return;
      if (!forceRefresh && this.tabContentLoaded[t]) return;
      if (this.tabContentLoading[t]) return;
      const mount = this.tabMountElement(t);
      if (!mount) return;
      this.tabContentLoading[t] = true;
      mount.innerHTML = '<div class="card shadow-sm mb-3"><div class="card-body small text-body-secondary">Loading...</div></div>';
      try {
        const r = await this.apiFetch('/admin/tab/' + encodeURIComponent(t), {headers:this.headers()});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (!r.ok) {
          mount.innerHTML = '<div class="card shadow-sm mb-3"><div class="card-body small text-danger">Failed to load tab content.</div></div>';
          return;
        }
        const html = await r.text();
        mount.innerHTML = html || '';
        if (window.Alpine && typeof window.Alpine.initTree === 'function') {
          window.Alpine.initTree(mount);
        }
        this.tabContentLoaded[t] = true;
      } catch (_) {
        mount.innerHTML = '<div class="card shadow-sm mb-3"><div class="card-body small text-danger">Failed to load tab content.</div></div>';
      } finally {
        this.tabContentLoading[t] = false;
      }
    },
    activateTabData(tab) {
      if (tab === 'status') {
        this.renderStats();
        this.loadStats(false);
      } else if (tab === 'models' && !this.modelsInitialized) {
        this.loadModelsCatalog(true);
      } else if (tab === 'performance') {
        if (!this.modelsInitialized) this.loadModelsCatalog(true);
        else this.renderPerformanceCatalog();
      } else if (tab === 'benchmark') {
        this.loadBenchmarkStatus();
      } else if (tab === 'quota') {
        this.loadStats(false);
      } else if (tab === 'providers') {
        this.loadProviders();
      } else if (tab === 'access') {
        this.loadAccessTokens();
        this.loadSecuritySettings();
        this.loadTLSSettings();
      } else if (tab === 'network') {
        this.refreshNetworkTabFromConfig();
      } else if (tab === 'conversations') {
        this.loadConversationsSettings();
        this.loadConversations(true);
      } else if (tab === 'log') {
        this.loadLogSettings();
        this.loadLogs();
      }
    },
    async selectTab(tab) {
      this.activeTab = tab;
      this.persistActiveTab();
      await this.ensureTabContentLoaded(tab, false);
      this.activateTabData(tab);
      this.$nextTick(() => this.ensureActiveTabVisible());
    },
    async restoreActiveTab() {
      try {
        const tab = window.localStorage.getItem(this.activeTabCacheKey);
        if (tab === 'status' || tab === 'conversations' || tab === 'log' || tab === 'quota' || tab === 'providers' || tab === 'access' || tab === 'network' || tab === 'models' || tab === 'performance' || tab === 'benchmark') {
          this.activeTab = tab;
        }
      } catch (_) {}
      await this.ensureTabContentLoaded(this.activeTab, false);
      this.activateTabData(this.activeTab);
      this.$nextTick(() => this.ensureActiveTabVisible());
    },
    persistActiveTab() {
      try {
        window.localStorage.setItem(this.activeTabCacheKey, this.activeTab);
      } catch (_) {}
    },
    initTabScroll() {
      const el = document.getElementById('adminTabsScroll');
      if (!el) return;
      el.scrollLeft = 0;
      window.requestAnimationFrame(() => this.updateTabScrollButtons());
      window.setTimeout(() => this.updateTabScrollButtons(), 80);
    },
    ensureActiveTabVisible() {
      const wrap = document.getElementById('adminTabsScroll');
      if (!wrap) return;
      const btn = wrap.querySelector('.nav-link.active');
      if (!btn || typeof btn.scrollIntoView !== 'function') {
        this.updateTabScrollButtons();
        return;
      }
      try {
        btn.scrollIntoView({block: 'nearest', inline: 'center', behavior: 'auto'});
      } catch (_) {}
      this.updateTabScrollButtons();
    },
    updateTabScrollButtons() {
      const el = document.getElementById('adminTabsScroll');
      if (!el) {
        this.tabBarCanScrollLeft = false;
        this.tabBarCanScrollRight = false;
        return;
      }
      const maxScrollLeft = Math.max(0, el.scrollWidth - el.clientWidth);
      this.tabBarCanScrollLeft = maxScrollLeft > 2 && el.scrollLeft > 2;
      this.tabBarCanScrollRight = maxScrollLeft > 2 && el.scrollLeft < (maxScrollLeft - 2);
    },
    scrollTabsLeft() {
      const el = document.getElementById('adminTabsScroll');
      if (!el) return;
      const maxScrollLeft = Math.max(0, el.scrollWidth - el.clientWidth);
      const delta = Math.max(120, Math.floor(el.clientWidth * 0.6));
      const target = Math.max(0, Math.min(maxScrollLeft, el.scrollLeft - delta));
      if (target <= 2) this.tabBarCanScrollLeft = false;
      el.scrollTo({left: target, behavior: 'smooth'});
      window.setTimeout(() => this.updateTabScrollButtons(), 180);
    },
    scrollTabsRight() {
      const el = document.getElementById('adminTabsScroll');
      if (!el) return;
      const maxScrollLeft = Math.max(0, el.scrollWidth - el.clientWidth);
      const delta = Math.max(120, Math.floor(el.clientWidth * 0.6));
      const target = Math.max(0, Math.min(maxScrollLeft, el.scrollLeft + delta));
      if (target >= (maxScrollLeft - 2)) this.tabBarCanScrollRight = false;
      el.scrollTo({left: target, behavior: 'smooth'});
      window.setTimeout(() => this.updateTabScrollButtons(), 180);
    },
    openBenchmarkModal() {
      this.showBenchmarkModal = true;
    },
    closeBenchmarkModal() {
      this.showBenchmarkModal = false;
    },
    renderBenchmark() {
      const r = this.benchmarkRun;
      if (!r || !Array.isArray(r.models) || r.models.length === 0) {
        this.benchmarkHtml = '<div class="small text-body-secondary">No benchmark running.</div>';
        return;
      }
      const status = this.escapeHtml(String(r.status || 'unknown'));
      const totals = 'Total tokens: ' + Number(r.total_tokens || 0) + ' (prompt ' + Number(r.total_prompt_tokens || 0) + ', completion ' + Number(r.total_completion_tokens || 0) + ')';
      const rows = r.models.map((m) => {
        const chats = Number(m.chats_done || 0) + '/' + Number(m.chats_total || 0);
        const msgs = Number(m.messages_done || 0) + '/' + Number(m.messages_total || 0);
        const toks = Number(m.total_tokens || 0);
        const st = this.escapeHtml(String(m.status || 'unknown'));
        const err = String(m.last_error || '').trim();
        return '<tr>' +
          '<td>' + this.escapeHtml(String(m.model || '')) + '</td>' +
          '<td>' + st + '</td>' +
          '<td class="text-end">' + chats + '</td>' +
          '<td class="text-end">' + msgs + '</td>' +
          '<td class="text-end">' + toks + '</td>' +
          '<td class="small text-danger">' + this.escapeHtml(err) + '</td>' +
        '</tr>';
      }).join('');
      this.benchmarkHtml =
        '<div class="small mb-2">Status: <strong>' + status + '</strong> · ' + this.escapeHtml(totals) + '</div>' +
        '<div class="table-responsive"><table class="table table-sm align-middle mb-0">' +
        '<thead><tr><th>Model</th><th>Status</th><th class="text-end">Chats</th><th class="text-end">Messages</th><th class="text-end">Tokens</th><th>Error</th></tr></thead>' +
        '<tbody>' + rows + '</tbody></table></div>';
    },
    async loadBenchmarkStatus() {
      const r = await this.apiFetch('/admin/api/benchmark', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json();
      this.benchmarkRunning = !!body.running;
      this.benchmarkRun = body.run || null;
      this.renderBenchmark();
    },
    async startBenchmark() {
      const models = (this.modelsCatalog || [])
        .map((m) => ({p: String(m.provider || '').trim(), id: String(m.model || '').trim()}))
        .filter((x) => x.p && x.id && x.id !== '-')
        .map((x) => x.p + '/' + x.id);
      if (!models.length) {
        this.toastError('No models to benchmark.');
        return;
      }
      const payload = {
        chats_per_model: Math.max(1, Number(this.benchmarkDraft.chats_per_model || 1)),
        messages_per_chat: Math.max(1, Number(this.benchmarkDraft.messages_per_chat || 1)),
        stop_after_tokens: Math.max(0, Number(this.benchmarkDraft.stop_after_tokens || 0)),
        models
      };
      const r = await this.apiFetch('/admin/api/benchmark/start', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        this.toastError('Failed to start benchmark.');
        return;
      }
      this.closeBenchmarkModal();
      this.activeTab = 'benchmark';
      this.persistActiveTab();
      await this.loadBenchmarkStatus();
    },
    async cancelBenchmark() {
      const r = await this.apiFetch('/admin/api/benchmark/cancel', {method:'POST', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      await this.loadBenchmarkStatus();
    },
    restoreThemeMode() {
      try {
        const v = window.localStorage.getItem(this.themeCacheKey);
        if (v === 'auto' || v === 'light' || v === 'dark') this.themeMode = v;
      } catch (_) {}
      this.applyThemeMode();
    },
    persistThemeMode() {
      try {
        window.localStorage.setItem(this.themeCacheKey, this.themeMode);
      } catch (_) {}
    },
    restoreUsageChartGroup() {
      try {
        const v = String(window.localStorage.getItem(this.usageChartGroupCacheKey) || '').trim();
        if (v === 'model' || v === 'provider' || v === 'api_key_name' || v === 'client_ip' || v === 'user_agent') {
          this.usageChartGroupBy = v;
        }
      } catch (_) {}
    },
    persistUsageChartGroup() {
      try {
        window.localStorage.setItem(this.usageChartGroupCacheKey, this.usageChartGroupBy);
      } catch (_) {}
    },
    restoreStatusUpdateSpeed() {
      try {
        const raw = String(window.localStorage.getItem(this.statusUpdateSpeedCacheKey) || '').trim();
        if (raw === 'realtime' || raw === '2s' || raw === '10s' || raw === '30s' || raw === '1m' || raw === '5m' || raw === '15m' || raw === 'disabled') {
          this.statusUpdateSpeed = raw;
        }
      } catch (_) {}
    },
    persistStatusUpdateSpeed() {
      try {
        window.localStorage.setItem(this.statusUpdateSpeedCacheKey, String(this.statusUpdateSpeed || 'realtime'));
      } catch (_) {}
    },
    setStatusUpdateSpeed(v) {
      const raw = String(v || '').trim();
      if (raw !== 'realtime' && raw !== '2s' && raw !== '10s' && raw !== '30s' && raw !== '1m' && raw !== '5m' && raw !== '15m' && raw !== 'disabled') return;
      this.statusUpdateSpeed = raw;
      this.persistStatusUpdateSpeed();
      this.configureStatusUpdates();
    },
    restoreLogLevelFilter() {
      try {
        const raw = String(window.localStorage.getItem(this.logLevelFilterCacheKey) || '').trim().toLowerCase();
        if (raw === 'all') {
          this.logLevelFilter = 'trace';
          this.persistLogLevelFilter();
          return;
        }
        if (raw === 'trace' || raw === 'debug' || raw === 'info' || raw === 'warn' || raw === 'error' || raw === 'fatal') {
          this.logLevelFilter = raw;
        }
      } catch (_) {}
    },
    persistLogLevelFilter() {
      try {
        window.localStorage.setItem(this.logLevelFilterCacheKey, String(this.logLevelFilter || 'trace'));
      } catch (_) {}
    },
    setLogLevelFilter(v) {
      const raw = String(v || '').trim().toLowerCase();
      if (raw !== 'trace' && raw !== 'debug' && raw !== 'info' && raw !== 'warn' && raw !== 'error' && raw !== 'fatal') return;
      this.logLevelFilter = raw;
      this.persistLogLevelFilter();
      this.logsPage = 1;
      this.loadLogs();
    },
    restoreStatsRangeHours() {
      try {
        const raw = String(window.localStorage.getItem(this.statsRangeCacheKey) || '').trim();
        if (raw === '1' || raw === '4' || raw === '8' || raw === '24' || raw === '72') {
          this.statsRangeHours = raw;
        }
      } catch (_) {}
    },
    persistStatsRangeHours() {
      try {
        window.localStorage.setItem(this.statsRangeCacheKey, String(this.statsRangeHours || '8'));
      } catch (_) {}
    },
    setStatsRangeHours(v) {
      const raw = String(v || '').trim();
      if (raw !== '1' && raw !== '4' && raw !== '8' && raw !== '24' && raw !== '72') return;
      this.statsRangeHours = raw;
      this.persistStatsRangeHours();
      this.loadStats(true);
    },
    restorePerformanceRangeHours() {
      try {
        const raw = String(window.localStorage.getItem(this.performanceRangeCacheKey) || '').trim();
        if (raw === '1' || raw === '4' || raw === '8' || raw === '24' || raw === '72') {
          this.performanceRangeHours = raw;
        }
      } catch (_) {}
    },
    persistPerformanceRangeHours() {
      try {
        window.localStorage.setItem(this.performanceRangeCacheKey, String(this.performanceRangeHours || '8'));
      } catch (_) {}
    },
    setPerformanceRangeHours(v) {
      const raw = String(v || '').trim();
      if (raw !== '1' && raw !== '4' && raw !== '8' && raw !== '24' && raw !== '72') return;
      this.performanceRangeHours = raw;
      this.persistPerformanceRangeHours();
      this.loadModelsCatalog(false, false);
    },
    resolvedThemeMode() {
      if (this.themeMode === 'auto') {
        try {
          return (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light';
        } catch (_) {
          return 'light';
        }
      }
      return this.themeMode;
    },
    applyThemeMode() {
      const resolved = this.resolvedThemeMode();
      document.documentElement.setAttribute('data-bs-theme', resolved);
    },
    themeButtonLabel() {
      if (this.themeMode === 'light') return 'Theme: Light';
      if (this.themeMode === 'dark') return 'Theme: Dark';
      return 'Theme: Auto';
    },
    themeButtonIcon() {
      if (this.themeMode === 'light') {
        return '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M8 1a.5.5 0 0 1 .5.5V3a.5.5 0 0 1-1 0V1.5A.5.5 0 0 1 8 1Zm0 10.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Zm0 3.5a.5.5 0 0 1-.5-.5V13a.5.5 0 0 1 1 0v1.5a.5.5 0 0 1-.5.5ZM2.343 3.757a.5.5 0 0 1 .707 0l1.06 1.06a.5.5 0 1 1-.707.708l-1.06-1.061a.5.5 0 0 1 0-.707Zm9.547 9.546a.5.5 0 0 1 .707 0l1.06 1.061a.5.5 0 0 1-.707.707l-1.06-1.06a.5.5 0 0 1 0-.708ZM1 8a.5.5 0 0 1 .5-.5H3a.5.5 0 0 1 0 1H1.5A.5.5 0 0 1 1 8Zm10.5 0a.5.5 0 0 1 .5-.5H13.5a.5.5 0 0 1 0 1H12a.5.5 0 0 1-.5-.5ZM2.343 12.243a.5.5 0 0 1 0 .707l-1.06 1.06a.5.5 0 0 1-.707-.707l1.06-1.06a.5.5 0 0 1 .707 0Zm9.547-9.546a.5.5 0 0 1 0 .707l-1.06 1.061a.5.5 0 0 1-.707-.708l1.06-1.06a.5.5 0 0 1 .707 0Z"/></svg>';
      }
      if (this.themeMode === 'dark') {
        return '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M6 0a.75.75 0 0 1 .73.96 6.5 6.5 0 0 0 8.31 8.31.75.75 0 0 1 .96.73A7.5 7.5 0 1 1 6 0Z"/></svg>';
      }
      return '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16"><path d="M8 1.5a.5.5 0 0 1 .5.5V3a.5.5 0 0 1-1 0V2a.5.5 0 0 1 .5-.5Zm0 11a.5.5 0 0 1 .5.5V14a.5.5 0 0 1-1 0v-1a.5.5 0 0 1 .5-.5ZM3 7.5a.5.5 0 0 1 0 1H2a.5.5 0 0 1 0-1h1Zm11 0a.5.5 0 0 1 0 1h-1a.5.5 0 0 1 0-1h1Zm-8.657-4.95a.5.5 0 0 1 .707 0l.707.707a.5.5 0 0 1-.707.707l-.707-.707a.5.5 0 0 1 0-.707Zm5.657 5.657a.5.5 0 0 1 .707 0l.707.707a.5.5 0 0 1-.707.707l-.707-.707a.5.5 0 0 1 0-.707ZM3.257 9.864a.5.5 0 0 1 .707 0l.707.707a.5.5 0 1 1-.707.707l-.707-.707a.5.5 0 0 1 0-.707Zm8.486-8.486a.5.5 0 0 1 .707 0l.707.707a.5.5 0 0 1-.707.707l-.707-.707a.5.5 0 0 1 0-.707ZM8 4.5a3.5 3.5 0 0 0 0 7 .5.5 0 0 0 0-1 2.5 2.5 0 1 1 0-5 .5.5 0 0 0 0-1Z"/></svg>';
    },
    cycleTheme() {
      if (this.themeMode === 'auto') this.themeMode = 'light';
      else if (this.themeMode === 'light') this.themeMode = 'dark';
      else this.themeMode = 'auto';
      this.persistThemeMode();
      this.applyThemeMode();
    },
    hydrateModelsFromCache() {
      try {
        const raw = window.localStorage.getItem(this.modelsCacheKey);
        if (!raw) return;
        const parsed = JSON.parse(raw);
        if (!Array.isArray(parsed)) return;
        this.modelsCatalog = parsed;
        this.modelsInitialized = parsed.length > 0;
        this.renderModelsCatalog();
      } catch (_) {}
    },
    persistModelsToCache() {
      try {
        window.localStorage.setItem(this.modelsCacheKey, JSON.stringify(this.modelsCatalog || []));
      } catch (_) {}
    },
    restoreModelsFreeOnly() {
      try {
        const raw = String(window.localStorage.getItem(this.modelsFreeOnlyCacheKey) || '').trim().toLowerCase();
        this.modelsFreeOnly = raw === '1' || raw === 'true';
      } catch (_) {}
    },
    persistModelsFreeOnly() {
      try {
        window.localStorage.setItem(this.modelsFreeOnlyCacheKey, this.modelsFreeOnly ? '1' : '0');
      } catch (_) {}
    },
    escapeHtml(v) {
      return String(v || '').replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
    },
    providerIconSrc(name) {
      const raw = String(name || '').trim();
      if (!raw) {
        return 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';
      }
      const id = raw.replace(/\.svg$/i, '').toLowerCase();
      return '/admin/static/' + encodeURIComponent(id) + '.svg';
    },
    providerIconID(name) {
      return String(name || '').trim().replace(/\.svg$/i, '').toLowerCase();
    },
    providerIconNeedsDarkInvert(name) {
      const id = this.providerIconID(name);
      const mono = new Set([
        'anthropic',
        'baseten',
        'cortecs',
        'deepinfra',
        'deepseek',
        'github-copilot',
        'helicone',
        'io-net',
        'lmstudio',
        'moonshotai',
        'nebius',
        'ollama-cloud',
        'venice',
        'vercel-ai-gateway',
        'xai',
        'zai',
        'zenmux'
      ]);
      return mono.has(id);
    },
    providerIconError(event) {
      const img = event && event.target ? event.target : null;
      if (!img) return;
      img.onerror = null;
      img.src = 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';
    },
    editActionIconSVG() {
      return '' +
        '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true">' +
          '<path d="M15.502 1.94a.5.5 0 0 1 0 .706l-1.043 1.043-2-2L13.502.646a.5.5 0 0 1 .707 0z"/>' +
          '<path d="M13.752 4.396l-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12z"/>' +
          '<path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11A1.5 1.5 0 0 0 15 13.5V6a.5.5 0 0 0-1 0v7.5a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H10a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5z"/>' +
        '</svg>';
    },
    formatPrice(v, currency) {
      if (v === undefined || v === null || Number.isNaN(Number(v))) return '-';
      const n = Number(v);
      const c = String(currency || '').trim().toUpperCase();
      if (!c || c === 'USD') return '$' + n.toFixed(2);
      return c + ' ' + n.toFixed(2);
    },
    formatAge(checkedAt) {
      return this.formatRelativeShort(checkedAt, '');
    },
    formatRelativeAge(ts) {
      return this.formatRelativeShort(ts, '-');
    },
    conversationProviderDisplayName(providerName) {
      const provider = String(providerName || '').trim();
      if (!provider) return '-';
      const configured = (this.providers || []).find((p) => String((p && p.name) || '').trim() === provider);
      if (configured) {
        const display = String((configured && configured.display_name) || '').trim();
        if (display) return display;
      }
      const preset = (this.popularProviders || []).find((p) => String((p && p.name) || '').trim() === provider);
      if (preset) {
        const display = String((preset && preset.display_name) || '').trim();
        if (display) return display;
      }
      return provider;
    },
    conversationModelDisplayName(providerName, modelName) {
      const model = String(modelName || '').trim();
      if (!model) return '-';
      const provider = String(providerName || '').trim();
      const match = (this.modelsCatalog || []).find((m) =>
        String((m && m.provider) || '').trim() === provider &&
        String((m && m.model) || '').trim() === model
      );
      if (match) {
        const display = String((match && (match.model_display_name || match.display_name)) || '').trim();
        if (display) return display;
      }
      return model;
    },
    formatUntil(targetAt) {
      if (!targetAt) return '';
      const t = new Date(targetAt);
      if (Number.isNaN(t.getTime())) return '';
      let sec = Math.floor((t.getTime() - Date.now()) / 1000);
      if (sec <= 0) return 'in <1m';
      const day = 24 * 60 * 60;
      const hour = 60 * 60;
      const minute = 60;
      const days = Math.floor(sec / day);
      sec -= days * day;
      const hours = Math.floor(sec / hour);
      sec -= hours * hour;
      const mins = Math.floor(sec / minute);
      if (days > 0) {
        if (hours > 0) return 'in ' + days + 'd ' + hours + 'h';
        return 'in ' + days + 'd';
      }
      if (hours > 0) {
        if (mins > 0) return 'in ' + hours + 'h ' + mins + 'm';
        return 'in ' + hours + 'h';
      }
      if (mins > 0) return 'in ' + mins + 'm';
      return 'in ' + sec + 's';
    },
    formatRelativeShort(raw, emptyValue) {
      const s = String(raw || '').trim();
      if (!s) return (emptyValue === undefined ? '' : emptyValue);
      const t = new Date(s);
      if (Number.isNaN(t.getTime())) return s;
      const diffSec = Math.floor((t.getTime() - Date.now()) / 1000);
      if (diffSec >= 0) return this.formatUntil(t.toISOString());
      let sec = Math.abs(diffSec);
      const day = 24 * 60 * 60;
      const hour = 60 * 60;
      const minute = 60;
      if (sec >= day) return Math.floor(sec / day) + 'd ago';
      if (sec >= hour) return Math.floor(sec / hour) + 'h ago';
      if (sec >= minute) return Math.floor(sec / minute) + 'm ago';
      return sec + 's ago';
    },
    parsePageSize(v) {
      const raw = String(v ?? '').trim().toLowerCase();
      if (raw === 'all' || raw === '0') return 0;
      const n = Number(raw);
      if (n === 25 || n === 50 || n === 100) return n;
      return 25;
    },
    paginateRows(allRows, page, pageSize) {
      const totalRows = (allRows || []).length;
      const size = this.parsePageSize(pageSize);
      if (size === 0) {
        return {
          rows: allRows || [],
          totalRows,
          totalPages: 1,
          page: 1,
          pageSize: 0
        };
      }
      const totalPages = Math.max(1, Math.ceil(totalRows / size));
      let currentPage = Number(page || 1);
      if (!Number.isFinite(currentPage)) currentPage = 1;
      currentPage = Math.min(totalPages, Math.max(1, Math.floor(currentPage)));
      const start = (currentPage - 1) * size;
      const end = start + size;
      return {
        rows: (allRows || []).slice(start, end),
        totalRows,
        totalPages,
        page: currentPage,
        pageSize: size
      };
    },
    setProvidersPage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.providersPage = Math.max(1, Math.floor(n));
      this.renderProviders();
    },
    setProvidersPageSize(v) {
      this.providersPageSize = this.parsePageSize(v);
      this.providersPage = 1;
      this.renderProviders();
    },
    setModelsPage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.modelsPage = Math.max(1, Math.floor(n));
      this.renderModelsCatalog();
    },
    setAccessPage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.accessPage = Math.max(1, Math.floor(n));
      this.renderAccessTokens();
    },
    setAccessPageSize(v) {
      this.accessPageSize = this.parsePageSize(v);
      this.accessPage = 1;
      this.renderAccessTokens();
    },
    setModelsPageSize(v) {
      this.modelsPageSize = this.parsePageSize(v);
      this.modelsPage = 1;
      this.renderModelsCatalog();
    },
    setPerformancePage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.performancePage = Math.max(1, Math.floor(n));
      this.renderPerformanceCatalog();
    },
    setPerformancePageSize(v) {
      this.performancePageSize = this.parsePageSize(v);
      this.performancePage = 1;
      this.renderPerformanceCatalog();
    },
    setConversationsPage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.conversationsPage = Math.max(1, Math.floor(n));
      this.renderConversationsList();
    },
    setConversationsPageSize(v) {
      this.conversationsPageSize = this.parsePageSize(v);
      this.conversationsPage = 1;
      this.renderConversationsList();
    },
    setLogsPage(v) {
      let n = Number(v);
      if (!Number.isFinite(n)) n = 1;
      this.logsPage = Math.max(1, Math.floor(n));
      this.loadLogs();
    },
    setLogsPageSize(v) {
      this.logsPageSize = this.parsePageSize(v);
      this.logsPage = 1;
      this.loadLogs();
    },
    setUsageChartGroup(v) {
      const key = String(v || '').trim();
      if (key !== 'model' && key !== 'provider' && key !== 'api_key_name' && key !== 'client_ip' && key !== 'user_agent') return;
      this.usageChartGroupBy = key;
      this.persistUsageChartGroup();
      this.renderStats();
    },
    renderStatsLoading() {
      if (this.usageChart) {
        try { this.usageChart.destroy(); } catch (_) {}
        this.usageChart = null;
      }
      this.statsSummaryHtml =
        '<div class="row g-2">' +
          '<div class="col-3"><div class="border rounded p-2 bg-body placeholder-glow" style="min-height:78px;"><span class="placeholder col-8"></span><span class="placeholder col-5"></span></div></div>' +
          '<div class="col-3"><div class="border rounded p-2 bg-body placeholder-glow" style="min-height:78px;"><span class="placeholder col-8"></span><span class="placeholder col-6"></span></div></div>' +
          '<div class="col-3"><div class="border rounded p-2 bg-body placeholder-glow" style="min-height:78px;"><span class="placeholder col-8"></span><span class="placeholder col-4"></span></div></div>' +
          '<div class="col-3"><div class="border rounded p-2 bg-body placeholder-glow" style="min-height:78px;"><span class="placeholder col-8"></span><span class="placeholder col-4"></span></div></div>' +
        '</div>' +
        '<div class="border rounded p-3 bg-body mt-2 placeholder-glow">' +
          '<span class="placeholder col-3 mb-2"></span>' +
          '<div style="height:280px;" class="bg-body-tertiary rounded"></div>' +
        '</div>' +
        '<div class="row g-3 mt-1">' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
        '</div>';
      this.quotaSummaryHtml = '<div class="small text-body-secondary placeholder-glow"><span class="placeholder col-3"></span></div>';
    },
    renderPager(totalRows, page, totalPages, pageSize, key) {
      const disabledFirst = page <= 1 ? ' disabled' : '';
      const disabledLast = page >= totalPages ? ' disabled' : '';
      const sizeValue = pageSize === 0 ? 'all' : String(pageSize);
      const firstFn = '__admin' + key + 'FirstPage';
      const prevFn = '__admin' + key + 'PrevPage';
      const setPageFn = '__adminSet' + key + 'Page';
      const nextFn = '__admin' + key + 'NextPage';
      const lastFn = '__admin' + key + 'LastPage';
      const setPageSizeFn = '__adminSet' + key + 'PageSize';
      return '' +
        '<div class="d-flex flex-wrap align-items-center justify-content-between gap-2 mt-2">' +
          '<div class="d-flex align-items-center gap-1">' +
            '<button class="icon-btn"' + disabledFirst + ' onclick="window.' + firstFn + '()" title="First page" aria-label="First page">&laquo;</button>' +
            '<button class="icon-btn"' + disabledFirst + ' onclick="window.' + prevFn + '()" title="Previous page" aria-label="Previous page">&lsaquo;</button>' +
            '<input class="form-control form-control-sm" type="number" min="1" max="' + totalPages + '" value="' + page + '" style="max-width:86px;" onchange="window.' + setPageFn + '(this.value)" />' +
            '<button class="icon-btn"' + disabledLast + ' onclick="window.' + nextFn + '()" title="Next page" aria-label="Next page">&rsaquo;</button>' +
            '<button class="icon-btn"' + disabledLast + ' onclick="window.' + lastFn + '()" title="Last page" aria-label="Last page">&raquo;</button>' +
          '</div>' +
          '<div class="d-flex align-items-center gap-2 small text-body-secondary">' +
            '<span>Page ' + page + ' of ' + totalPages + '</span>' +
            '<span>(' + totalRows + ' rows)</span>' +
            '<select class="form-select form-select-sm" style="width:auto;" onchange="window.' + setPageSizeFn + '(this.value)">' +
              '<option value="25"' + (sizeValue === '25' ? ' selected' : '') + '>25</option>' +
              '<option value="50"' + (sizeValue === '50' ? ' selected' : '') + '>50</option>' +
              '<option value="100"' + (sizeValue === '100' ? ' selected' : '') + '>100</option>' +
              '<option value="all"' + (sizeValue === 'all' ? ' selected' : '') + '>all</option>' +
            '</select>' +
          '</div>' +
        '</div>';
    },
    renderProvidersPager(totalRows, page, totalPages, pageSize) {
      return this.renderPager(totalRows, page, totalPages, pageSize, 'Providers');
    },
    renderModelsPager(totalRows, page, totalPages, pageSize) {
      return this.renderPager(totalRows, page, totalPages, pageSize, 'Models');
    },
    renderPerformancePager(totalRows, page, totalPages, pageSize) {
      return this.renderPager(totalRows, page, totalPages, pageSize, 'Performance');
    },
    renderAccessPager(totalRows, page, totalPages, pageSize) {
      return this.renderPager(totalRows, page, totalPages, pageSize, 'Access');
    },
    resetDraft() {
      this.cancelOAuthPolling();
      this.draft = {name:'',provider_type:'',base_url:'',api_key:'',auth_token:'',refresh_token:'',token_expires_at:'',account_id:'',device_auth_url:'',device_code:'',device_auth_id:'',device_code_url:'',device_token_url:'',device_client_id:'',device_scope:'',device_grant_type:'',oauth_authorize_url:'',oauth_token_url:'',oauth_client_id:'',oauth_client_secret:'',oauth_scope:'',enabled:true,timeout_seconds:''};
      this.addProviderStep = 'pick_provider';
      this.selectedPreset = '';
      this.presetInfoHtml = '';
      this.overrideProviderSettings = false;
      this.authMode = 'api_key';
      this.providerSearch = '';
      this.editingProviderName = '';
      this.modalStatusHtml = '';
      this.oauthAdvanced = false;
    },
    openAddProviderModal() {
      this.resetDraft();
      this.showAddProviderModal = true;
    },
    closeAddProviderModal() {
      this.cancelOAuthPolling();
      this.showAddProviderModal = false;
    },
    addProviderTitle() {
      if (this.addProviderStep === 'choose_auth') return 'Add Provider - Choose Authentication';
      if (this.addProviderStep === 'api_key') return 'Add Provider - API Key';
      if (this.addProviderStep === 'oauth_browser') return 'Add Provider - Browser OAuth';
      if (this.addProviderStep === 'device_auth') return 'Add Provider - Device Auth';
      return 'Add Provider';
    },
    getSelectedPreset() {
      return (this.popularProviders || []).find((p) => p.name === this.selectedPreset) || null;
    },
    selectedPresetDisplayName() {
      const preset = this.getSelectedPreset();
      return preset ? (preset.display_name || preset.name || '') : '';
    },
    selectedPresetGetKeyURL() {
      const preset = this.getSelectedPreset();
      return preset ? String(preset.get_api_key_url || '').trim() : '';
    },
    selectedPresetApiKeyOptional() {
      const preset = this.getSelectedPreset();
      return !!(preset && preset.public_free_no_auth);
    },
    selectedPresetBaseURLTemplate() {
      const preset = this.getSelectedPreset();
      return preset ? String(preset.base_url_template || '').trim() : '';
    },
    selectedPresetBaseURLHint() {
      const preset = this.getSelectedPreset();
      return preset ? String(preset.base_url_hint || '').trim() : '';
    },
    selectedPresetBaseURLExample() {
      const preset = this.getSelectedPreset();
      return preset ? String(preset.base_url_example || '').trim() : '';
    },
    filteredPopularProviders() {
      const all = (this.popularProviders || []).slice().sort((a, b) => {
        const ad = String((a && a.display_name) || (a && a.name) || '').toLowerCase();
        const bd = String((b && b.display_name) || (b && b.name) || '').toLowerCase();
        if (ad < bd) return -1;
        if (ad > bd) return 1;
        return 0;
      });
      const q = String(this.providerSearch || '').trim().toLowerCase();
      if (!q) return all;
      return all.filter((p) => {
        const name = String((p && p.name) || '').toLowerCase();
        const display = String((p && p.display_name) || '').toLowerCase();
        const docs = String((p && p.docs_url) || '').toLowerCase();
        return name.includes(q) || display.includes(q) || docs.includes(q);
      });
    },
    filteredPopularProvidersCount() {
      return this.filteredPopularProviders().length;
    },
    selectedPresetRequiresBaseURLInput() {
      return !!this.selectedPresetBaseURLTemplate();
    },
    presetSupportsDeviceAuth(preset) {
      return !!(preset && String(preset.device_binding_url || '').trim());
    },
    presetSupportsOAuthBrowser(preset) {
      if (!preset) return false;
      return !!(String(preset.oauth_authorize_url || '').trim() && String(preset.oauth_token_url || '').trim());
    },
    presetSupportsAPIKey(preset) {
      if (!preset) return true;
      if (String(preset.get_api_key_url || '').trim()) return true;
      if (String(preset.api_key_env || '').trim()) return true;
      if (preset.public_free_no_auth) return true;
      return false;
    },
    alternateAuthTitle() {
      const preset = this.getSelectedPreset();
      if (this.presetSupportsOAuthBrowser(preset)) {
        if (preset && String(preset.name || '').trim() === 'openai') return 'Use browser OAuth (ChatGPT/Codex)';
        return 'Use browser OAuth';
      }
      return 'Use OAuth / device auth';
    },
    alternateAuthDescription() {
      const preset = this.getSelectedPreset();
      if (this.presetSupportsOAuthBrowser(preset)) return 'Sign in via browser and auto-capture OAuth tokens.';
      return 'Open device binding, complete login, then save token.';
    },
    applySelectedPreset() {
      const preset = (this.popularProviders || []).find((p) => p.name === this.selectedPreset);
      if (!preset) {
        this.presetInfoHtml = '';
        return;
      }
      this.draft.name = preset.name || '';
      this.draft.provider_type = preset.name || '';
      this.draft.base_url = String(preset.base_url || '').trim();
      if (!this.draft.base_url && String(preset.base_url_template || '').trim()) {
        this.draft.base_url = String(preset.base_url_template || '').trim();
      }
      this.draft.enabled = true;
      this.draft.timeout_seconds = '';
      this.draft.api_key = '';
      this.draft.auth_token = '';
      this.draft.refresh_token = '';
      this.draft.token_expires_at = '';
      this.draft.account_id = '';
      this.draft.device_auth_url = preset.device_binding_url || '';
      this.draft.device_code = '';
      this.draft.device_auth_id = '';
      this.draft.device_code_url = preset.device_code_url || '';
      this.draft.device_token_url = preset.device_token_url || '';
      this.draft.device_client_id = preset.device_client_id || '';
      this.draft.device_scope = preset.device_scope || '';
      this.draft.device_grant_type = preset.device_grant_type || '';
      this.draft.oauth_authorize_url = preset.oauth_authorize_url || '';
      this.draft.oauth_token_url = preset.oauth_token_url || '';
      this.draft.oauth_client_id = preset.oauth_client_id || '';
      this.draft.oauth_client_secret = preset.oauth_client_secret || '';
      this.draft.oauth_scope = preset.oauth_scope || '';
      this.overrideProviderSettings = false;
      this.authMode = 'api_key';
      this.presetInfoHtml = this.renderPresetInfo(preset);
    },
    selectProviderPreset(name) {
      this.selectedPreset = String(name || '').trim();
      this.applySelectedPreset();
      const preset = this.getSelectedPreset();
      const hasDevice = this.presetSupportsDeviceAuth(preset);
      const hasOAuth = this.presetSupportsOAuthBrowser(preset);
      const hasAPI = this.presetSupportsAPIKey(preset);
      if ((hasDevice || hasOAuth) && hasAPI) {
        this.addProviderStep = 'choose_auth';
        return;
      }
      if (hasOAuth) {
        this.authMode = 'oauth';
        this.addProviderStep = 'oauth_browser';
        return;
      }
      if (hasDevice) {
        this.authMode = 'device';
        this.addProviderStep = 'device_auth';
        return;
      }
      this.authMode = 'api_key';
      this.addProviderStep = 'api_key';
    },
    chooseApiKeyAuth() {
      this.authMode = 'api_key';
      this.modalStatusHtml = '';
      this.addProviderStep = 'api_key';
    },
    chooseAlternateAuth() {
      const preset = this.getSelectedPreset();
      if (this.presetSupportsOAuthBrowser(preset)) {
        this.authMode = 'oauth';
        this.modalStatusHtml = '';
        this.addProviderStep = 'oauth_browser';
        this.draft.base_url = String(preset.oauth_base_url || '').trim() || String(preset.base_url || '').trim();
        return;
      }
      this.authMode = 'device';
      this.modalStatusHtml = '';
      this.addProviderStep = 'device_auth';
    },
    goBackFromChoice() {
      this.addProviderStep = 'pick_provider';
    },
    goBackFromForm() {
      const preset = this.getSelectedPreset();
      if ((this.presetSupportsDeviceAuth(preset) || this.presetSupportsOAuthBrowser(preset)) && this.presetSupportsAPIKey(preset)) {
        this.addProviderStep = 'choose_auth';
        return;
      }
      this.addProviderStep = 'pick_provider';
    },
    selectedPresetSupportsDeviceAuth() {
      return this.presetSupportsDeviceAuth(this.getSelectedPreset());
    },
    selectedPresetSupportsDeviceCodeFetch() {
      const preset = this.getSelectedPreset();
      if (preset && String(preset.device_code_url || '').trim()) return true;
      return !!String(this.draft.device_code_url || '').trim();
    },
    selectedPresetSupportsDeviceTokenPolling() {
      const tokenURL = String(this.draft.device_token_url || '').trim();
      const clientID = String(this.draft.device_client_id || '').trim();
      const deviceCode = String(this.draft.device_code || '').trim();
      return !!(tokenURL && clientID && deviceCode);
    },
    deviceCodeParamName() {
      const preset = this.getSelectedPreset();
      const raw = preset ? String(preset.device_code_param || '').trim() : '';
      if (raw) return raw;
      if (preset && preset.name === 'google-gemini') return 'user_code';
      return 'code';
    },
    deviceAuthURLWithCode() {
      const base = String(this.draft.device_auth_url || '').trim();
      const code = String(this.draft.device_code || '').trim();
      if (!base || !code) return '';
      try {
        const u = new URL(base);
        u.searchParams.set(this.deviceCodeParamName(), code);
        return u.toString();
      } catch (_) {
        return '';
      }
    },
    async copyDeviceCode() {
      const code = String(this.draft.device_code || '').trim();
      if (!code) return;
      try {
        await navigator.clipboard.writeText(code);
        this.toastSuccess('Device code copied.');
      } catch (_) {
        this.toastError('Could not copy device code.');
      }
    },
    openDeviceAuth(withCode) {
      const base = String(this.draft.device_auth_url || '').trim();
      if (!base) return;
      const target = withCode ? (this.deviceAuthURLWithCode() || base) : base;
      window.open(target, '_blank', 'noopener,noreferrer');
    },
    cancelOAuthPolling() {
      this.oauthLoginInProgress = false;
      this.oauthPollState = '';
    },
    async startOAuthBrowserLogin() {
      if (this.oauthLoginInProgress) return;
      const provider = String(this.selectedPreset || '').trim();
      if (!provider) return;
      this.oauthLoginInProgress = true;
      this.modalStatusHtml = '<span class="text-body-secondary">Starting browser OAuth flow...</span>';
      try {
        const startResp = await this.apiFetch('/admin/api/providers/oauth/start', {method:'POST', headers:this.headers(), body:JSON.stringify({
          provider,
          oauth_authorize_url: String(this.draft.oauth_authorize_url || '').trim(),
          oauth_token_url: String(this.draft.oauth_token_url || '').trim(),
          oauth_client_id: String(this.draft.oauth_client_id || '').trim(),
          oauth_client_secret: String(this.draft.oauth_client_secret || '').trim(),
          oauth_scope: String(this.draft.oauth_scope || '').trim()
        })});
        if (startResp.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        const startBody = await startResp.json().catch(() => ({}));
        if (!(startResp.ok && startBody.ok)) {
          this.toastError(startBody.error || 'Failed to start OAuth flow.');
          this.oauthLoginInProgress = false;
          return;
        }
        const state = String(startBody.state || '').trim();
        const authURL = String(startBody.auth_url || '').trim();
        if (!state || !authURL) {
          this.toastError('OAuth start response was incomplete.');
          this.oauthLoginInProgress = false;
          return;
        }
        this.oauthPollState = state;
        window.open(authURL, '_blank', 'noopener,noreferrer');
        this.modalStatusHtml = '<span class="text-body-secondary">Complete login in browser tab. Waiting for callback...</span>';
        for (let i = 0; i < 180; i++) {
          if (!this.oauthLoginInProgress || this.oauthPollState !== state) return;
          await new Promise((resolve) => setTimeout(resolve, 1000));
          const rr = await this.apiFetch('/admin/api/providers/oauth/result?state=' + encodeURIComponent(state), {headers:this.headers()});
          if (rr.status === 401) { window.location = '/admin/login?next=/admin'; return; }
          const rb = await rr.json().catch(() => ({}));
          if (rr.status === 404) {
            this.toastError('OAuth session expired or not found. Retry.');
            this.oauthLoginInProgress = false;
            return;
          }
          if (!(rr.ok && rb.ok)) {
            this.toastError(rb.error || 'OAuth flow failed.');
            this.oauthLoginInProgress = false;
            return;
          }
          if (rb.pending) continue;
          this.draft.auth_token = String(rb.auth_token || '').trim();
          this.draft.refresh_token = String(rb.refresh_token || '').trim();
          this.draft.token_expires_at = String(rb.token_expires_at || '').trim();
          this.draft.account_id = String(rb.account_id || '').trim();
          if (String(rb.base_url || '').trim()) this.draft.base_url = String(rb.base_url || '').trim();
          this.toastSuccess('OAuth login completed. Token captured.');
          this.modalStatusHtml = '';
          this.oauthLoginInProgress = false;
          return;
        }
        this.toastError('Timed out waiting for OAuth callback.');
      } finally {
        this.oauthLoginInProgress = false;
      }
    },
    async fetchDeviceCode() {
      if (this.deviceCodeFetchInProgress) return;
      this.deviceCodeFetchInProgress = true;
      this.modalStatusHtml = '<span class="text-body-secondary">Requesting device code...</span>';
      try {
        const payload = {
          provider: String(this.selectedPreset || '').trim(),
          device_code_url: String(this.draft.device_code_url || '').trim(),
          client_id: String(this.draft.device_client_id || '').trim(),
          scope: String(this.draft.device_scope || '').trim()
        };
        const r = await this.apiFetch('/admin/api/providers/device-code', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        const body = await r.json().catch(() => ({}));
        if (!(r.ok && body.ok)) {
          this.toastError(body.error || 'Failed to fetch device code.');
          return;
        }
        const userCode = String(body.user_code || '').trim();
        const deviceAuthID = String(body.device_auth_id || '').trim();
        const verificationComplete = String(body.verification_uri_complete || '').trim();
        const verificationURL = String(body.verification_uri || body.verification_url || '').trim();
        const expiresIn = Number(body.expires_in || 0);
        if (userCode) this.draft.device_code = userCode;
        if (deviceAuthID) this.draft.device_auth_id = deviceAuthID;
        if (verificationComplete) this.draft.device_auth_url = verificationComplete;
        else if (verificationURL) this.draft.device_auth_url = verificationURL;
        let msg = 'Device code fetched.';
        if (expiresIn > 0) msg += ' Expires in ' + expiresIn + 's.';
        this.toastSuccess(msg);
        this.modalStatusHtml = '';
      } finally {
        this.deviceCodeFetchInProgress = false;
      }
    },
    async pollDeviceToken() {
      if (this.deviceTokenPollInProgress) return;
      const payload = {
        provider: String(this.selectedPreset || '').trim(),
        device_token_url: String(this.draft.device_token_url || '').trim(),
        client_id: String(this.draft.device_client_id || '').trim(),
        device_code: String(this.draft.device_code || '').trim(),
        device_auth_id: String(this.draft.device_auth_id || '').trim(),
        grant_type: String(this.draft.device_grant_type || '').trim() || 'urn:ietf:params:oauth:grant-type:device_code'
      };
      if (!payload.device_token_url || !payload.client_id || !payload.device_code) {
        this.toastError('Device token URL, client_id, and device_code are required.');
        return;
      }
      if (payload.provider === 'openai' && !payload.device_auth_id) {
        this.toastError('OpenAI headless flow requires device_auth_id. Fetch device code first.');
        return;
      }
      this.deviceTokenPollInProgress = true;
      this.modalStatusHtml = '<span class="text-body-secondary">Polling device auth token...</span>';
      try {
        for (let i = 0; i < 90; i++) {
          const r = await this.apiFetch('/admin/api/providers/device-token', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
          if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
          const body = await r.json().catch(() => ({}));
          if (!(r.ok && body.ok)) {
            this.toastError(body.error || 'Device token exchange failed.');
            return;
          }
          if (!body.pending && String(body.auth_token || '').trim()) {
            this.draft.auth_token = String(body.auth_token || '').trim();
            this.draft.refresh_token = String(body.refresh_token || '').trim();
            this.draft.account_id = String(body.account_id || '').trim();
            if (Number(body.expires_in || 0) > 0) {
              const when = new Date(Date.now() + Number(body.expires_in) * 1000).toISOString();
              this.draft.token_expires_at = when;
            }
            this.toastSuccess('Device auth token received.');
            this.modalStatusHtml = '';
            return;
          }
          const waitSec = Math.max(1, Number(body.interval || 5));
          await new Promise((resolve) => setTimeout(resolve, waitSec * 1000));
        }
        this.toastError('Timed out waiting for device auth token.');
      } finally {
        this.deviceTokenPollInProgress = false;
      }
    },
    renderPresetInfo(preset) {
      const links = [];
      if (preset.docs_url) links.push('<a href="' + this.escapeHtml(preset.docs_url) + '" target="_blank" rel="noopener noreferrer">Docs</a>');
      if (preset.get_api_key_url) links.push('<a href="' + this.escapeHtml(preset.get_api_key_url) + '" target="_blank" rel="noopener noreferrer">Get API key</a>');
      if (preset.auth_portal_url) links.push('<a href="' + this.escapeHtml(preset.auth_portal_url) + '" target="_blank" rel="noopener noreferrer">Auth portal</a>');
      if (preset.device_binding_url) links.push('<a href="' + this.escapeHtml(preset.device_binding_url) + '" target="_blank" rel="noopener noreferrer">Device binding</a>');
      if (preset.oauth_authorize_url) links.push('Browser OAuth');
      const notes = [];
      if (preset.base_url_template) notes.push('Base URL template: <code>' + this.escapeHtml(preset.base_url_template) + '</code>');
      if (preset.base_url_hint) notes.push(this.escapeHtml(preset.base_url_hint));
      if (preset.public_free_no_auth) notes.push('Public free (no auth)');
      if (preset.free_tier_with_key) notes.push('Free tier (with key)');
      if (preset.trial_credits) notes.push('Trial credits');
      if (preset.last_verified_at) notes.push('Verified: ' + this.escapeHtml(preset.last_verified_at));
      if (preset.source_url) notes.push('<a href="' + this.escapeHtml(preset.source_url) + '" target="_blank" rel="noopener noreferrer">Pricing source</a>');
      return links.join(' · ') + (notes.length ? '<div class="mt-1">' + notes.join(' · ') + '</div>' : '');
    },
    renderStats() {
      const s = this.stats || {};
      const req = Number(s.requests || 0);
      const prompt = Number(s.prompt_tokens || 0);
      const generated = Number(s.completion_tokens || 0);
      const latency = Number(s.avg_latency_ms || 0).toFixed(1);
      const pp = Number(s.avg_prompt_tps || 0).toFixed(2);
      const tg = Number(s.avg_generation_tps || 0).toFixed(2);
      const periodHours = Math.max(1, Number(this.statsRangeHours || 8));
      const providersAvailable = Number(s.providers_available || 0);
      const providersOnline = Number(s.providers_online || 0);
      const providerChart = this.renderUsageChart(s.requests_per_provider || {}, 'Providers');
      const buckets = Array.isArray(s.buckets) ? s.buckets : [];
      const groupField = this.usageChartGroupBy || 'model';
      const groupLabel = this.usageChartGroupLabel(groupField);
      const bucketMs = this.usageBucketMs(buckets);
      const bucketLabel = this.usageBucketLabel(bucketMs);
      const renderToken = ++this.statsRenderToken;
      this.statsSummaryHtml =
        '<div class="admin-stats-grid">' +
          '<div class="border rounded p-2 bg-body d-flex flex-column align-items-center justify-content-center text-center admin-stats-tile"><div class="small text-body-secondary">Requests</div><div class="fw-semibold">' + req + '</div></div>' +
          '<div class="border rounded p-2 bg-body d-flex flex-column align-items-center justify-content-center text-center admin-stats-tile"><div class="small text-body-secondary">Prompt / Generated</div><div class="fw-semibold">' + prompt + ' / ' + generated + '</div></div>' +
          '<div class="border rounded p-2 bg-body d-flex flex-column align-items-center justify-content-center text-center admin-stats-tile"><div class="small text-body-secondary">Avg latency ms</div><div class="fw-semibold">' + latency + '</div></div>' +
          '<div class="border rounded p-2 bg-body d-flex flex-column align-items-center justify-content-center text-center admin-stats-tile"><div class="small text-body-secondary">Avg PP/s</div><div class="fw-semibold">' + pp + '</div></div>' +
          '<div class="border rounded p-2 bg-body d-flex flex-column align-items-center justify-content-center text-center admin-stats-tile"><div class="small text-body-secondary">Avg TG/s</div><div class="fw-semibold">' + tg + '</div></div>' +
        '</div>' +
        '<div class="border rounded p-3 bg-body mt-2">' +
          '<div class="d-flex justify-content-between align-items-center mb-2">' +
            '<div class="fw-semibold">Usage by ' + this.escapeHtml(groupLabel.toLowerCase()) + ' (tokens / ' + this.escapeHtml(bucketLabel) + ')</div>' +
            '<div class="d-flex align-items-center">' +
              '<select class="form-select form-select-sm" style="width:auto;" onchange="window.__adminSetUsageChartGroup(this.value)">' +
                '<option value="model"' + (groupField === 'model' ? ' selected' : '') + '>Model</option>' +
                '<option value="provider"' + (groupField === 'provider' ? ' selected' : '') + '>Provider</option>' +
                '<option value="api_key_name"' + (groupField === 'api_key_name' ? ' selected' : '') + '>Token Name</option>' +
                '<option value="client_ip"' + (groupField === 'client_ip' ? ' selected' : '') + '>Remote IP</option>' +
                '<option value="user_agent"' + (groupField === 'user_agent' ? ' selected' : '') + '>User Agent</option>' +
              '</select>' +
            '</div>' +
          '</div>' +
          '<div style="height:280px;"><canvas id="modelUsageChart"></canvas></div>' +
        '</div>' +
        '<div class="small text-body-secondary mt-2">Providers available: <strong>' + providersAvailable + '</strong> · Online: <strong>' + providersOnline + '</strong></div>' +
        '<div id="usageStatsTables" class="row g-3 mt-1">' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
          '<div class="col-lg-6"><div class="border rounded p-3 bg-body placeholder-glow"><span class="placeholder col-4"></span><span class="placeholder col-12"></span><span class="placeholder col-10"></span><span class="placeholder col-11"></span></div></div>' +
        '</div>';
      if (!String(this.quotaSummaryHtml || '').trim()) {
        this.quotaSummaryHtml = '<div class="small text-body-secondary">Loading quota...</div>';
      }
      const quotaMap = this.quotaMapWithFallback(s.provider_quotas || {});
      setTimeout(() => {
        if (renderToken !== this.statsRenderToken) return;
        this.quotaSummaryHtml = this.renderQuotaPanel(quotaMap);
        this.renderUsageTimelineChart(buckets, groupField);
      }, 0);
      setTimeout(() => {
        if (renderToken !== this.statsRenderToken) return;
        const providersTable = this.renderUsageMetricTable('Providers', this.aggregateUsageRowsBy(buckets, 'provider'), 'Provider');
        const modelsTable = this.renderUsageMetricTable('Models', this.aggregateUsageRowsBy(buckets, 'model'), 'Model');
        const tokenNamesTable = this.renderUsageMetricTable('Token Names', this.aggregateUsageRowsBy(buckets, 'api_key_name'), 'Token Name');
        const remoteIPsTable = this.renderUsageMetricTable('Remote IPs', this.aggregateUsageRowsBy(buckets, 'client_ip'), 'Remote IP');
        const userAgentsTable = this.renderUsageMetricTable('User Agents', this.aggregateUsageRowsBy(buckets, 'user_agent'), 'User-Agent');
        const tablesNode = document.getElementById('usageStatsTables');
        if (!tablesNode) return;
        tablesNode.innerHTML =
          '<div class="col-lg-6">' + providersTable + '</div>' +
          '<div class="col-lg-6">' + modelsTable + '</div>' +
          '<div class="col-lg-6">' + tokenNamesTable + '</div>' +
          '<div class="col-lg-6">' + remoteIPsTable + '</div>' +
          '<div class="col-lg-6">' + userAgentsTable + '</div>';
      }, 25);
    },
    usageChartGroupLabel(field) {
      if (field === 'provider') return 'Provider';
      if (field === 'api_key_name') return 'Token Name';
      if (field === 'client_ip') return 'Remote IP';
      if (field === 'user_agent') return 'User Agent';
      return 'Model';
    },
    usageBucketMs(buckets) {
      let minSec = 0;
      (Array.isArray(buckets) ? buckets : []).forEach((b) => {
        const sec = Number(b.slot_seconds || 0);
        if (!Number.isFinite(sec) || sec <= 0) return;
        if (minSec <= 0 || sec < minSec) minSec = sec;
      });
      if (minSec > 0) return minSec * 1000;
      if (Math.max(1, Number(this.statsRangeHours || 8)) <= 1) return 60 * 1000;
      return 5 * 60 * 1000;
    },
    usageBucketLabel(bucketMs) {
      const sec = Math.max(1, Math.round(Number(bucketMs || 0) / 1000));
      if (sec % 3600 === 0) return String(sec / 3600) + 'h';
      if (sec % 60 === 0) return String(sec / 60) + 'm';
      return String(sec) + 's';
    },
    renderQuotaPanel(quotaMap) {
      const chart = this.renderQuotaChart(quotaMap || {});
      if (!chart) return '<div class="small text-body-secondary">No quota data.</div>';
      return chart;
    },
    quotaMapWithFallback(quotaMap) {
      const input = (quotaMap && typeof quotaMap === 'object') ? quotaMap : {};
      const out = {};
      Object.entries(input).forEach(([provider, snap]) => {
        const cur = (snap && typeof snap === 'object') ? snap : {};
        const status = String(cur.status || '').trim().toLowerCase();
        const prev = this.lastGoodQuotaByProvider[provider];
        if (status === 'loading' && prev && String(prev.status || '').trim().toLowerCase() === 'ok') {
          out[provider] = prev;
          return;
        }
        out[provider] = cur;
        if (status === 'ok') {
          this.lastGoodQuotaByProvider[provider] = cur;
        }
      });
      return out;
    },
    quotaValueText(v) {
      const n = Number(v);
      if (!Number.isFinite(n)) return '';
      const rounded = Math.round(n);
      if (Math.abs(n - rounded) < 0.000001) return String(rounded);
      return n.toFixed(2);
    },
    quotaTooltipText(metric) {
      const m = metric || {};
      const unit = String(m.unit || '').trim();
      const limit = Number(m.limit_value);
      const remaining = Number(m.remaining_value);
      const used = Number(m.used_value);
      if (!Number.isFinite(limit) || limit <= 0) return '';
      const usedVal = Number.isFinite(used) ? used : Math.max(0, limit - (Number.isFinite(remaining) ? remaining : 0));
      const remVal = Number.isFinite(remaining) ? remaining : Math.max(0, limit - usedVal);
      const suffix = unit ? (' ' + unit) : '';
      return 'Used: ' + this.quotaValueText(usedVal) + suffix +
        ' / Limit: ' + this.quotaValueText(limit) + suffix +
        ' (Remaining: ' + this.quotaValueText(remVal) + suffix + ')';
    },
    quotaMetricKind(metric) {
      const m = metric || {};
      const unit = String(m.unit || '').trim().toLowerCase();
      const feature = String(m.metered_feature || '').trim().toLowerCase();
      if (unit === 'tokens' || feature.includes('token')) return 'tokens';
      if (unit === 'requests' || feature.includes('request')) return 'requests';
      return 'other';
    },
    metricLeftPercent(metric) {
      if (!metric || typeof metric !== 'object') return null;
      const limitVal = Number(metric.limit_value || 0);
      const remainVal = Number(metric.remaining_value);
      let left = Number(metric.left_percent);
      if (Number.isFinite(limitVal) && limitVal > 0 && Number.isFinite(remainVal)) {
        left = (remainVal / limitVal) * 100;
      }
      if (!Number.isFinite(left)) return null;
      return Math.max(0, Math.min(100, left));
    },
    hasAnyProviderOutOfQuota() {
      const map = (this.stats && this.stats.provider_quotas && typeof this.stats.provider_quotas === 'object')
        ? this.stats.provider_quotas
        : {};
      const snaps = Object.values(map);
      for (let i = 0; i < snaps.length; i++) {
        const q = snaps[i] || {};
        if (String(q.status || '').trim().toLowerCase() !== 'ok') continue;
        const metrics = Array.isArray(q.metrics) ? q.metrics : [];
        if (metrics.length > 0) {
          const leftVals = metrics
            .map((m) => this.metricLeftPercent(m))
            .filter((v) => Number.isFinite(v));
          if (leftVals.length > 0 && leftVals.every((v) => v <= 0.000001)) {
            return true;
          }
          continue;
        }
        const left = Number(q.left_percent);
        if (Number.isFinite(left) && left <= 0.000001) {
          return true;
        }
      }
      return false;
    },
    hasAnyProviderQuotaErrors() {
      const map = (this.stats && this.stats.provider_quotas && typeof this.stats.provider_quotas === 'object')
        ? this.stats.provider_quotas
        : {};
      const snaps = Object.values(map);
      for (let i = 0; i < snaps.length; i++) {
        const q = snaps[i] || {};
        const status = String(q.status || '').trim().toLowerCase();
        if (!status || status === 'ok' || status === 'loading') continue;
        return true;
      }
      return false;
    },
    hasAnyQuotaAlerts() {
      return this.hasAnyProviderOutOfQuota() || this.hasAnyProviderQuotaErrors();
    },
    failingManagedProviders() {
      const list = Array.isArray(this.providers) ? this.providers : [];
      const failing = [];
      for (let i = 0; i < list.length; i++) {
        const p = list[i] || {};
        if (!p.managed) continue;
        if (!this.providerHasProblem(p)) continue;
        failing.push(p);
      }
      return failing;
    },
    hasAnyProviderFailing() {
      return this.failingManagedProviders().length > 0;
    },
    providersFailureTooltip() {
      const list = this.failingManagedProviders();
      const failing = [];
      for (let i = 0; i < list.length; i++) {
        const p = list[i] || {};
        const name = String((p.display_name || p.name) || '').trim() || 'provider';
        const status = String(p.status || '').trim().toLowerCase() || 'unknown';
        failing.push(name + ' (' + status + ')');
      }
      if (!failing.length) return 'No managed provider failures.';
      return 'Failing providers: ' + failing.join(', ');
    },
    providerHasProblem(provider) {
      const p = provider || {};
      const s = String((p && p.status) || '').trim().toLowerCase();
      return s === 'offline' || s === 'auth problem' || s === 'blocked';
    },
    chartColor(seed) {
      let h = 0;
      const str = String(seed || '');
      for (let i = 0; i < str.length; i++) h = (h * 31 + str.charCodeAt(i)) >>> 0;
      const hue = h % 360;
      return 'hsl(' + hue + ', 70%, 55%)';
    },
    renderUsageTimelineChart(buckets, groupField) {
      const canvas = document.getElementById('modelUsageChart');
      if (!canvas || typeof Chart === 'undefined') return;
      const now = Date.now();
      const rangeMs = Math.max(1, Number(this.statsRangeHours || 8)) * 3600 * 1000;
      const bucketMs = this.usageBucketMs(buckets);
      const endBucket = Math.floor(now / bucketMs) * bucketMs;
      const startBucket = Math.floor((endBucket - rangeMs) / bucketMs) * bucketMs;
      const byGroup = {};
      const field = String(groupField || 'model').trim();
      (Array.isArray(buckets) ? buckets : []).forEach((b) => {
        const raw = String(b[field] || '').trim();
        const name = raw || '(unknown)';
        const tRaw = new Date(String(b.start_at || '')).getTime();
        if (!Number.isFinite(tRaw)) return;
        const t = Math.floor(tRaw / bucketMs) * bucketMs;
        if (t < startBucket || t > endBucket) return;
        const y = Number(b.total_tokens || 0);
        if (!byGroup[name]) byGroup[name] = {};
        byGroup[name][t] = Number(byGroup[name][t] || 0) + y;
      });
      const times = [];
      for (let t = startBucket; t <= endBucket; t += bucketMs) times.push(t);
      const labels = times.map((t) => {
        const d = new Date(t);
        const hh = String(d.getHours()).padStart(2, '0');
        const mm = String(d.getMinutes()).padStart(2, '0');
        return hh + ':' + mm;
      });
      const names = Object.keys(byGroup).sort((a, b) => {
        const sumA = Object.values(byGroup[a] || {}).reduce((acc, v) => acc + Number(v || 0), 0);
        const sumB = Object.values(byGroup[b] || {}).reduce((acc, v) => acc + Number(v || 0), 0);
        return (sumB - sumA) || a.localeCompare(b);
      });
      const datasets = names.map((name) => {
        const color = this.chartColor(name);
        const seriesMap = byGroup[name] || {};
        const data = times.map((t) => Number(seriesMap[t] || 0));
        return {
          label: name,
          data,
          borderColor: color,
          backgroundColor: color,
          borderWidth: 1
        };
      });
      if (this.usageChart) {
        try { this.usageChart.destroy(); } catch (_) {}
        this.usageChart = null;
      }
      this.usageChart = new Chart(canvas.getContext('2d'), {
        type: 'bar',
        data: {labels, datasets},
        options: {
          animation: false,
          maintainAspectRatio: false,
          scales: {
            x: {
              stacked: true
            },
            y: {
              beginAtZero: true,
              stacked: true,
              title: {display: true, text: 'Tokens'}
            }
          },
          plugins: {
            legend: {display: true, position: 'bottom'},
            tooltip: {
              callbacks: {
                title: (items) => {
                  if (!items || !items.length) return '';
                  return String(items[0].label || '');
                },
                label: (ctx) => (ctx.dataset.label + ': ' + Math.round(Number(ctx.raw || 0)) + ' tokens')
              }
            }
          }
        }
      });
    },
    renderQuotaChart(quotaMap) {
      const items = Object.values(quotaMap || {});
      if (!items.length) return '';
      const quotaSearch = String(this.quotaSearch || '').trim().toLowerCase();
      const groupField = String(this.usageChartGroupBy || 'model').trim().toLowerCase();
      const groups = {};
      items.forEach((q) => {
        const status = String(q.status || '');
        const providerName = String(q.display_name || q.provider || 'provider').trim();
        if (status !== 'ok') {
          const key = providerName || 'quota';
          if (!groups[key]) groups[key] = {name: key, provider: providerName, requestMetrics: [], tokenMetrics: [], other: [], status: status || 'unknown', error: String(q.error || '').trim()};
          if (!groups[key].status || groups[key].status === 'ok') groups[key].status = status || 'unknown';
          if (!groups[key].error) groups[key].error = String(q.error || '').trim();
          return;
        }
        const metrics = (Array.isArray(q.metrics) && q.metrics.length)
          ? q.metrics
          : [{metered_feature: 'quota', window: '', left_percent: q.left_percent, reset_at: q.reset_at}];
        metrics.forEach((m) => {
          const feature = String(m.metered_feature || '').trim();
          const windowLabel = String(m.window || '').trim();
          let groupName = providerName;
          if (groupField === 'provider') {
            groupName = providerName;
          } else if (feature && !feature.toLowerCase().includes('request') && !feature.toLowerCase().includes('token')) {
            groupName = feature;
          }
          const key = groupName || providerName || 'quota';
          if (!groups[key]) groups[key] = {name: key, provider: providerName, requestMetrics: [], tokenMetrics: [], other: [], status: 'ok', error: ''};
          const kind = this.quotaMetricKind(m);
          if (kind === 'requests') groups[key].requestMetrics.push(m);
          else if (kind === 'tokens') groups[key].tokenMetrics.push(m);
          else groups[key].other.push(m);
        });
      });
      const visibleGroups = Object.values(groups)
        .filter((g) => {
          if (!quotaSearch) return true;
          const haystack = [
            String(g.name || ''),
            String(g.provider || ''),
            String(g.status || ''),
            String(g.error || '')
          ].join(' ').toLowerCase();
          return haystack.includes(quotaSearch);
        })
        .sort((a, b) => String(a.name || '').localeCompare(String(b.name || '')));
      const cards = visibleGroups.map((g) => {
        if (String(g.status || '').toLowerCase() !== 'ok') {
          const statusNorm = String(g.status || '').trim().toLowerCase();
          const isLoading = statusNorm === 'loading';
          const rawMsg = String(g.error || '').trim();
          const msg = this.escapeHtml(rawMsg);
          const statusLabel = isLoading ? 'Loading quota...' : 'Quota unavailable';
          const msgClass = isLoading ? 'small text-body-secondary mt-1' : 'small text-danger mt-1';
          return '<div class="border rounded p-2 bg-body">' +
            '<div class="fw-semibold small mb-2" title="' + this.escapeHtml(g.name || 'quota') + '" style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">' + this.escapeHtml(g.name || 'quota') + '</div>' +
            '<div class="small text-body-secondary">' + this.escapeHtml(statusLabel) + '</div>' +
            (msg ? ('<div class="' + msgClass + '">' + msg + '</div>') : '') +
          '</div>';
        }
        const ringColorFromUsed = (used) => (used >= 90 ? '#dc3545' : (used >= 70 ? '#fd7e14' : '#198754'));
        const metricLabel = (metric, fallback) => {
          const m = metric || {};
          const feature = String(m.metered_feature || '').trim();
          const windowLabel = String(m.window || '').trim();
          if (feature && windowLabel) return feature + ' · ' + windowLabel;
          if (feature) return feature;
          if (windowLabel) return fallback + ' · ' + windowLabel;
          return fallback;
        };
        const renderCircle = (metric, label) => {
          if (!metric) return '';
          const left = Number(this.metricLeftPercent(metric) || 0);
          const used = Math.max(0, Math.min(100, 100 - left));
          const ringColor = ringColorFromUsed(used);
          const tip = this.escapeHtml(this.quotaTooltipText(metric));
          const resetAge = this.formatUntil(metric && metric.reset_at);
          return '<div class="d-flex align-items-center gap-2">' +
            '<div' + (tip ? (' title="' + tip + '"') : '') + ' style="width:62px;height:62px;border-radius:999px;background:conic-gradient(' + ringColor + ' ' + used + '%, rgba(128,128,128,0.25) 0);display:flex;align-items:center;justify-content:center;cursor:help;">' +
              '<div class="bg-body rounded-circle d-flex align-items-center justify-content-center fw-semibold" style="width:48px;height:48px;font-size:12px;">' + Math.round(left) + '%</div>' +
            '</div>' +
            '<div class="small"><div class="fw-semibold">' + this.escapeHtml(label) + '</div><div class="text-body-secondary">' + (resetAge ? ('resets ' + this.escapeHtml(resetAge)) : 'reset unknown') + '</div></div>' +
          '</div>';
        };
        const renderDualCircle = (reqMetric, tokMetric, label) => {
          if (!reqMetric || !tokMetric) return '';
          const reqLeft = Number(this.metricLeftPercent(reqMetric) || 0);
          const tokLeft = Number(this.metricLeftPercent(tokMetric) || 0);
          const reqUsed = Math.max(0, Math.min(100, 100 - reqLeft));
          const tokUsed = Math.max(0, Math.min(100, 100 - tokLeft));
          const reqColor = ringColorFromUsed(reqUsed);
          const tokColor = ringColorFromUsed(tokUsed);
          const reqReset = this.formatUntil(reqMetric && reqMetric.reset_at);
          const tokReset = this.formatUntil(tokMetric && tokMetric.reset_at);
          const tip = this.escapeHtml('Tokens: ' + this.quotaTooltipText(tokMetric) + ' | Requests: ' + this.quotaTooltipText(reqMetric));
          return '<div class="d-flex align-items-center gap-2">' +
            '<div' + (tip ? (' title="' + tip + '"') : '') + ' style="width:72px;height:72px;border-radius:999px;background:conic-gradient(' + tokColor + ' ' + tokUsed + '%, rgba(128,128,128,0.25) 0);display:flex;align-items:center;justify-content:center;cursor:help;">' +
              '<div class="bg-body rounded-circle d-flex align-items-center justify-content-center" style="width:54px;height:54px;">' +
                '<div style="width:42px;height:42px;border-radius:999px;background:conic-gradient(' + reqColor + ' ' + reqUsed + '%, rgba(96,96,96,0.38) 0);display:flex;align-items:center;justify-content:center;">' +
                  '<div class="bg-body rounded-circle d-flex align-items-center justify-content-center fw-semibold" style="width:28px;height:28px;font-size:9px;">' + Math.round(reqLeft) + '%</div>' +
                '</div>' +
              '</div>' +
            '</div>' +
            '<div class="small">' +
              '<div class="fw-semibold">' + this.escapeHtml(label || 'Tokens') + ' ' + Math.round(tokLeft) + '%</div>' +
              '<div class="text-body-secondary">' + (tokReset ? ('resets ' + this.escapeHtml(tokReset)) : 'reset unknown') + '</div>' +
              '<div class="fw-semibold mt-1">Requests ' + Math.round(reqLeft) + '%</div>' +
              '<div class="text-body-secondary">' + (reqReset ? ('resets ' + this.escapeHtml(reqReset)) : 'reset unknown') + '</div>' +
            '</div>' +
          '</div>';
        };
        const requestMetrics = Array.isArray(g.requestMetrics) ? g.requestMetrics : [];
        const tokenMetrics = Array.isArray(g.tokenMetrics) ? g.tokenMetrics : [];
        const windows = {};
        requestMetrics.forEach((m) => {
          const key = String((m && m.window) || '').trim() || 'quota';
          if (!windows[key]) windows[key] = {request: null, token: null};
          windows[key].request = m;
        });
        tokenMetrics.forEach((m) => {
          const key = String((m && m.window) || '').trim() || 'quota';
          if (!windows[key]) windows[key] = {request: null, token: null};
          windows[key].token = m;
        });
        const metricBlocks = [];
        Object.keys(windows).sort().forEach((w) => {
          const pair = windows[w] || {};
          if (pair.request && pair.token) {
            metricBlocks.push(renderDualCircle(pair.request, pair.token, metricLabel(pair.token, 'Tokens')));
            return;
          }
          if (pair.request) {
            metricBlocks.push(renderCircle(pair.request, metricLabel(pair.request, 'Requests')));
            return;
          }
          if (pair.token) {
            metricBlocks.push(renderCircle(pair.token, metricLabel(pair.token, 'Tokens')));
          }
        });
        (Array.isArray(g.other) ? g.other : []).forEach((m) => {
          metricBlocks.push(renderCircle(m, metricLabel(m, 'Quota')));
        });
        const fallbackCircle = metricBlocks.length ? '' : renderCircle(null, 'Quota');
        const subtitle = (String(g.provider || '').trim() && String(g.provider || '').trim() !== String(g.name || '').trim())
          ? ('<div class="small text-body-secondary text-break">' + this.escapeHtml(g.provider) + '</div>')
          : '';
        return '<div class="border rounded p-2 bg-body">' +
          subtitle +
          '<div class="fw-semibold small mb-2" title="' + this.escapeHtml(g.name || 'quota') + '" style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis;">' + this.escapeHtml(g.name || 'quota') + '</div>' +
          '<div class="d-flex flex-wrap gap-3 align-items-center mt-2">' + (metricBlocks.join('') + fallbackCircle) + '</div>' +
        '</div>';
      }).join('');
      const emptyMsg = quotaSearch ? 'No matching quotas.' : 'No quota data.';
      return '<div class="mt-2" style="width:100%;display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:.5rem;align-items:stretch;">' + (cards || ('<div class="small text-body-secondary">' + this.escapeHtml(emptyMsg) + '</div>')) + '</div>';
    },
    renderUsageChart(sourceMap, title) {
      const colors = ['#0d6efd', '#198754', '#dc3545', '#fd7e14', '#20c997', '#6f42c1', '#0dcaf0', '#ffc107', '#6c757d', '#6610f2'];
      let items = Object.entries(sourceMap || {}).map(([name, count]) => ({name, count: Number(count || 0)})).filter((x) => x.count > 0);
      items.sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
      if (items.length > 8) items = items.slice(0, 8);
      if (items.length === 0) {
        return '<div class="border rounded p-3 bg-body"><div class="fw-semibold mb-2">' + this.escapeHtml(title) + '</div><div class="small text-body-secondary">No usage yet.</div></div>';
      }
      const maxCount = items[0].count || 1;
      const rows = items.map((item, i) => {
        const color = colors[i % colors.length];
        const w = Math.max(3, Math.round((item.count / maxCount) * 100));
        return '<div class="mb-2">' +
          '<div class="d-flex justify-content-between small mb-1">' +
            '<span><span class="d-inline-block me-1" style="width:10px;height:10px;border-radius:2px;background:' + color + ';"></span>' + this.escapeHtml(item.name) + '</span>' +
            '<span class="text-body-secondary">' + item.count + '</span>' +
          '</div>' +
          '<div class="progress" style="height:8px;"><div class="progress-bar" role="progressbar" style="width:' + w + '%;background:' + color + ';"></div></div>' +
        '</div>';
      }).join('');
      return '<div class="border rounded p-3 bg-body"><div class="fw-semibold mb-2">' + this.escapeHtml(title) + '</div>' + rows + '</div>';
    },
    aggregateUsageRowsBy(buckets, field) {
      const now = Date.now();
      const rangeMs = Math.max(1, Number(this.statsRangeHours || 8)) * 3600 * 1000;
      const cutoff = now - rangeMs;
      const map = {};
      (Array.isArray(buckets) ? buckets : []).forEach((b) => {
        const t = new Date(String(b.start_at || '')).getTime();
        if (!Number.isFinite(t) || t < cutoff) return;
        const raw = String(b[field] || '').trim();
        const name = raw || '(unknown)';
        if (!map[name]) map[name] = {name, requests: 0, pp: 0, tg: 0};
        map[name].requests += Number(b.requests || 0);
        map[name].pp += Number(b.prompt_tokens || 0);
        map[name].tg += Number(b.completion_tokens || 0);
      });
      const rows = Object.values(map).filter((x) => x.requests > 0 || x.pp > 0 || x.tg > 0);
      rows.sort((a, b) => (b.requests - a.requests) || (b.pp - a.pp) || (b.tg - a.tg) || a.name.localeCompare(b.name));
      return rows;
    },
    renderUsageMetricTable(title, rows, nameHeader) {
      const bodyRows = (Array.isArray(rows) ? rows : []).slice(0, 50).map((r) => {
        return '<tr>' +
          '<td class="text-break">' + this.escapeHtml(r.name || '') + '</td>' +
          '<td class="text-end">' + Number(r.requests || 0) + '</td>' +
          '<td class="text-end">' + Number(r.pp || 0) + '</td>' +
          '<td class="text-end">' + Number(r.tg || 0) + '</td>' +
        '</tr>';
      }).join('');
      return '<div class="border rounded p-3 bg-body">' +
        '<div class="fw-semibold mb-2">' + this.escapeHtml(title) + '</div>' +
        '<div class="table-responsive">' +
          '<table class="table table-sm align-middle mb-0">' +
            '<thead><tr><th>' + this.escapeHtml(nameHeader) + '</th><th class="text-end">Requests</th><th class="text-end">PP</th><th class="text-end">TG</th></tr></thead>' +
            '<tbody>' + (bodyRows || '<tr><td colspan="4" class="text-body-secondary">No usage yet.</td></tr>') + '</tbody>' +
          '</table>' +
        '</div>' +
      '</div>';
    },
    renderModelStatsTable(buckets) {
      const now = Date.now();
      const rangeMs = Math.max(1, Number(this.statsRangeHours || 8)) * 3600 * 1000;
      const cutoff = now - rangeMs;
      const perModel = {};
      (Array.isArray(buckets) ? buckets : []).forEach((b) => {
        const model = String(b.model || '').trim();
        if (!model) return;
        const t = new Date(String(b.start_at || '')).getTime();
        if (!Number.isFinite(t) || t < cutoff) return;
        if (!perModel[model]) perModel[model] = {
          model,
          requests: 0,
          prompt_tokens: 0,
          completion_tokens: 0,
          total_tokens: 0,
          latency_ms_sum: 0,
          prompt_tps_sum: 0,
          generation_tps_sum: 0
        };
        const x = perModel[model];
        x.requests += Number(b.requests || 0);
        x.prompt_tokens += Number(b.prompt_tokens || 0);
        x.completion_tokens += Number(b.completion_tokens || 0);
        x.total_tokens += Number(b.total_tokens || 0);
        x.latency_ms_sum += Number(b.latency_ms_sum || 0);
        x.prompt_tps_sum += Number(b.prompt_tps_sum || 0);
        x.generation_tps_sum += Number(b.generation_tps_sum || 0);
      });
      const rowsData = Object.values(perModel).filter((x) => x.requests > 0);
      rowsData.sort((a, b) => (b.total_tokens - a.total_tokens) || (b.requests - a.requests) || a.model.localeCompare(b.model));
      if (!rowsData.length) return '';
      const rows = rowsData.map((x) => {
        const avgLatency = x.requests > 0 ? (x.latency_ms_sum / x.requests).toFixed(1) : '0.0';
        const avgPromptTPS = x.requests > 0 ? (x.prompt_tps_sum / x.requests).toFixed(2) : '0.00';
        const avgGenTPS = x.requests > 0 ? (x.generation_tps_sum / x.requests).toFixed(2) : '0.00';
        const color = this.chartColor(x.model);
        return '<tr>' +
          '<td class="text-break"><span class="d-inline-block me-2 align-middle" style="width:10px;height:10px;border-radius:2px;background:' + color + ';"></span><span class="align-middle">' + this.escapeHtml(x.model) + '</span></td>' +
          '<td class="text-end">' + Number(x.requests || 0) + '</td>' +
          '<td class="text-end">' + Number(x.prompt_tokens || 0) + '</td>' +
          '<td class="text-end">' + Number(x.completion_tokens || 0) + '</td>' +
          '<td class="text-end">' + this.escapeHtml(avgLatency) + '</td>' +
          '<td class="text-end">' + this.escapeHtml(avgPromptTPS) + '</td>' +
          '<td class="text-end">' + this.escapeHtml(avgGenTPS) + '</td>' +
        '</tr>';
      }).join('');
      return '<div class="border rounded p-3 bg-body mt-2">' +
        '<div class="fw-semibold mb-2">Per-model usage</div>' +
        '<div class="table-responsive">' +
          '<table class="table table-sm align-middle mb-0">' +
            '<thead><tr><th>Model</th><th class="text-end">Requests</th><th class="text-end">Prompt</th><th class="text-end">Generated</th><th class="text-end">Avg latency ms</th><th class="text-end">Avg prompt tok/s</th><th class="text-end">Avg gen tok/s</th></tr></thead>' +
            '<tbody>' + rows + '</tbody>' +
          '</table>' +
        '</div>' +
      '</div>';
    },
    renderProviders() {
      const sortedProviders = (Array.isArray(this.providers) ? this.providers.slice() : []).sort((a, b) => {
        const ap = this.providerHasProblem(a) ? 1 : 0;
        const bp = this.providerHasProblem(b) ? 1 : 0;
        if (ap !== bp) return bp - ap;
        const an = String((a && (a.display_name || a.name)) || '').toLowerCase();
        const bn = String((b && (b.display_name || b.name)) || '').toLowerCase();
        return an.localeCompare(bn);
      });
      const rows = sortedProviders.map((p) => {
        const rawName = String(p.name || '').trim();
        const name = this.escapeHtml(rawName);
        const display = this.escapeHtml(p.display_name || p.name);
        const iconName = this.escapeHtml(String(p.provider_type || p.name || '').trim());
        const iconCls = this.providerIconNeedsDarkInvert(iconName) ? ' provider-icon-invert-dark' : '';
        const providerLabel = '<span class="d-inline-flex align-items-center gap-2"><img src="' + this.providerIconSrc(iconName) + '" class="' + iconCls.trim() + '" onerror="this.onerror=null;this.src=&quot;data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==&quot;" alt="" width="16" height="16" style="object-fit:contain;" /><span>' + display + '</span></span>';
        const statusRaw = String(p.status || 'unknown');
        const ageText = this.formatAge(p.checked_at);
        const modelCount = Number(p.model_count || 0);
        const pricedCount = Number(p.priced_models || 0);
        const freeCount = Number(p.free_models || 0);
        let totalModels = modelCount;
        if (totalModels <= 0 && pricedCount > 0) totalModels = pricedCount;
        const unknownCount = Math.max(0, totalModels - pricedCount);
        const paidCount = Math.max(0, pricedCount - freeCount);
        const totalModelsParts = [];
        if (freeCount > 0) totalModelsParts.push(freeCount + ' free');
        if (paidCount > 0) totalModelsParts.push(paidCount + ' paid');
        if (unknownCount > 0) totalModelsParts.push(unknownCount + ' unknown');
        const totalModelsText = totalModelsParts.length > 0 ? totalModelsParts.join(', ') : '0';
        const responseMSValue = Number(p.response_ms || 0);
        const responseMS = responseMSValue > 0 ? responseMSValue + 'ms' : '';
        let status = this.escapeHtml(statusRaw);
        if (statusRaw === 'online') {
          const detail = [responseMS, ageText].filter(Boolean).join(' ');
          status = this.escapeHtml(detail || 'online');
        }
        const statusProblem = this.providerHasProblem(p);
        const statusAlert = statusProblem
          ? '<span class="ms-1 d-inline-flex align-items-center justify-content-center rounded-circle bg-danger text-white fw-bold align-middle" style="width:1.05rem;height:1.05rem;line-height:1.05rem;font-size:0.74rem;" title="Provider has a problem">!</span>'
          : '';
        const pricingAge = this.formatAge(p.pricing_last_update);
        const pricingUpdated = this.escapeHtml(pricingAge || '');
        const actionNameAttr = this.escapeHtml(rawName);
        const actions = p.managed
          ? '<div class="d-flex justify-content-end gap-1">' +
              '<button class="icon-btn" type="button" title="Edit provider" aria-label="Edit provider" data-provider-name="' + actionNameAttr + '" onclick="window.__adminEditProvider(this.getAttribute(\'data-provider-name\'))">' + this.editActionIconSVG() + '</button>' +
              '<button class="icon-btn icon-btn-danger" type="button" title="Delete provider" aria-label="Delete provider" data-provider-name="' + actionNameAttr + '" onclick="window.__adminRemoveProvider(this.getAttribute(\'data-provider-name\'))"><svg xmlns="http://www.w3.org/2000/svg" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true"><path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5Zm2.5.5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6Zm2 .5a.5.5 0 0 1 1 0v6a.5.5 0 0 1-1 0V6Z"/><path d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 1 1 0-2H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1ZM4 4v9a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4H4Z"/></svg></button>' +
            '</div>'
          : '<span class="badge text-bg-light border">Auto</span>';
        return '<tr>' +
          '<td>' + providerLabel + '</td>' +
          '<td>' + status + statusAlert + '</td>' +
          '<td>' + this.escapeHtml(totalModelsText) + '</td>' +
          '<td><small>' + pricingUpdated + '</small></td>' +
          '<td>' + actions + '</td>' +
          '</tr>';
      });
      const page = this.paginateRows(rows, this.providersPage, this.providersPageSize);
      this.providersPage = page.page;
      this.providersPageSize = page.pageSize;
      const tableRows = page.rows.join('');
      this.providersTableHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr><th>Name</th><th>Status</th><th>Models</th><th>Pricing Updated</th><th></th></tr></thead>' +
          '<tbody>' + (tableRows || '<tr><td colspan="5" class="text-body-secondary">No providers configured.</td></tr>') + '</tbody>' +
        '</table>' +
        this.renderProvidersPager(page.totalRows, page.page, page.totalPages, page.pageSize);
    },
    renderAccessTokenRoleBadge(role) {
      const r = String(role || '').trim().toLowerCase();
      if (r === 'admin') return '<span class="badge text-bg-danger">admin</span>';
      if (r === 'keymaster') return '<span class="badge text-bg-info">keymaster</span>';
      return '<span class="badge text-bg-secondary">inferrer</span>';
    },
    formatAccessTokenCountShort(n) {
      const v = Number(n || 0);
      if (!Number.isFinite(v) || v <= 0) return '0';
      if (v >= 1000000000) return Math.round(v / 100000000) / 10 + 'b';
      if (v >= 1000000) return Math.round(v / 100000) / 10 + 'm';
      if (v >= 1000) return Math.round(v / 100) / 10 + 'k';
      return String(Math.round(v));
    },
    formatAccessTokenIntervalShort(sec) {
      const s = Math.max(0, Number(sec || 0));
      if (!Number.isFinite(s) || s <= 0) return '';
      if (s % 86400 === 0) return (s / 86400) + 'd';
      if (s % 3600 === 0) return (s / 3600) + 'h';
      if (s % 60 === 0) return (s / 60) + 'm';
      return s + 's';
    },
    renderAccessTokenQuotaSummary(quotaObj) {
      const q = quotaObj && typeof quotaObj === 'object' ? quotaObj : {};
      const qr = q && q.requests ? q.requests : {};
      const qt = q && q.tokens ? q.tokens : {};
      const parts = [];
      const full = [];
      const reqLimit = Number(qr.limit || 0);
      const reqInt = Number(qr.interval_seconds || 0);
      if (Number.isFinite(reqLimit) && reqLimit > 0) {
        const shortInt = this.formatAccessTokenIntervalShort(reqInt);
        const left = this.formatAccessTokenCountShort(reqLimit) + ' req' + (shortInt ? ('/' + shortInt) : '');
        parts.push(left);
        full.push(Math.round(reqLimit) + ' requests' + (shortInt ? (' per ' + shortInt) : ''));
      }
      const tokLimit = Number(qt.limit || 0);
      const tokInt = Number(qt.interval_seconds || 0);
      if (Number.isFinite(tokLimit) && tokLimit > 0) {
        const shortInt = this.formatAccessTokenIntervalShort(tokInt);
        const left = this.formatAccessTokenCountShort(tokLimit) + ' tok' + (shortInt ? ('/' + shortInt) : '');
        parts.push(left);
        full.push(Math.round(tokLimit) + ' tokens' + (shortInt ? (' per ' + shortInt) : ''));
      }
      if (!parts.length) return {short: 'no quota', full: 'No quota'};
      return {short: parts.join(', '), full: full.join(', ')};
    },
    flattenAccessTokens(items) {
      const byParent = {};
      const byID = {};
      items.forEach((t) => {
        const parent = t.parent_id;
        if (!byParent[parent]) byParent[parent] = [];
        byParent[parent].push(t);
        if (t.id) byID[t.id] = t;
      });
      const roots = items.filter((t) => !t.parent_id || !byID[t.parent_id]);
      roots.sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
      Object.keys(byParent).forEach((k) => {
        byParent[k].sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
      });
      const out = [];
      const walk = (token, depth) => {
        out.push({token, depth});
        (byParent[token.id] || []).forEach((child) => walk(child, depth + 1));
      };
      roots.forEach((r) => walk(r, 0));
      return out;
    },
    renderAccessTokenRow(token, depth) {
      const t = token || {};
      const id = this.escapeHtml(t.id);
      const name = this.escapeHtml(t.name);
      const expiryRaw = String(t.expires_at || '').trim();
      const expiry = this.escapeHtml(this.formatRelativeShort(expiryRaw, 'none'));
      const expiryTitle = this.escapeHtml(this.formatTimestamp(expiryRaw));
      const quota = this.renderAccessTokenQuotaSummary(t.quota);
      const quotaShort = this.escapeHtml(quota.short);
      const quotaFull = this.escapeHtml(quota.full);
      const indent = depth > 0 ? (' style="padding-left:' + (depth * 20) + 'px;"') : '';
      const marker = depth > 0 ? '<span class="text-body-secondary me-1">↳</span>' : '';
      return '<tr>' +
        '<td' + indent + '>' + marker + name + '</td>' +
        '<td>' + this.renderAccessTokenRoleBadge(t.role) + '</td>' +
        '<td title="' + quotaFull + '">' + quotaShort + '</td>' +
        '<td title="' + expiryTitle + '">' + expiry + '</td>' +
        '<td class="text-end">' +
          '<button class="icon-btn me-1" type="button" title="Edit token" aria-label="Edit token" data-token-id="' + id + '" onclick="window.__adminEditAccessToken(this.getAttribute(\'data-token-id\'))">' +
            this.editActionIconSVG() +
          '</button>' +
          '<button class="icon-btn icon-btn-danger" type="button" title="Delete token" aria-label="Delete token" data-token-id="' + id + '" onclick="window.__adminDeleteAccessToken(this.getAttribute(\'data-token-id\'))">' +
            '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="currentColor" viewBox="0 0 16 16" aria-hidden="true"><path d="M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5Zm2.5.5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6Zm2 .5a.5.5 0 0 1 1 0v6a.5.5 0 0 1-1 0V6Z"/><path d="M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 1 1 0-2H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1ZM4 4v9a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4H4Z"/></svg>' +
          '</button>' +
        '</td>' +
      '</tr>';
    },
    renderAccessTokens() {
      const items = (this.accessTokens || []).map((t) => ({
        id: String(t.id || '').trim(),
        name: String(t.name || '').trim() || 'Token',
        role: String(t.role || '').trim().toLowerCase() || 'inferrer',
        parent_id: String(t.parent_id || '').trim(),
        expires_at: String(t.expires_at || '').trim(),
        quota: (t && typeof t.quota === 'object' && t.quota) ? t.quota : null
      }));
      const flattened = this.flattenAccessTokens(items);
      const allRows = flattened.map((entry) => this.renderAccessTokenRow(entry.token, entry.depth));
      const page = this.paginateRows(allRows, this.accessPage, this.accessPageSize);
      this.accessPage = page.page;
      this.accessPageSize = page.pageSize;
      this.accessTokensTableHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr><th>Name</th><th>Type</th><th>Quota</th><th>Expiry</th><th></th></tr></thead>' +
          '<tbody>' + (page.rows.join('') || '<tr><td colspan="5" class="text-body-secondary">No tokens found.</td></tr>') + '</tbody>' +
        '</table>';
      this.accessTokensPagerHtml = this.renderAccessPager(page.totalRows, page.page, page.totalPages, page.pageSize);
      window.__adminDeleteAccessToken = (id) => this.removeAccessToken(id);
      window.__adminEditAccessToken = (id) => this.openEditAccessTokenModal(id);
    },
    randomTokenKey() {
      const prefix = 'tr_';
      try {
        const bytes = new Uint8Array(48);
        if (window.crypto && window.crypto.getRandomValues) {
          window.crypto.getRandomValues(bytes);
        } else {
          for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
        }
        let b64 = btoa(String.fromCharCode(...bytes)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
        if (b64.length < 64) b64 = (b64 + b64 + b64).slice(0, 64);
        return prefix + b64.slice(0, 64);
      } catch (_) {
        return prefix + String(Date.now()) + '_' + Math.random().toString(36).slice(2).padEnd(48, 'x');
      }
    },
    openAddAccessTokenModal() {
      this.showAddAccessTokenModal = true;
      const initialSetup = this.requiresInitialTokenSetup && (!Array.isArray(this.accessTokens) || this.accessTokens.length === 0);
      this.accessTokenDraft = {
        id:'',
        name: initialSetup ? 'Admin' : '',
        key:this.randomTokenKey(),
        role: initialSetup ? 'admin' : 'inferrer',
        expiry_preset:'never',
        expires_at:'',
        quota_enabled:false,
        quota_requests_limit:'',
        quota_requests_interval_seconds:'0',
        quota_tokens_limit:'',
        quota_tokens_interval_seconds:'0',
        disable_localhost_no_auth: initialSetup
      };
    },
    openEditAccessTokenModal(id) {
      const sid = String(id || '').trim();
      const t = (this.accessTokens || []).find((x) => String(x.id || '').trim() === sid);
      if (!t) return;
      const expiresAt = String(t.expires_at || '').trim();
      this.showAddAccessTokenModal = true;
      const q = t && t.quota ? t.quota : {};
      const qr = q && q.requests ? q.requests : {};
      const qt = q && q.tokens ? q.tokens : {};
      const hasQuota = Number(qr.limit || 0) > 0 || Number(qt.limit || 0) > 0;
      this.accessTokenDraft = {
        id: sid,
        name: String(t.name || '').trim(),
        key: '',
        role: String(t.role || 'inferrer').trim().toLowerCase() || 'inferrer',
        expiry_preset: expiresAt ? 'custom' : 'never',
        expires_at: expiresAt,
        quota_enabled: hasQuota,
        quota_requests_limit: Number(qr.limit || 0) > 0 ? String(qr.limit) : '',
        quota_requests_interval_seconds: String(Number(qr.interval_seconds || 0)),
        quota_tokens_limit: Number(qt.limit || 0) > 0 ? String(qt.limit) : '',
        quota_tokens_interval_seconds: String(Number(qt.interval_seconds || 0))
      };
    },
    closeAddAccessTokenModal() {
      if (this.requiresInitialTokenSetup) this.initialSetupDialogDismissed = true;
      this.showAddAccessTokenModal = false;
      this.accessTokenDraft = {id:'', name:'', key:'', role:'inferrer', expiry_preset:'never', expires_at:'', quota_enabled:false, quota_requests_limit:'', quota_requests_interval_seconds:'0', quota_tokens_limit:'', quota_tokens_interval_seconds:'0', disable_localhost_no_auth:false};
    },
    persistInitialSetupDialogDismissed() {
      try {
        if (this.initialSetupDialogDismissed) window.localStorage.setItem(this.initialSetupDialogDismissCacheKey, '1');
        else window.localStorage.removeItem(this.initialSetupDialogDismissCacheKey);
      } catch (_) {}
    },
    restoreInitialSetupDialogDismissed() {
      try {
        this.initialSetupDialogDismissed = window.localStorage.getItem(this.initialSetupDialogDismissCacheKey) === '1';
      } catch (_) {
        this.initialSetupDialogDismissed = false;
      }
    },
    dismissInitialSetupDialogForever() {
      this.initialSetupDialogDismissed = true;
      this.persistInitialSetupDialogDismissed();
      this.closeAddAccessTokenModal();
    },
    buildAccessTokenQuotaPayload() {
      if (!this.accessTokenDraft.quota_enabled) return null;
      const reqLimit = Math.max(0, Number(this.accessTokenDraft.quota_requests_limit || 0));
      const reqInterval = Math.max(0, Number(this.accessTokenDraft.quota_requests_interval_seconds || 0));
      const tokLimit = Math.max(0, Number(this.accessTokenDraft.quota_tokens_limit || 0));
      const tokInterval = Math.max(0, Number(this.accessTokenDraft.quota_tokens_interval_seconds || 0));
      const quota = {};
      if (reqLimit > 0) {
        quota.requests = {
          limit: Math.floor(reqLimit),
          interval_seconds: Math.floor(reqInterval)
        };
      }
      if (tokLimit > 0) {
        quota.tokens = {
          limit: Math.floor(tokLimit),
          interval_seconds: Math.floor(tokInterval)
        };
      }
      if (!quota.requests && !quota.tokens) return null;
      return quota;
    },
    regenerateAccessTokenKey() {
      this.accessTokenDraft.key = this.randomTokenKey();
    },
    async copyAccessTokenKey() {
      const key = String(this.accessTokenDraft.key || '').trim();
      if (!key) return;
      try {
        await navigator.clipboard.writeText(key);
        this.toastSuccess('Key copied.');
      } catch (_) {
        this.toastError('Could not copy key.');
      }
    },
    expiryPresetToRFC3339(preset) {
      const p = String(preset || 'never').trim();
      if (p === 'never') return '';
      const now = new Date();
      const out = new Date(now.getTime());
      if (p === '1d') out.setDate(out.getDate() + 1);
      else if (p === '1w') out.setDate(out.getDate() + 7);
      else if (p === '1m') out.setMonth(out.getMonth() + 1);
      else if (p === '3m') out.setMonth(out.getMonth() + 3);
      else if (p === '12m') out.setMonth(out.getMonth() + 12);
      else return '';
      return out.toISOString();
    },
    renderModelsCatalog() {
      const search = this.modelsSearch.trim().toLowerCase();
      let rows = (this.modelsCatalog || []).filter((m) => {
        if (!search) return true;
        return String(m.provider || '').toLowerCase().includes(search) || String(m.model || '').toLowerCase().includes(search);
      });
      if (this.modelsFreeOnly) {
        rows = rows.filter((m) => Number(m.input_per_1m) === 0 && Number(m.output_per_1m) === 0);
      }
      const sortBy = this.modelsSortBy;
      const dir = this.modelsSortAsc ? 1 : -1;
      rows.sort((a, b) => {
        const av = a[sortBy];
        const bv = b[sortBy];
        if (sortBy === 'input_per_1m' || sortBy === 'output_per_1m') {
          return (Number(av || 0) - Number(bv || 0)) * dir;
        }
        return String(av || '').localeCompare(String(bv || '')) * dir;
      });
      const htmlRows = rows.map((m) => {
        const providerDisplay = this.escapeHtml(m.provider_display_name || m.provider);
        const iconName = this.escapeHtml(String(m.provider_type || m.provider || '').trim());
        const iconCls = this.providerIconNeedsDarkInvert(iconName) ? ' provider-icon-invert-dark' : '';
        const providerLabel = '<span class="d-inline-flex align-items-center gap-2"><img src="' + this.providerIconSrc(iconName) + '" class="' + iconCls.trim() + '" onerror="this.onerror=null;this.src=&quot;data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==&quot;" alt="" width="16" height="16" style="object-fit:contain;" /><span>' + providerDisplay + '</span></span>';
        const model = this.escapeHtml(m.model || '-');
        const statusRaw = String(m.status || 'unknown');
        const ageText = this.formatAge(m.checked_at);
        const responseMSValue = Number(m.response_ms || 0);
        const responseMS = responseMSValue > 0 ? responseMSValue + 'ms' : '';
        let status = this.escapeHtml(statusRaw);
        if (statusRaw === 'online') {
          status = this.escapeHtml([responseMS, ageText].filter(Boolean).join(' ') || 'online');
        }
        const hasInput = !(m.input_per_1m === undefined || m.input_per_1m === null);
        const hasOutput = !(m.output_per_1m === undefined || m.output_per_1m === null);
        const inputNum = hasInput ? Number(m.input_per_1m) : null;
        const outputNum = hasOutput ? Number(m.output_per_1m) : null;
        const isFree = hasInput && hasOutput && inputNum === 0 && outputNum === 0;
        const input = this.formatPrice(inputNum, m.currency);
        const output = this.formatPrice(outputNum, m.currency);
        return '<tr>' +
          '<td>' + providerLabel + '</td>' +
          '<td>' + model + '</td>' +
          '<td>' + status + '</td>' +
          '<td class="text-end' + (isFree ? ' text-success fw-semibold' : '') + '">' + input + '</td>' +
          '<td class="text-end' + (isFree ? ' text-success fw-semibold' : '') + '">' + output + '</td>' +
          '</tr>';
      });
      const page = this.paginateRows(htmlRows, this.modelsPage, this.modelsPageSize);
      this.modelsPage = page.page;
      this.modelsPageSize = page.pageSize;
      const tableRows = page.rows.join('');
      const sortArrow = this.modelsSortAsc ? ' ▲' : ' ▼';
      const providerSort = this.modelsSortBy === 'provider' ? sortArrow : '';
      const modelSort = this.modelsSortBy === 'model' ? sortArrow : '';
      const inputSort = this.modelsSortBy === 'input_per_1m' ? sortArrow : '';
      const outputSort = this.modelsSortBy === 'output_per_1m' ? sortArrow : '';
      this.modelsTableHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr>' +
          '<th role="button" class="text-decoration-underline" onclick="window.__adminSortModels(\'provider\')">Provider' + providerSort + '</th>' +
          '<th role="button" class="text-decoration-underline" onclick="window.__adminSortModels(\'model\')">Model' + modelSort + '</th>' +
          '<th>Status</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortModels(\'input_per_1m\')">Input / 1M' + inputSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortModels(\'output_per_1m\')">Output / 1M' + outputSort + '</th>' +
          '</tr></thead>' +
          '<tbody>' + (tableRows || '<tr><td colspan="5" class="text-body-secondary">No models available.</td></tr>') + '</tbody>' +
        '</table>' +
        this.renderModelsPager(page.totalRows, page.page, page.totalPages, page.pageSize);
    },
    renderPerformanceCatalog() {
      const search = this.performanceSearch.trim().toLowerCase();
      let rows = (this.modelsCatalog || []).filter((m) => {
        if (Number(m.perf_requests || 0) <= 0) return false;
        if (!search) return true;
        return String(m.provider || '').toLowerCase().includes(search) || String(m.model || '').toLowerCase().includes(search);
      });
      const sortBy = this.performanceSortBy;
      const dir = this.performanceSortAsc ? 1 : -1;
      rows.sort((a, b) => {
        const av = a[sortBy];
        const bv = b[sortBy];
        if (sortBy === 'failure_rate' || sortBy === 'avg_prompt_tps' || sortBy === 'avg_generation_tps' || sortBy === 'prompt_tokens' || sortBy === 'completion_tokens') {
          return (Number(av || 0) - Number(bv || 0)) * dir;
        }
        return String(av || '').localeCompare(String(bv || '')) * dir;
      });
      const htmlRows = rows.map((m) => {
        const providerDisplay = this.escapeHtml(m.provider_display_name || m.provider);
        const iconName = this.escapeHtml(String(m.provider_type || m.provider || '').trim());
        const iconCls = this.providerIconNeedsDarkInvert(iconName) ? ' provider-icon-invert-dark' : '';
        const providerLabel = '<span class="d-inline-flex align-items-center gap-2"><img src="' + this.providerIconSrc(iconName) + '" class="' + iconCls.trim() + '" onerror="this.onerror=null;this.src=&quot;data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==&quot;" alt="" width="16" height="16" style="object-fit:contain;" /><span>' + providerDisplay + '</span></span>';
        const model = this.escapeHtml(m.model || '-');
        const promptTokens = Number(m.prompt_tokens || 0);
        const completionTokens = Number(m.completion_tokens || 0);
        const failureRateNum = Number(m.failure_rate || 0);
        const failureRate = Number.isFinite(failureRateNum) ? (failureRateNum.toFixed(1) + '%') : '-';
        const avgPP = Number(m.avg_prompt_tps || 0);
        const avgTG = Number(m.avg_generation_tps || 0);
        const avgPPText = (Number.isFinite(avgPP) && avgPP > 0) ? avgPP.toFixed(1) : '-';
        const avgTGText = (Number.isFinite(avgTG) && avgTG > 0) ? avgTG.toFixed(1) : '-';
        return '<tr>' +
          '<td>' + providerLabel + '</td>' +
          '<td>' + model + '</td>' +
          '<td class="text-end">' + promptTokens + '</td>' +
          '<td class="text-end">' + completionTokens + '</td>' +
          '<td class="text-end">' + this.escapeHtml(avgPPText) + '</td>' +
          '<td class="text-end">' + this.escapeHtml(avgTGText) + '</td>' +
          '<td class="text-end">' + this.escapeHtml(failureRate) + '</td>' +
          '</tr>';
      });
      const page = this.paginateRows(htmlRows, this.performancePage, this.performancePageSize);
      this.performancePage = page.page;
      this.performancePageSize = page.pageSize;
      const tableRows = page.rows.join('');
      const sortArrow = this.performanceSortAsc ? ' ▲' : ' ▼';
      const providerSort = this.performanceSortBy === 'provider' ? sortArrow : '';
      const modelSort = this.performanceSortBy === 'model' ? sortArrow : '';
      const ppTokSort = this.performanceSortBy === 'prompt_tokens' ? sortArrow : '';
      const tgTokSort = this.performanceSortBy === 'completion_tokens' ? sortArrow : '';
      const ppSort = this.performanceSortBy === 'avg_prompt_tps' ? sortArrow : '';
      const tgSort = this.performanceSortBy === 'avg_generation_tps' ? sortArrow : '';
      const failSort = this.performanceSortBy === 'failure_rate' ? sortArrow : '';
      this.performanceTableHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr>' +
          '<th role="button" class="text-decoration-underline" onclick="window.__adminSortPerformance(\'provider\')">Provider' + providerSort + '</th>' +
          '<th role="button" class="text-decoration-underline" onclick="window.__adminSortPerformance(\'model\')">Model' + modelSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortPerformance(\'prompt_tokens\')">PP Tokens' + ppTokSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortPerformance(\'completion_tokens\')">TG Tokens' + tgTokSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortPerformance(\'avg_prompt_tps\')">Avg PP/s' + ppSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortPerformance(\'avg_generation_tps\')">Avg TG/s' + tgSort + '</th>' +
          '<th role="button" class="text-decoration-underline text-end" onclick="window.__adminSortPerformance(\'failure_rate\')">Failure Rate' + failSort + '</th>' +
          '</tr></thead>' +
          '<tbody>' + (tableRows || '<tr><td colspan="7" class="text-body-secondary">No performance data.</td></tr>') + '</tbody>' +
        '</table>' +
        this.renderPerformancePager(page.totalRows, page.page, page.totalPages, page.pageSize);
    },
    renderModelsFreshness(fetchedAt, pricingUpdatedAt) {
      const fetched = fetchedAt ? new Date(fetchedAt) : null;
      const pricing = pricingUpdatedAt ? new Date(pricingUpdatedAt) : null;
      let fetchedText = 'unknown';
      if (fetched && !Number.isNaN(fetched.getTime())) {
        fetchedText = fetched.toLocaleString() + ' (' + this.formatRelativeShort(fetched.toISOString(), '-') + ')';
      }
      let pricingText = 'unknown';
      if (pricing && !Number.isNaN(pricing.getTime())) {
        pricingText = pricing.toLocaleString() + ' (' + this.formatRelativeShort(pricing.toISOString(), '-') + ')';
      }
      this.modelsFreshnessHtml = 'Catalog fetched: <strong>' + this.escapeHtml(fetchedText) + '</strong> · Pricing cache: <strong>' + this.escapeHtml(pricingText) + '</strong>';
    },
    toggleModelsSortDir() {
      this.modelsSortAsc = !this.modelsSortAsc;
      this.renderModelsCatalog();
    },
    sortModelsBy(col) {
      if (this.modelsSortBy === col) {
        this.modelsSortAsc = !this.modelsSortAsc;
      } else {
        this.modelsSortBy = col;
        this.modelsSortAsc = true;
      }
      this.renderModelsCatalog();
    },
    sortPerformanceBy(col) {
      if (this.performanceSortBy === col) this.performanceSortAsc = !this.performanceSortAsc;
      else {
        this.performanceSortBy = col;
        this.performanceSortAsc = true;
      }
      this.renderPerformanceCatalog();
    },
    toggleFreeOnly() {
      this.modelsFreeOnly = !this.modelsFreeOnly;
      this.persistModelsFreeOnly();
      this.modelsPage = 1;
      this.renderModelsCatalog();
    },
    async loadStats(force) {
      const sec = Math.max(1, Number(this.statsRangeHours || 8)) * 3600;
      const u = force ? ('/admin/api/stats?period_seconds=' + sec + '&force=1') : ('/admin/api/stats?period_seconds=' + sec);
      this.statsLoading = true;
      const hasExistingStats = !!(this.stats && typeof this.stats === 'object' && Object.keys(this.stats).length > 0);
      if (!hasExistingStats) {
        this.renderStatsLoading();
      }
      const r = await this.apiFetch(u, {headers:this.headers()});
      if (r.status === 401) {
        this.statsLoading = false;
        window.location = '/admin/login?next=/admin';
        return;
      }
      if (!r.ok) {
        this.statsLoading = false;
        this.statsSummaryHtml = '<div class="small text-danger">Failed to load usage stats.</div>';
        return;
      }
      this.stats = await r.json();
      this.statsLoading = false;
      this.renderStats();
    },
    async refreshQuota() {
      if (this.quotaRefreshInProgress) return;
      this.quotaRefreshInProgress = true;
      const startedAt = Date.now();
      try {
        await this.loadStats(true);
      } finally {
        const elapsed = Date.now() - startedAt;
        const minVisibleMs = 600;
        if (elapsed < minVisibleMs) {
          await new Promise((resolve) => setTimeout(resolve, minVisibleMs - elapsed));
        }
        this.quotaRefreshInProgress = false;
      }
    },
    async loadSecuritySettings() {
      const r = await this.apiFetch('/admin/api/settings/security', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json();
      this.allowLocalhostNoAuthEffective = !!body.allow_localhost_no_auth;
      this.allowHostDockerInternalNoAuthEffective = !!body.allow_host_docker_internal_no_auth;
      this.allowLocalhostNoAuth = this.allowLocalhostNoAuthEffective;
      this.allowHostDockerInternalNoAuth = this.allowHostDockerInternalNoAuthEffective;
      this.autoEnablePublicFreeModels = !!body.auto_enable_public_free_models;
      this.autoDetectLocalServers = (body.auto_detect_local_servers === undefined) ? true : !!body.auto_detect_local_servers;
      this.autoRemoveExpiredTokens = !!body.auto_remove_expired_tokens;
      this.autoRemoveEmptyQuotaTokens = !!body.auto_remove_empty_quota_tokens;
    },
    async saveSecuritySettings() {
      if (this.securitySaveInProgress) return;
      this.securitySaveInProgress = true;
      try {
        const payload = {
          allow_localhost_no_auth: !!this.allowLocalhostNoAuth,
          allow_host_docker_internal_no_auth: !!this.allowHostDockerInternalNoAuth,
          auto_enable_public_free_models: !!this.autoEnablePublicFreeModels,
          auto_detect_local_servers: !!this.autoDetectLocalServers,
          auto_remove_expired_tokens: !!this.autoRemoveExpiredTokens,
          auto_remove_empty_quota_tokens: !!this.autoRemoveEmptyQuotaTokens
        };
        const r = await this.apiFetch('/admin/api/settings/security', {method:'PUT', headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (r.ok) {
          const body = await r.json().catch(() => ({}));
          this.allowLocalhostNoAuthEffective = !!body.allow_localhost_no_auth;
          this.allowHostDockerInternalNoAuthEffective = !!body.allow_host_docker_internal_no_auth;
          this.allowLocalhostNoAuth = this.allowLocalhostNoAuthEffective;
          this.allowHostDockerInternalNoAuth = this.allowHostDockerInternalNoAuthEffective;
          this.autoEnablePublicFreeModels = !!body.auto_enable_public_free_models;
          this.autoDetectLocalServers = (body.auto_detect_local_servers === undefined) ? true : !!body.auto_detect_local_servers;
          this.autoRemoveExpiredTokens = !!body.auto_remove_expired_tokens;
          this.autoRemoveEmptyQuotaTokens = !!body.auto_remove_empty_quota_tokens;
          await this.loadAccessTokens();
          this.toastSuccess('Security settings saved.');
          return true;
        }
        this.toastError('Failed to save security settings.');
        return false;
      } finally {
        this.securitySaveInProgress = false;
      }
    },
    async openAccessSettingsModal() {
      await this.loadSecuritySettings();
      this.showAccessSettingsModal = true;
    },
    closeAccessSettingsModal() {
      if (this.securitySaveInProgress) return;
      this.showAccessSettingsModal = false;
    },
    async saveAccessSettings() {
      const ok = await this.saveSecuritySettings();
      if (ok) this.showAccessSettingsModal = false;
    },
    async openProvidersSettingsModal() {
      await this.loadSecuritySettings();
      this.showProvidersSettingsModal = true;
    },
    closeProvidersSettingsModal() {
      if (this.securitySaveInProgress) return;
      this.showProvidersSettingsModal = false;
    },
    async saveProvidersSettings() {
      const ok = await this.saveSecuritySettings();
      if (ok) this.showProvidersSettingsModal = false;
    },
    async loadTLSSettings() {
      const r = await this.apiFetch('/admin/api/settings/tls', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json();
      const mode = String(body.mode || '').trim().toLowerCase();
      this.tlsSettings.mode = (mode === 'letsencrypt' || mode === 'self_signed' || mode === 'pem') ? mode : 'letsencrypt';
      const bindMode = String(body.bind_mode || '').trim().toLowerCase();
      this.tlsBindMode = (bindMode === 'localhost' || bindMode === 'all' || bindMode === 'custom') ? bindMode : 'localhost';
      this.tlsCustomBind = String(body.custom_bind || '').trim();
      const port = Number(body.port || 443);
      this.tlsPort = (Number.isFinite(port) && port > 0) ? Math.trunc(port) : 443;
      if (this.tlsBindMode === 'custom' && this.tlsCustomBind && this.networkDetectedAddrs.includes(this.tlsCustomBind)) {
        this.tlsBindSelection = 'ip:' + this.tlsCustomBind;
      } else {
        this.tlsBindSelection = this.tlsBindMode;
      }
      this.tlsSettings = {
        enabled: !!body.enabled,
        mode: this.tlsSettings.mode,
        domain: String(body.domain || '').trim(),
        email: String(body.email || '').trim(),
        cache_dir: String(body.cache_dir || '').trim(),
        cert_pem: '',
        key_pem: '',
        cert_configured: !!body.cert_configured,
        key_configured: !!body.key_configured
      };
    },
    async refreshNetworkTabFromConfig() {
      await this.loadNetworkSettings();
      await this.loadTLSSettings();
    },
    async loadNetworkSettings() {
      const r = await this.apiFetch('/admin/api/settings/network', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      const mode = String(body.bind_mode || '').trim().toLowerCase();
      this.networkBindMode = (mode === 'localhost' || mode === 'all' || mode === 'custom') ? mode : 'localhost';
      this.networkCustomBind = String(body.custom_bind || '').trim();
      this.networkDetectedAddrs = Array.isArray(body.local_addrs) ? body.local_addrs.map((x) => String(x || '').trim()).filter(Boolean) : [];
      if (this.networkBindMode === 'custom' && this.networkCustomBind && this.networkDetectedAddrs.includes(this.networkCustomBind)) {
        this.networkBindSelection = 'ip:' + this.networkCustomBind;
      } else {
        this.networkBindSelection = this.networkBindMode;
      }
      const p = Number(body.port || 7050);
      this.networkPort = (Number.isFinite(p) && p > 0) ? Math.trunc(p) : 7050;
      const httpMode = String(body.http_mode || '').trim().toLowerCase();
      this.networkHTTPMode = (httpMode === 'enabled' || httpMode === 'when_required' || httpMode === 'disabled') ? httpMode : 'enabled';
      this.networkActiveAddrs = Array.isArray(body.active) ? body.active.map((x) => String(x || '').trim()).filter(Boolean) : [];
      this.networkPending = (body.pending && typeof body.pending === 'object') ? body.pending : null;
    },
    onNetworkBindSelectionChange() {
      const sel = String(this.networkBindSelection || '').trim();
      if (sel === 'localhost' || sel === 'all' || sel === 'custom') {
        this.networkBindMode = sel;
        return;
      }
      if (sel.startsWith('ip:')) {
        this.networkBindMode = 'custom';
        this.networkCustomBind = sel.slice(3);
      }
    },
    onTLSBindSelectionChange() {
      const sel = String(this.tlsBindSelection || '').trim();
      if (sel === 'localhost' || sel === 'all' || sel === 'custom') {
        this.tlsBindMode = sel;
        return;
      }
      if (sel.startsWith('ip:')) {
        this.tlsBindMode = 'custom';
        this.tlsCustomBind = sel.slice(3);
      }
    },
    async onTLSCertFileChange(event) {
      const file = event && event.target && event.target.files && event.target.files[0];
      if (!file) return;
      const text = await file.text().catch(() => '');
      if (!text.trim()) {
        this.toastError('Certificate PEM file is empty.');
        return;
      }
      this.tlsSettings.cert_pem = text;
      this.toastSuccess('Certificate PEM loaded.');
    },
    async onTLSKeyFileChange(event) {
      const file = event && event.target && event.target.files && event.target.files[0];
      if (!file) return;
      const text = await file.text().catch(() => '');
      if (!text.trim()) {
        this.toastError('Private key PEM file is empty.');
        return;
      }
      this.tlsSettings.key_pem = text;
      this.toastSuccess('Private key PEM loaded.');
    },
    async applyNetworkSettings() {
      if (this.networkApplyInProgress) return;
      const mode = String(this.networkBindMode || '').trim().toLowerCase();
      const httpMode = String(this.networkHTTPMode || '').trim().toLowerCase();
      const port = Math.trunc(Number(this.networkPort || 0));
      if (mode !== 'localhost' && mode !== 'all' && mode !== 'custom') {
        this.toastError('Bind mode must be localhost, all, or custom.');
        return false;
      }
      if (httpMode !== 'enabled' && httpMode !== 'when_required' && httpMode !== 'disabled') {
        this.toastError('HTTP mode must be enabled, when_required, or disabled.');
        return false;
      }
      if (!Number.isFinite(port) || port < 1 || port > 65535) {
        this.toastError('Port must be between 1 and 65535.');
        return false;
      }
      if (mode === 'custom' && !String(this.networkCustomBind || '').trim()) {
        this.toastError('Custom bind host is required.');
        return false;
      }
      this.networkApplyInProgress = true;
      try {
        const payload = {
          bind_mode: mode,
          custom_bind: String(this.networkCustomBind || '').trim(),
          port,
          http_mode: httpMode
        };
        const applyResp = await this.apiFetch('/admin/api/settings/network/apply', {
          method: 'POST',
          headers: this.headers(),
          body: JSON.stringify(payload)
        });
        if (applyResp.status === 401) { window.location = '/admin/login?next=/admin'; return 'redirect'; }
        if (!applyResp.ok) {
          const apiErr = await this.readAPIError(applyResp, 'Failed to apply network settings.');
          const body = apiErr.body || {};
          const status = String(body.status || '').trim();
          const redirectURL = String(body.https_url || body.http_url || '').trim();
          if (applyResp.status === 409 && status === 'transition_required' && redirectURL) {
            this.openConfirmModal({
              title: 'Switch Admin URL',
              message: String(body.error || 'Switch admin URL before applying network changes.'),
              confirmLabel: 'Switch'
            }, async () => {
              const payload = this.buildLocalStorageHandoffPayload();
              window.location = this.appendLocalStorageHandoffToURL(redirectURL, payload);
            });
            return 'failed';
          }
          this.toastError(apiErr.message);
          return 'failed';
        }
        const applyBody = await applyResp.json().catch(() => ({}));
        const switchID = String(applyBody.switch_id || '').trim();
        const expectedInstance = String(applyBody.instance_id || '').trim();
        if (String(applyBody.status || '').trim() === 'ok' && !switchID) {
          this.toastSuccess('Network settings saved.');
          await this.loadNetworkSettings();
          return 'ok';
        }
        if (!switchID) {
          this.toastError('Network apply did not return switch id.');
          return 'failed';
        }

        const probeOK = await this.probeNewBinding(mode, String(applyBody.custom_bind || '').trim(), Number(applyBody.port || port), expectedInstance);
        const confirmResp = await this.apiFetch('/admin/api/settings/network/confirm', {
          method: 'POST',
          headers: this.headers(),
          body: JSON.stringify({switch_id: switchID, probe_ok: !!probeOK})
        });
        if (confirmResp.status === 401) { window.location = '/admin/login?next=/admin'; return 'redirect'; }
        if (!confirmResp.ok) {
          const apiErr = await this.readAPIError(confirmResp, 'Failed to confirm network switch.');
          this.toastError(apiErr.message);
          return 'failed';
        }
        const confirmBody = await confirmResp.json().catch(() => ({}));
        if (!probeOK) {
          this.toastError('Could not verify new bind address; keeping existing listener.');
          await this.loadNetworkSettings();
          return 'failed';
        }
        const redirectURL = String(confirmBody.redirect_url || '').trim();
        if (redirectURL) {
          const payload = this.buildLocalStorageHandoffPayload();
          window.location = this.appendLocalStorageHandoffToURL(redirectURL, payload);
          return 'redirect';
        }
        this.requestUIReload('default');
        return 'redirect';
      } finally {
        this.networkApplyInProgress = false;
      }
    },
    async applyNetworkPageSettings() {
      if (this.networkApplyInProgress || this.tlsSaveInProgress || this.tlsActionInProgress) return;
      const networkResult = await this.applyNetworkSettings();
      if (networkResult !== 'ok') return;
      await this.saveTLSSettings();
    },
    async probeNewBinding(mode, customHost, port, expectedInstance) {
      const proto = window.location.protocol === 'https:' ? 'https:' : 'http:';
      const currentHost = window.location.hostname || '127.0.0.1';
      const hosts = [];
      const pushHost = (h) => {
        const v = String(h || '').trim();
        if (!v) return;
        if (!hosts.includes(v)) hosts.push(v);
      };
      if (mode === 'localhost') {
        pushHost('127.0.0.1');
        pushHost('localhost');
      } else if (mode === 'all') {
        pushHost(currentHost);
        pushHost('127.0.0.1');
        pushHost('localhost');
      } else {
        pushHost(customHost);
      }
      for (let i = 0; i < hosts.length; i++) {
        const host = hosts[i];
        const origin = proto + '//' + host + ':' + String(port);
        const url = origin + '/admin/api/network/probe';
        try {
          const r = await fetch(url, {method:'GET', mode:'cors', cache:'no-store', credentials:'omit'});
          if (!r.ok) continue;
          const body = await r.json().catch(() => ({}));
          const inst = String((body && body.instance) || '').trim();
          const expect = String(expectedInstance || this.runtimeInstanceID || '').trim();
          if (expect && inst && inst === expect) return true;
        } catch (_) {}
      }
      return false;
    },
    buildLocalStorageHandoffPayload() {
      try {
        const data = {};
        const maxItems = 64;
        let count = 0;
        for (let i = 0; i < window.localStorage.length; i++) {
          if (count >= maxItems) break;
          const k = String(window.localStorage.key(i) || '');
          if (!k.startsWith('opp_')) continue;
          const v = window.localStorage.getItem(k);
          if (v === null || v === undefined) continue;
          data[k] = String(v);
          count++;
        }
        const raw = JSON.stringify({v:1, t:Date.now(), data});
        const encoded = btoa(unescape(encodeURIComponent(raw))).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
        if (encoded.length > 7000) return '';
        return encoded;
      } catch (_) {
        return '';
      }
    },
    appendLocalStorageHandoffToURL(url, payload) {
      const target = String(url || '').trim();
      const p = String(payload || '').trim();
      if (!target || !p) return target;
      const sep = target.includes('#') ? '&' : '#';
      return target + sep + 'lsm=' + encodeURIComponent(p);
    },
    restoreLocalStorageFromHandoff() {
      try {
        const hash = String(window.location.hash || '');
        if (!hash || hash.indexOf('lsm=') === -1) return;
        const params = new URLSearchParams(hash.startsWith('#') ? hash.slice(1) : hash);
        const token = String(params.get('lsm') || '').trim();
        if (!token) return;
        const normalized = token.replace(/-/g, '+').replace(/_/g, '/');
        const pad = normalized.length % 4 === 0 ? '' : '='.repeat(4 - (normalized.length % 4));
        const decoded = decodeURIComponent(escape(atob(normalized + pad)));
        const parsed = JSON.parse(decoded);
        const data = parsed && typeof parsed === 'object' ? parsed.data : null;
        if (data && typeof data === 'object') {
          Object.keys(data).forEach((k) => {
            const key = String(k || '');
            if (!key.startsWith('opp_')) return;
            const val = data[k];
            if (val === null || val === undefined) return;
            try { window.localStorage.setItem(key, String(val)); } catch (_) {}
          });
        }
        params.delete('lsm');
        const rest = params.toString();
        const next = window.location.pathname + window.location.search + (rest ? ('#' + rest) : '');
        window.history.replaceState(null, '', next);
      } catch (_) {}
    },
    async loadVersion() {
      const r = await this.apiFetch('/admin/api/version', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      this.appVersion = String(body.version || '').trim();
    },
    async saveTLSSettings() {
      if (this.tlsSaveInProgress) return;
      this.tlsSaveInProgress = true;
      try {
        const mode = String(this.tlsSettings.mode || '').trim().toLowerCase();
        const bindMode = String(this.tlsBindMode || '').trim().toLowerCase();
        const port = Math.trunc(Number(this.tlsPort || 0));
        if (mode !== 'letsencrypt' && mode !== 'self_signed' && mode !== 'pem') {
          this.toastError('TLS mode must be letsencrypt, self_signed, or pem.');
          return false;
        }
        if (bindMode !== 'localhost' && bindMode !== 'all' && bindMode !== 'custom') {
          this.toastError('TLS bind mode must be localhost, all, or custom.');
          return false;
        }
        if (!Number.isFinite(port) || port < 1 || port > 65535) {
          this.toastError('TLS port must be between 1 and 65535.');
          return false;
        }
        if (bindMode === 'custom' && !String(this.tlsCustomBind || '').trim()) {
          this.toastError('Custom TLS bind host is required.');
          return false;
        }
        const payload = {
          enabled: !!this.tlsSettings.enabled,
          mode,
          bind_mode: bindMode,
          custom_bind: String(this.tlsCustomBind || '').trim(),
          port,
          domain: String(this.tlsSettings.domain || '').trim(),
          email: String(this.tlsSettings.email || '').trim(),
          cert_pem: String(this.tlsSettings.cert_pem || ''),
          key_pem: String(this.tlsSettings.key_pem || '')
        };
        const r = await this.apiFetch('/admin/api/settings/tls', {method:'PUT', headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        const apiErr = r.ok ? null : await this.readAPIError(r, 'Failed to save TLS settings.');
        const body = (apiErr && apiErr.body) ? apiErr.body : {};
        if (r.ok) this.toastSuccess('TLS settings saved.');
        else if (r.status === 409 && String(body.status || '').trim() === 'transition_required' && String(body.http_url || '').trim()) {
          const target = String(body.http_url || '').trim();
          this.openConfirmModal({
            title: 'Switch To HTTP',
            message: 'Apply HTTPS listener changes from HTTP admin. Switch now?',
            confirmLabel: 'Switch'
          }, async () => {
            const payload = this.buildLocalStorageHandoffPayload();
            window.location = this.appendLocalStorageHandoffToURL(target, payload);
          });
          return false;
        } else {
          this.toastError((apiErr && apiErr.message) ? apiErr.message : 'Failed to save TLS settings.');
          return false;
        }
        if (r.ok) {
          await this.loadTLSSettings();
          return true;
        }
      } finally {
        this.tlsSaveInProgress = false;
      }
    },
    async testTLSCertificate() {
      await this.runTLSAction('/admin/api/settings/tls/test-certificate', 'Test certificate obtained.');
    },
    async renewTLSCertificate() {
      await this.runTLSAction('/admin/api/settings/tls/renew', 'Certificate renewed.');
    },
    async runTLSAction(url, fallbackMessage) {
      if (this.tlsActionInProgress) return;
      this.tlsActionInProgress = true;
      try {
        const r = await this.apiFetch(url, {method:'POST', headers:this.headers()});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        const body = await r.json().catch(() => ({}));
        if (!r.ok || String(body.status || '').trim() !== 'ok') {
          const msg = String(body.error || '').trim() || 'TLS action failed.';
          this.toastError(msg);
          return;
        }
        const msg = String(body.message || '').trim() || fallbackMessage;
        this.toastSuccess(msg);
      } finally {
        this.tlsActionInProgress = false;
      }
    },
    async loadProviders() {
      const r = await this.apiFetch('/admin/api/providers', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      this.providers = await r.json();
      this.renderProviders();
      window.__adminRemoveProvider = (name) => this.removeProvider(name);
      window.__adminEditProvider = (name) => this.openEditProviderModal(name);
      window.__adminSortModels = (col) => this.sortModelsBy(col);
    },
    async loadAccessTokens() {
      const r = await this.apiFetch('/admin/api/access-tokens', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      this.accessTokens = await r.json();
      this.renderAccessTokens();
      this.requiresInitialTokenSetup = !Array.isArray(this.accessTokens) || this.accessTokens.length === 0;
      if (this.requiresInitialTokenSetup) {
        if (!this.initialSetupDialogDismissed) {
          this.activeTab = 'access';
          this.persistActiveTab();
        }
        if (!this.showAddAccessTokenModal && !this.initialSetupDialogDismissed) {
          this.openAddAccessTokenModal();
          this.toastWarning('Create your first admin key now, or cancel to continue localhost unauthenticated.');
        }
      } else {
        this.initialSetupDialogDismissed = false;
        this.persistInitialSetupDialogDismissed();
      }
    },
    debouncedLoadConversations() {
      if (this.conversationsDebounceTimer) clearTimeout(this.conversationsDebounceTimer);
      this.conversationsDebounceTimer = setTimeout(() => this.loadConversations(true), 250);
    },
    debouncedLoadLogs() {
      if (this.logDebounceTimer) clearTimeout(this.logDebounceTimer);
      this.logDebounceTimer = setTimeout(() => {
        this.logsPage = 1;
        this.loadLogs();
      }, 250);
    },
    async loadLogSettings() {
      const r = await this.apiFetch('/admin/api/settings/logs', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      this.logMaxLines = Number(body.max_lines || 5000);
    },
    async saveLogSettings() {
      if (this.logSaveInProgress) return;
      const maxLines = Number(this.logMaxLines || 0);
      if (!Number.isFinite(maxLines) || maxLines < 100 || maxLines > 200000) {
        this.toastError('Max lines must be between 100 and 200000.');
        return false;
      }
      this.logSaveInProgress = true;
      try {
        const r = await this.apiFetch('/admin/api/settings/logs', {
          method:'PUT',
          headers:this.headers(),
          body:JSON.stringify({max_lines: Math.trunc(maxLines)})
        });
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (!r.ok) {
          const txt = await r.text();
          this.toastError(txt || 'Failed to save log settings.');
          return false;
        }
        this.toastSuccess('Log settings saved.');
        await this.loadLogSettings();
        await this.loadLogs();
        return true;
      } finally {
        this.logSaveInProgress = false;
      }
    },
    async openLogSettingsModal() {
      await this.loadLogSettings();
      this.showLogSettingsModal = true;
    },
    closeLogSettingsModal() {
      if (this.logSaveInProgress) return;
      this.showLogSettingsModal = false;
    },
    async saveLogSettingsFromModal() {
      const ok = await this.saveLogSettings();
      if (ok) this.showLogSettingsModal = false;
    },
    async loadLogs() {
      const params = new URLSearchParams();
      const level = String(this.logLevelFilter || 'trace').trim().toLowerCase();
      const query = String(this.logSearch || '').trim();
      if (level) params.set('level', level);
      if (query) params.set('q', query);
      params.set('page', String(Math.max(1, Number(this.logsPage || 1))));
      params.set('page_size', String(this.logsPageSize === 0 ? 0 : this.parsePageSize(this.logsPageSize)));
      const u = '/admin/api/logs' + (params.toString() ? ('?' + params.toString()) : '');
      const r = await this.apiFetch(u, {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      this.logEntries = Array.isArray(body.entries) ? body.entries : [];
      const serverPage = Number(body.page || 1);
      if (Number.isFinite(serverPage) && serverPage > 0) this.logsPage = Math.floor(serverPage);
      const serverPageSize = Number(body.page_size);
      if (serverPageSize === 0 || serverPageSize === 25 || serverPageSize === 50 || serverPageSize === 100) {
        this.logsPageSize = serverPageSize;
      }
      this.logEntriesTotalCount = Number(body.total || this.logEntries.length || 0);
      this.renderLogs();
    },
    async clearLogs() {
      this.openConfirmModal({
        title: 'Clear Log',
        message: 'Delete all persisted log entries?',
        confirmLabel: 'Clear log'
      }, async () => this.clearLogsConfirmed());
    },
    async clearLogsConfirmed() {
      const r = await this.apiFetch('/admin/api/logs', {method:'DELETE', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text();
        this.toastError(txt || 'Failed to clear log.');
        return;
      }
      this.logEntries = [];
      this.logEntriesTotalCount = 0;
      this.renderLogs();
      this.toastSuccess('Log cleared.');
    },
    formatStructuredLogMessage(input) {
      const raw = String(input || '').trim();
      if (!raw) return '';
      const tokens = raw.split(/\s+/);
      let structured = 0;
      const out = tokens.map((tok) => {
        const idx = tok.indexOf('=');
        if (idx > 0) {
          const key = tok.slice(0, idx);
          const val = tok.slice(idx + 1);
          if (/^[A-Za-z_][A-Za-z0-9_.:-]*$/.test(key)) {
            structured++;
            return '<span class="text-body-secondary">' + this.escapeHtml(key) + '=</span><span class="text-info-emphasis">' + this.escapeHtml(val) + '</span>';
          }
        }
        return this.escapeHtml(tok);
      });
      if (structured === 0) return this.escapeHtml(raw);
      return out.join(' ');
    },
    renderLogs() {
      const rows = (this.logEntries || []).map((e) => {
        const level = String(e.level || 'info').trim().toLowerCase();
        const badgeCls =
          level === 'trace' ? 'text-bg-light border text-dark' :
          level === 'debug' ? 'text-bg-secondary' :
          level === 'info' ? 'text-bg-primary' :
          level === 'warn' ? 'text-bg-warning text-dark' :
          level === 'error' ? 'text-bg-danger' :
          level === 'fatal' ? 'text-bg-dark' : 'text-bg-secondary';
        const tsRaw = String(e.timestamp || '').trim();
        const ts = this.escapeHtml(this.formatRelativeShort(tsRaw, '-'));
        const tsTitle = this.escapeHtml(this.formatTimestamp(tsRaw));
        const msgRaw = String(e.message || '').trim();
        const msg = this.formatStructuredLogMessage(msgRaw);
        return '' +
          '<tr>' +
            '<td class="small text-nowrap" title="' + tsTitle + '">' + ts + '</td>' +
            '<td class="small text-nowrap"><span class="badge ' + badgeCls + '">' + this.escapeHtml(level || 'info') + '</span></td>' +
            '<td class="small" title="' + this.escapeHtml(msgRaw) + '"><code style="white-space:pre-wrap;">' + msg + '</code></td>' +
          '</tr>';
      });
      const totalRows = Math.max(0, Number(this.logEntriesTotalCount || 0));
      const pageSize = this.parsePageSize(this.logsPageSize);
      const totalPages = pageSize === 0 ? 1 : Math.max(1, Math.ceil(totalRows / pageSize));
      const page = Math.min(totalPages, Math.max(1, Number(this.logsPage || 1)));
      this.logsPage = page;
      this.logsPageSize = pageSize;
      this.logEntriesShownCount = (rows || []).length;
      this.logEntriesHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr><th>Time</th><th>Level</th><th>Message</th></tr></thead>' +
          '<tbody>' + (rows.join('') || '<tr><td colspan="3" class="text-body-secondary small">No log entries.</td></tr>') + '</tbody>' +
        '</table>';
      this.logPagerHtml = this.renderPager(totalRows, page, totalPages, pageSize, 'Logs');
    },
    formatTimestamp(raw) {
      const s = String(raw || '').trim();
      if (!s) return '';
      const t = new Date(s);
      if (Number.isNaN(t.getTime())) return s;
      return t.toLocaleString();
    },
    async openConversationsSettingsModal() {
      await this.loadConversationsSettings();
      this.showConversationsSettingsModal = true;
    },
    closeConversationsSettingsModal() {
      this.showConversationsSettingsModal = false;
    },
    closeConversationDetailModal() {
      this.showConversationDetailModal = false;
      this.showConversationRawModal = false;
      this.conversationDetailMessageCount = 0;
    },
    closeConversationRawModal() {
      this.showConversationRawModal = false;
    },
    async loadConversationsSettings() {
      const r = await this.apiFetch('/admin/api/settings/conversations', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      this.conversationsSettings = {
        enabled: !!body.enabled,
        max_items: Number(body.max_items || 5000),
        max_age_days: Number(body.max_age_days || 30)
      };
    },
    async saveConversationsSettings() {
      if (this.conversationsSaveInProgress) return;
      const maxItems = Number(this.conversationsSettings.max_items || 0);
      const maxAgeDays = Number(this.conversationsSettings.max_age_days || 0);
      if (!Number.isFinite(maxItems) || maxItems < 100 || maxItems > 200000) {
        this.toastError('Max items must be between 100 and 200000.');
        return;
      }
      if (!Number.isFinite(maxAgeDays) || maxAgeDays < 1) {
        this.toastError('Max age days must be at least 1.');
        return;
      }
      this.conversationsSaveInProgress = true;
      try {
        const payload = {
          enabled: !!this.conversationsSettings.enabled,
          max_items: Math.trunc(maxItems),
          max_age_days: Math.trunc(maxAgeDays)
        };
        const r = await this.apiFetch('/admin/api/settings/conversations', {method:'PUT', headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (!r.ok) {
          const txt = await r.text();
          this.toastError(txt || 'Failed to save conversation settings.');
          return;
        }
        this.toastSuccess('Conversation settings saved.');
        await this.loadConversationsSettings();
        await this.loadConversations(true);
        this.closeConversationsSettingsModal();
      } finally {
        this.conversationsSaveInProgress = false;
      }
    },
    async loadConversations(resetSelection) {
      const params = new URLSearchParams();
      const q = String(this.conversationsSearch || '').trim();
      if (q) params.set('q', q);
      params.set('limit', '5000');
      const url = '/admin/api/conversations' + (params.toString() ? ('?' + params.toString()) : '');
      let r = null;
      try {
        r = await this.apiFetch(url, {headers:this.headers()});
      } catch (_) {
        this.conversationsListHtml = '<div class=\"small text-danger\">Failed to load conversations (network error).</div>';
        this.conversationDetailHtml = '<div class=\"small text-body-secondary\">Select a conversation to inspect chat flow.</div>';
        this.conversationsPagerHtml = '';
        return;
      }
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const msg = await r.text().catch(() => '');
        this.conversationsListHtml = '<div class=\"small text-danger\">Failed to load conversations: ' + this.escapeHtml(msg || ('HTTP ' + r.status)) + '</div>';
        this.conversationDetailHtml = '<div class=\"small text-body-secondary\">Select a conversation to inspect chat flow.</div>';
        this.conversationsPagerHtml = '';
        return;
      }
      const body = await r.json().catch(() => ({}));
      this.conversationThreads = Array.isArray(body.threads) ? body.threads : [];
      this.conversationsCaptureEnabled = !!body.enabled;
      this.conversationsPage = 1;
      if (resetSelection) {
        this.selectedConversationKey = '';
      }
      if (this.selectedConversationKey && !this.conversationThreads.some((x) => String(x.conversation_key || '').trim() === this.selectedConversationKey)) {
        this.selectedConversationKey = '';
      }
      this.renderConversationsList();
      if (this.selectedConversationKey) {
        await this.loadConversationDetail(this.selectedConversationKey);
      } else {
        this.conversationRecords = [];
        this.conversationDetailHtml = this.conversationThreads.length
          ? '<div class="small text-body-secondary">Select a conversation above to inspect details.</div>'
          : '<div class="small text-body-secondary">No conversation data.</div>';
      }
    },
    async loadConversationDetail(conversationKey) {
      const key = String(conversationKey || '').trim();
      if (!key) return;
      const params = new URLSearchParams();
      if (this.conversationIncludeInternal) params.set('include_internal', '1');
      const url = '/admin/api/conversations/' + encodeURIComponent(key) + (params.toString() ? ('?' + params.toString()) : '');
      const r = await this.apiFetch(url, {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      const body = await r.json().catch(() => ({}));
      this.selectedConversationKey = key;
      this.conversationTitle = String(body.title || '').trim();
      this.conversationRecords = Array.isArray(body.records) ? body.records : [];
      this.conversationThinkVisible = {};
      this.conversationSystemVisible = {};
      this.showConversationRawModal = false;
      this.renderConversationsList();
      this.renderConversationDetail();
      this.showConversationDetailModal = true;
    },
    conversationModalTitle() {
      const title = String(this.conversationTitle || '').trim();
      const count = Math.max(0, Number(this.conversationDetailMessageCount || 0));
      if (!title) return 'Conversation';
      return title + ' - ' + String(count) + ' messages';
    },
    async toggleConversationIncludeInternal() {
      this.conversationIncludeInternal = !this.conversationIncludeInternal;
      const key = String(this.selectedConversationKey || '').trim();
      if (!key) {
        this.renderConversationDetail();
        return;
      }
      await this.loadConversationDetail(key);
    },
    toggleConversationThink(recordIndex) {
      const idx = Math.max(0, Math.floor(Number(recordIndex || 0)));
      const cur = !!(this.conversationThinkVisible && this.conversationThinkVisible[idx]);
      this.conversationThinkVisible = Object.assign({}, this.conversationThinkVisible || {}, {[idx]: !cur});
      this.renderConversationDetail();
    },
    toggleConversationSystem(recordIndex) {
      const idx = Math.max(0, Math.floor(Number(recordIndex || 0)));
      const cur = !!(this.conversationSystemVisible && this.conversationSystemVisible[idx]);
      this.conversationSystemVisible = Object.assign({}, this.conversationSystemVisible || {}, {[idx]: !cur});
      this.renderConversationDetail();
    },
    async removeConversation(conversationKey) {
      const key = String(conversationKey || '').trim();
      if (!key) return;
      this.openConfirmModal({
        title: 'Delete Conversation',
        message: 'Delete this conversation and all its records?',
        confirmLabel: 'Delete'
      }, async () => this.removeConversationConfirmed(key));
    },
    async removeConversationConfirmed(key) {
      const r = await this.apiFetch('/admin/api/conversations/' + encodeURIComponent(key), {method:'DELETE', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text().catch(() => '');
        this.toastError('Failed to delete conversation: ' + (txt || ('HTTP ' + r.status)));
        return;
      }
      this.toastSuccess('Conversation deleted.');
      if (this.selectedConversationKey === key) {
        this.selectedConversationKey = '';
        this.conversationTitle = '';
        this.conversationDetailMessageCount = 0;
        this.conversationRecords = [];
        this.showConversationDetailModal = false;
        this.conversationDetailHtml = '<div class=\"small text-body-secondary\">Select a conversation to inspect chat flow.</div>';
      }
      await this.loadConversations(false);
    },
    async clearConversations() {
      this.openConfirmModal({
        title: 'Clear Conversations',
        message: 'Delete all persisted conversations and records?',
        confirmLabel: 'Clear conversations'
      }, async () => this.clearConversationsConfirmed());
    },
    async clearConversationsConfirmed() {
      const r = await this.apiFetch('/admin/api/conversations', {method:'DELETE', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text().catch(() => '');
        this.toastError('Failed to clear conversations: ' + (txt || ('HTTP ' + r.status)));
        return;
      }
      this.conversationThreads = [];
      this.conversationRecords = [];
      this.conversationTitle = '';
      this.conversationDetailMessageCount = 0;
      this.selectedConversationKey = '';
      this.showConversationDetailModal = false;
      this.conversationsListHtml = '<div class=\"small text-body-secondary\">No conversations yet.</div>';
      this.conversationDetailHtml = '<div class=\"small text-body-secondary\">Select a conversation to inspect chat flow.</div>';
      this.conversationsPagerHtml = '';
      this.toastSuccess('Conversations cleared.');
      await this.loadConversations(true);
    },
    renderConversationsList() {
      const allRows = (this.conversationThreads || []).map((t) => {
        const key = String(t.conversation_key || '').trim();
        const selected = key && key === this.selectedConversationKey;
        const rowClass = selected ? 'table-primary' : '';
        const updated = this.escapeHtml(this.formatRelativeAge(t.last_at || ''));
        const count = Math.max(0, Number(t.count || 0));
        const tokenCount = Math.max(0, Number(t.token_count || 0));
        const title = this.escapeHtml(String(t.title || '').trim());
        return '' +
          '<tr class="' + rowClass + '" style="cursor:pointer;" data-conversation-key="' + this.escapeHtml(key) + '" onclick="window.__adminConversationRowClick(event,this.getAttribute(\'data-conversation-key\'))">' +
            '<td>' + title + '</td>' +
            '<td class="text-nowrap">' + updated + '</td>' +
            '<td class="text-end text-nowrap">' + count + '</td>' +
            '<td class="text-end text-nowrap">' + tokenCount + '</td>' +
            '<td class="text-end">' +
              '<button class=\"icon-btn icon-btn-danger\" type=\"button\" title=\"Delete conversation\" aria-label=\"Delete conversation\" data-conversation-key=\"' + this.escapeHtml(key) + '\" onclick=\"window.__adminDeleteConversation(this.getAttribute(\'data-conversation-key\'))\">' +
                '<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"currentColor\" viewBox=\"0 0 16 16\" aria-hidden=\"true\"><path d=\"M5.5 5.5A.5.5 0 0 1 6 6v6a.5.5 0 0 1-1 0V6a.5.5 0 0 1 .5-.5Zm2.5.5a.5.5 0 0 0-1 0v6a.5.5 0 0 0 1 0V6Zm2 .5a.5.5 0 0 1 1 0v6a.5.5 0 0 1-1 0V6Z\"/><path d=\"M14.5 3a1 1 0 0 1-1 1H13v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V4h-.5a1 1 0 1 1 0-2H6a1 1 0 0 1 1-1h2a1 1 0 0 1 1 1h3.5a1 1 0 0 1 1 1ZM4 4v9a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V4H4Z\"/></svg>' +
              '</button>' +
            '</td>' +
          '</tr>';
      });
      const page = this.paginateRows(allRows, this.conversationsPage, this.conversationsPageSize);
      this.conversationsPage = page.page;
      this.conversationsPageSize = page.pageSize;
      this.conversationsListHtml =
        '<table class="table table-sm align-middle mb-0">' +
          '<thead><tr><th>Title</th><th>Timestamp</th><th class="text-end">Messages</th><th class="text-end">Tokens</th><th class="text-end"></th></tr></thead>' +
          '<tbody>' + (page.rows.join('') || '<tr><td colspan="5" class="text-body-secondary">No conversation data.</td></tr>') + '</tbody>' +
        '</table>';
      this.conversationsPagerHtml = this.renderPager(page.totalRows, page.page, page.totalPages, page.pageSize, 'Conversations');
      window.__adminOpenConversation = (key) => this.loadConversationDetail(key);
      window.__adminDeleteConversation = (key) => this.removeConversation(key);
      window.__adminConversationRowClick = (ev, key) => {
        if (ev && ev.target && ev.target.closest && ev.target.closest('button')) return;
        this.loadConversationDetail(key);
      };
    },
    renderConversationDetail() {
      const records = this.conversationRecords || [];
      if (!records.length) {
        this.conversationDetailMessageCount = 0;
        this.conversationDetailHtml = '<div class=\"small text-body-secondary\">No messages captured for this conversation.</div>';
        return;
      }
      const rows = [];
      records.forEach((rec, idx) => {
        const ts = this.escapeHtml(this.formatRelativeAge(rec.created_at || ''));
        const tsTitle = this.escapeHtml(this.formatTimestamp(rec.created_at || ''));
        const status = this.escapeHtml(String(rec.status_code || ''));
        const latency = this.escapeHtml(String(rec.latency_ms || 0) + 'ms');
        const rawSent = String(rec.request_text_markdown || '');
        const rawRecv = String(rec.response_text_markdown || '');
        const systemText = String(rec.request_system_markdown || '').trim();
        const systemVisible = !!(this.conversationSystemVisible && this.conversationSystemVisible[idx]);
        const systemTokens = systemText ? Math.max(1, Math.ceil(systemText.length / 4)) : 0;
        const systemButton = systemText
          ? '<button class=\"btn btn-sm btn-outline-secondary py-0 px-2\" type=\"button\" data-record-index=\"' + String(idx) + '\" onclick=\"window.__adminToggleConversationSystem(this.dataset.recordIndex)\">System (' + String(systemTokens) + ' tok)</button>'
          : '';
        const systemBlock = (systemText && systemVisible)
          ? '<div class=\"mb-2 border rounded p-2 bg-body-tertiary\"><div class=\"small text-body-secondary mb-1\">System Prompt</div><div class=\"small\">' + this.renderMarkdown(systemText) + '</div></div>'
          : '';
        const sentText = String(rec.request_render_markdown || rec.request_delta_markdown || rawSent || '').trim();
        const recvText = String(rec.response_render_markdown || rec.response_delta_markdown || rawRecv || '').trim();
        if (!sentText && !recvText) return;
        const sent = this.renderMarkdown(sentText);
        const recv = this.renderMarkdown(recvText);
        const internalBadge = rec.is_internal
          ? '<span class=\"badge text-bg-warning\">Internal' + (rec.internal_kind ? (': ' + this.escapeHtml(String(rec.internal_kind))) : '') + '</span>'
          : '';
        const thinkText = String(rec.response_think_markdown || '').trim();
        const thinkVisible = !!(this.conversationThinkVisible && this.conversationThinkVisible[idx]);
        const thinkTokens = thinkText ? Math.max(1, Math.ceil(thinkText.length / 4)) : 0;
        const thinkBtnClass = thinkVisible ? 'btn-primary' : 'btn-outline-secondary';
        const thinkButton = thinkText
          ? '<button class=\"btn btn-sm ' + thinkBtnClass + ' py-0 px-2\" type=\"button\" data-record-index=\"' + String(idx) + '\" onclick=\"window.__adminToggleConversationThink(this.dataset.recordIndex)\">Think (' + String(thinkTokens) + ' tok)</button>'
          : '';
        const recvMetaRow = '<div class=\"mb-2 d-flex flex-wrap gap-2 align-items-center\"><span class=\"badge text-bg-secondary\">Status ' + status + '</span><span class=\"badge text-bg-light border\">Latency ' + latency + '</span>' + thinkButton + '</div>';
        const thinkBlock = (thinkText && thinkVisible)
          ? '<div class=\"mt-2 border rounded p-2 bg-body-tertiary\"><div class=\"small text-body-secondary mb-1\">Reasoning</div><div class=\"small\">' + this.renderMarkdown(thinkText) + '</div></div>'
          : '';
        rows.push('' +
          '<div class=\"d-flex justify-content-center align-items-center gap-2 small text-body-secondary mb-2\" title=\"' + tsTitle + '\">· ' + ts + ' ·<button class=\"btn btn-sm btn-outline-secondary py-0 px-2\" type=\"button\" data-record-index=\"' + String(idx) + '\" onclick=\"window.__adminOpenConversationRaw(this.dataset.recordIndex)\">Raw</button>' + systemButton + internalBadge + '</div>' +
          systemBlock +
          '<div class=\"d-flex justify-content-end mb-2\">' +
            '<div class=\"rounded-3 p-2 border bg-primary-subtle\" style=\"max-width:78%;\">' +
              '<div class=\"small conversation-markdown\">' + sent + '</div>' +
            '</div>' +
          '</div>' +
          '<div class=\"d-flex justify-content-start mb-3\">' +
            '<div class=\"rounded-3 p-2 border bg-body-tertiary\" style=\"max-width:78%;\">' +
              recvMetaRow +
              (thinkBlock ? ('<div class=\"conversation-markdown\">' + thinkBlock + '</div>') : '') +
              '<div class=\"small conversation-markdown\">' + recv + '</div>' +
            '</div>' +
		          '</div>');
	      });
      if (!rows.length) {
        this.conversationDetailMessageCount = 0;
        this.conversationDetailHtml = '<div class=\"small text-body-secondary\">No new messages to display after deduplication.</div>';
        return;
      }
      this.conversationDetailMessageCount = rows.length;
      this.conversationDetailHtml = rows.join('');
      window.__adminOpenConversationRaw = (recordIndex) => this.openConversationRawModal(recordIndex);
      window.__adminToggleConversationThink = (recordIndex) => this.toggleConversationThink(recordIndex);
      window.__adminToggleConversationSystem = (recordIndex) => this.toggleConversationSystem(recordIndex);
    },
    openConversationRawModal(recordIndex) {
      const idx = Math.max(0, Math.floor(Number(recordIndex || 0)));
      const rec = (this.conversationRecords || [])[idx];
      if (!rec) return;
      const title = this.escapeHtml(this.formatTimestamp(rec.created_at || ''));
      const reqHeaders = this.escapeHtml(String(rec.request_headers_raw || '').trim());
      const reqBody = this.escapeHtml(String(rec.request_payload_raw || '').trim());
      const respHeaders = this.escapeHtml(String(rec.response_headers_raw || '').trim());
      const respBody = this.escapeHtml(String(rec.response_payload_raw || '').trim());
      this.conversationRawHtml = '' +
        '<div class="small text-body-secondary mb-3">Message timestamp: ' + title + '</div>' +
        '<div class="mb-3">' +
          '<div class="fw-semibold mb-1">Outgoing Headers</div>' +
          '<pre class="small border rounded p-2 bg-body-tertiary mb-0" style="max-height:28vh; overflow:auto;"><code>' + reqHeaders + '</code></pre>' +
        '</div>' +
        '<div class="mb-3">' +
          '<div class="fw-semibold mb-1">Outgoing Body</div>' +
          '<pre class="small border rounded p-2 bg-body-tertiary mb-0" style="max-height:28vh; overflow:auto;"><code>' + reqBody + '</code></pre>' +
        '</div>' +
        '<div class="mb-3">' +
          '<div class="fw-semibold mb-1">Incoming Headers</div>' +
          '<pre class="small border rounded p-2 bg-body-tertiary mb-0" style="max-height:28vh; overflow:auto;"><code>' + respHeaders + '</code></pre>' +
        '</div>' +
        '<div>' +
          '<div class="fw-semibold mb-1">Incoming Body</div>' +
          '<pre class="small border rounded p-2 bg-body-tertiary mb-0" style="max-height:28vh; overflow:auto;"><code>' + respBody + '</code></pre>' +
        '</div>';
      this.showConversationRawModal = true;
    },
    collapseRepeatedConversationText(currentText, previousText) {
      const cur = String(currentText || '').replaceAll('\r\n', '\n');
      const prev = String(previousText || '').replaceAll('\r\n', '\n');
      if (!cur) return '';
      if (!prev) return cur.trim();
      if (cur === prev) return '';
      if (cur.startsWith(prev)) return cur.slice(prev.length).trimStart();
      const prevTailLimit = 12000;
      const prevTail = prev.length > prevTailLimit ? prev.slice(prev.length - prevTailLimit) : prev;
      const max = Math.min(prevTail.length, cur.length);
      const minOverlap = 40;
      for (let n = max; n >= minOverlap; n--) {
        if (prevTail.slice(prevTail.length - n) === cur.slice(0, n)) {
          return cur.slice(n).trimStart();
        }
      }
      return cur.trim();
    },
    renderMarkdown(input) {
      const md = String(input || '').trim();
      if (!md) return '<span class=\"text-body-secondary\">(empty)</span>';
      try {
        let html = '';
        if (window.marked && typeof window.marked.parse === 'function') {
          html = window.marked.parse(md, {gfm:true, breaks:false});
        } else {
          html = this.escapeHtml(md).replaceAll('\\n', '<br>');
        }
        if (window.DOMPurify && typeof window.DOMPurify.sanitize === 'function') {
          html = window.DOMPurify.sanitize(html);
        }
        if (html.includes('<table')) {
          const wrap = document.createElement('div');
          wrap.innerHTML = html;
          wrap.querySelectorAll('table').forEach((table) => {
            table.classList.add('table', 'table-sm', 'table-bordered', 'align-middle', 'mb-2');
            if (!(table.parentElement && table.parentElement.classList && table.parentElement.classList.contains('table-responsive'))) {
              const holder = document.createElement('div');
              holder.className = 'table-responsive';
              table.parentNode.insertBefore(holder, table);
              holder.appendChild(table);
            }
          });
          html = wrap.innerHTML;
        }
        return html;
      } catch (_) {
        return this.escapeHtml(md).replaceAll('\\n', '<br>');
      }
    },
    async saveAccessToken() {
      if (String(this.accessTokenDraft.id || '').trim()) {
        await this.saveAccessTokenEdit();
        return;
      }
      const name = String(this.accessTokenDraft.name || '').trim();
      if (!name) {
        this.toastError('Name is required.');
        return;
      }
      const key = String(this.accessTokenDraft.key || '').trim();
      if (!key) {
        this.toastError('Key is required.');
        return;
      }
      const role = String(this.accessTokenDraft.role || '').trim().toLowerCase();
      if (role !== 'admin' && role !== 'keymaster' && role !== 'inferrer') {
        this.toastError('Type is required.');
        return;
      }
      const expiresAt = this.expiryPresetToRFC3339(this.accessTokenDraft.expiry_preset);
      const payload = {
        name,
        key,
        role,
        expires_at: expiresAt,
        quota: this.buildAccessTokenQuotaPayload()
      };
      if (this.accessTokenDraft.quota_enabled && !payload.quota) {
        this.toastError('Set at least one quota limit.');
        return;
      }
      const r = await this.apiFetch('/admin/api/access-tokens', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text();
        this.toastError(txt || 'Failed to add key');
        return;
      }
      if (this.requiresInitialTokenSetup && this.accessTokenDraft.disable_localhost_no_auth) {
        this.allowLocalhostNoAuth = false;
        await this.saveSecuritySettings();
      }
      this.initialSetupDialogDismissed = false;
      this.closeAddAccessTokenModal();
      this.toastSuccess('Key added.');
      await this.loadAccessTokens();
    },
    async saveAccessTokenEdit() {
      const id = String(this.accessTokenDraft.id || '').trim();
      if (!id) return;
      const name = String(this.accessTokenDraft.name || '').trim();
      if (!name) {
        this.toastError('Name is required.');
        return;
      }
      const role = String(this.accessTokenDraft.role || '').trim().toLowerCase();
      if (role !== 'admin' && role !== 'keymaster' && role !== 'inferrer') {
        this.toastError('Type is required.');
        return;
      }
      const preset = String(this.accessTokenDraft.expiry_preset || 'never').trim();
      const expiresAt = preset === 'custom'
        ? String(this.accessTokenDraft.expires_at || '').trim()
        : this.expiryPresetToRFC3339(preset);
      const payload = {
        name,
        role,
        expires_at: expiresAt,
        quota: this.buildAccessTokenQuotaPayload()
      };
      if (this.accessTokenDraft.quota_enabled && !payload.quota) {
        this.toastError('Set at least one quota limit.');
        return;
      }
      const r = await this.apiFetch('/admin/api/access-tokens/' + encodeURIComponent(id), {method:'PUT', headers:this.headers(), body:JSON.stringify(payload)});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text();
        this.toastError(txt || 'Failed to update key');
        return;
      }
      this.closeAddAccessTokenModal();
      this.toastSuccess('Key updated.');
      await this.loadAccessTokens();
    },
    async removeAccessToken(id) {
      const tokenId = String(id || '').trim();
      if (!tokenId) return;
      this.openConfirmModal({
        title: 'Delete Access Token',
        message: 'Delete this access token?',
        confirmLabel: 'Delete'
      }, async () => this.removeAccessTokenConfirmed(tokenId));
    },
    async removeAccessTokenConfirmed(tokenId) {
      const r = await this.apiFetch('/admin/api/access-tokens/' + encodeURIComponent(tokenId), {method:'DELETE', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        const txt = await r.text();
        this.toastError(txt || 'Failed to delete token');
        return;
      }
      this.toastSuccess('Token deleted.');
      await this.loadAccessTokens();
    },
    openEditProviderModal(name) {
      const row = (this.providers || []).find((p) => String(p.name || '') === String(name || ''));
      if (!row) return;
      this.resetDraft();
      this.editingProviderName = String(row.name || '').trim();
      this.selectedPreset = String(row.provider_type || '').trim();
      if (!(this.popularProviders || []).some((p) => p.name === this.selectedPreset)) {
        this.selectedPreset = '';
      }
      this.draft.name = String(row.name || '').trim();
      this.draft.provider_type = String(row.provider_type || '').trim();
      this.draft.base_url = String(row.base_url || '').trim();
      this.draft.enabled = true;
      this.draft.timeout_seconds = Number(row.timeout_seconds || 0) > 0 ? Number(row.timeout_seconds) : '';
      this.overrideProviderSettings = true;
      this.authMode = 'api_key';
      this.addProviderStep = 'api_key';
      const preset = this.getSelectedPreset();
      this.presetInfoHtml = preset ? this.renderPresetInfo(preset) : '';
      this.modalStatusHtml = '<span class="text-body-secondary">Edit provider settings. Re-enter credentials to update them.</span>';
      this.showAddProviderModal = true;
    },
    async loadModelsCatalog(firstLoad, forceRefresh) {
      if (firstLoad && this.modelsCatalog.length === 0) this.modelsInitialLoadInProgress = true;
      const sec = Math.max(1, Number(this.performanceRangeHours || 8)) * 3600;
      const u = forceRefresh ? ('/admin/api/models?refresh=1&period_seconds=' + sec) : ('/admin/api/models?period_seconds=' + sec);
      const r = await this.apiFetch(u, {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) {
        if (firstLoad) this.modelsInitialLoadInProgress = false;
        return;
      }
      const body = await r.json();
      this.modelsCatalog = body.data || [];
      this.modelsInitialized = true;
      if (firstLoad || forceRefresh) this.modelsPage = 1;
      this.renderModelsCatalog();
      this.renderPerformanceCatalog();
      this.renderModelsFreshness(body.fetched_at, body.pricing_cache_updated_at);
      this.persistModelsToCache();
      if (firstLoad) this.modelsInitialLoadInProgress = false;
    },
    async refreshPricingAndModels() {
      if (this.modelsRefreshInProgress) return;
      this.modelsRefreshInProgress = true;
      try {
        await this.loadModelsCatalog(false, true);
        await this.loadProviders();
      } finally {
        this.modelsRefreshInProgress = false;
      }
    },
    async loadPopularProviders() {
      const r = await this.apiFetch('/admin/api/providers/popular', {headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (!r.ok) return;
      this.popularProviders = await r.json();
    },
    buildProviderPayload() {
      const payload = {name: String(this.draft.name || '').trim(), enabled: !!this.draft.enabled};
      if (this.selectedPreset) {
        payload.provider_type = String(this.selectedPreset || '').trim();
      } else if (String(this.draft.provider_type || '').trim()) {
        payload.provider_type = String(this.draft.provider_type || '').trim();
      }
      if (this.authMode === 'oauth') {
        payload.auth_token = String(this.draft.auth_token || '').trim();
        payload.refresh_token = String(this.draft.refresh_token || '').trim();
        payload.token_expires_at = String(this.draft.token_expires_at || '').trim();
        payload.account_id = String(this.draft.account_id || '').trim();
      } else if (this.authMode === 'device' && this.selectedPresetSupportsDeviceAuth()) {
        payload.auth_token = String(this.draft.auth_token || '').trim();
        payload.refresh_token = String(this.draft.refresh_token || '').trim();
        payload.token_expires_at = String(this.draft.token_expires_at || '').trim();
        payload.account_id = String(this.draft.account_id || '').trim();
        payload.device_auth_url = String(this.draft.device_auth_url || '').trim();
      } else {
        payload.api_key = String(this.draft.api_key || '').trim();
      }
      if (this.authMode === 'oauth') {
        payload.base_url = String(this.draft.base_url || '').trim();
        const tout = Number(this.draft.timeout_seconds || 0);
        payload.timeout_seconds = (Number.isFinite(tout) && tout > 0) ? tout : 60;
      } else if (this.selectedPresetRequiresBaseURLInput() || this.overrideProviderSettings || !this.selectedPreset) {
        payload.base_url = String(this.draft.base_url || '').trim();
        const tout = Number(this.draft.timeout_seconds || 0);
        if (Number.isFinite(tout) && tout > 0) payload.timeout_seconds = tout;
      }
      return payload;
    },
    async testProvider() {
      const payload = this.buildProviderPayload();
      const r = await this.apiFetch('/admin/api/providers/test', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      const body = await r.json().catch(() => ({}));
      if (r.ok && body.ok) {
        this.toastSuccess('Connection successful. Models found: ' + Number(body.model_count || 0));
        return;
      }
      const err = body.error || 'Connection failed.';
      this.toastError(err);
    },
    async saveProvider() {
      const payload = this.buildProviderPayload();
      const isEdit = String(this.editingProviderName || '').trim() !== '';
      const endpoint = isEdit ? ('/admin/api/providers/' + encodeURIComponent(this.editingProviderName)) : '/admin/api/providers';
      const method = isEdit ? 'PUT' : 'POST';
      const isDisabling = !payload.enabled;
      if (this.authMode === 'oauth') {
        if (!String(payload.auth_token || '').trim()) {
          this.toastError('Run browser OAuth login first.');
          return;
        }
        const r = await this.apiFetch(endpoint, {method, headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (r.ok) this.toastSuccess('Provider ' + (isEdit ? 'updated' : 'added') + '.');
        else this.toastError('Failed to ' + (isEdit ? 'update' : 'add') + ' provider.');
        if (!r.ok) return;
        this.closeAddProviderModal();
        this.loadProviders();
        return;
      }
      if (isDisabling) {
        const r = await this.apiFetch(endpoint, {method, headers:this.headers(), body:JSON.stringify(payload)});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (r.ok) this.toastSuccess('Provider disabled.');
        else this.toastError('Failed to disable provider.');
        if (!r.ok) return;
        this.closeAddProviderModal();
        this.loadProviders();
        return;
      }
      this.modalStatusHtml = '<span class="text-body-secondary">Testing provider credentials...</span>';
      const tr = await this.apiFetch('/admin/api/providers/test', {method:'POST', headers:this.headers(), body:JSON.stringify(payload)});
      if (tr.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      const tb = await tr.json().catch(() => ({}));
      if (!(tr.ok && tb.ok)) {
        const err = tb.error || 'Provider authentication failed.';
        this.toastError(err);
        return;
      }
      const r = await this.apiFetch(endpoint, {method, headers:this.headers(), body:JSON.stringify(payload)});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (r.ok) this.toastSuccess('Provider ' + (isEdit ? 'updated' : 'added') + '.');
      else this.toastError('Failed to ' + (isEdit ? 'update' : 'add') + ' provider.');
      if (!r.ok) return;
      this.closeAddProviderModal();
      this.loadProviders();
    },
    async removeProvider(name) {
      const providerName = String(name || '').trim();
      if (!providerName) return;
      this.openConfirmModal({
        title: 'Delete Provider',
        message: 'Delete provider "' + providerName + '"? This cannot be undone.',
        confirmLabel: 'Delete provider'
      }, async () => this.removeProviderConfirmed(providerName));
    },
    async removeProviderConfirmed(providerName) {
      const r = await this.apiFetch('/admin/api/providers/'+encodeURIComponent(providerName), {method:'DELETE', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (r.ok) this.toastSuccess('Provider removed.');
      else this.toastError('Failed to remove provider.');
      if (!r.ok) return;
      this.loadProviders();
    },
    async refreshModels() {
      const r = await this.apiFetch('/admin/api/models/refresh', {method:'POST', headers:this.headers()});
      if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
      if (r.ok) this.toastSuccess('Model refresh completed.');
      else this.toastError('Model refresh failed.');
      if (r.ok && (this.activeTab === 'models' || this.activeTab === 'performance')) {
        this.loadModelsCatalog(false);
      }
    },
    async refreshProviders() {
      if (this.providersRefreshInProgress) return;
      this.providersRefreshInProgress = true;
      try {
        const r = await this.apiFetch('/admin/api/providers/refresh', {method:'POST', headers:this.headers()});
        if (r.status === 401) { window.location = '/admin/login?next=/admin'; return; }
        if (r.ok) this.toastSuccess('Provider refresh completed.');
        else this.toastError('Provider refresh failed.');
        if (r.ok) {
          await this.loadProviders();
        }
      } finally {
        this.providersRefreshInProgress = false;
      }
    }
  }
}
