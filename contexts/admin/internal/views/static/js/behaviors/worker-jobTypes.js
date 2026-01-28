document.addEventListener("htmx:beforeOnLoad", leave);
function leave(e) {
  if (!e.detail.pathInfo.finalRequestPath.includes("/admin/jobs/workers")) {
    // user leaves page
    document.removeEventListener("htmx:load", initDetails);
    document.removeEventListener("htmx:beforeOnLoad", leave);
    Arrower.Jobs = null; // gc this state
  }
}

//required when: load page => go to another page => come back to this page. htmx:load will only be executed after the next trigger refresh
initDetails();
document.addEventListener("htmx:load", initDetails);
function initDetails() {
  if (!window.Arrower) window.Arrower = {};
  if (!window.Arrower.Jobs) {
    window.Arrower.Jobs = {
      Worker: {},
    };
  }

  document.querySelectorAll("[data-js-worker-jobTypes]").forEach((elem) => {
    if (elem.getAttribute("data-initialised")) {
      // already initialed
      return;
    }

    elem.setAttribute("data-initialised", true);

    let worker = elem.dataset.worker;
    if (!Arrower.Jobs.Worker[worker]) {
      Arrower.Jobs.Worker[worker] = false;
    }

    // console.log(worker, Arrower.Jobs.Worker[worker])

    if (Arrower.Jobs.Worker[worker]) {
      elem.nextElementSibling.classList.remove("hidden");
      elem.nextElementSibling.nextElementSibling.classList.remove("hidden");
      elem.nextElementSibling.nextElementSibling.nextElementSibling.classList.remove(
        "hidden",
      );
    }

    elem.addEventListener("click", () => {
      elem.nextElementSibling.classList.toggle("hidden");
      elem.nextElementSibling.nextElementSibling.classList.toggle("hidden");
      elem.nextElementSibling.nextElementSibling.nextElementSibling.classList.toggle(
        "hidden",
      );
      Arrower.Jobs.Worker[worker] = !Arrower.Jobs.Worker[worker];
    });
  });
}
