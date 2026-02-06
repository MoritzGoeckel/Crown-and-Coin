let ws = null;
let currentUser = null;
let currentSecret = null;

const loginScreen = document.getElementById('login-screen');
const gameScreen = document.getElementById('game-screen');
const usernameInput = document.getElementById('username');
const secretInput = document.getElementById('secret');
const registerBtn = document.getElementById('register-btn');
const connectBtn = document.getElementById('connect-btn');
const loginError = document.getElementById('login-error');
const userInfo = document.getElementById('user-info');
const stateDisplay = document.getElementById('state-display');
const actionsList = document.getElementById('actions-list');
const consoleOutput = document.getElementById('console-output');
const consoleInput = document.getElementById('console-input');
const sendBtn = document.getElementById('send-btn');

registerBtn.addEventListener('click', register);
connectBtn.addEventListener('click', connect);
sendBtn.addEventListener('click', sendConsoleMessage);
consoleInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') sendConsoleMessage();
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
        log('Connected to server', 'received');
        refreshState();
        refreshActions();
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        log('Received: ' + JSON.stringify(data, null, 2), data.success === false ? 'error' : 'received');

        if (data.state) {
            stateDisplay.textContent = JSON.stringify(data.state, null, 2);
        }

        if (data.actions) {
            renderActions(data.actions);
        }
    };

    ws.onerror = (err) => {
        log('WebSocket error', 'error');
    };

    ws.onclose = () => {
        log('Disconnected from server', 'error');
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
    send({ type: 'get_actions', player_id: currentUser });
}

function renderActions(actions) {
    actionsList.innerHTML = '';

    if (actions.length === 0) {
        const empty = document.createElement('div');
        empty.textContent = 'No actions available';
        empty.style.color = '#666';
        actionsList.appendChild(empty);
        return;
    }

    actions.forEach(action => {
        const btn = document.createElement('button');
        btn.textContent = formatAction(action);
        btn.addEventListener('click', () => {
            send({ type: 'submit', actions: [action] });
            setTimeout(refreshState, 100);
            setTimeout(refreshActions, 100);
        });
        actionsList.appendChild(btn);
    });
}

function formatAction(action) {
    switch (action.type) {
        case 'tax_peasants_low':
            return 'Tax Peasants (Low)';
        case 'tax_peasants_high':
            return 'Tax Peasants (High)';
        case 'tax_merchants':
            return `Tax ${action.merchant_id}`;
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
