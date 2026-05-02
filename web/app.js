const refreshIntervalMs = 2000;

const parameterNames = {
    pressure: "Pressure",
    moisture: "Moisture",
    barrel_temperature_zone_1: "Barrel temperature zone 1",
    barrel_temperature_zone_2: "Barrel temperature zone 2",
    barrel_temperature_zone_3: "Barrel temperature zone 3",
    screw_speed: "Screw speed",
    drive_load: "Drive load",
    outlet_temperature: "Outlet temperature"
};

let lastHistoryReadings = [];
let historyIsLoading = false;

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

async function fetchJSON(url, options = {}) {
    const response = await fetch(url, options);

    if (!response.ok) {
        const text = await response.text();

        throw new Error(`${url}: ${response.status} ${text}`);
    }

    return response.json();
}

function formatDate(value) {
    if (!value) {
        return "—";
    }

    return new Date(value).toLocaleString();
}

function formatNumber(value) {
    if (typeof value !== "number") {
        return value;
    }

    return Number.isInteger(value) ? value.toString() : value.toFixed(2);
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

    valueElement.textContent = `Quality Index: ${formatNumber(quality.value)}`;
    stateElement.textContent = quality.state;

    stateElement.className = "quality-state";
    stateElement.classList.add(quality.state || "unknown");
}

function renderParameters(readings) {
    const container = document.getElementById("parameters");

    if (!readings || readings.length === 0) {
        container.innerHTML = `<div class="empty">No telemetry readings yet</div>`;
        return;
    }

    container.innerHTML = readings.map((reading) => {
        const name = parameterNames[reading.parameterType] || reading.parameterType;

        return `
      <article class="parameter-card">
        <div class="parameter-name">${name}</div>
        <div class="parameter-value">${formatNumber(reading.value)} ${reading.unit}</div>
        <div class="parameter-meta">
          Source: ${reading.sourceId}<br />
          Measured at: ${formatDate(reading.measuredAt)}
        </div>
      </article>
    `;
    }).join("");
}

function renderEvents(events) {
    const container = document.getElementById("events");

    if (!events || events.length === 0) {
        container.innerHTML = `<div class="empty">No active events</div>`;
        return;
    }

    container.innerHTML = events.map((event) => {
        const isAcknowledged = event.status === "acknowledged";
        const buttonText = isAcknowledged ? "Acknowledged" : "Acknowledge";
        const disabled = isAcknowledged ? "disabled" : "";

        return `
      <article class="event-card">
        <div class="event-header">
          <div class="event-level">${event.level} · ${event.parameterType}</div>
          <div class="event-status">${event.status}</div>
        </div>

        <div class="event-message">${event.message}</div>

        <div class="event-meta">
          Created at: ${formatDate(event.createdAt)}<br />
          Value: ${formatNumber(event.value)} ${event.unit}<br />
          Source: ${event.sourceId}
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
        container.innerHTML = `<div class="empty">No active anomalies</div>`;
        return;
    }

    container.innerHTML = anomalies.map((anomaly) => {
        return `
      <article class="event-card">
        <div class="event-header">
          <div class="event-level">${anomaly.level} · ${anomaly.type}</div>
          <div class="event-status">${anomaly.status}</div>
        </div>

        <div class="event-message">${anomaly.message}</div>

        <div class="event-meta">
          Parameter: ${anomaly.parameterType}<br />
          Observed at: ${formatDate(anomaly.observedAt)}<br />
          Updated at: ${formatDate(anomaly.updatedAt)}<br />
          Source: ${anomaly.sourceId}
        </div>
      </article>
    `;
    }).join("");
}

function renderHistoryList(readings) {
    const container = document.getElementById("history");

    if (!readings || readings.length === 0) {
        container.innerHTML = `<div class="empty">No history found for selected parameter</div>`;
        return;
    }

    container.innerHTML = readings.slice().reverse().map((reading) => {
        const name = parameterNames[reading.parameterType] || reading.parameterType;

        return `
      <article class="history-card">
        <div class="history-header">
          <div class="history-value">${formatNumber(reading.value)} ${reading.unit}</div>
          <div class="history-time">${formatDate(reading.measuredAt)}</div>
        </div>

        <div class="history-meta">
          Parameter: ${name}<br />
          Source: ${reading.sourceId}<br />
          Created at: ${formatDate(reading.createdAt)}
        </div>
      </article>
    `;
    }).join("");
}

function drawHistoryChart(readings) {
    const canvas = document.getElementById("historyChart");
    const wrapper = canvas.parentElement;
    const ctx = canvas.getContext("2d");

    const width = wrapper.clientWidth - 24;
    const height = 260;
    const ratio = window.devicePixelRatio || 1;

    canvas.width = width * ratio;
    canvas.height = height * ratio;
    canvas.style.width = `${width}px`;
    canvas.style.height = `${height}px`;

    ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    ctx.clearRect(0, 0, width, height);

    const paddingLeft = 44;
    const paddingRight = 16;
    const paddingTop = 20;
    const paddingBottom = 36;

    const plotWidth = width - paddingLeft - paddingRight;
    const plotHeight = height - paddingTop - paddingBottom;

    ctx.strokeStyle = "#d0d5dd";
    ctx.lineWidth = 1;

    ctx.beginPath();
    ctx.moveTo(paddingLeft, paddingTop);
    ctx.lineTo(paddingLeft, paddingTop + plotHeight);
    ctx.lineTo(paddingLeft + plotWidth, paddingTop + plotHeight);
    ctx.stroke();

    if (!readings || readings.length === 0) {
        ctx.fillStyle = "#667085";
        ctx.font = "14px Arial";
        ctx.fillText("No data", paddingLeft + 10, paddingTop + 30);
        return;
    }

    const values = readings.map((reading) => reading.value);
    let minValue = Math.min(...values);
    let maxValue = Math.max(...values);

    if (minValue === maxValue) {
        minValue -= 1;
        maxValue += 1;
    }

    const valueRange = maxValue - minValue;

    ctx.fillStyle = "#667085";
    ctx.font = "12px Arial";
    ctx.fillText(formatNumber(maxValue), 4, paddingTop + 4);
    ctx.fillText(formatNumber(minValue), 4, paddingTop + plotHeight);

    ctx.strokeStyle = "#2563eb";
    ctx.lineWidth = 2;
    ctx.beginPath();

    readings.forEach((reading, index) => {
        const x = readings.length === 1
            ? paddingLeft + plotWidth / 2
            : paddingLeft + (index / (readings.length - 1)) * plotWidth;

        const y = paddingTop + plotHeight - ((reading.value - minValue) / valueRange) * plotHeight;

        if (index === 0) {
            ctx.moveTo(x, y);
            return;
        }

        ctx.lineTo(x, y);
    });

    ctx.stroke();

    ctx.fillStyle = "#2563eb";

    readings.forEach((reading, index) => {
        const x = readings.length === 1
            ? paddingLeft + plotWidth / 2
            : paddingLeft + (index / (readings.length - 1)) * plotWidth;

        const y = paddingTop + plotHeight - ((reading.value - minValue) / valueRange) * plotHeight;

        ctx.beginPath();
        ctx.arc(x, y, 3, 0, Math.PI * 2);
        ctx.fill();
    });

    const firstTime = readings[0]?.measuredAt;
    const lastTime = readings[readings.length - 1]?.measuredAt;

    ctx.fillStyle = "#667085";
    ctx.font = "12px Arial";

    ctx.fillText(
        firstTime ? new Date(firstTime).toLocaleTimeString() : "",
        paddingLeft,
        height - 10
    );

    const lastLabel = lastTime ? new Date(lastTime).toLocaleTimeString() : "";
    const lastLabelWidth = ctx.measureText(lastLabel).width;

    ctx.fillText(
        lastLabel,
        paddingLeft + plotWidth - lastLabelWidth,
        height - 10
    );
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

        const [quality, telemetry, events, anomalies] = await Promise.all([
            fetchJSON("/api/quality/latest"),
            fetchJSON("/api/telemetry/latest"),
            fetchJSON("/api/events/active"),
            fetchJSON("/api/anomalies/active")
        ]);

        renderQuality(quality);
        renderParameters(telemetry);
        renderEvents(events);
        renderAnomalies(anomalies);
    } catch (error) {
        showError(error.message);
    }
}

async function loadTelemetryHistory() {
    const button = document.getElementById("loadHistoryButton");

    if (historyIsLoading) {
        return;
    }

    try {
        historyIsLoading = true;
        hideError();

        button.disabled = true;
        button.textContent = "Loading...";

        const parameter = document.getElementById("historyParameter").value;
        const limit = document.getElementById("historyLimit").value || "200";
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
        button.textContent = "Load history";
    }
}

function startDashboard() {
    updateHistoryRangeControls();

    refreshDashboard();
    setInterval(refreshDashboard, refreshIntervalMs);

    setInterval(() => {
        const autoRefreshEnabled = document.getElementById("historyAutoRefresh")?.checked;

        if (autoRefreshEnabled) {
            loadTelemetryHistory();
        }
    }, refreshIntervalMs);

    window.addEventListener("resize", () => {
        drawHistoryChart(lastHistoryReadings);
    });
}

window.acknowledgeEvent = acknowledgeEvent;
window.loadTelemetryHistory = loadTelemetryHistory;
window.updateHistoryRangeControls = updateHistoryRangeControls;

startDashboard();
