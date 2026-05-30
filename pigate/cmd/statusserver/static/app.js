const state = {
  pending: false,
};

const els = {
  location: document.querySelector("#location"),
  mqttBadge: document.querySelector("#mqttBadge"),
  dbBadge: document.querySelector("#dbBadge"),
  gateState: document.querySelector("#gateState"),
  gateTime: document.querySelector("#gateTime"),
  credentialStatus: document.querySelector("#credentialStatus"),
  credentialTime: document.querySelector("#credentialTime"),
  lastCommand: document.querySelector("#lastCommand"),
  lastCommandTime: document.querySelector("#lastCommandTime"),
  serverTime: document.querySelector("#serverTime"),
  notice: document.querySelector("#notice"),
  buttons: Array.from(document.querySelectorAll("[data-command]")),
};

const statusLabels = {
  opened: "Open",
  locked_open: "Locked Open",
  closed: "Closed",
  unknown: "Unknown",
};

const commandLabels = {
  open: "Open",
  lock_open: "Lock Open",
  hold_open: "Lock Open",
  close: "Close",
};

function labelFor(value, labels) {
  return labels[value] || titleCase(value || "unknown");
}

function titleCase(value) {
  return String(value)
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}

function setBadge(el, label, ok) {
  el.textContent = label;
  el.classList.toggle("ok", ok);
  el.classList.toggle("bad", !ok);
}

function setNotice(message, isError = false) {
  els.notice.textContent = message;
  els.notice.classList.toggle("error", isError);
}

function setPending(pending) {
  state.pending = pending;
  els.buttons.forEach((button) => {
    button.disabled = pending;
  });
}

function renderStatus(data) {
  els.location.textContent = data.location_id || "Unknown location";
  setBadge(els.mqttBadge, data.mqtt_connected ? "MQTT Online" : "MQTT Offline", data.mqtt_connected);
  setBadge(els.dbBadge, data.db_connected ? "Postgres Online" : "Postgres Offline", data.db_connected);

  const gateStatus = data.gate_status || "unknown";
  els.gateState.textContent = labelFor(gateStatus, statusLabels);
  els.gateTime.textContent = data.gate_status_at ? `Updated ${formatTime(data.gate_status_at)}` : "No status yet";

  els.credentialStatus.textContent = labelFor(data.credential_status, {});
  els.credentialTime.textContent = data.credential_status_at ? `Updated ${formatTime(data.credential_status_at)}` : "No update yet";

  els.lastCommand.textContent = data.last_command ? labelFor(data.last_command, commandLabels) : "None";
  els.lastCommandTime.textContent = data.last_command_at ? `Sent ${formatTime(data.last_command_at)}` : "No command yet";
  els.serverTime.textContent = data.server_time ? `Refreshed ${formatTime(data.server_time)}` : "Waiting for status";

  document.body.classList.remove("gate-open", "gate-locked-open", "gate-closed", "gate-unknown");
  if (gateStatus === "opened") document.body.classList.add("gate-open");
  else if (gateStatus === "locked_open") document.body.classList.add("gate-locked-open");
  else if (gateStatus === "closed") document.body.classList.add("gate-closed");
  else document.body.classList.add("gate-unknown");
}

async function refreshStatus() {
  try {
    const response = await fetch("/api/status", { cache: "no-store" });
    if (!response.ok) throw new Error("Status request failed");
    const data = await response.json();
    renderStatus(data);
    if (!state.pending) setNotice(data.db_error ? data.db_error : "");
  } catch (error) {
    setNotice(error.message, true);
  }
}

async function sendCommand(command) {
  setPending(true);
  setNotice(`Sending ${labelFor(command, commandLabels)}`);
  try {
    const response = await fetch("/api/command", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ command }),
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(body.error || "Command failed");
    setNotice(`${labelFor(body.command, commandLabels)} sent`);
    await refreshStatus();
  } catch (error) {
    setNotice(error.message, true);
  } finally {
    setPending(false);
  }
}

els.buttons.forEach((button) => {
  button.addEventListener("click", () => sendCommand(button.dataset.command));
});

refreshStatus();
setInterval(refreshStatus, 3000);
