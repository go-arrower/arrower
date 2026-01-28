// initAllPendingJobsPieCharts();
document.addEventListener("htmx:load", initAllPendingJobsPieCharts);
document.addEventListener("htmx:beforeOnLoad", leave);

function initAllPendingJobsPieCharts() {
  document
    .querySelectorAll("[data-js-pending-jobs-by-queue]")
    .forEach((elem) => {
      elem.setAttribute("data-chart-initialised", true);

      let queuesChart = echarts.init(elem);

      queuesChart.showLoading();
      queuesChart.setOption({
        tooltip: {
          trigger: "item",
          formatter: "<b>{b}</b>: {c} ({d}%)",
        },
        title: {
          text: "Pending Jobs per Queue",
          right: "25%",
        },
        series: [
          {
            type: "pie",
            data: [],
            emphasis: {
              itemStyle: {
                shadowBlur: 10,
                shadowOffsetX: 0,
                shadowColor: "rgba(0, 0, 0, 0.5)",
              },
            },
          },
        ],
        // roseType: 'area'
      });

      updateAllPendingJobsPieCharts(queuesChart);
    });
}

function updateAllPendingJobsPieCharts() {
  document
    .querySelectorAll("[data-js-pending-jobs-by-queue]")
    .forEach((elem) => {
      let queuesChart = echarts.init(elem);
      updatePendingJobsPieChart(queuesChart);
    });
}

async function updatePendingJobsPieChart(queuesChart) {
  const response = await fetch("/admin/jobs/data/pending");
  const data = await response.json();

  queuesChart.hideLoading();
  queuesChart.setOption({
    series: [
      {
        data: data,
      },
    ],
  });
}

function leave(e) {
  if (e.detail.pathInfo.finalRequestPath !== "/admin/jobs") {
    // user leaves page
    document.removeEventListener("htmx:beforeOnLoad", leave);
    document.removeEventListener("htmx:load", updateAllPendingJobsPieCharts);
  }
}
