// Quant IDX Pro FinTech Terminal Application JS

let currentSelectedSymbol = 'BBCA';
let sentimentChartInstance = null;
let detailChartInstance = null;
let currentSignalFilter = ''; // '', 'BUY', 'SELL', 'HOLD'
let searchQuery = '';

let appInitialized = false;

function safeInitApp() {
  if (appInitialized) return;
  appInitialized = true;

  console.log('[QuantTerminal] Initializing application...');

  try { initHealthCheck(); } catch (e) { console.error('HealthCheck init error:', e); }
  try { initSignalFilters(); } catch (e) { console.error('SignalFilters init error:', e); }
  try { initNewsFilters(); } catch (e) { console.error('NewsFilters init error:', e); }
  try { initIngestionTrigger(); } catch (e) { console.error('IngestionTrigger init error:', e); }
  try { initReprocessTrigger(); } catch (e) { console.error('ReprocessTrigger init error:', e); }
  try { initSearchAndTabs(); } catch (e) { console.error('SearchAndTabs init error:', e); }
  try { initImportModal(); } catch (e) { console.error('ImportModal init error:', e); }
  try { initSSEProcessStream(); } catch (e) { console.error('SSEProcessStream init error:', e); }
  try { initStockDetailPanel(); } catch (e) { console.error('StockDetailPanel init error:', e); }

  // Load initial data
  loadSignals();
  loadHistory(currentSelectedSymbol);
  loadNews();
  loadIHSG();

  // Periodic health check ping
  setInterval(initHealthCheck, 15000);
  setInterval(loadIHSG, 5 * 60 * 1000);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', safeInitApp);
}
safeInitApp();

// ─── HEALTH CHECK ──────────────────────────────────────

async function initHealthCheck() {
  const badge = document.getElementById('healthBadge');
  const statusText = document.getElementById('healthStatusText');
  const latencyText = document.getElementById('healthLatencyText');

  try {
    const res = await fetch('/readyz');
    const data = await res.json();

    if (badge) badge.classList.remove('hidden');

    if (data.status === 'ok') {
      const dbLatency = data.checks?.postgres?.latency_ms || 0;
      if (statusText) statusText.innerText = 'System Ready';
      if (latencyText) latencyText.innerText = `${dbLatency}ms DB`;
    } else {
      if (statusText) statusText.innerText = 'Service Degraded';
    }
  } catch (err) {
    if (badge) badge.classList.remove('hidden');
    if (statusText) statusText.innerText = 'Backend Offline';
  }
}

let cachedSignalsData = null;
let lastFetchedPeriod = '';
let lastFetchedSector = '';

// ─── SEARCH & SIGNAL FILTER TABS ──────────────────────

function initSearchAndTabs() {
  const searchInput = document.getElementById('tickerSearchInput');
  if (searchInput) {
    searchInput.addEventListener('input', (e) => {
      searchQuery = e.target.value.trim().toLowerCase();
      renderSignals();
    });
  }

  const signalBtns = document.querySelectorAll('[data-signal-filter]');
  signalBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      signalBtns.forEach(b => {
        b.classList.remove('bg-cyan-500/20', 'text-cyan-400');
        b.classList.add('text-slate-400');
      });

      btn.classList.remove('text-slate-400');
      btn.classList.add('bg-cyan-500/20', 'text-cyan-400');

      currentSignalFilter = btn.getAttribute('data-signal-filter') || '';
      renderSignals();
    });
  });
}

function initSignalFilters() {
  const periodSel = document.getElementById('signalPeriodFilter');
  const sectorSel = document.getElementById('signalSectorFilter');

  if (periodSel) periodSel.addEventListener('change', () => loadSignals(true));
  if (sectorSel) sectorSel.addEventListener('change', () => loadSignals(true));
}

async function loadSignals(forceRefresh = false) {
  const container = document.getElementById('signalsContainer');
  const period = document.getElementById('signalPeriodFilter')?.value || '7d';
  const sector = document.getElementById('signalSectorFilter')?.value || '';

  // Use in-memory cache if period and sector haven't changed
  if (!forceRefresh && cachedSignalsData && lastFetchedPeriod === period && lastFetchedSector === sector) {
    renderSignals();
    return;
  }

  let url = `/api/v1/signals?period=${period}`;
  if (sector) url += `&sector=${encodeURIComponent(sector)}`;

  try {
    const res = await fetch(url);
    const result = await res.json();

    if (!result.success || !result.data || result.data.length === 0) {
      cachedSignalsData = [];
      container.innerHTML = `
        <div class="col-span-full text-center py-12 glass-card rounded-xl text-slate-400">
          <i data-lucide="info" class="w-8 h-8 mx-auto mb-2 text-cyan-400"></i>
          <p class="font-medium text-xs">Belum ada sinyal teranalisis untuk periode ${period}.</p>
          <p class="text-[11px] text-slate-500 mt-1">Klik tombol <strong>Run Scrape AI</strong> atau <strong>Reprocess AI</strong> di atas untuk menganalisis berita!</p>
        </div>
      `;
      if (window.lucide) window.lucide.createIcons();
      return;
    }

    cachedSignalsData = result.data;
    lastFetchedPeriod = period;
    lastFetchedSector = sector;

    renderSignals();
  } catch (err) {
    console.error('Failed to load signals:', err);
    if (container) {
      container.innerHTML = `
        <div class="col-span-full text-center py-8 glass-card rounded-xl text-rose-400 text-xs font-mono">
          Gagal memuat sinyal kuantitatif: ${err.message || err}
        </div>
      `;
    }
  }
}

function renderSignals() {
  const container = document.getElementById('signalsContainer');
  if (!container || !cachedSignalsData) return;

  try {
    // Filter by signal type and search query
  let filteredData = [...cachedSignalsData];

  if (currentSignalFilter) {
    filteredData = filteredData.filter(item => item.signal === currentSignalFilter);
  }

  if (searchQuery) {
    filteredData = filteredData.filter(item => 
      item.symbol.toLowerCase().includes(searchQuery) ||
      (item.company_name && item.company_name.toLowerCase().includes(searchQuery))
    );
  }

  // Strictly sort by Composite Score descending (highest score first)
  filteredData.sort((a, b) => {
    const scoreA = a.composite_score !== undefined && a.composite_score !== null ? a.composite_score : (a.average_score || 0);
    const scoreB = b.composite_score !== undefined && b.composite_score !== null ? b.composite_score : (b.average_score || 0);
    return scoreB - scoreA;
  });

  if (filteredData.length === 0) {
    container.innerHTML = `
      <div class="col-span-full text-center py-8 glass-card rounded-xl text-slate-500 text-xs font-mono">
        Tidak ada emiten yang sesuai dengan filter '${searchQuery || currentSignalFilter}'.
      </div>
    `;
    return;
  }

  // Calculate overview stats
  let totalScore = 0;
  let totalArticles = 0;

    const cardsHTML = filteredData.map(item => {
      totalScore += item.composite_score || item.average_score;
      totalArticles += item.article_count;

      // Use composite score for gauge pointer (-1.0 -> 0%, 0.0 -> 50%, +1.0 -> 100%)
      const displayScore = item.composite_score || item.average_score;
      const pointerPct = Math.min(Math.max(((displayScore + 1.0) / 2.0) * 100, 5), 95);

      let badgeClass = 'badge-hold';
      if (item.signal === 'BUY') badgeClass = 'badge-buy';
      if (item.signal === 'SELL') badgeClass = 'badge-sell';

      const isSelected = item.symbol === currentSelectedSymbol ? 'border-cyan-500/80 shadow-lg shadow-cyan-500/10 bg-[#12192a]' : '';

      // Technical indicator badges
      let techBadges = '';
      if (item.rsi14 != null) {
        let rsiColor = 'text-amber-400 bg-amber-500/10';
        let rsiLabel = 'Normal';
        if (item.rsi14 < 30) { rsiColor = 'text-emerald-400 bg-emerald-500/10'; rsiLabel = 'Oversold'; }
        else if (item.rsi14 < 40) { rsiColor = 'text-emerald-400 bg-emerald-500/10'; rsiLabel = 'Low'; }
        else if (item.rsi14 > 80) { rsiColor = 'text-rose-400 bg-rose-500/10'; rsiLabel = 'Overbought!'; }
        else if (item.rsi14 > 70) { rsiColor = 'text-rose-400 bg-rose-500/10'; rsiLabel = 'High'; }
        techBadges += `<span class="px-1 py-0.5 rounded text-[9px] font-mono ${rsiColor}" title="RSI(14): ${item.rsi14.toFixed(1)}">RSI:${item.rsi14.toFixed(0)}</span>`;
      }
      if (item.ma20 != null && item.last_price != null) {
        const aboveMA = item.last_price > item.ma20;
        const maColor = aboveMA ? 'text-emerald-400 bg-emerald-500/10' : 'text-rose-400 bg-rose-500/10';
        const maArrow = aboveMA ? '↑' : '↓';
        techBadges += `<span class="px-1 py-0.5 rounded text-[9px] font-mono ${maColor}" title="Price vs MA20">MA20${maArrow}</span>`;
      }

      // Price info
      let priceInfo = '';
      if (item.last_price != null) {
        const priceChg = item.price_change_pct || 0;
        const priceColor = priceChg >= 0 ? 'text-emerald-400' : 'text-rose-400';
        const priceSign = priceChg >= 0 ? '+' : '';
        priceInfo = `<span class="text-[10px] ${priceColor} font-mono">${priceSign}${priceChg.toFixed(1)}%</span>`;
      }

      // Confidence bar width
      const confidence = item.confidence || 0;
      const confPct = Math.round(confidence * 100);

      return `
        <div class="glass-card glass-card-hover rounded-xl p-3.5 space-y-2.5 cursor-pointer ${isSelected}" onclick="selectStock('${item.symbol}')">
          <div class="flex items-start justify-between">
            <div>
              <div class="flex items-center space-x-2">
                <span class="text-base font-bold text-white font-mono tracking-tight">${item.symbol}</span>
                ${item.sector ? `<span class="text-[10px] text-slate-400 bg-slate-800/80 px-1.5 py-0.5 rounded font-mono truncate max-w-[100px]">${item.sector}</span>` : ''}
                ${priceInfo}
              </div>
              <p class="text-[11px] text-slate-400 truncate max-w-[170px] mt-0.5">${item.company_name}</p>
            </div>
            <span class="px-2.5 py-0.5 rounded text-xs font-bold font-mono tracking-wide ${badgeClass}">
              ${item.signal}
            </span>
          </div>

          <!-- Technical Badges -->
          ${techBadges ? `<div class="flex items-center gap-1">${techBadges}</div>` : ''}

          <!-- Composite Score Gauge -->
          <div class="space-y-1.5">
            <div class="flex justify-between items-center text-[11px] font-mono">
              <span class="text-slate-400 text-[10px] uppercase font-sans">Composite Score</span>
              <span class="${displayScore > 0.15 ? 'text-emerald-400' : (displayScore < -0.15 ? 'text-rose-400' : 'text-amber-400')} font-bold">
                ${displayScore > 0 ? '+' : ''}${displayScore.toFixed(2)}
              </span>
            </div>
            <div class="gauge-bar-track">
              <div class="gauge-bar-pointer" style="left: ${pointerPct}%"></div>
            </div>
            ${confidence > 0 ? `<div class="flex items-center gap-1.5"><span class="text-[9px] text-slate-500 font-mono">Confidence</span><div class="flex-1 h-1 bg-slate-800 rounded-full overflow-hidden"><div class="h-full rounded-full ${confPct > 70 ? 'bg-emerald-500' : (confPct > 40 ? 'bg-amber-500' : 'bg-rose-500')}" style="width:${confPct}%"></div></div><span class="text-[9px] text-slate-500 font-mono">${confPct}%</span></div>` : ''}
          </div>

          <!-- Breakdown Footer -->
          <div class="flex justify-between items-center text-[11px] text-slate-400 pt-2 border-t border-slate-800/80 font-mono">
            <span>News: <strong class="text-slate-200">${item.article_count}</strong></span>
            <div class="flex items-center space-x-2">
              <span class="text-emerald-400 font-semibold">▲ ${item.bullish_articles}</span>
              <span class="text-rose-400 font-semibold">▼ ${item.bearish_articles}</span>
              <button onclick="event.stopPropagation(); openStockDetail('${item.symbol}')" class="px-2 py-0.5 bg-cyan-500/20 text-cyan-400 border border-cyan-500/30 rounded text-[10px] font-bold hover:bg-cyan-500/30 transition" title="Lihat Detail ${item.symbol}">Detail</button>
            </div>
          </div>
        </div>
      `;
    }).join('');

    container.innerHTML = cardsHTML;
    if (window.lucide) window.lucide.createIcons();

    // Update overview metrics
    const avgMarketScore = (totalScore / (cachedSignalsData.length || 1)) || 0;
    const avgScoreEl = document.getElementById('metricAvgSentiment');
    const avgLabelEl = document.getElementById('metricSentimentLabel');
    const articleCountEl = document.getElementById('metricArticleCount');

    if (avgScoreEl) avgScoreEl.innerText = `${avgMarketScore >= 0 ? '+' : ''}${avgMarketScore.toFixed(2)}`;
    if (avgLabelEl) avgLabelEl.innerText = avgMarketScore >= 0.1 ? 'BULLISH BIAS' : (avgMarketScore <= -0.1 ? 'BEARISH BIAS' : 'NEUTRAL BIAS');
    if (articleCountEl) articleCountEl.innerText = totalArticles;

    // Update Top Marquee Ticker Tape
    updateTickerMarquee(cachedSignalsData);
  } catch (err) {
    console.error('Failed to render signals:', err);
  }
}

function updateTickerMarquee(data) {
  const marquee = document.getElementById('tickerMarquee');
  if (!marquee || !data || data.length === 0) return;

  const tickerHTML = data.map(item => {
    let colorClass = 'text-amber-400';
    if (item.signal === 'BUY') colorClass = 'text-emerald-400';
    if (item.signal === 'SELL') colorClass = 'text-rose-400';

    return `<span class="inline-flex items-center space-x-1.5"><strong class="text-white">${item.symbol}</strong> <span class="${colorClass}">${item.average_score > 0 ? '+' : ''}${item.average_score.toFixed(2)} ${item.signal}</span></span>`;
  }).join('');

  marquee.innerHTML = tickerHTML + tickerHTML; // Duplicate for smooth looping marquee
}

function selectStock(symbol) {
  currentSelectedSymbol = symbol;
  const badge = document.getElementById('selectedSymbolBadge');
  if (badge) badge.innerText = symbol;
  loadHistory(symbol);
  loadSignals(); // Refresh highlight border
}

// ─── SENTIMENT & PRICE DUAL-AXIS CHART ────────────────────

async function loadHistory(symbol) {
  try {
    const res = await fetch(`/api/v1/signals/${symbol}/history?days=30`);
    const result = await res.json();

    let points = result.data || [];

    // Check if price data is missing for this symbol
    const hasPrice = points.some(p => p.close_price && p.close_price > 0);
    if (!hasPrice) {
      try {
        const fetchRes = await fetch(`/api/v1/import/fetch-prices/${symbol}`, { method: 'POST' });
        const fetchResult = await fetchRes.json();
        if (fetchResult.success) {
          const reloadRes = await fetch(`/api/v1/signals/${symbol}/history?days=30`);
          const reloadResult = await reloadRes.json();
          points = reloadResult.data || [];
        }
      } catch (e) {
        console.warn('Auto-fetching Yahoo prices failed:', e);
      }
    }

    renderChart(points, symbol);
  } catch (err) {
    console.error('Failed to load history:', err);
  }
}

function renderChart(points, symbol) {
  const ctx = document.getElementById('sentimentChart')?.getContext('2d');
  if (!ctx) return;

  if (sentimentChartInstance) {
    sentimentChartInstance.destroy();
  }

  const labels = points.map(p => p.date);
  const sentimentScores = points.map(p => p.average_score);
  const priceData = points.map(p => p.close_price || null);

  // Gradient fill for sentiment
  const gradient = ctx.createLinearGradient(0, 0, 0, 250);
  gradient.addColorStop(0, 'rgba(6, 182, 212, 0.35)');
  gradient.addColorStop(1, 'rgba(6, 182, 212, 0.0)');

  const datasets = [
    {
      label: 'Sentiment Score',
      data: sentimentScores.length > 0 ? sentimentScores : [0],
      borderColor: '#06b6d4',
      borderWidth: 2,
      backgroundColor: gradient,
      fill: true,
      tension: 0.3,
      spanGaps: true,
      pointBackgroundColor: '#06b6d4',
      pointBorderColor: '#07090e',
      pointBorderWidth: 2,
      pointRadius: 4,
      yAxisID: 'y'
    }
  ];

  const hasPriceValues = priceData.some(p => p !== null);
  if (hasPriceValues) {
    datasets.push({
      label: 'Harga Saham (Rp)',
      data: priceData,
      borderColor: '#10b981',
      borderWidth: 2,
      backgroundColor: 'transparent',
      borderDash: [4, 4],
      tension: 0.2,
      spanGaps: true,
      pointBackgroundColor: '#10b981',
      pointBorderColor: '#07090e',
      pointBorderWidth: 2,
      pointRadius: 3,
      yAxisID: 'y1'
    });
  }

  sentimentChartInstance = new Chart(ctx, {
    type: 'line',
    data: {
      labels: labels.length > 0 ? labels : ['No Data'],
      datasets: datasets
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false
      },
      plugins: {
        legend: {
          display: true,
          position: 'top',
          align: 'end',
          labels: {
            color: '#94a3b8',
            font: { size: 10, family: 'Plus Jakarta Sans' },
            boxWidth: 12,
            usePointStyle: true
          }
        },
        tooltip: {
          backgroundColor: '#0e131f',
          borderColor: 'rgba(255, 255, 255, 0.1)',
          borderWidth: 1,
          titleColor: '#ffffff',
          bodyColor: '#cbd5e1',
          titleFont: { size: 11, weight: 'bold' },
          bodyFont: { size: 11, family: 'JetBrains Mono' },
          padding: 10,
          callbacks: {
            label: (context) => {
              if (context.dataset.yAxisID === 'y') {
                const score = context.parsed.y;
                return `Sentimen AI: ${score > 0 ? '+' : ''}${score.toFixed(2)}`;
              }
              if (context.dataset.yAxisID === 'y1') {
                return `Harga: Rp ${context.parsed.y.toLocaleString('id-ID')}`;
              }
              return `${context.dataset.label}: ${context.parsed.y}`;
            }
          }
        }
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { color: '#64748b', font: { size: 9, family: 'JetBrains Mono' }, maxRotation: 0 }
        },
        y: {
          type: 'linear',
          display: true,
          position: 'left',
          min: -1,
          max: 1,
          title: {
            display: false
          },
          grid: { color: 'rgba(255, 255, 255, 0.05)' },
          ticks: {
            color: '#06b6d4',
            font: { size: 9, family: 'JetBrains Mono' },
            callback: (val) => `${val > 0 ? '+' : ''}${val.toFixed(1)}`
          }
        },
        y1: {
          type: 'linear',
          display: hasPriceValues,
          position: 'right',
          title: {
            display: false
          },
          grid: { drawOnChartArea: false },
          ticks: {
            color: '#10b981',
            font: { size: 9, family: 'JetBrains Mono' },
            callback: (val) => `Rp ${val.toLocaleString('id-ID')}`
          }
        }
      }
    }
  });
}

// ─── NEWS FEED ─────────────────────────────────────────

function initNewsFilters() {
  const sel = document.getElementById('newsSentimentFilter');
  if (sel) sel.addEventListener('change', loadNews);
}

async function loadNews() {
  const container = document.getElementById('newsContainer');
  const sentiment = document.getElementById('newsSentimentFilter')?.value || '';

  let url = `/api/v1/articles?limit=10`;
  if (sentiment) url += `&sentiment=${sentiment}`;

  try {
    const res = await fetch(url);
    const result = await res.json();

    if (!result.success || !result.data || result.data.length === 0) {
      container.innerHTML = `<p class="text-xs text-slate-500 text-center py-4 font-mono">Belum ada berita terproses.</p>`;
      return;
    }

    const newsHTML = result.data.map(item => {
      let labelClass = 'text-amber-400 bg-amber-400/10 border-amber-400/20';
      if (item.sentiment_label === 'Bullish') labelClass = 'text-emerald-400 bg-emerald-400/10 border-emerald-400/20';
      if (item.sentiment_label === 'Bearish') labelClass = 'text-rose-400 bg-rose-400/10 border-rose-400/20';

      return `
        <div class="p-3 rounded-lg bg-slate-900/80 border border-slate-800 hover:border-slate-700 transition-all space-y-1.5">
          <div class="flex items-center justify-between text-[10px]">
            <span class="px-2 py-0.5 rounded font-mono uppercase font-bold border ${labelClass}">
              ${item.sentiment_label || 'Neutral'}
            </span>
            <span class="text-slate-500 font-mono">${item.source || 'news'}</span>
          </div>
          <a href="${item.url}" target="_blank" class="text-xs font-semibold text-slate-200 hover:text-cyan-400 transition-colors line-clamp-2">
            ${item.title}
          </a>
          ${item.summary ? `<p class="text-[11px] text-slate-400 line-clamp-2 leading-relaxed">${item.summary}</p>` : ''}
        </div>
      `;
    }).join('');

    container.innerHTML = newsHTML;
  } catch (err) {
    console.error('Failed to load news:', err);
  }
}

// ─── MANUAL SCRAPE & REPROCESS TRIGGERS ─────────────────

function initIngestionTrigger() {
  const btn = document.getElementById('triggerIngestBtn');
  const icon = document.getElementById('triggerBtnIcon');

  if (!btn) return;

  btn.addEventListener('click', async () => {
    btn.disabled = true;
    if (icon) icon.classList.add('animate-spin');

    showToast('info', 'Mulai scraping RSS & analisis Gemini AI...');

    try {
      const res = await fetch('/api/v1/ingestion/trigger', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          feeds: [
            { name: "CNBC Indonesia Market", url: "https://www.cnbcindonesia.com/market/rss" },
            { name: "Detik Finance", url: "https://finance.detik.com/rss" },
            { name: "Kontan", url: "https://rss.kontan.co.id/news/investasi" }
          ]
        })
      });

      const data = await res.json();

      if (data.success) {
        showToast('success', 'Scraping dimulai di background...');
      } else if (data.error?.code !== 'CONFLICT') {
        showToast('error', `Gagal: ${data.error?.message || 'Error occurred'}`);
      }
    } catch (err) {
      if (err.name !== 'AbortError') {
        console.warn('Ingestion trigger notice:', err);
      }
    } finally {
      btn.disabled = false;
      if (icon) icon.classList.remove('animate-spin');
    }
  });
}

function initReprocessTrigger() {
  const btn = document.getElementById('reprocessBtn');
  const icon = document.getElementById('reprocessBtnIcon');

  if (!btn) return;

  btn.addEventListener('click', async () => {
    btn.disabled = true;
    if (icon) icon.classList.add('animate-spin');

    try {
      const res = await fetch('/api/v1/ingestion/reprocess', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });

      const data = await res.json();

      if (data.success) {
        loadSignals(true);
        loadNews();
        if (currentSelectedSymbol) loadHistory(currentSelectedSymbol);
      } else if (data.error?.code !== 'CONFLICT') {
        showToast('error', `Gagal: ${data.error?.message || 'Error occurred'}`);
      }
    } catch (err) {
      if (err.name !== 'AbortError') {
        console.warn('Reprocess trigger notice:', err);
      }
    } finally {
      btn.disabled = false;
      if (icon) icon.classList.remove('animate-spin');
      setTimeout(() => {
        const banner = document.getElementById('processBanner');
        if (banner) banner.classList.add('hidden');
      }, 1200);
    }
  });
}

// ─── IMPORT DATA MODAL ─────────────────────────────────

function initImportModal() {
  const modal = document.getElementById('importModal');
  const openBtn = document.getElementById('openImportModalBtn');
  const closeBtn = document.getElementById('closeImportModalBtn');

  if (openBtn && modal) {
    openBtn.addEventListener('click', () => modal.classList.remove('hidden'));
  }
  if (closeBtn && modal) {
    closeBtn.addEventListener('click', () => modal.classList.add('hidden'));
  }

  // Modal Tabs
  const tabPricesBtn = document.getElementById('tabPricesBtn');
  const tabBrokerBtn = document.getElementById('tabBrokerBtn');
  const tabPricesContent = document.getElementById('tabPricesContent');
  const tabBrokerContent = document.getElementById('tabBrokerContent');

  if (tabPricesBtn && tabBrokerBtn) {
    tabPricesBtn.addEventListener('click', () => {
      tabPricesBtn.classList.add('text-cyan-400', 'border-b-2', 'border-cyan-400');
      tabPricesBtn.classList.remove('text-slate-400');
      tabBrokerBtn.classList.remove('text-cyan-400', 'border-b-2', 'border-cyan-400');
      tabBrokerBtn.classList.add('text-slate-400');
      tabPricesContent.classList.remove('hidden');
      tabBrokerContent.classList.add('hidden');
    });

    tabBrokerBtn.addEventListener('click', () => {
      tabBrokerBtn.classList.add('text-cyan-400', 'border-b-2', 'border-cyan-400');
      tabBrokerBtn.classList.remove('text-slate-400');
      tabPricesBtn.classList.remove('text-cyan-400', 'border-b-2', 'border-cyan-400');
      tabPricesBtn.classList.add('text-slate-400');
      tabBrokerContent.classList.remove('hidden');
      tabPricesContent.classList.add('hidden');
    });
  }

  // Fetch Yahoo Prices Button
  const fetchYfBtn = document.getElementById('fetchYfBtn');
  if (fetchYfBtn) {
    fetchYfBtn.addEventListener('click', async () => {
      const symbolInput = document.getElementById('yfSymbolInput');
      const symbol = symbolInput?.value.trim().toUpperCase();
      if (!symbol) {
        showToast('error', 'Masukkan kode ticker saham (contoh: BBCA)');
        return;
      }

      fetchYfBtn.disabled = true;
      const rangeStr = document.getElementById('yfRangeSelect')?.value || '1y';
      const rangeLabels = {'1mo':'1 bulan','6mo':'6 bulan','1y':'1 tahun','5y':'5 tahun','max':'all-time'};
      showToast('info', `Menarik harga ${rangeLabels[rangeStr] || rangeStr} ${symbol} dari Yahoo Finance...`);

      try {
        const res = await fetch(`/api/v1/import/fetch-prices/${symbol}?range=${rangeStr}`, { method: 'POST' });
        const data = await res.json();
        if (data.success) {
          showToast('success', `Berhasil mengimpor ${data.data.imported} data harga untuk ${symbol}!`);
          if (modal) modal.classList.add('hidden');
          loadHistory(symbol);
        } else {
          showToast('error', `Gagal: ${data.error?.message}`);
        }
      } catch (err) {
        showToast('error', 'Terjadi kesalahan koneksi.');
      } finally {
        fetchYfBtn.disabled = false;
      }
    });
  }

  // Upload CSV Prices Button
  const uploadCsvBtn = document.getElementById('uploadCsvBtn');
  if (uploadCsvBtn) {
    uploadCsvBtn.addEventListener('click', async () => {
      const fileInput = document.getElementById('csvFileInput');
      if (!fileInput || !fileInput.files[0]) {
        showToast('error', 'Pilih file CSV terlebih dahulu');
        return;
      }

      const formData = new FormData();
      formData.append('file', fileInput.files[0]);

      uploadCsvBtn.disabled = true;
      showToast('info', 'Mengunggah file CSV harga saham...');

      try {
        const res = await fetch('/api/v1/import/prices', {
          method: 'POST',
          body: formData
        });
        const data = await res.json();
        if (data.success) {
          showToast('success', data.data.message);
          if (modal) modal.classList.add('hidden');
          loadHistory(currentSelectedSymbol);
        } else {
          showToast('error', `Gagal: ${data.error?.message}`);
        }
      } catch (err) {
        showToast('error', 'Terjadi kesalahan koneksi.');
      } finally {
        uploadCsvBtn.disabled = false;
      }
    });
  }

  // Upload Broker CSV Button
  const uploadBrokerCsvBtn = document.getElementById('uploadBrokerCsvBtn');
  if (uploadBrokerCsvBtn) {
    uploadBrokerCsvBtn.addEventListener('click', async () => {
      const fileInput = document.getElementById('brokerCsvFileInput');
      if (!fileInput || !fileInput.files[0]) {
        showToast('error', 'Pilih file CSV terlebih dahulu');
        return;
      }

      const formData = new FormData();
      formData.append('file', fileInput.files[0]);

      uploadBrokerCsvBtn.disabled = true;
      showToast('info', 'Mengunggah file CSV broker summary...');

      try {
        const res = await fetch('/api/v1/import/broker-summary', {
          method: 'POST',
          body: formData
        });
        const data = await res.json();
        if (data.success) {
          showToast('success', data.data.message);
          if (modal) modal.classList.add('hidden');
          loadHistory(currentSelectedSymbol);
        } else {
          showToast('error', `Gagal: ${data.error?.message}`);
        }
      } catch (err) {
        showToast('error', 'Terjadi kesalahan koneksi.');
      } finally {
        uploadBrokerCsvBtn.disabled = false;
      }
    });
  }
}

// ─── TOAST NOTIFICATIONS ──────────────────────────────

function showToast(type, message) {
  const container = document.getElementById('toastContainer');
  if (!container) return;

  const toast = document.createElement('div');
  let bgClass = 'bg-[#0e131f] border-cyan-500/50 text-cyan-200';
  if (type === 'success') bgClass = 'bg-[#0e131f] border-emerald-500/50 text-emerald-200';
  if (type === 'error') bgClass = 'bg-[#0e131f] border-rose-500/50 text-rose-200';

  toast.className = `px-4 py-3 rounded-xl border ${bgClass} shadow-2xl text-xs font-mono font-medium backdrop-blur flex items-center space-x-2 transition-all transform translate-y-2 opacity-0`;
  toast.innerHTML = `<span>${message}</span>`;

  container.appendChild(toast);

  setTimeout(() => {
    toast.classList.remove('translate-y-2', 'opacity-0');
  }, 50);

  setTimeout(() => {
    toast.classList.add('opacity-0', 'translate-y-2');
    setTimeout(() => toast.remove(), 300);
  }, 4000);
}

// ─── REAL-TIME PROCESS STREAM ───────────────────────────

let liveProcessLogs = [];

function initSSEProcessStream() {
  const banner = document.getElementById('processBanner');
  const bannerMsg = document.getElementById('processBannerMsg');
  const bannerBar = document.getElementById('processBannerBar');
  const bannerPct = document.getElementById('processBannerPct');
  const stopBtn = document.getElementById('stopProcessBtn');

  const bannerClickable = document.getElementById('processBannerClickable');
  const modal = document.getElementById('processModal');
  const modalClose1 = document.getElementById('closeProcessModalBtn');
  const modalClose2 = document.getElementById('closeProcessModalBtn2');
  const modalStopBtn = document.getElementById('modalStopBtn');
  const modalStage = document.getElementById('modalStage');
  const modalCount = document.getElementById('modalCount');
  const modalLogList = document.getElementById('modalLogList');

  const handleStop = async () => {
    showToast('info', 'Mengirim perintah hentikan proses...');
    try {
      const res = await fetch('/api/v1/ingestion/cancel', { method: 'POST' });
      const data = await res.json();
      showToast('success', data.data?.message || 'Proses dihentikan');
    } catch (err) {
      showToast('error', 'Gagal menghentikan proses');
    } finally {
      if (banner) banner.classList.add('hidden');
      if (modal) modal.classList.add('hidden');
    }
  };

  if (stopBtn) stopBtn.addEventListener('click', handleStop);
  if (modalStopBtn) modalStopBtn.addEventListener('click', handleStop);

  if (bannerClickable && modal) {
    bannerClickable.addEventListener('click', () => {
      modal.classList.remove('hidden');
      renderProcessModalLogs();
    });
  }
  if (modalClose1) modalClose1.addEventListener('click', () => modal.classList.add('hidden'));
  if (modalClose2) modalClose2.addEventListener('click', () => modal.classList.add('hidden'));

  function renderProcessModalLogs() {
    if (!modalLogList) return;
    if (liveProcessLogs.length === 0) {
      modalLogList.innerHTML = `<div class="text-slate-500 text-center py-12">Menunggu log pemrosesan data real-time...</div>`;
      return;
    }

    modalLogList.innerHTML = liveProcessLogs.slice().reverse().map(item => `
      <div class="pt-2 pb-1 flex items-start justify-between space-x-3">
        <div class="space-y-0.5 min-w-0 flex-1">
          <div class="flex items-center space-x-2">
            <span class="text-[10px] px-1.5 py-0.5 rounded ${item.stage === 'SCRAPING' ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/30' : 'bg-purple-500/20 text-purple-300 border border-purple-500/30'} font-bold">${item.stage || 'PROCESS'}</span>
            <span class="text-slate-400 text-[11px] font-mono">${item.timestamp || ''}</span>
          </div>
          <p class="text-slate-200 font-sans font-medium text-xs truncate leading-snug">${item.message || ''}</p>
        </div>
        ${item.sentiment ? `<span class="text-[10px] px-2 py-0.5 rounded font-bold ${item.sentiment === 'Bullish' ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30' : (item.sentiment === 'Bearish' ? 'bg-rose-500/20 text-rose-400 border border-rose-500/30' : 'bg-amber-500/20 text-amber-400 border border-amber-500/30')}">${item.sentiment}</span>` : ''}
      </div>
    `).join('');
  }

  try {
    const eventSource = new EventSource('/api/v1/ingestion/stream');

    eventSource.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);

        if (data.type === 'start') {
          liveProcessLogs = [];
          if (banner) banner.classList.remove('hidden');
          if (bannerMsg) bannerMsg.innerText = data.message;
          if (bannerBar) bannerBar.style.width = '0%';
          if (bannerPct) bannerPct.innerText = '0%';
          if (modalStage) modalStage.innerText = data.stage || 'SCRAPING';
          if (modalCount) modalCount.innerText = `0 / ${data.total || 0}`;

          if (window._scrapeTimeout) clearTimeout(window._scrapeTimeout);
          window._scrapeTimeout = setTimeout(() => {
            if (banner) banner.classList.add('hidden');
            showToast('error', 'Proses timeout, silakan cek log server.');
          }, 180000);
        }

        if (data.type === 'start' || data.type === 'progress' || data.type === 'log') {
          if (banner) banner.classList.remove('hidden');
          if (bannerMsg) bannerMsg.innerText = data.message;
          const pct = data.total ? Math.min(Math.round((data.current / data.total) * 100), 100) : 0;
          if (bannerBar) bannerBar.style.width = `${pct}%`;
          if (bannerPct) bannerPct.innerText = `${pct}%`;

          if (modalStage) modalStage.innerText = data.stage || 'PROCESS';
          if (modalCount) modalCount.innerText = `${data.current || 0} / ${data.total || 0}`;

          liveProcessLogs.push(data);
          if (liveProcessLogs.length > 150) liveProcessLogs.shift();
          if (modal && !modal.classList.contains('hidden')) {
            renderProcessModalLogs();
          }
        }

        if (data.type === 'done' || data.type === 'cancelled') {
          if (window._scrapeTimeout) clearTimeout(window._scrapeTimeout);

          setTimeout(() => {
            if (banner) banner.classList.add('hidden');
            if (modal) modal.classList.add('hidden');
          }, 1500);

          showToast(data.type === 'done' ? 'success' : 'info', data.message);
          loadSignals(true);
          loadNews();
          if (currentSelectedSymbol) loadHistory(currentSelectedSymbol);
        }
      } catch (err) {
        console.error('Parsing SSE event error:', err);
      }
    };
  } catch (err) {
    console.error('EventSource initialization failed:', err);
  }
}

// ─── STOCK DETAIL PANEL ──────────────────────────────────

function initStockDetailPanel() {
  const panel = document.getElementById('stockDetailPanel');
  const overlay = document.getElementById('stockDetailOverlay');
  const closeBtn = document.getElementById('closeDetailPanel');

  if (closeBtn) closeBtn.addEventListener('click', closeStockDetail);
  if (overlay) overlay.addEventListener('click', closeStockDetail);

  // Range buttons in detail panel
  const rangeBtns = document.getElementById('detailRangeBtns');
  if (rangeBtns) {
    rangeBtns.querySelectorAll('button').forEach(btn => {
      btn.addEventListener('click', () => {
        const range = btn.dataset.range;
        rangeBtns.querySelectorAll('button').forEach(b => {
          b.className = 'px-2 py-0.5 text-[10px] font-mono rounded bg-slate-800 text-slate-400 hover:text-cyan-400 transition';
        });
        btn.className = 'px-2 py-0.5 text-[10px] font-mono rounded bg-cyan-500/20 text-cyan-400 border border-cyan-500/30 transition';
        loadDetailPriceChart(currentSelectedSymbol, range);
      });
    });
  }
}

function closeStockDetail() {
  const panel = document.getElementById('stockDetailPanel');
  const overlay = document.getElementById('stockDetailOverlay');
  if (panel) panel.classList.add('translate-x-full');
  if (overlay) overlay.classList.add('hidden');
  document.body.style.overflow = '';
}

async function openStockDetail(symbol) {
  const panel = document.getElementById('stockDetailPanel');
  const overlay = document.getElementById('stockDetailOverlay');
  if (!panel) return;

  // Update header
  document.getElementById('detailSymbol').textContent = symbol;
  document.getElementById('detailNewsSymbol').textContent = symbol;
  document.getElementById('detailCompanyName').textContent = 'Memuat...';

  // Reset sections
  document.getElementById('detailProfileContent').innerHTML = '<p class="text-slate-500 text-center py-4">Memuat profil...</p>';
  document.getElementById('detailMetricsContent').innerHTML = '<p class="col-span-2 text-slate-500 text-center py-4">Memuat metrik...</p>';
  document.getElementById('detailExecContent').innerHTML = '<p class="text-slate-500 text-center py-4">Memuat eksekutif...</p>';
  document.getElementById('detailNewsContent').innerHTML = '<p class="text-slate-500 text-center py-4 text-xs">Memuat berita...</p>';

  // Show panel
  if (overlay) overlay.classList.remove('hidden');
  panel.classList.remove('translate-x-full');
  document.body.style.overflow = 'hidden';
  panel.scrollTop = 0;

  // Re-init icons
  if (window.lucide) window.lucide.createIcons();

  // Load all data in parallel
  try {
    const detailRes = await fetch(`/api/v1/stocks/${symbol}/detail`);
    const detailData = await detailRes.json();

    if (detailData.success && detailData.data) {
      renderStockDetail(detailData.data, symbol);
    } else {
      // Stock might not exist yet, try to fetch profile from Yahoo
      document.getElementById('detailCompanyName').textContent = symbol;
      showToast('info', `Mengambil data ${symbol} dari Yahoo Finance...`);
      try {
        await fetch(`/api/v1/stocks/${symbol}/profile`, { method: 'POST' });
        const retry = await fetch(`/api/v1/stocks/${symbol}/detail`);
        const retryData = await retry.json();
        if (retryData.success && retryData.data) {
          renderStockDetail(retryData.data, symbol);
        }
      } catch (e) {
        console.warn('Profile fetch failed:', e);
      }
    }
  } catch (err) {
    console.error('Failed to load stock detail:', err);
  }

  // Load price chart (1y default)
  loadDetailPriceChart(symbol, '1y');
}

function renderStockDetail(data, symbol) {
  const stock = data.stock;
  const profile = data.profile;
  const executives = data.executives || [];
  const news = data.news || [];

  // Header
  document.getElementById('detailCompanyName').textContent = stock?.company_name || symbol;

  // Signal badge from current signals
  const badge = document.getElementById('detailSignalBadge');
  if (badge) {
    badge.className = 'px-2 py-0.5 rounded text-[10px] font-bold font-mono';
    badge.textContent = stock?.sector || '—';
    badge.classList.add('bg-slate-800', 'text-slate-300');
  }

  // Profile section
  if (profile) {
    renderProfile(profile);
    renderMetrics(profile);
  } else {
    document.getElementById('detailProfileContent').innerHTML = `
      <div class="text-center py-3">
        <p class="text-slate-500 text-xs mb-2">Profil belum tersedia</p>
        <button onclick="fetchAndRefreshProfile('${symbol}')" class="px-3 py-1.5 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg text-xs font-bold transition">Fetch dari Yahoo Finance</button>
      </div>`;
    document.getElementById('detailMetricsContent').innerHTML = '<p class="col-span-2 text-slate-500 text-center py-3 text-xs">Fetch profil untuk melihat metrik</p>';
  }

  // Executives
  renderExecutives(executives);

  // News
  renderDetailNews(news, symbol);

  if (window.lucide) window.lucide.createIcons();
}

function renderProfile(p) {
  const el = document.getElementById('detailProfileContent');
  if (!el) return;

  const rows = [];
  if (p.industry) rows.push(`<div class="flex justify-between py-1 border-b border-slate-800/50"><span class="text-slate-400">Industri</span><span class="text-white font-medium">${p.industry}</span></div>`);
  if (p.city || p.country) rows.push(`<div class="flex justify-between py-1 border-b border-slate-800/50"><span class="text-slate-400">Lokasi</span><span class="text-white font-medium">${[p.city, p.country].filter(Boolean).join(', ')}</span></div>`);
  if (p.employees) rows.push(`<div class="flex justify-between py-1 border-b border-slate-800/50"><span class="text-slate-400">Karyawan</span><span class="text-white font-medium">${p.employees.toLocaleString('id-ID')}</span></div>`);
  if (p.website) rows.push(`<div class="flex justify-between py-1 border-b border-slate-800/50"><span class="text-slate-400">Website</span><a href="${p.website}" target="_blank" class="text-cyan-400 hover:underline truncate max-w-[200px]">${p.website.replace(/^https?:\/\//, '')}</a></div>`);
  if (p.description) rows.push(`<div class="pt-2"><p class="text-slate-300 text-[11px] leading-relaxed line-clamp-4">${p.description}</p></div>`);

  el.innerHTML = rows.length > 0 ? rows.join('') : '<p class="text-slate-500 text-center py-3">Data profil tidak tersedia</p>';
}

function formatRupiah(val) {
  if (!val && val !== 0) return '—';
  const absVal = Math.abs(val);
  const sign = val < 0 ? '-' : '';
  if (absVal >= 1e12) return `${sign}Rp ${(absVal / 1e12).toFixed(1)}T`;
  if (absVal >= 1e9) return `${sign}Rp ${(absVal / 1e9).toFixed(1)}M`;
  if (absVal >= 1e6) return `${sign}Rp ${(absVal / 1e6).toFixed(1)}Jt`;
  return `${sign}Rp ${absVal.toLocaleString('id-ID')}`;
}

function renderMetrics(p) {
  const el = document.getElementById('detailMetricsContent');
  if (!el) return;

  const metricCard = (label, value, color = 'text-white') => `
    <div class="bg-slate-900/80 rounded-lg p-2.5 border border-slate-800/60">
      <p class="text-[10px] text-slate-400 uppercase tracking-wider">${label}</p>
      <p class="text-sm font-bold font-mono ${color} mt-0.5">${value}</p>
    </div>`;

  const cards = [];
  if (p.market_cap) cards.push(metricCard('Market Cap', formatRupiah(p.market_cap), 'text-cyan-400'));
  if (p.trailing_pe) cards.push(metricCard('PER (P/E)', `${p.trailing_pe.toFixed(1)}x`, 'text-white'));
  if (p.price_to_book) cards.push(metricCard('PBV (P/B)', `${p.price_to_book.toFixed(2)}x`, 'text-white'));
  if (p.trailing_eps) cards.push(metricCard('EPS', `Rp ${p.trailing_eps.toFixed(0)}`, 'text-emerald-400'));
  if (p.dividend_yield) cards.push(metricCard('Dividen Yield', `${(p.dividend_yield * 100).toFixed(2)}%`, 'text-amber-400'));
  if (p.return_on_equity) cards.push(metricCard('ROE', `${(p.return_on_equity * 100).toFixed(1)}%`, 'text-emerald-400'));
  if (p.debt_to_equity) cards.push(metricCard('DER', `${p.debt_to_equity.toFixed(1)}%`, p.debt_to_equity > 100 ? 'text-rose-400' : 'text-white'));
  if (p.total_revenue) cards.push(metricCard('Revenue', formatRupiah(p.total_revenue), 'text-white'));
  if (p.net_income) cards.push(metricCard('Net Income', formatRupiah(p.net_income), p.net_income > 0 ? 'text-emerald-400' : 'text-rose-400'));
  if (p.total_debt) cards.push(metricCard('Total Utang', formatRupiah(p.total_debt), 'text-rose-400'));
  if (p.total_assets) cards.push(metricCard('Total Aset', formatRupiah(p.total_assets), 'text-white'));
  if (p.week_52_high) cards.push(metricCard('52W High', `Rp ${p.week_52_high.toLocaleString('id-ID')}`, 'text-emerald-400'));
  if (p.week_52_low) cards.push(metricCard('52W Low', `Rp ${p.week_52_low.toLocaleString('id-ID')}`, 'text-rose-400'));
  if (p.shares_outstanding) cards.push(metricCard('Saham Beredar', `${(p.shares_outstanding / 1e9).toFixed(2)}B`, 'text-white'));

  el.innerHTML = cards.length > 0 ? cards.join('') : '<p class="col-span-2 text-slate-500 text-center py-3">Metrik tidak tersedia</p>';
}

function renderExecutives(execs) {
  const el = document.getElementById('detailExecContent');
  if (!el) return;

  if (!execs || execs.length === 0) {
    el.innerHTML = '<p class="text-slate-500 text-center py-3">Data eksekutif belum tersedia</p>';
    return;
  }

  el.innerHTML = execs.map(e => `
    <div class="flex items-center space-x-2.5 py-1.5 border-b border-slate-800/40">
      <div class="w-7 h-7 rounded-full bg-gradient-to-br from-cyan-500/20 to-purple-500/20 border border-slate-700 flex items-center justify-center text-[10px] font-bold text-cyan-400">
        ${e.name.charAt(0)}
      </div>
      <div class="min-w-0">
        <p class="text-white font-medium truncate">${e.name}</p>
        <p class="text-[10px] text-slate-400">${e.title || '—'}</p>
      </div>
    </div>
  `).join('');
}

function renderDetailNews(news, symbol) {
  const el = document.getElementById('detailNewsContent');
  if (!el) return;

  if (!news || news.length === 0) {
    el.innerHTML = `<p class="text-slate-500 text-center py-4 text-xs">Belum ada berita terkait ${symbol}</p>`;
    return;
  }

  el.innerHTML = news.map(n => {
    const label = n.sentiment_label || 'Neutral';
    let labelClass = 'text-slate-400 bg-slate-800';
    if (label === 'Bullish') labelClass = 'text-emerald-400 bg-emerald-500/10';
    if (label === 'Bearish') labelClass = 'text-rose-400 bg-rose-500/10';

    const dateStr = n.published_at ? new Date(n.published_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }) : '';

    return `
      <a href="${n.url}" target="_blank" class="block p-2.5 rounded-lg bg-slate-900/60 border border-slate-800/60 hover:border-cyan-500/30 transition group">
        <div class="flex items-start justify-between gap-2">
          <p class="text-[11px] text-slate-200 font-medium leading-snug group-hover:text-cyan-300 transition line-clamp-2">${n.title}</p>
          <span class="px-1.5 py-0.5 rounded text-[9px] font-bold font-mono whitespace-nowrap ${labelClass}">${label}</span>
        </div>
        <div class="flex items-center space-x-2 mt-1.5 text-[10px] text-slate-500">
          <span>${n.source || ''}</span>
          ${dateStr ? `<span>· ${dateStr}</span>` : ''}
          ${n.sentiment_score != null ? `<span>· Score: ${n.sentiment_score > 0 ? '+' : ''}${n.sentiment_score.toFixed(2)}</span>` : ''}
        </div>
      </a>`;
  }).join('');
}

async function fetchAndRefreshProfile(symbol) {
  showToast('info', `Mengambil profil ${symbol} dari Yahoo Finance...`);
  try {
    const res = await fetch(`/api/v1/stocks/${symbol}/profile`, { method: 'POST' });
    const data = await res.json();
    if (data.success) {
      showToast('success', `Profil ${symbol} berhasil diperbarui!`);
      openStockDetail(symbol);
    } else {
      showToast('error', `Gagal: ${data.error?.message || 'Unknown error'}`);
    }
  } catch (err) {
    showToast('error', 'Gagal mengambil profil dari Yahoo Finance');
  }
}

async function loadDetailPriceChart(symbol, range) {
  const ctx = document.getElementById('detailPriceChart')?.getContext('2d');
  if (!ctx) return;

  if (detailChartInstance) {
    detailChartInstance.destroy();
    detailChartInstance = null;
  }

  try {
    // First try to fetch prices for this range
    await fetch(`/api/v1/import/fetch-prices/${symbol}?range=${range}`, { method: 'POST' });

    // Get days mapping
    const daysMap = { '1mo': 30, '6mo': 180, '1y': 365, '5y': 1825, 'max': 9999 };
    const days = daysMap[range] || 365;

    const res = await fetch(`/api/v1/signals/${symbol}/history?days=${days}`);
    const result = await res.json();
    const points = result.data || [];

    const labels = points.map(p => p.date);
    const priceData = points.map(p => p.close_price || null);

    const gradient = ctx.createLinearGradient(0, 0, 0, 180);
    gradient.addColorStop(0, 'rgba(16, 185, 129, 0.25)');
    gradient.addColorStop(1, 'rgba(16, 185, 129, 0.0)');

    detailChartInstance = new Chart(ctx, {
      type: 'line',
      data: {
        labels: labels.length > 0 ? labels : ['No Data'],
        datasets: [{
          label: 'Harga (Rp)',
          data: priceData.length > 0 ? priceData : [0],
          borderColor: '#10b981',
          borderWidth: 2,
          backgroundColor: gradient,
          fill: true,
          tension: 0.2,
          spanGaps: true,
          pointRadius: points.length > 60 ? 0 : 2,
          pointHoverRadius: 4,
          pointBackgroundColor: '#10b981',
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: 'index', intersect: false },
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: '#0e131f',
            borderColor: 'rgba(255,255,255,0.1)',
            borderWidth: 1,
            titleColor: '#fff',
            bodyColor: '#cbd5e1',
            bodyFont: { size: 11, family: 'JetBrains Mono' },
            callbacks: {
              label: (ctx) => `Rp ${ctx.parsed.y?.toLocaleString('id-ID') || '—'}`
            }
          }
        },
        scales: {
          x: {
            grid: { display: false },
            ticks: { color: '#64748b', font: { size: 8, family: 'JetBrains Mono' }, maxRotation: 0, maxTicksLimit: 8 }
          },
          y: {
            grid: { color: 'rgba(255,255,255,0.04)' },
            ticks: {
              color: '#10b981',
              font: { size: 9, family: 'JetBrains Mono' },
              callback: (val) => `Rp ${(val/1000).toFixed(0)}k`
            }
          }
        }
      }
    });
  } catch (err) {
    console.error('Failed to load detail price chart:', err);
  }
}

// ─── IHSG MARKET OVERVIEW ───────────────────────────────

let ihsgChartInstance = null;

async function loadIHSG() {
  try {
    const res = await fetch('/api/v1/market/ihsg');
    if (!res.ok) return;
    const result = await res.json();
    if (!result.success || !result.data) return;

    const data = result.data;
    
    const formatter = new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    
    const priceEl = document.getElementById('ihsgPrice');
    const changeEl = document.getElementById('ihsgChange');
    const changePctEl = document.getElementById('ihsgChangePct');
    const updateTimeEl = document.getElementById('ihsgUpdateTime');
    
    if (priceEl) priceEl.innerText = formatter.format(data.price);
    
    if (changeEl && changePctEl) {
      const isUp = data.change >= 0;
      const sign = isUp ? '+' : '';
      const colorClass = isUp ? 'text-emerald-400' : 'text-rose-400';
      const bgClass = isUp ? 'bg-emerald-500/10' : 'bg-rose-500/10';
      
      changeEl.innerText = `${sign}${formatter.format(data.change)}`;
      changeEl.className = `text-sm font-bold font-mono ${colorClass}`;
      
      changePctEl.innerText = `${sign}${formatter.format(data.change_pct)}%`;
      changePctEl.className = `text-[11px] font-mono px-1.5 py-0.5 rounded ${colorClass} ${bgClass}`;
    }
    
    if (updateTimeEl && data.date) {
      updateTimeEl.innerText = `Last update: ${data.date}`;
    }

    const ctx = document.getElementById('ihsgSparkline')?.getContext('2d');
    if (ctx && data.sparkline && data.sparkline.length > 0) {
      if (ihsgChartInstance) ihsgChartInstance.destroy();
      
      const chartData = data.sparkline;
      const labels = data.spark_dates || chartData.map((_, i) => i);
      
      const gradient = ctx.createLinearGradient(0, 0, 0, 40);
      gradient.addColorStop(0, 'rgba(34, 211, 238, 0.3)');
      gradient.addColorStop(1, 'rgba(34, 211, 238, 0)');

      ihsgChartInstance = new Chart(ctx, {
        type: 'line',
        data: {
          labels: labels,
          datasets: [{
            data: chartData,
            borderColor: '#22d3ee',
            borderWidth: 1.5,
            backgroundColor: gradient,
            fill: true,
            tension: 0.4,
            pointRadius: 0
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { display: false }, tooltip: { enabled: false } },
          scales: {
            x: { display: false },
            y: { display: false, min: Math.min(...chartData) * 0.999, max: Math.max(...chartData) * 1.001 }
          },
          layout: { padding: 0 }
        }
      });
    }
  } catch (err) {
    console.error('Failed to load IHSG:', err);
  }
}
