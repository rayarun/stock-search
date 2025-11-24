console.log('MarketFinder Mobile v1.0');

const searchInput = document.getElementById('mobileSearchInput');
const resultsContainer = document.getElementById('mobileResults');
const homeScreen = document.getElementById('home-screen');
const detailScreen = document.getElementById('detail-screen');

let debounceTimer;
let currentChart = null;

// Search functionality
if (searchInput) {
    searchInput.addEventListener('input', (e) => {
        clearTimeout(debounceTimer);
        const query = e.target.value.trim();

        if (query.length === 0) {
            resultsContainer.innerHTML = `
                <div class="empty-state">
                    <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <line x1="12" y1="1" x2="12" y2="23"></line>
                        <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
                    </svg>
                    <p>Start typing to search stocks</p>
                </div>
            `;
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
        resultsContainer.innerHTML = '<div class="error">Error fetching results. Please try again.</div>';
    }
}

function displayResults(stocks) {
    resultsContainer.innerHTML = '';

    if (!stocks || stocks.length === 0) {
        resultsContainer.innerHTML = `
            <div class="empty-state">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="11" cy="11" r="8"></circle>
                    <path d="m21 21-4.35-4.35"></path>
                </svg>
                <p>No stocks found</p>
            </div>
        `;
        return;
    }

    stocks.forEach(stock => {
        const card = document.createElement('div');
        card.className = 'stock-card';
        card.onclick = () => showStockDetail(stock.symbol, stock.exchange);
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

function showStockDetail(symbol, exchange) {
    homeScreen.classList.remove('active');
    detailScreen.classList.add('active');

    document.getElementById('stockName').textContent = symbol;
    document.getElementById('stockExchange').textContent = exchange;

    fetchStockDetails(symbol, '1D', exchange);
}

function goBack() {
    detailScreen.classList.remove('active');
    homeScreen.classList.add('active');

    if (currentChart) {
        currentChart.destroy();
        currentChart = null;
    }
}

async function fetchStockDetails(symbol, period = '1D', exchange = '') {
    const container = document.getElementById('stockDetailContent');

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

        // Format price
        const priceFormatter = new Intl.NumberFormat('en-IN', {
            style: 'currency',
            currency: 'INR'
        });
        const formattedPrice = priceFormatter.format(data.currentPrice);

        // Calculate percentage change
        let startPrice = data.previousDayClose;
        if (period !== '1D' && data.history && data.history.length > 0) {
            startPrice = data.history[0].price;
        }

        let priceChangeValue = 0;
        let percentageChange = 0;
        let isUp = false;

        if (startPrice && startPrice > 0) {
            priceChangeValue = data.currentPrice - startPrice;
            percentageChange = (priceChangeValue / startPrice) * 100;
            isUp = percentageChange >= 0;
        }

        const priceChangeFormatted = new Intl.NumberFormat('en-IN', {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        }).format(Math.abs(priceChangeValue));

        const percentChangeFormatted = Math.abs(percentageChange).toFixed(2);
        const sign = isUp ? '+' : '-';
        const colorClass = isUp ? 'positive' : 'negative';

        container.innerHTML = `
            <div class="price-section">
                <span class="current-price">${formattedPrice}</span>
                <div class="price-change ${colorClass}">
                    ${sign}${priceChangeFormatted} (${percentChangeFormatted}%)
                    <span class="time-period-label">${period}</span>
                </div>
            </div>

            <div class="period-filters">
                <button class="period-btn ${period === '1D' ? 'active' : ''}" onclick="changePeriod('${symbol}', '1D', '${exchange}')">1D</button>
                <button class="period-btn ${period === '1W' ? 'active' : ''}" onclick="changePeriod('${symbol}', '1W', '${exchange}')">1W</button>
                <button class="period-btn ${period === '1M' ? 'active' : ''}" onclick="changePeriod('${symbol}', '1M', '${exchange}')">1M</button>
                <button class="period-btn ${period === '6M' ? 'active' : ''}" onclick="changePeriod('${symbol}', '6M', '${exchange}')">6M</button>
                <button class="period-btn ${period === 'YTD' ? 'active' : ''}" onclick="changePeriod('${symbol}', 'YTD', '${exchange}')">YTD</button>
                <button class="period-btn ${period === '1Y' ? 'active' : ''}" onclick="changePeriod('${symbol}', '1Y', '${exchange}')">1Y</button>
                <button class="period-btn ${period === '5Y' ? 'active' : ''}" onclick="changePeriod('${symbol}', '5Y', '${exchange}')">5Y</button>
            </div>

            <div class="chart-container">
                <canvas id="mobileChart"></canvas>
            </div>
        `;

        if (data.history && data.history.length > 0) {
            renderChart(data.history, period, data.previousDayClose, isUp);
        }

    } catch (error) {
        console.error('Error fetching details:', error);
        container.innerHTML = `<div class="error">Could not load details for ${symbol}</div>`;
    }
}

function changePeriod(symbol, period, exchange) {
    fetchStockDetails(symbol, period, exchange);
}

function renderChart(historyData, period, previousClose, isUp) {
    if (currentChart) {
        currentChart.destroy();
    }

    const ctx = document.getElementById('mobileChart');
    if (!ctx) return;

    const lineColor = isUp ? '#00d09c' : '#eb5b3c';
    const labels = historyData.map(p => formatDateLabel(p.date, period));

    currentChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Price',
                data: historyData.map(p => p.price),
                borderColor: lineColor,
                backgroundColor: 'transparent',
                borderWidth: 2,
                fill: false,
                tension: 0.1,
                pointRadius: 0,
                pointHoverRadius: 5,
                pointHoverBackgroundColor: '#fff',
                pointHoverBorderColor: lineColor,
                pointHoverBorderWidth: 2
            }]
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
                    padding: 12,
                    callbacks: {
                        title: function (context) {
                            const index = context[0].dataIndex;
                            const date = new Date(historyData[index].date);

                            if (['6M', '1Y', '5Y', 'YTD'].includes(period)) {
                                return date.toLocaleDateString('en-IN', {
                                    year: 'numeric',
                                    month: 'short',
                                    day: 'numeric'
                                });
                            }

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
                        display: true,
                        color: 'rgba(0, 0, 0, 0.05)',
                        drawBorder: false
                    },
                    ticks: {
                        display: true,
                        color: '#7c7e8c',
                        font: {
                            size: 10
                        },
                        padding: 8,
                        callback: function (value) {
                            return '₹' + value.toFixed(0);
                        }
                    }
                },
                x: {
                    display: true,
                    grid: {
                        display: false,
                        drawBorder: false
                    },
                    ticks: {
                        display: true,
                        color: '#7c7e8c',
                        font: {
                            size: 9
                        },
                        maxRotation: 0,
                        autoSkip: true,
                        maxTicksLimit: 5
                    }
                }
            }
        }
    });
}

function formatDateLabel(dateStr, period) {
    const date = new Date(dateStr);
    switch (period) {
        case '1D':
            return date.toLocaleTimeString('en-IN', { hour: '2-digit', minute: '2-digit', hour12: false });
        case '1W':
        case '1M':
            return date.toLocaleDateString('en-IN', { day: 'numeric', month: 'short' });
        case '6M':
        case '1Y':
        case 'YTD':
            return date.toLocaleDateString('en-IN', { month: 'short', year: 'numeric' });
        case '5Y':
            return date.toLocaleDateString('en-IN', { year: 'numeric' });
        default:
            return date.toLocaleDateString('en-IN');
    }
}
