console.log('app.js v8 loaded - with UI updates');

const searchInput = document.getElementById('searchInput');
const resultsContainer = document.getElementById('results');

let debounceTimer;

if (searchInput) {
    searchInput.addEventListener('input', (e) => {
        clearTimeout(debounceTimer);
        const query = e.target.value.trim();

        if (query.length === 0) {
            resultsContainer.innerHTML = '<div class="empty-state"><p>Start typing to search...</p></div>';
            return;
        }

        debounceTimer = setTimeout(() => {
            fetchStocks(query);
        }, 300);
    });
}

async function fetchStocks(query) {
    try {
        const response = await fetch(`/search?q=${encodeURIComponent(query)}`);
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const stocks = await response.json();
        displayResults(stocks);
    } catch (error) {
        console.error('Error fetching stocks:', error);
        resultsContainer.innerHTML = '<div class="empty-state"><p>Error fetching results. Please try again.</p></div>';
    }
}

function displayResults(stocks) {
    resultsContainer.innerHTML = '';

    if (!stocks || stocks.length === 0) {
        resultsContainer.innerHTML = '<div class="empty-state"><p>No stocks found.</p></div>';
        return;
    }

    stocks.forEach(stock => {
        const card = document.createElement('a');
        card.className = 'stock-card';
        card.href = `/stock.html?symbol=${encodeURIComponent(stock.symbol)}&exchange=${encodeURIComponent(stock.exchange)}`;
        card.innerHTML = `
            <div class="card-header">
                <span class="symbol">${stock.symbol}</span>
                <span class="exchange-badge">${stock.exchange}</span>
            </div>
            <div class="name">${stock.name}</div>
        `;
        resultsContainer.appendChild(card);
    });
}

async function fetchStockDetails(symbol, period = '1D', exchange = '') {
    const container = document.getElementById('stock-detail');
    try {
        let url = `/api/stock?symbol=${encodeURIComponent(symbol)}&period=${encodeURIComponent(period)}`;
        if (exchange) {
            url += `&exchange=${encodeURIComponent(exchange)}`;
        }
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error('Stock not found');
        }
        const data = await response.json();

        // Format Price
        const priceFormatter = new Intl.NumberFormat('en-IN', {
            style: 'currency',
            currency: 'INR'
        });
        const formattedPrice = priceFormatter.format(data.currentPrice);

        // Calculate percentage change
        let percentageChange = 0;
        let changeClass = '';
        let changeSymbol = '';
        let priceChangeValue = 0;
        let isUp = false;

        // Determine start price for calculation
        let startPrice = data.previousDayClose;
        if (period !== '1D' && data.history && data.history.length > 0) {
            startPrice = data.history[0].price;
        }

        if (startPrice && startPrice > 0) {
            const currentPrice = data.currentPrice;
            priceChangeValue = currentPrice - startPrice;
            percentageChange = (priceChangeValue / startPrice) * 100;
            isUp = percentageChange >= 0;
            changeClass = isUp ? 'positive' : 'negative';
        }

        const priceChangeFormatted = new Intl.NumberFormat('en-IN', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }).format(Math.abs(priceChangeValue));

        const percentChangeFormatted = Math.abs(percentageChange).toFixed(2);
        const sign = isUp ? '+' : '-';
        const colorClass = isUp ? 'positive' : 'negative';

        container.innerHTML = `
            <div class="detail-header">
                <div class="detail-symbol">${data.name}</div>
                <div class="price-container">
                    <span class="current-price">${formattedPrice}</span>
                    <span class="price-change ${colorClass}">
                        ${sign}${priceChangeFormatted} (${percentChangeFormatted}%)
                        <span class="time-period-label">${period}</span>
                    </span>
                </div>
            </div>

            <div class="chart-section">
                <div class="chart-container">
                    <canvas id="stockChart"></canvas>
                </div>
                
                <div class="controls-container">
                    <div class="left-controls">
                         <button class="control-btn">${data.exchange}</button>
                    </div>
                    
                    <div class="time-period-filters">
                        <button class="period-btn ${period === '1D' ? 'active' : ''}" data-period="1D">1D</button>
                        <button class="period-btn ${period === '1W' ? 'active' : ''}" data-period="1W">1W</button>
                        <button class="period-btn ${period === '1M' ? 'active' : ''}" data-period="1M">1M</button>
                        <button class="period-btn ${period === '6M' ? 'active' : ''}" data-period="6M">6M</button>
                        <button class="period-btn ${period === 'YTD' ? 'active' : ''}" data-period="YTD">YTD</button>
                        <button class="period-btn ${period === '1Y' ? 'active' : ''}" data-period="1Y">1Y</button>
                        <button class="period-btn ${period === '5Y' ? 'active' : ''}" data-period="5Y">5Y</button>
                    </div>

                    <div class="right-controls">
                        <button class="control-btn terminal-btn">
                            Terminal 
                            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                        </button>
                    </div>
                </div>
            </div>
        `;

        // Validate history data
        if (!data.history || data.history.length === 0) {
            // container.innerHTML += '<div class="error">No historical data available for this stock.</div>';
            // Don't show error, just empty chart or something?
            return;
        }

        // Store symbol and exchange globally for re-fetching
        window.stockSymbol = data.symbol;
        window.stockExchange = data.exchange;

        // Render chart with fetched data
        if (period === 'YTD') {
            console.log('YTD History Data Sample:', data.history.slice(0, 3), data.history.slice(-3));
        }
        renderStockChart(data.history, period, data.previousDayClose, isUp);

        // Add event listeners to period buttons
        document.querySelectorAll('.period-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const selectedPeriod = e.currentTarget.dataset.period;
                // Map new periods to closest supported if needed, or just pass through
                // For now, pass through. Backend might default to 1Y if not found, or we can handle it.
                // Actually, let's map 3M -> 6M, 3Y -> 5Y, All -> 5Y for now if backend doesn't support them to avoid errors.
                // But better to try fetching and let backend handle or default.
                fetchStockDetails(window.stockSymbol, selectedPeriod, window.stockExchange);
            });
        });

    } catch (error) {
        console.error('Error fetching details:', error);
        container.innerHTML = `<div class="error">Could not load details for ${symbol}</div>`;
    }
}

function formatDateLabel(dateStr, period) {
    const date = new Date(dateStr);
    switch (period) {
        case '1D':
            // Show time for 1 day view (e.g., "10:30")
            return date.toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit', hour12: false });
        case '1W':
        case '1M':
            // For hourly/daily data in short range, show day and month
            return date.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
        case '6M':
        case '1Y':
        case 'YTD':
            // Show month and year for longer views
            return date.toLocaleDateString('en-IN', { month: 'short', year: 'numeric' });
        case '5Y':
            // Show year for 5Y view (e.g., "2024")
            return date.toLocaleDateString('en-IN', { year: 'numeric' });
        default:
            return date.toLocaleDateString('en-IN');
    }
}

function renderStockChart(historyData, period, previousClose, isUp) {
    const ctx = document.getElementById('stockChart').getContext('2d');

    if (window.myStockChart) {
        window.myStockChart.destroy();
    }

    if (!historyData || historyData.length === 0) {
        return;
    }

    // Colors
    const lineColor = isUp ? '#00d09c' : '#eb5b3c'; // Green if up, Red if down
    const baselineColor = '#d1d5db'; // Light gray for dashed line

    // Format labels based on period
    const labels = historyData.map(p => formatDateLabel(p.date, period));

    // Determine max number of ticks based on period
    let maxTicksLimit;
    switch (period) {
        case '1D':
            maxTicksLimit = 8; // Show ~8 time labels for 1 day
            break;
        case '1W':
            maxTicksLimit = 7; // Show all 7 days
            break;
        case '1M':
            maxTicksLimit = 10; // Show ~10 dates for 1 month
            break;
        case '6M':
            maxTicksLimit = 6; // Show ~6 months
            break;
        case '1Y':
            maxTicksLimit = 12; // Show ~12 months
            break;
        case 'YTD':
            maxTicksLimit = 8; // Show ~8 months/points
            break;
        case '5Y':
            maxTicksLimit = 5; // Show ~5 years
            break;
        default:
            maxTicksLimit = 10;
    }

    // Prepare data for baseline (previous close)
    // We want a dashed line across the chart at the previousClose level
    // But Chart.js annotation plugin is better for this. 
    // Since we might not have the plugin, we can add a dataset or just rely on the main line.
    // The design shows a dashed line. Let's try to add a dataset with constant value.

    const baselineData = previousClose ? new Array(historyData.length).fill(previousClose) : [];

    window.myStockChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [
                {
                    label: 'Price',
                    data: historyData.map(p => p.price),
                    borderColor: lineColor,
                    backgroundColor: 'transparent',
                    borderWidth: 2,
                    fill: false,
                    tension: 0.1,
                    pointRadius: 0,
                    pointHoverRadius: 4,
                    pointHoverBackgroundColor: '#fff',
                    pointHoverBorderColor: lineColor,
                    pointHoverBorderWidth: 2
                },
                // Baseline dataset
                ...(previousClose ? [{
                    label: 'Previous Close',
                    data: baselineData,
                    borderColor: baselineColor,
                    borderWidth: 1,
                    borderDash: [5, 5],
                    pointRadius: 0,
                    fill: false,
                    tension: 0
                }] : [])
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                intersect: false,
                mode: 'index',
            },
            plugins: {
                legend: {
                    display: false
                },
                tooltip: {
                    backgroundColor: '#fff',
                    titleColor: '#444',
                    bodyColor: '#444',
                    borderColor: '#e5e7eb',
                    borderWidth: 1,
                    displayColors: false,
                    padding: 10,
                    callbacks: {
                        title: function (context) {
                            // Show full date and time in tooltip
                            const index = context[0].dataIndex;
                            const date = new Date(historyData[index].date);
                            // console.log('Tooltip:', index, historyData[index].date, date);

                            // For daily/weekly intervals (6M+), show only date
                            if (['6M', '1Y', '5Y', 'YTD'].includes(period)) {
                                return date.toLocaleDateString('en-IN', {
                                    year: 'numeric',
                                    month: 'short',
                                    day: 'numeric'
                                });
                            }

                            // For intraday (1D, 1W, 1M), show date and time
                            return date.toLocaleString('en-IN', {
                                year: 'numeric',
                                month: 'short',
                                day: 'numeric',
                                hour: '2-digit',
                                minute: '2-digit',
                                hour12: true
                            });
                        },
                        label: function (context) {
                            if (context.dataset.label === 'Previous Close') return 'Prev Close: ₹' + context.parsed.y.toFixed(2);
                            return 'Price: ₹' + context.parsed.y.toFixed(2);
                        }
                    }
                }
            },
            scales: {
                y: {
                    display: true,
                    position: 'right',
                    grid: {
                        display: false,
                        drawBorder: false
                    },
                    ticks: {
                        display: false // Hide Y axis labels as per clean design? Or keep them? 
                        // The image has no Y axis labels visible, or very subtle. 
                        // Let's hide them for now to match the "clean" look, or maybe just very subtle.
                        // Actually, let's keep them but make them subtle.
                        // Wait, looking at the image again... there are NO Y-axis labels.
                    }
                },
                x: {
                    display: true,
                    grid: {
                        display: false,
                        drawBorder: false
                    },
                    ticks: {
                        display: false // Hide X axis labels too? The image has NO X axis labels!
                        // Wait, the image DOES NOT have date labels on the X axis.
                        // It just has the line.
                        // Okay, I will hide X axis ticks.
                    }
                }
            }
        }
    });
}

