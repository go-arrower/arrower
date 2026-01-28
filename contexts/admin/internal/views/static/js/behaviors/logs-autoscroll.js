if (!window.Arrower) window.Arrower = {};

document.addEventListener("htmx:beforeOnLoad", leave);

Arrower.Log = {
  auto: true,
  scrollableElem: null,
  autoScrollerButton: document.getElementById("autoScroll"),
  autoScrollerRenderer: null,

  startLive: function () {
    this.auto = true;
    this.autoScrollerButton.innerHTML = Arrower.Log.autoScrollerButtonLive;
    this.scrollableElem.scrollTop = Arrower.Log.scrollableElem.scrollHeight;
  },
  pauseLive: function () {
    this.auto = false;
    this.autoScrollerButton.innerHTML = this.autoScrollerButtonPause;
  },

  autoScrollerButtonLive: `<span class="flex border px-4 py-2 text-green-600 hover:bg-green-200">
                 <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6">
                   <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.348a1.125 1.125 0 010 1.971l-11.54 6.347a1.125 1.125 0 01-1.667-.985V5.653z" />
                 </svg> Live</span>`,
  autoScrollerButtonPause: `<span class="flex border px-4 py-2 hover:text-green-600 hover:border-green-200" onclick="Arrower.Log.startLive()">
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25v13.5m-7.5-13.5v13.5" />
                  </svg> Live</span>`,
};

document.querySelectorAll("[data-js-logs-autoscroll]").forEach((elem) => {
  Arrower.Log.scrollableElem = elem;

  elem.addEventListener("wheel", (e) => {
    if (elem.scrollTop + elem.clientHeight >= elem.scrollHeight) {
      Arrower.Log.startLive();
    } else {
      Arrower.Log.pauseLive();
    }
  });

  elem.addEventListener("scroll", (e) => {
    if (elem.scrollTop + elem.clientHeight >= elem.scrollHeight) {
      Arrower.Log.startLive();
    } else {
      Arrower.Log.pauseLive();
    }
  });

  elem.addEventListener("keydown", (e) => {
    if (e.code === "PageUp" || e.code === "Home" || e.code === "ArrowUp") {
      Arrower.Log.pauseLive();
    }
  });

  elem.addEventListener("keydown", (e) => {
    if (e.code === "PageDown" || e.code === "End" || e.code === "ArrowDown") {
      if (
        !Arrower.Log.auto &&
        elem.scrollTop + elem.clientHeight >= elem.scrollHeight
      ) {
        Arrower.Log.startLive();
      }
    }
  });

  Arrower.Log.autoScrollerRenderer = window.setInterval(function () {
    if (Arrower.Log.auto) {
      Arrower.Log.autoScrollerButton.innerHTML =
        Arrower.Log.autoScrollerButtonLive;
      elem.scrollTop = elem.scrollHeight;
    }
  }, 100);
});

function leave(e) {
  if (!e.detail.pathInfo.finalRequestPath.startsWith("/admin/logs")) {
    // user leaves page
    window.clearInterval(Arrower.Log.autoScrollerRenderer);
    document.removeEventListener("htmx:beforeOnLoad", leave);
    Arrower.Log = null; // gc this state
  }
}
