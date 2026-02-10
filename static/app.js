let ws = null;
let currentUser = null;
let currentSecret = null;
let gameState = null;
let lastStateJSON = '';
let connectedPlayers = [];
let refreshInterval = null;

const loginScreen = document.getElementById('login-screen');
const gameScreen = document.getElementById('game-screen');
const loginUsernameInput = document.getElementById('login-username');
const loginSecretInput = document.getElementById('login-secret');
const signupUsernameInput = document.getElementById('signup-username');
const signupSecretInput = document.getElementById('signup-secret');
const loginBtn = document.getElementById('login-btn');
const signupBtn = document.getElementById('signup-btn');
const loginError = document.getElementById('login-error');
const signupError = document.getElementById('signup-error');
const userInfo = document.getElementById('user-info');
const phaseInfo = document.getElementById('phase-info');
const countriesDisplay = document.getElementById('countries-display');
const merchantsDisplay = document.getElementById('merchants-display');
const actionsList = document.getElementById('actions-list');
const queuedActionsList = document.getElementById('queued-actions-list');
const adminPanel = document.getElementById('admin-panel');
const logoutBtn = document.getElementById('logout-btn');

loginBtn.addEventListener('click', login);
signupBtn.addEventListener('click', signup);
logoutBtn.addEventListener('click', logout);
loginUsernameInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') login();
});
loginSecretInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') login();
});
signupUsernameInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') signup();
});
signupSecretInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') signup();
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
    const merchantId = document.getElementById('new-merchant-id').value;
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

async function signup() {
    const name = signupUsernameInput.value.trim();
    const secret = signupSecretInput.value.trim();

    if (!name || !secret) {
        signupError.textContent = 'Please enter username and secret';
        signupError.style.color = '#ff6b6b';
        return;
    }

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, secret })
        });

        if (response.ok) {
            signupError.textContent = '';
            // Auto-login after successful registration
            connectToServer(name, secret);
        } else {
            const text = await response.text();
            signupError.style.color = '#ff6b6b';
            signupError.textContent = text;
        }
    } catch (err) {
        signupError.style.color = '#ff6b6b';
        signupError.textContent = 'Connection error';
    }
}

async function login() {
    const name = loginUsernameInput.value.trim();
    const secret = loginSecretInput.value.trim();

    if (!name || !secret) {
        loginError.textContent = 'Please enter username and secret';
        loginError.style.color = '#ff6b6b';
        return;
    }

    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, secret })
        });

        if (response.ok) {
            loginError.textContent = '';
            connectToServer(name, secret);
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

function connectToServer(name, secret) {
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
            document.getElementById('game-content').style.gridTemplateColumns = '1fr 1fr 1fr';
        } else {
            document.getElementById('game-content').style.gridTemplateColumns = '1fr 1fr';
        }

        log('Connected to server', 'received');
        refreshState();
        refreshActions();
        refreshQueuedActions();

        refreshInterval = setInterval(() => {
            refreshState();
            refreshActions();
            refreshQueuedActions();
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
            updateMerchantSelect();
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
            // Check if this is a queued actions response (has phase but no player_id)
            if (data.phase !== undefined && data.player_id === undefined && data.state === undefined) {
                renderQueuedActions(data.actions);
            } else {
                renderActions(data.actions);
            }
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
}

function log(message, type = 'received') {
    const prefix = type === 'sent' ? '→' : type === 'error' ? '✗' : '←';
    console.log(`${prefix} ${message}`);
}

function refreshState() {
    send({ type: 'get_state' });
}

function refreshActions() {
    if (currentUser && currentUser !== 'admin') {
        send({ type: 'get_actions', player_id: currentUser });
    }
}

function refreshQueuedActions() {
    if (currentUser === 'admin') {
        // Admin sees all queued actions
        send({ type: 'get_queued' });
    } else if (currentUser) {
        // Players see only their queued actions
        send({ type: 'get_queued', player_id: currentUser });
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

    gameScreen.classList.add('hidden');
    adminPanel.classList.add('hidden');
    loginScreen.classList.remove('hidden');

    loginUsernameInput.value = '';
    loginSecretInput.value = '';
    signupUsernameInput.value = '';
    signupSecretInput.value = '';
    loginError.textContent = '';
    loginError.style.color = '#ff6b6b';
    signupError.textContent = '';
    signupError.style.color = '#ff6b6b';

    countriesDisplay.innerHTML = '';
    merchantsDisplay.innerHTML = '';
    actionsList.innerHTML = '';
    queuedActionsList.innerHTML = '';
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

    // Add all active players (merchants and monarchs)
    const activePlayers = new Set();

    // Add merchants
    for (const merchantId of Object.keys(gameState.merchants || {})) {
        activePlayers.add(merchantId);
    }

    // Add monarchs
    for (const country of Object.values(gameState.countries || {})) {
        if (country.monarch_id && !country.is_republic) {
            activePlayers.add(country.monarch_id);
        }
    }

    // Populate dropdown with all active players
    Array.from(activePlayers).sort().forEach(playerId => {
        const option = document.createElement('option');
        option.value = playerId;
        option.textContent = playerId;
        playerSelect.appendChild(option);
    });

    if (prevPlayer) playerSelect.value = prevPlayer;

    updateMerchantSelect();
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

function updateMerchantSelect() {
    const select = document.getElementById('new-merchant-id');
    if (!select) return;

    const prevValue = select.value;
    select.innerHTML = '<option value="">Select Merchant...</option>';

    // Get current monarchs
    const monarchs = new Set();
    if (gameState && gameState.countries) {
        for (const country of Object.values(gameState.countries)) {
            if (country.monarch_id && !country.is_republic) {
                monarchs.add(country.monarch_id);
            }
        }
    }

    // Get existing merchants
    const existingMerchants = new Set();
    if (gameState && gameState.merchants) {
        for (const merchant of Object.values(gameState.merchants)) {
            existingMerchants.add(merchant.player_id);
        }
    }

    // Filter available players
    connectedPlayers.forEach(name => {
        if (!monarchs.has(name) && !existingMerchants.has(name)) {
            const option = document.createElement('option');
            option.value = name;
            option.textContent = name;
            select.appendChild(option);
        }
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

function getActionKey(action) {
    // Create a unique identifier for this action to preserve input values
    return `${action.type}_${action.player_id || ''}_${action.merchant_id || ''}_${action.country_id || ''}`;
}

function renderActions(actions) {
    // Save current input values before clearing
    const savedValues = {};
    actionsList.querySelectorAll('.amount-input').forEach(input => {
        const key = input.dataset.actionKey;
        if (key) {
            savedValues[key] = input.value;
        }
    });

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
            input.className = 'amount-input';

            // Generate action key and store it on the input
            const actionKey = getActionKey(action);
            input.dataset.actionKey = actionKey;

            // Restore saved value if it exists and is valid, otherwise use max
            const savedValue = savedValues[actionKey];
            if (savedValue !== undefined) {
                const numValue = parseInt(savedValue);
                if (!isNaN(numValue) && numValue >= range.min && numValue <= range.max) {
                    input.value = numValue;
                } else {
                    input.value = range.max > 0 ? range.max : range.min;
                }
            } else {
                input.value = range.max > 0 ? range.max : range.min;
            }

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

function renderQueuedActions(actions) {
    queuedActionsList.innerHTML = '';

    if (!actions || actions.length === 0) {
        const empty = document.createElement('div');
        empty.textContent = 'No queued actions';
        empty.style.color = '#666';
        empty.style.fontSize = '0.9em';
        queuedActionsList.appendChild(empty);
        return;
    }

    actions.forEach(action => {
        const item = document.createElement('div');
        item.className = 'queued-action-item';
        item.style.padding = '8px';
        item.style.marginBottom = '4px';
        item.style.backgroundColor = '#2a2a2a';
        item.style.borderRadius = '4px';
        item.style.fontSize = '0.9em';

        const playerLabel = document.createElement('span');
        playerLabel.style.color = '#4ecdc4';
        playerLabel.style.fontWeight = 'bold';
        playerLabel.textContent = action.player_id + ': ';

        const actionText = document.createElement('span');
        actionText.style.color = '#e0e0e0';
        actionText.textContent = formatAction(action);

        item.appendChild(playerLabel);
        item.appendChild(actionText);
        queuedActionsList.appendChild(item);
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
        case 'no_attack':
            return 'No Attack';
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
