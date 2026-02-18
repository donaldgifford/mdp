// mdp preview client — handles WebSocket/SSE connection and DOM updates.
"use strict";

(function () {
  var content = document.getElementById("content");
  var statusEl = document.getElementById("connection-status");
  var reconnectDelay = 250;
  var maxReconnectDelay = 5000;
  var useSSE = false;

  function connectWebSocket() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      reconnectDelay = 250;
      useSSE = false;
      setConnectionStatus(true);
    };

    ws.onmessage = function (event) {
      updateContent(event.data);
    };

    ws.onclose = function () {
      setConnectionStatus(false);
      scheduleReconnect();
    };

    ws.onerror = function () {
      // If WebSocket fails, try SSE on next reconnect.
      useSSE = true;
      ws.close();
    };
  }

  function connectSSE() {
    var source = new EventSource("/events");

    source.onopen = function () {
      reconnectDelay = 250;
      setConnectionStatus(true);
    };

    source.onmessage = function (event) {
      updateContent(event.data);
    };

    source.onerror = function () {
      source.close();
      setConnectionStatus(false);
      scheduleReconnect();
    };
  }

  function connect() {
    if (useSSE) {
      connectSSE();
    } else {
      connectWebSocket();
    }
  }

  function scheduleReconnect() {
    setTimeout(function () {
      reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
      connect();
    }, reconnectDelay);
  }

  function updateContent(html) {
    if (content) {
      content.innerHTML = html;
    }
  }

  function setConnectionStatus(connected) {
    if (!statusEl) return;
    if (connected) {
      statusEl.className = "connected";
    } else {
      statusEl.className = "disconnected";
      statusEl.textContent = "Disconnected \u2014 reconnecting\u2026";
    }
  }

  // Connect on page load.
  connect();
})();
