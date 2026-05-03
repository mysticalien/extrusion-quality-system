const refreshIntervalMs = 2000;
const authTokenStorageKey = "authToken";

const parameterNames = {
    pressure: "Давление",
    moisture: "Влажность",
    barrel_temperature_zone_1: "Температура зоны 1",
    barrel_temperature_zone_2: "Температура зоны 2",
    barrel_temperature_zone_3: "Температура зоны 3",
    screw_speed: "Скорость шнека",
    drive_load: "Нагрузка привода",
    outlet_temperature: "Температура продукта на выходе",
    process_risk: "Риск нестабильности процесса"
};

const unitNames = {
    bar: "бар",
    percent: "%",
    celsius: "°C",
    rpm: "об/мин"
};

const stateNames = {
    stable: "стабильно",
    warning: "предупреждение",
    unstable: "нестабильно",
    critical: "критично",
    normal: "норма"
};

const roleNames = {
    operator: "оператор",
    technologist: "технолог",
    admin: "администратор"
};

const eventLevelNames = {
    warning: "предупреждение",
    critical: "критично"
};

const eventStatusNames = {
    active: "активно",
    acknowledged: "подтверждено",
    resolved: "устранено"
};

const anomalyTypeNames = {
    jump: "скачок",
    drift: "дрейф",
    combined_risk: "комбинированный риск"
};

let currentUser = null;
let lastHistoryReadings = [];
let historyIsLoading = false;
let dashboardIntervalID = null;
let historyIntervalID = null;
let setpoints = [];
let users = [];
let activeTab = "overview";

function canManageSetpoints() {
    return currentUser?.role === "technologist" || currentUser?.role === "admin";
}

function canViewAnomalies() {
    return currentUser?.role === "technologist" || currentUser?.role === "admin";
}

function canViewHistory() {
    return currentUser?.role === "technologist" || currentUser?.role === "admin";
}

function canManageUsers() {
    return currentUser?.role === "admin";
}

function canViewQualityWeights() {
    return currentUser?.role === "technologist" || currentUser?.role === "admin";
}

function formatUnit(unit) {
    return unitNames[unit] || unit || "—";
}

function formatState(state) {
    return stateNames[state] || state || "неизвестно";
}

function formatRole(role) {
    return roleNames[role] || role || "—";
}

function formatParameter(parameterType) {
    return parameterNames[parameterType] || parameterType || "—";
}

function formatEventLevel(level) {
    return eventLevelNames[level] || level || "—";
}

function formatEventStatus(status) {
    return eventStatusNames[status] || status || "—";
}

function formatAnomalyType(type) {
    return anomalyTypeNames[type] || type || "аномалия";
}

function formatTelemetryValue(value, unit) {
    return `${formatNumber(value)} ${formatUnit(unit)}`;
}

function formatAlertMessage(event) {
    const parameter = formatParameter(event.parameterType);
    const value = formatTelemetryValue(event.value, event.unit);

    if (event.level === "critical") {
        return `Критическое отклонение параметра «${parameter}»: ${value}.`;
    }

    if (event.level === "warning") {
        return `Предупредительное отклонение параметра «${parameter}»: ${value}.`;
    }

    return `Отклонение параметра «${parameter}»: ${value}.`;
}

function formatAnomalyMessage(anomaly) {
    const parameter = formatParameter(anomaly.parameterType);

    if (anomaly.type === "jump") {
        return `Обнаружен резкий скачок параметра «${parameter}».`;
    }

    if (anomaly.type === "drift") {
        return `Обнаружен устойчивый дрейф параметра «${parameter}» за последние измерения.`;
    }

    if (anomaly.type === "combined_risk") {
        return "Обнаружен комбинированный риск нестабильности процесса: влажность снижается, а давление и нагрузка привода растут.";
    }

    return anomaly.message || "Обнаружена аномалия технологического процесса.";
}

function switchTab(tabName) {
    activeTab = tabName;

    document.querySelectorAll(".tab-button").forEach((button) => {
        button.classList.toggle("active", button.dataset.tab === tabName);
    });

    document.querySelectorAll(".tab-panel").forEach((panel) => {
        panel.classList.remove("active");
    });

    const activePanel = document.getElementById(`${tabName}Tab`);
    if (activePanel) {
        activePanel.classList.add("active");
    }

    if (tabName === "history") {
        drawHistoryChart(lastHistoryReadings);
    }
}

function applyRolePermissions() {
    const setpointsSection = document.getElementById("setpointsSection");
    if (setpointsSection) {
        setpointsSection.classList.toggle("hidden", !canManageSetpoints());
    }

    const anomaliesSection = document.getElementById("anomaliesSection");
    if (anomaliesSection) {
        anomaliesSection.classList.toggle("hidden", !canViewAnomalies());
    }

    const historySection = document.getElementById("historySection");
    if (historySection) {
        historySection.classList.toggle("hidden", !canViewHistory());
    }

    const usersSection = document.getElementById("usersSection");
    if (usersSection) {
        usersSection.classList.toggle("hidden", !canManageUsers());
    }

    const usersTabButton = document.querySelector('[data-tab="users"]');
    if (usersTabButton) {
        usersTabButton.classList.toggle("hidden", !canManageUsers());
    }

    const qualityWeightsSection = document.getElementById("qualityWeightsSection");
    if (qualityWeightsSection) {
        qualityWeightsSection.classList.toggle("hidden", !canViewQualityWeights());
    }
}

async function loadCurrentUser() {
    currentUser = await fetchJSON("/api/me");
    applyRolePermissions();
}

function renderUserPanel() {
    const loginSection = document.getElementById("loginSection");
    const userPanel = document.getElementById("userPanel");
    const dashboard = document.getElementById("dashboard");
    const currentUserInfo = document.getElementById("currentUserInfo");

    if (!currentUser) {
        loginSection.classList.remove("hidden");
        userPanel.classList.add("hidden");
        dashboard.classList.add("hidden");
        return;
    }

    loginSection.classList.add("hidden");
    userPanel.classList.remove("hidden");
    dashboard.classList.remove("hidden");

    currentUserInfo.textContent = `${currentUser.username} · ${formatRole(currentUser.role)}`;
}

function showError(message) {
    const errorElement = document.getElementById("error");
    errorElement.textContent = message;
    errorElement.style.display = "block";
}

function hideError() {
    const errorElement = document.getElementById("error");
    errorElement.textContent = "";
    errorElement.style.display = "none";
}

async function readAPIError(response) {
    let code = "";
    let serverMessage = "";

    try {
        const body = await response.json();

        if (typeof body.error === "string") {
            serverMessage = body.error;
        } else if (body.error && typeof body.error === "object") {
            code = body.error.code || "";
            serverMessage = body.error.message || "";
        }
    } catch {
        try {
            serverMessage = await response.text();
        } catch {
            serverMessage = "";
        }
    }

    switch (code || serverMessage) {
        case "invalid_username_or_password":
        case "invalid username or password":
            return "Неверный логин или пароль";

        case "missing_authorization_token":
        case "missing authorization token":
            return "Необходимо войти в систему";

        case "invalid_authorization_token":
        case "invalid authorization token":
            return "Сессия недействительна. Войдите заново";

        case "authorization_token_expired":
        case "authorization token expired":
            return "Сессия истекла. Войдите заново";

        case "user is inactive or not found":
            return "Пользователь не найден или деактивирован";

        case "forbidden":
            return "Недостаточно прав для выполнения действия";

        case "user_already_exists":
        case "user already exists":
            return "Пользователь с таким логином уже существует";

        case "password must contain at least 12 characters":
        case "new_password_too_short":
        case "new password must contain at least 12 characters":
            return "Пароль должен содержать минимум 12 символов";

        case "old_password_required":
        case "old password is required":
            return "Введите текущий пароль";

        case "old_password_incorrect":
        case "old password is incorrect":
            return "Текущий пароль указан неверно";

        case "new_password_same_as_old":
        case "new password must be different from old password":
            return "Новый пароль должен отличаться от текущего";

        case "username is required":
            return "Введите логин";

        case "invalid_user_role":
        case "invalid user role":
            return "Некорректная роль пользователя";

        case "invalid_json_body":
        case "invalid JSON body":
            return "Некорректные данные формы";

        case "setpoint_not_found":
        case "setpoint not found":
            return "Уставка не найдена";

        case "quality_weight_not_found":
        case "quality weight not found":
            return "Вес параметра не найден";

        case "weight_must_be_positive":
        case "weight must be positive":
            return "Вес должен быть больше нуля";

        case "weight_too_large":
        case "weight must not be greater than 10":
            return "Вес не должен быть больше 10";

        case "validation_error":
            return serverMessage || "Ошибка валидации";

        default:
            return serverMessage || `Ошибка запроса: ${response.status}`;
    }
}

async function fetchJSON(url, options = {}) {
    const token = localStorage.getItem(authTokenStorageKey);

    const headers = {
        ...(options.headers || {})
    };

    if (token) {
        headers.Authorization = `Bearer ${token}`;
    }

    const response = await fetch(url, {
        ...options,
        headers
    });

    if (!response.ok) {
        const message = await readAPIError(response);
        throw new Error(message);
    }

    return response.json();
}

async function login() {
    const usernameInput = document.getElementById("loginUsername");
    const passwordInput = document.getElementById("loginPassword");
    const loginButton = document.getElementById("loginButton");

    try {
        hideError();

        loginButton.disabled = true;
        loginButton.textContent = "Входим...";

        const response = await fetch("/api/login", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                username: usernameInput.value.trim(),
                password: passwordInput.value
            })
        });

        if (!response.ok) {
            const message = await readAPIError(response);
            throw new Error(message);
        }

        const result = await response.json();

        localStorage.setItem(authTokenStorageKey, result.token);
        passwordInput.value = "";

        await initializeAuthenticatedDashboard();
    } catch (error) {
        showError(error.message);
    } finally {
        loginButton.disabled = false;
        loginButton.textContent = "Войти";
    }
}

function logout() {
    localStorage.removeItem(authTokenStorageKey);
    currentUser = null;
    lastHistoryReadings = [];
    setpoints = [];
    users = [];

    stopPolling();
    clearSensitiveForms();
    renderUserPanel();
    applyRolePermissions();
    hideError();
}

function clearSensitiveForms() {
    const passwordFields = [
        "loginPassword",
        "changeOldPassword",
        "changeNewPassword",
        "newPassword"
    ];

    for (const fieldID of passwordFields) {
        const element = document.getElementById(fieldID);
        if (element) {
            element.value = "";
        }
    }

    const resultElements = [
        "changePasswordResult",
        "userResult",
        "setpointResult"
    ];

    for (const elementID of resultElements) {
        const element = document.getElementById(elementID);
        if (element) {
            element.textContent = "";
        }
    }
}

async function changeOwnPassword() {
    const oldPasswordInput = document.getElementById("changeOldPassword");
    const newPasswordInput = document.getElementById("changeNewPassword");
    const resultElement = document.getElementById("changePasswordResult");

    try {
        hideError();

        const result = await fetchJSON("/api/me/change-password", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                oldPassword: oldPasswordInput.value,
                newPassword: newPasswordInput.value
            })
        });

        oldPasswordInput.value = "";
        newPasswordInput.value = "";

        resultElement.textContent = `Пароль успешно изменён для пользователя ${result.username}`;
    } catch (error) {
        resultElement.textContent = error.message;
    }
}

async function initializeAuthenticatedDashboard() {
    await loadCurrentUser();

    renderUserPanel();
    applyRolePermissions();

    if (canViewQualityWeights()) {
        await loadQualityWeights();
    }

    if (canManageSetpoints()) {
        await loadSetpoints();
    }

    if (canManageUsers()) {
        await loadUsers();
    }

    await refreshDashboard();

    startPolling();
}

function startPolling() {
    stopPolling();

    dashboardIntervalID = setInterval(refreshDashboard, refreshIntervalMs);

    historyIntervalID = setInterval(() => {
        const autoRefreshEnabled = document.getElementById("historyAutoRefresh")?.checked;

        if (autoRefreshEnabled && canViewHistory()) {
            loadTelemetryHistory();
        }
    }, refreshIntervalMs);
}

function stopPolling() {
    if (dashboardIntervalID !== null) {
        clearInterval(dashboardIntervalID);
        dashboardIntervalID = null;
    }

    if (historyIntervalID !== null) {
        clearInterval(historyIntervalID);
        historyIntervalID = null;
    }
}

function formatDate(value) {
    if (!value) {
        return "—";
    }

    return new Date(value).toLocaleString();
}

function formatTime(value) {
    if (!value) {
        return "";
    }

    return new Date(value).toLocaleTimeString();
}

function formatNumber(value) {
    if (typeof value !== "number") {
        return value;
    }

    return Number.isInteger(value) ? value.toString() : value.toFixed(2);
}

function escapeHTML(value) {
    return String(value ?? "")
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#039;");
}

function toRFC3339FromDateTimeLocal(value) {
    if (!value) {
        return "";
    }

    return new Date(value).toISOString();
}

function toDateTimeLocalValue(date) {
    const timezoneOffsetMs = date.getTimezoneOffset() * 60 * 1000;
    const localDate = new Date(date.getTime() - timezoneOffsetMs);

    return localDate.toISOString().slice(0, 16);
}

function startOfToday() {
    const now = new Date();

    return new Date(
        now.getFullYear(),
        now.getMonth(),
        now.getDate(),
        0,
        0,
        0,
        0
    );
}

function updateHistoryRangeControls() {
    const range = document.getElementById("historyRange").value;
    const isCustom = range === "custom";

    document.getElementById("customFromGroup").classList.toggle("hidden", !isCustom);
    document.getElementById("customToGroup").classList.toggle("hidden", !isCustom);

    if (!isCustom) {
        return;
    }

    const fromInput = document.getElementById("historyFrom");
    const toInput = document.getElementById("historyTo");

    if (!fromInput.value && !toInput.value) {
        const now = new Date();
        const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);

        fromInput.value = toDateTimeLocalValue(oneHourAgo);
        toInput.value = toDateTimeLocalValue(now);
    }
}

function resolveHistoryRange() {
    const range = document.getElementById("historyRange").value;
    const now = new Date();

    switch (range) {
        case "last_1h":
            return {
                from: new Date(now.getTime() - 60 * 60 * 1000).toISOString(),
                to: now.toISOString()
            };

        case "last_2h":
            return {
                from: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
                to: now.toISOString()
            };

        case "today":
            return {
                from: startOfToday().toISOString(),
                to: now.toISOString()
            };

        case "last_2d":
            return {
                from: new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000).toISOString(),
                to: now.toISOString()
            };

        case "last_7d":
            return {
                from: new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                to: now.toISOString()
            };

        case "custom":
            return {
                from: toRFC3339FromDateTimeLocal(document.getElementById("historyFrom").value),
                to: toRFC3339FromDateTimeLocal(document.getElementById("historyTo").value)
            };

        default:
            return {
                from: "",
                to: ""
            };
    }
}

function renderQuality(quality) {
    const valueElement = document.getElementById("qualityValue");
    const stateElement = document.getElementById("qualityState");

    valueElement.textContent = formatNumber(quality.value);
    stateElement.textContent = formatState(quality.state);

    stateElement.className = "quality-state";
    stateElement.classList.add(quality.state || "unknown");
}

async function loadQualityWeights() {
    if (!canViewQualityWeights()) {
        return;
    }

    const weights = await fetchJSON("/api/quality/weights");
    renderQualityWeights(weights);
}

async function saveQualityWeight(weightId) {
    if (!canViewQualityWeights()) {
        showError("Раздел весов качества недоступен для текущей роли");
        return;
    }

    const input = document.getElementById(`qualityWeight_${weightId}`);
    const weight = Number(input.value);

    try {
        hideError();

        await fetchJSON(`/api/quality/weights/${weightId}`, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({ weight })
        });

        await loadQualityWeights();
    } catch (error) {
        showError(error.message);
    }
}

function renderQualityWeights(weights) {
    const container = document.getElementById("qualityWeights");

    if (!container) {
        return;
    }

    if (!weights || weights.length === 0) {
        container.innerHTML = `<div class="empty">Веса параметров качества не настроены.</div>`;
        return;
    }

    container.innerHTML = weights.map((item) => {
        const name = formatParameter(item.parameterType);

        return `
      <article class="parameter-card">
        <div class="parameter-name">${escapeHTML(name)}</div>

        <div class="control-group">
          <label for="qualityWeight_${item.id}">Вес</label>
          <input
            id="qualityWeight_${item.id}"
            type="number"
            min="0.1"
            max="10"
            step="0.1"
            value="${formatNumber(item.weight)}"
          >
        </div>

        <div class="parameter-meta">
          Параметр: ${escapeHTML(name)}<br />
          Обновлено: ${formatDate(item.updatedAt)}<br />
          Кем обновлено: ${escapeHTML(item.updatedBy || "—")}
        </div>

        <button onclick="saveQualityWeight(${item.id})">
          Сохранить вес
        </button>
      </article>
    `;
    }).join("");
}

function renderParameters(readings) {
    const container = document.getElementById("parameters");

    if (!readings || readings.length === 0) {
        container.innerHTML = `<div class="empty">Измерения телеметрии пока не поступали.</div>`;
        return;
    }

    container.innerHTML = readings.map((reading) => {
        const name = formatParameter(reading.parameterType);

        return `
      <article class="parameter-card">
        <div class="parameter-name">${escapeHTML(name)}</div>
        <div class="parameter-value">${escapeHTML(formatTelemetryValue(reading.value, reading.unit))}</div>
        <div class="parameter-meta">
          Источник: ${escapeHTML(reading.sourceId)}<br />
          Измерено: ${formatDate(reading.measuredAt)}
        </div>
      </article>
    `;
    }).join("");
}

function renderEvents(events) {
    const container = document.getElementById("events");

    if (!events || events.length === 0) {
        container.innerHTML = `<div class="empty">Активных событий нет.</div>`;
        return;
    }

    container.innerHTML = events.map((event) => {
        const isAcknowledged = event.status === "acknowledged";
        const buttonText = isAcknowledged ? "Подтверждено" : "Подтвердить";
        const disabled = isAcknowledged ? "disabled" : "";
        const parameter = formatParameter(event.parameterType);
        const level = formatEventLevel(event.level);
        const status = formatEventStatus(event.status);
        const message = formatAlertMessage(event);
        const eventClass = event.level === "critical" ? "event-critical" : "event-warning";

        return `
      <article class="event-card ${eventClass}">
        <div class="event-header">
          <div class="event-level">${escapeHTML(level)} · ${escapeHTML(parameter)}</div>
          <div class="event-status">${escapeHTML(status)}</div>
        </div>

        <div class="event-message">${escapeHTML(message)}</div>

        <div class="event-meta">
          Создано: ${formatDate(event.createdAt)}<br />
          Значение: ${escapeHTML(formatTelemetryValue(event.value, event.unit))}<br />
          Источник: ${escapeHTML(event.sourceId)}
        </div>

        <button ${disabled} onclick="acknowledgeEvent(${event.id})">
          ${buttonText}
        </button>
      </article>
    `;
    }).join("");
}

function renderAnomalies(anomalies) {
    const container = document.getElementById("anomalies");

    if (!container) {
        return;
    }

    if (!anomalies || anomalies.length === 0) {
        container.innerHTML = `<div class="empty">Активных аномалий нет.</div>`;
        return;
    }

    container.innerHTML = anomalies.map((anomaly) => {
        const level = formatEventLevel(anomaly.level);
        const type = formatAnomalyType(anomaly.type);
        const status = formatEventStatus(anomaly.status);
        const parameter = formatParameter(anomaly.parameterType);
        const message = formatAnomalyMessage(anomaly);
        const eventClass = anomaly.level === "critical" ? "event-critical" : "event-warning";

        return `
      <article class="event-card ${eventClass}">
        <div class="event-header">
          <div class="event-level">${escapeHTML(level)} · ${escapeHTML(type)}</div>
          <div class="event-status">${escapeHTML(status)}</div>
        </div>

        <div class="event-message">${escapeHTML(message)}</div>

        <div class="event-meta">
          Параметр: ${escapeHTML(parameter)}<br />
          Обнаружено: ${formatDate(anomaly.observedAt)}<br />
          Обновлено: ${formatDate(anomaly.updatedAt)}<br />
          Источник: ${escapeHTML(anomaly.sourceId)}
        </div>
      </article>
    `;
    }).join("");
}

function renderHistoryList(readings) {
    const container = document.getElementById("history");

    if (!readings || readings.length === 0) {
        container.innerHTML = `<div class="empty">История по выбранному параметру не найдена.</div>`;
        return;
    }

    container.innerHTML = readings.slice().reverse().map((reading) => {
        const name = formatParameter(reading.parameterType);

        return `
      <article class="history-card">
        <div class="history-header">
          <div class="history-value">${escapeHTML(formatTelemetryValue(reading.value, reading.unit))}</div>
          <div class="history-time">${formatDate(reading.measuredAt)}</div>
        </div>

        <div class="history-meta">
          Параметр: ${escapeHTML(name)}<br />
          Источник: ${escapeHTML(reading.sourceId)}<br />
          Сохранено: ${formatDate(reading.createdAt)}
        </div>
      </article>
    `;
    }).join("");
}

function drawHistoryChart(readings) {
    const canvas = document.getElementById("historyChart");

    if (!canvas) {
        return;
    }

    const wrapper = canvas.parentElement;
    if (!wrapper) {
        return;
    }

    const ctx = canvas.getContext("2d");

    const width = Math.max(wrapper.clientWidth - 28, 320);
    const height = 250;
    const ratio = window.devicePixelRatio || 1;

    canvas.width = width * ratio;
    canvas.height = height * ratio;
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;

    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    ctx.clearRect(0, 0, width, height);

    const paddingLeft = 48;
    const paddingRight = 20;
    const paddingTop = 26;
    const paddingBottom = 42;

    const plotWidth = width - paddingLeft - paddingRight;
    const plotHeight = height - paddingTop - paddingBottom;

    ctx.strokeStyle = "rgba(148, 163, 184, 0.36)";
    ctx.lineWidth = 1;

    ctx.beginPath();
    ctx.moveTo(paddingLeft, paddingTop);
    ctx.lineTo(paddingLeft, paddingTop + plotHeight);
    ctx.lineTo(paddingLeft + plotWidth, paddingTop + plotHeight);
    ctx.stroke();

    if (!readings || readings.length === 0) {
        ctx.fillStyle = "#64748b";
        ctx.font = "14px Arial";
        ctx.fillText("Нет данных для построения графика", paddingLeft + 10, paddingTop + 30);
        return;
    }

    const values = readings.map((reading) => Number(reading.value)).filter((value) => Number.isFinite(value));

    if (values.length === 0) {
        ctx.fillStyle = "#64748b";
        ctx.font = "14px Arial";
        ctx.fillText("Нет числовых данных", paddingLeft + 10, paddingTop + 30);
        return;
    }

    let minValue = Math.min(...values);
    let maxValue = Math.max(...values);

    if (minValue === maxValue) {
        const delta = Math.max(Math.abs(minValue) * 0.08, 1);
        minValue -= delta;
        maxValue += delta;
    } else {
        const padding = (maxValue - minValue) * 0.12;
        minValue -= padding;
        maxValue += padding;
    }

    const valueRange = maxValue - minValue;

    ctx.strokeStyle = "rgba(148, 163, 184, 0.18)";
    ctx.lineWidth = 1;

    for (let i = 1; i <= 3; i++) {
        const y = paddingTop + (plotHeight / 4) * i;
        ctx.beginPath();
        ctx.moveTo(paddingLeft, y);
        ctx.lineTo(paddingLeft + plotWidth, y);
        ctx.stroke();
    }

    ctx.fillStyle = "#64748b";
    ctx.font = "12px Arial";
    ctx.fillText(formatNumber(maxValue), 6, paddingTop + 4);
    ctx.fillText(formatNumber(minValue), 6, paddingTop + plotHeight);

    const gradient = ctx.createLinearGradient(paddingLeft, 0, paddingLeft + plotWidth, 0);
    gradient.addColorStop(0, "#64748b");
    gradient.addColorStop(0.45, "#b45f82");
    gradient.addColorStop(1, "#9a647c");

    ctx.strokeStyle = gradient;
    ctx.lineWidth = 2.4;
    ctx.beginPath();

    readings.forEach((reading, index) => {
        const value = Number(reading.value);

        const x = readings.length === 1
            ? paddingLeft + plotWidth / 2
            : paddingLeft + (index / (readings.length - 1)) * plotWidth;

        const y = paddingTop + plotHeight - ((value - minValue) / valueRange) * plotHeight;

        if (index === 0) {
            ctx.moveTo(x, y);
            return;
        }

        ctx.lineTo(x, y);
    });

    ctx.stroke();

    ctx.fillStyle = "#9a647c";

    readings.forEach((reading, index) => {
        const value = Number(reading.value);

        const x = readings.length === 1
            ? paddingLeft + plotWidth / 2
            : paddingLeft + (index / (readings.length - 1)) * plotWidth;

        const y = paddingTop + plotHeight - ((value - minValue) / valueRange) * plotHeight;

        ctx.beginPath();
        ctx.arc(x, y, 3.2, 0, Math.PI * 2);
        ctx.fill();
    });

    const firstTime = readings[0]?.measuredAt;
    const lastTime = readings[readings.length - 1]?.measuredAt;

    ctx.fillStyle = "#64748b";
    ctx.font = "12px Arial";

    ctx.fillText(formatTime(firstTime), paddingLeft, height - 14);

    const lastLabel = formatTime(lastTime);
    const lastLabelWidth = ctx.measureText(lastLabel).width;

    ctx.fillText(lastLabel, paddingLeft + plotWidth - lastLabelWidth, height - 14);
}

async function acknowledgeEvent(eventId) {
    try {
        hideError();

        await fetchJSON(`/api/events/${eventId}/ack`, {
            method: "POST"
        });

        await refreshDashboard();
    } catch (error) {
        showError(error.message);
    }
}

async function refreshDashboard() {
    try {
        hideError();

        const [quality, telemetry, events] = await Promise.all([
            fetchJSON("/api/quality/latest"),
            fetchJSON("/api/telemetry/latest"),
            fetchJSON("/api/events/active")
        ]);

        renderQuality(quality);
        renderParameters(telemetry);
        renderEvents(events);

        if (canViewAnomalies()) {
            const anomalies = await fetchJSON("/api/anomalies/active");
            renderAnomalies(anomalies);
        }
    } catch (error) {
        showError(error.message);
    }
}

async function loadTelemetryHistory() {
    if (!canViewHistory()) {
        showError("История недоступна для текущей роли");
        return;
    }

    const button = document.getElementById("loadHistoryButton");

    if (historyIsLoading) {
        return;
    }

    try {
        historyIsLoading = true;
        hideError();

        button.disabled = true;
        button.textContent = "Загружаем...";

        const parameter = document.getElementById("historyParameter").value;
        const limit = document.getElementById("historyLimit").value || "30";
        const range = resolveHistoryRange();

        const params = new URLSearchParams();

        params.set("parameter", parameter);
        params.set("limit", limit);

        if (range.from) {
            params.set("from", range.from);
        }

        if (range.to) {
            params.set("to", range.to);
        }

        const readings = await fetchJSON(`/api/telemetry/history?${params.toString()}`);

        lastHistoryReadings = readings;

        renderHistoryList(readings);
        drawHistoryChart(readings);
    } catch (error) {
        showError(error.message);
    } finally {
        historyIsLoading = false;
        button.disabled = false;
        button.textContent = "Загрузить историю";
    }
}

async function loadSetpoints() {
    if (!canManageSetpoints()) {
        return;
    }

    setpoints = await fetchJSON("/api/setpoints");

    const select = document.getElementById("setpointSelect");
    select.innerHTML = "";

    for (const setpoint of setpoints) {
        const option = document.createElement("option");
        option.value = setpoint.id;
        option.textContent = `${formatParameter(setpoint.parameterType)} (${formatUnit(setpoint.unit)})`;
        select.appendChild(option);
    }

    fillSetpointForm();
}

function fillSetpointForm() {
    const selectedId = Number(document.getElementById("setpointSelect").value);
    const setpoint = setpoints.find((item) => item.id === selectedId);

    if (!setpoint) {
        return;
    }

    document.getElementById("criticalMin").value = setpoint.criticalMin;
    document.getElementById("warningMin").value = setpoint.warningMin;
    document.getElementById("normalMin").value = setpoint.normalMin;
    document.getElementById("normalMax").value = setpoint.normalMax;
    document.getElementById("warningMax").value = setpoint.warningMax;
    document.getElementById("criticalMax").value = setpoint.criticalMax;
}

async function saveSetpoint() {
    if (!canManageSetpoints()) {
        showError("Уставки недоступны для текущей роли");
        return;
    }

    const selectedId = Number(document.getElementById("setpointSelect").value);

    const payload = {
        criticalMin: Number(document.getElementById("criticalMin").value),
        warningMin: Number(document.getElementById("warningMin").value),
        normalMin: Number(document.getElementById("normalMin").value),
        normalMax: Number(document.getElementById("normalMax").value),
        warningMax: Number(document.getElementById("warningMax").value),
        criticalMax: Number(document.getElementById("criticalMax").value)
    };

    try {
        const result = await fetchJSON(`/api/setpoints/${selectedId}`, {
            method: "PUT",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(payload)
        });

        document.getElementById("setpointResult").textContent =
            `Уставка сохранена: ${formatParameter(result.parameterType)}`;

        await loadSetpoints();
    } catch (error) {
        document.getElementById("setpointResult").textContent = error.message;
    }
}

async function loadUsers() {
    if (!canManageUsers()) {
        return;
    }

    users = await fetchJSON("/api/users");
    renderUsers(users);
}

function renderUsers(usersToRender) {
    const container = document.getElementById("users");

    if (!container) {
        return;
    }

    if (!usersToRender || usersToRender.length === 0) {
        container.innerHTML = `<div class="empty">Пользователи не найдены.</div>`;
        return;
    }

    container.innerHTML = usersToRender.map((user) => {
        const activeText = user.isActive ? "активен" : "деактивирован";
        const activeButton = user.isActive
            ? `<button class="danger-button" onclick="deactivateUser(${user.id})">Деактивировать</button>`
            : `<button class="success-button" onclick="activateUser(${user.id})">Активировать</button>`;

        return `
      <article class="user-card">
        <div class="user-header">
          <div class="user-name">#${user.id} · ${escapeHTML(user.username)}</div>
          <div class="user-status">${activeText}</div>
        </div>

        <div class="user-meta">
          Роль: ${escapeHTML(formatRole(user.role))}<br />
          Создан: ${formatDate(user.createdAt)}<br />
          Обновлён: ${formatDate(user.updatedAt)}
        </div>

        <div class="user-actions">
          <div class="control-group">
            <label for="userRole_${user.id}">Роль</label>
            <select id="userRole_${user.id}">
              <option value="operator" ${user.role === "operator" ? "selected" : ""}>Оператор</option>
              <option value="technologist" ${user.role === "technologist" ? "selected" : ""}>Технолог</option>
              <option value="admin" ${user.role === "admin" ? "selected" : ""}>Администратор</option>
            </select>
          </div>

          <div class="control-group">
            <button onclick="updateUserRole(${user.id})">Изменить роль</button>
          </div>

          <div class="control-group">
            ${activeButton}
          </div>

          <div class="control-group">
            <label for="resetPassword_${user.id}">Новый пароль</label>
            <input id="resetPassword_${user.id}" type="password" placeholder="минимум 12 символов">
          </div>

          <div class="control-group">
            <button class="secondary-button" onclick="resetUserPassword(${user.id})">Сбросить пароль</button>
          </div>
        </div>
      </article>
    `;
    }).join("");
}

async function createUser() {
    if (!canManageUsers()) {
        showError("Управление пользователями недоступно для текущей роли");
        return;
    }

    const resultElement = document.getElementById("userResult");

    const usernameInput = document.getElementById("newUsername");
    const passwordInput = document.getElementById("newPassword");
    const roleInput = document.getElementById("newUserRole");
    const isActiveInput = document.getElementById("newUserIsActive");

    try {
        const result = await fetchJSON("/api/users", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                username: usernameInput.value.trim(),
                password: passwordInput.value,
                role: roleInput.value,
                isActive: isActiveInput.checked
            })
        });

        resultElement.textContent = `Пользователь создан: ${result.username}`;

        usernameInput.value = "";
        passwordInput.value = "";
        roleInput.value = "operator";
        isActiveInput.checked = true;

        await loadUsers();
    } catch (error) {
        resultElement.textContent = error.message;
    }
}

async function updateUserRole(userId) {
    if (!canManageUsers()) {
        showError("Управление пользователями недоступно для текущей роли");
        return;
    }

    const role = document.getElementById(`userRole_${userId}`).value;

    try {
        const result = await fetchJSON(`/api/users/${userId}/role`, {
            method: "PATCH",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({ role })
        });

        document.getElementById("userResult").textContent =
            `Роль пользователя ${result.username} изменена на ${formatRole(result.role)}`;

        await loadUsers();
    } catch (error) {
        document.getElementById("userResult").textContent = error.message;
    }
}

async function activateUser(userId) {
    await setUserActive(userId, true);
}

async function deactivateUser(userId) {
    await setUserActive(userId, false);
}

async function setUserActive(userId, isActive) {
    if (!canManageUsers()) {
        showError("Управление пользователями недоступно для текущей роли");
        return;
    }

    const action = isActive ? "activate" : "deactivate";

    try {
        const result = await fetchJSON(`/api/users/${userId}/${action}`, {
            method: "POST"
        });

        document.getElementById("userResult").textContent =
            isActive
                ? `Пользователь ${result.username} активирован`
                : `Пользователь ${result.username} деактивирован`;

        await loadUsers();
    } catch (error) {
        document.getElementById("userResult").textContent = error.message;
    }
}

async function resetUserPassword(userId) {
    if (!canManageUsers()) {
        showError("Управление пользователями недоступно для текущей роли");
        return;
    }

    const passwordInput = document.getElementById(`resetPassword_${userId}`);
    const password = passwordInput.value;

    try {
        const result = await fetchJSON(`/api/users/${userId}/reset-password`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({ password })
        });

        passwordInput.value = "";

        document.getElementById("userResult").textContent =
            `Пароль пользователя ${result.username} сброшен`;

        await loadUsers();
    } catch (error) {
        document.getElementById("userResult").textContent = error.message;
    }
}

async function startDashboard() {
    updateHistoryRangeControls();

    const setpointSelect = document.getElementById("setpointSelect");
    if (setpointSelect) {
        setpointSelect.addEventListener("change", fillSetpointForm);
    }

    const token = localStorage.getItem(authTokenStorageKey);

    if (!token) {
        renderUserPanel();
        return;
    }

    try {
        await initializeAuthenticatedDashboard();
    } catch (error) {
        localStorage.removeItem(authTokenStorageKey);
        currentUser = null;
        renderUserPanel();
        showError("Сессия недействительна. Войдите заново.");
    }

    window.addEventListener("resize", () => {
        if (canViewHistory() && activeTab === "history") {
            drawHistoryChart(lastHistoryReadings);
        }
    });
}

window.login = login;
window.logout = logout;
window.changeOwnPassword = changeOwnPassword;

window.switchTab = switchTab;
window.acknowledgeEvent = acknowledgeEvent;

window.loadTelemetryHistory = loadTelemetryHistory;
window.updateHistoryRangeControls = updateHistoryRangeControls;

window.saveSetpoint = saveSetpoint;

window.createUser = createUser;
window.updateUserRole = updateUserRole;
window.activateUser = activateUser;
window.deactivateUser = deactivateUser;
window.resetUserPassword = resetUserPassword;

window.saveQualityWeight = saveQualityWeight;

startDashboard();