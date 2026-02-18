// mdp preview client — handles WebSocket connection and DOM updates.
"use strict";

(function () {
  var content = document.getElementById("content");
  var status = document.getElementById("connection-status");
  var ws = null;
  var reconnectDelay = 250;
  var maxReconnectDelay = 5000;

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      reconnectDelay = 250;
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
      if (ws) ws.close();
    };
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
    if (!status) return;
    if (connected) {
      status.className = "connected";
    } else {
      status.className = "disconnected";
      status.textContent = "Disconnected \u2014 reconnecting\u2026";
    }
  }

  // Connect on page load.
  connect();
})();
