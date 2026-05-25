(function () {
  var ws = null;
  var sse = null;
  var retry = 0;
  var maxRetry = 5;
  var baseDelay = 500;

  function reload() {
    location.reload();
  }

  function scheduleSSE() {
    if (sse) return;
    sse = new EventSource("/events");
    sse.onmessage = reload;
    sse.onerror = function () {
      sse.close();
      sse = null;
      setTimeout(scheduleSSE, baseDelay);
    };
  }

  function connectWS() {
    try {
      var proto = location.protocol === "https:" ? "wss:" : "ws:";
      ws = new WebSocket(proto + "//" + location.host + "/ws");
      ws.onopen = function () {
        retry = 0;
      };
      ws.onmessage = reload;
      ws.onclose = function () {
        ws = null;
        retry += 1;
        if (retry < maxRetry) {
          setTimeout(connectWS, baseDelay * retry);
        } else {
          scheduleSSE();
        }
      };
      ws.onerror = function () {
        if (ws) ws.close();
      };
    } catch (_e) {
      scheduleSSE();
    }
  }

  connectWS();
})();
