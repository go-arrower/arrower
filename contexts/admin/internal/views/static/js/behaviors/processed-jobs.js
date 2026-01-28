document.addEventListener("htmx:load", initAllProcessedJobsChart);
document.addEventListener("htmx:beforeOnLoad", leave);


function initAllProcessedJobsChart() {
  document.querySelectorAll("[data-js-processed-jobs]").forEach((elem) => {
    elem.setAttribute("data-chart-initialised", true);

    let interval = elem.getAttribute("data-interval");
    interval = interval ? interval : "";

    const charType = interval === "hour" ? "bar" : "line";

    let processedChart = echarts.init(elem);

    processedChart.showLoading();
    processedChart.setOption({
      tooltip: {
        trigger: "axis",
        // axisPointer: { type: 'cross' }
      },
      title: {
        text: `Processed Jobs this ${interval}`,
        right: "30%",
      },
      xAxis: {
        type: "category",
      },
      yAxis: {
        type: "value",
        name: "Jobs",
      },
      series: [
        {
          type: charType,
          smooth: true,
        },
      ],
      toolbox: {
        feature: {
          myTool1: {
            show: true,
            title: "Reload",
            icon: "path://M432.45,595.444c0,2.177-4.661,6.82-11.305,6.82c-6.475,0-11.306-4.567-11.306-6.82s4.852-6.812,11.306-6.812C427.841,588.632,432.452,593.191,432.45,595.444L432.45,595.444z M421.155,589.876c-3.009,0-5.448,2.495-5.448,5.572s2.439,5.572,5.448,5.572c3.01,0,5.449-2.495,5.449-5.572C426.604,592.371,424.165,589.876,421.155,589.876L421.155,589.876z M421.146,591.891c-1.916,0-3.47,1.589-3.47,3.549c0,1.959,1.554,3.548,3.47,3.548s3.469-1.589,3.469-3.548C424.614,593.479,423.062,591.891,421.146,591.891L421.146,591.891zM421.146,591.891",
            onclick: function() {
              updateProcessedLineChart(processedChart, interval);
            },
          },
        },
      },
    });

    updateProcessedLineChart(processedChart, interval);
  });
}

async function updateProcessedLineChart(processedChart, interval) {
  const response = await fetch("/admin/jobs/data/processed/" + interval);
  const data = await response.json();

  processedChart.hideLoading();
  processedChart.setOption({
    xAxis: {
      data: data.xAxis,
    },
    series: [
      {
        data: data.series,
      },
    ],
  });
}


function leave(e) {
  if (e.detail.pathInfo.finalRequestPath !== "/admin/jobs") {
    // user leaves page
    document.removeEventListener("htmx:beforeOnLoad", leave);
    document.removeEventListener("htmx:load", initAllProcessedJobsChart);
  }
}
