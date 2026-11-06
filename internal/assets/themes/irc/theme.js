// Theme: irc - nick colors, a live clock, and join/topic notices.
(function () {
  var NICK_COLORS = 12;

  function nickColor(nick) {
    var h = 0;
    for (var i = 0; i < nick.length; i++) {
      h = (h * 31 + nick.charCodeAt(i)) >>> 0;
    }
    return "var(--nick-" + (h % NICK_COLORS) + ")";
  }

  function pad(n) {
    return n < 10 ? "0" + n : "" + n;
  }

  function notice(html) {
    var line = document.createElement("div");
    line.className = "irc-notice";
    line.innerHTML = '<span class="irc-notice-mark">-!-</span> ' + html;
    return line;
  }

  function escapeHTML(s) {
    var d = document.createElement("div");
    d.textContent = s;
    return d.innerHTML;
  }

  document.querySelectorAll(".author a, .quote-author").forEach(function (el) {
    el.style.color = nickColor(el.textContent.trim().toLowerCase());
  });

  var clock = document.getElementById("irc-clock");
  function tick() {
    var now = new Date();
    if (clock) clock.textContent = "[" + pad(now.getHours()) + ":" + pad(now.getMinutes()) + "]";
  }
  tick();
  setInterval(tick, 15000);

  var content = document.getElementById("content");
  var channel = document.querySelector("#masthead a");
  var topic = document.querySelector(".irc-topic-text");
  if (content && channel) {
    var nicks = {};
    document.querySelectorAll(".item[data-user]").forEach(function (item) {
      nicks[item.getAttribute("data-user")] = true;
    });
    var chan = escapeHTML(channel.textContent.trim());
    var lines = [
      notice('<span class="irc-hl">you</span> [~you@tumble] has joined <span class="irc-hl">' + chan + "</span>"),
      notice("Topic for " + chan + ": " + escapeHTML(topic ? topic.textContent : ""))
    ];
    var nickCount = Object.keys(nicks).length;
    if (nickCount) {
      lines.push(notice(chan + ": Total of " + nickCount + " nicks talking on this page"));
    }
    for (var i = lines.length - 1; i >= 0; i--) {
      content.insertBefore(lines[i], content.firstChild);
    }

    var act = document.getElementById("irc-act");
    var count = document.querySelectorAll("#content .item").length;
    if (act && count) act.textContent = "[Act: " + count + "]";
  }
})();
