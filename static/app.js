let ws = null;
let currentUser = null;
let currentSecret = null;
let gameState = null;
let lastStateJSON = '';
let connectedPlayers = [];
let refreshInterval = null;
let pendingUserActions = 0;

const loginScreen = document.getElementById('login-screen');
const gameScreen = document.getElementById('game-screen');
const usernameInput = document.getElementById('username');
const secretInput = document.getElementById('secret');
const registerBtn = document.getElementById('register-btn');
const connectBtn = document.getElementById('connect-btn');
const loginError = document.getElementById('login-error');
const userInfo = document.getElementById('user-info');
const phaseInfo = document.getElementById('phase-info');
const countriesDisplay = document.getElementById('countries-display');
const merchantsDisplay = document.getElementById('merchants-display');
const actionsList = document.getElementById('actions-list');
const adminPanel = document.getElementById('admin-panel');
const consoleOutput = document.getElementById('console-output');
const consoleInput = document.getElementById('console-input');
const sendBtn = document.getElementById('send-btn');
const logoutBtn = document.getElementById('logout-btn');

registerBtn.addEventListener('click', register);
connectBtn.addEventListener('click', connect);
sendBtn.addEventListener('click', sendConsoleMessage);
logoutBtn.addEventListener('click', logout);
consoleInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') sendConsoleMessage();
});

document.getElementById('advance-btn').addEventListener('click', () => {
    send({ type: 'advance' });
});

document.getElementById('add-country-btn').addEventListener('click', () => {
    const countryId = document.getElementById('new-country-id').value.trim();
    const monarchId = document.getElementById('new-monarch-id').value;
    if (countryId && monarchId) {
        send({ type: 'add_country', country_id: countryId, monarch_id: monarchId });
        document.getElementById('new-country-id').value = '';
        document.getElementById('new-monarch-id').value = '';
    }
});

document.getElementById('add-merchant-btn').addEventListener('click', () => {
    const merchantId = document.getElementById('new-merchant-id').value.trim();
    const countryId = document.getElementById('merchant-country-select').value;
    if (merchantId && countryId) {
        send({ type: 'add_merchant', player_id: merchantId, country_id: countryId });
        document.getElementById('new-merchant-id').value = '';
    }
});

document.getElementById('remove-player-btn').addEventListener('click', () => {
    const playerId = document.getElementById('remove-player-select').value;
    if (playerId) {
        send({ type: 'remove_merchant', player_id: playerId });
    }
});

async function register() {
    const name = usernameInput.value.trim();
    const secret = secretInput.value.trim();

    if (!name || !secret) {
        loginError.textContent = 'Please enter username and secret';
        return;
    }

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, secret })
        });

        if (response.ok) {
            loginError.textContent = '';
            loginError.style.color = '#4ecdc4';
            loginError.textContent = 'Registration successful! Click Connect.';
        } else {
            const text = await response.text();
            loginError.style.color = '#ff6b6b';
            loginError.textContent = text;
        }
    } catch (err) {
        loginError.style.color = '#ff6b6b';
        loginError.textContent = 'Connection error';
    }
}

function connect() {
    const name = usernameInput.value.trim();
    const secret = secretInput.value.trim();

    if (!name || !secret) {
        loginError.textContent = 'Please enter username and secret';
        return;
    }

    currentUser = name;
    currentSecret = secret;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = () => {
        loginScreen.classList.add('hidden');
        gameScreen.classList.remove('hidden');
        userInfo.textContent = `Logged in as: ${currentUser}`;

        if (currentUser === 'admin') {
            adminPanel.classList.remove('hidden');
            document.getElementById('game-content').style.gridTemplateColumns = '1fr 1fr 1fr 1fr';
        } else {
            document.getElementById('game-content').style.gridTemplateColumns = '1fr 1fr 1fr';
        }

        log('Connected to server', 'received');
        refreshState();
        refreshActions();

        refreshInterval = setInterval(() => {
            refreshState();
            refreshActions();
        }, 5000);
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        log('Received: ' + JSON.stringify(data, null, 2), data.success === false ? 'error' : 'received');

        // Handle connected_players broadcast
        if (data.type === 'connected_players') {
            connectedPlayers = (data.players || []).sort();
            renderConnectedPlayers();
            updateMonarchSelect();
            return;
        }

        // Only re-render state if it actually changed
        if (data.state) {
            const newStateJSON = JSON.stringify(data.state);
            if (newStateJSON !== lastStateJSON) {
                lastStateJSON = newStateJSON;
                gameState = data.state;
                renderState(data.state);
                updateAdminSelects();
            }
        }

        if (data.actions !== undefined) {
            renderActions(data.actions);
        }

        // After receiving response to a user action, request fresh state
        if (pendingUserActions > 0) {
            pendingUserActions--;
            refreshState();
            refreshActions();
        }
    };

    ws.onerror = (err) => {
        log('WebSocket error', 'error');
    };

    ws.onclose = () => {
        log('Disconnected from server', 'error');
        if (refreshInterval) {
            clearInterval(refreshInterval);
            refreshInterval = null;
        }
    };
}

function send(payload) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        log('Not connected', 'error');
        return;
    }

    const message = {
        user: currentUser,
        secret: currentSecret,
        payload: payload
    };

    ws.send(JSON.stringify(message));
    log('Sent: ' + JSON.stringify(payload, null, 2), 'sent');

    // Track non-refresh messages so we can auto-refresh after response
    const type = payload.type;
    if (type !== 'get_state' && type !== 'get_actions' && type !== 'get_connected_players') {
        pendingUserActions++;
    }
}

function sendConsoleMessage() {
    const input = consoleInput.value.trim();
    if (!input) return;

    try {
        const payload = JSON.parse(input);
        send(payload);
        consoleInput.value = '';
    } catch (err) {
        log('Invalid JSON: ' + err.message, 'error');
    }
}

function log(message, type = 'received') {
    const entry = document.createElement('div');
    entry.className = `log-entry ${type}`;
    entry.textContent = message;
    consoleOutput.appendChild(entry);
    consoleOutput.scrollTop = consoleOutput.scrollHeight;
}

function refreshState() {
    send({ type: 'get_state' });
}

function refreshActions() {
    if (currentUser && currentUser !== 'admin') {
        send({ type: 'get_actions', player_id: currentUser });
    }
}

function logout() {
    if (ws) {
        ws.close();
        ws = null;
    }
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
    currentUser = null;
    currentSecret = null;
    gameState = null;
    lastStateJSON = '';
    connectedPlayers = [];
    pendingUserActions = 0;

    gameScreen.classList.add('hidden');
    adminPanel.classList.add('hidden');
    loginScreen.classList.remove('hidden');

    usernameInput.value = '';
    secretInput.value = '';
    loginError.textContent = '';
    loginError.style.color = '#ff6b6b';

    countriesDisplay.innerHTML = '';
    merchantsDisplay.innerHTML = '';
    actionsList.innerHTML = '';
    consoleOutput.innerHTML = '';
}

function renderState(state) {
    phaseInfo.textContent = `Turn ${state.turn} - ${formatPhase(state.phase)}`;

    countriesDisplay.innerHTML = '';
    for (const [id, country] of Object.entries(state.countries || {})) {
        const card = document.createElement('div');
        card.className = 'country-card';
        if (country.hp <= 0) card.classList.add('defeated');

        const status = country.is_republic ? 'Republic' : `Monarch: ${country.monarch_id}`;
        const healthPercent = Math.max(0, (country.hp / 10) * 100);

        card.innerHTML = `
            <div class="country-header">
                <span class="country-name">${country.country_id}</span>
                <span class="country-status">${status}</span>
            </div>
            <div class="health-bar">
                <div class="health-fill" style="width: ${healthPercent}%"></div>
                <span class="health-text">${country.hp} HP</span>
            </div>
            <div class="country-stats">
                <div class="stat">
                    <span class="stat-label">Gold</span>
                    <span class="stat-value">${country.gold}</span>
                </div>
                <div class="stat">
                    <span class="stat-label">Army</span>
                    <span class="stat-value">${country.army_strength}</span>
                </div>
                <div class="stat">
                    <span class="stat-label">Peasants</span>
                    <span class="stat-value">${country.peasants}</span>
                </div>
            </div>
        `;
        countriesDisplay.appendChild(card);
    }

    merchantsDisplay.innerHTML = '';
    for (const [id, merchant] of Object.entries(state.merchants || {})) {
        const card = document.createElement('div');
        card.className = 'merchant-card';

        card.innerHTML = `
            <div class="merchant-header">
                <span class="merchant-name">${merchant.player_id}</span>
                <span class="merchant-location">${merchant.country_id}</span>
            </div>
            <div class="merchant-stats">
                <div class="stat">
                    <span class="stat-label">Stored</span>
                    <span class="stat-value">${merchant.stored_gold}</span>
                </div>
                <div class="stat">
                    <span class="stat-label">Invested</span>
                    <span class="stat-value">${merchant.invested_gold}</span>
                </div>
            </div>
        `;
        merchantsDisplay.appendChild(card);
    }
}

function updateAdminSelects() {
    if (!gameState || currentUser !== 'admin') return;

    const countrySelect = document.getElementById('merchant-country-select');
    const prevCountry = countrySelect.value;
    countrySelect.innerHTML = '';
    for (const countryId of Object.keys(gameState.countries || {})) {
        const option = document.createElement('option');
        option.value = countryId;
        option.textContent = countryId;
        countrySelect.appendChild(option);
    }
    if (prevCountry) countrySelect.value = prevCountry;

    const playerSelect = document.getElementById('remove-player-select');
    const prevPlayer = playerSelect.value;
    playerSelect.innerHTML = '';
    for (const merchantId of Object.keys(gameState.merchants || {})) {
        const option = document.createElement('option');
        option.value = merchantId;
        option.textContent = merchantId;
        playerSelect.appendChild(option);
    }
    if (prevPlayer) playerSelect.value = prevPlayer;
}

function renderConnectedPlayers() {
    const list = document.getElementById('connected-players-list');
    if (!list) return;

    list.innerHTML = '';
    if (connectedPlayers.length === 0) {
        list.innerHTML = '<div class="no-players">No players connected</div>';
        return;
    }

    connectedPlayers.forEach(name => {
        const tag = document.createElement('span');
        tag.className = 'player-tag';
        tag.textContent = name;
        list.appendChild(tag);
    });
}

function updateMonarchSelect() {
    const select = document.getElementById('new-monarch-id');
    if (!select) return;

    const prevValue = select.value;
    select.innerHTML = '<option value="">Select Monarch...</option>';

    connectedPlayers.forEach(name => {
        const option = document.createElement('option');
        option.value = name;
        option.textContent = name;
        select.appendChild(option);
    });

    if (prevValue) select.value = prevValue;
}

function formatPhase(phase) {
    const phases = {
        'taxation': 'Taxation',
        'negotiation': 'Negotiation',
        'spending': 'Spending',
        'war': 'War',
        'assessment': 'Assessment'
    };
    return phases[phase] || phase;
}

function parseAmountRange(value) {
    if (typeof value === 'string') {
        const match = value.match(/^<AMOUNT:(\d+)-(\d+)>$/);
        if (match) {
            return { min: parseInt(match[1]), max: parseInt(match[2]) };
        }
    }
    return null;
}

function renderActions(actions) {
    actionsList.innerHTML = '';

    if (!actions || actions.length === 0) {
        const empty = document.createElement('div');
        empty.textContent = 'No actions available';
        empty.style.color = '#666';
        actionsList.appendChild(empty);
        return;
    }

    actions.forEach(action => {
        const range = parseAmountRange(action.amount);

        if (range) {
            const container = document.createElement('div');
            container.className = 'action-with-amount';

            const label = document.createElement('span');
            label.className = 'action-label';
            label.textContent = formatActionLabel(action);

            const input = document.createElement('input');
            input.type = 'number';
            input.min = range.min;
            input.max = range.max;
            input.value = range.max > 0 ? range.max : range.min;
            input.className = 'amount-input';

            const btn = document.createElement('button');
            btn.textContent = 'Go';
            btn.addEventListener('click', () => {
                const amount = parseInt(input.value);
                if (!isNaN(amount) && amount >= range.min && amount <= range.max) {
                    const submitAction = { ...action, amount };
                    send({ type: 'submit', actions: [submitAction] });
                }
            });

            container.appendChild(label);
            container.appendChild(input);
            container.appendChild(btn);
            actionsList.appendChild(container);
        } else {
            const btn = document.createElement('button');
            btn.textContent = formatAction(action);
            btn.addEventListener('click', () => {
                send({ type: 'submit', actions: [action] });
            });
            actionsList.appendChild(btn);
        }
    });
}

function formatActionLabel(action) {
    switch (action.type) {
        case 'tax_merchants':
            return `Tax ${action.merchant_id}`;
        case 'merchant_invest':
            return 'Invest';
        case 'build_army':
            return 'Build Army';
        default:
            return formatAction(action);
    }
}

function formatAction(action) {
    switch (action.type) {
        case 'tax_peasants_low':
            return 'Tax Peasants (Low)';
        case 'tax_peasants_high':
            return 'Tax Peasants (High)';
        case 'tax_merchants':
            return `Tax ${action.merchant_id} (${action.amount})`;
        case 'build_army':
            return `Build Army (${action.amount})`;
        case 'merchant_invest':
            return `Invest ${action.amount}`;
        case 'merchant_hide':
            return 'Hide Gold';
        case 'attack':
            return `Attack ${action.target_id}`;
        case 'remain':
            return 'Remain';
        case 'flee':
            return `Flee to ${action.target_id}`;
        case 'revolt':
            return 'Revolt!';
        default:
            return action.type;
    }
}
