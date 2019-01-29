function setCurrent(elementId, data, display) {
  const baseElement = document.getElementById(elementId);
  data.forEach(room => {
    const roomName = room.label;
    const current = room.data[room.data.length - 1].y;
    const currentText = display(current);

    const titleElement = document.createElement('dt');
    titleElement.classList.add('text-muted');
    titleElement.textContent = roomName;
    baseElement.appendChild(titleElement);

    const dataElement = document.createElement('dd');
    dataElement.textContent = currentText;
    baseElement.appendChild(dataElement);
  })
}

function buildChartOptions(chartName, min, max) {
  const options = {
    maintainAspectRatio: false,
    title: {
      display: true,
      text: chartName,
    },
    scales: {
      xAxes: [{
        type: 'time',
        scaleLabel: {
          display: true,
          labelString: 'Date'
        },
      }],
    },
  };
  if (min != null && max != null) {
    options.scales.yAxes = [{
      ticks: {
        suggestedMin: min,
        suggestedMax: max,
      },
    }];
  }
  return options;
}

function createChart(elementId, name, min, max, datasets) {
  const element = document.getElementById(elementId);
  const options = buildChartOptions(name, min, max);
  new Chart(element, { type: 'line', data: { datasets }, options });
}

fetch('/data.json' + document.location.search)
  .then(resp => resp.json())
  .then(resp => {
    if (resp.error) {
      console.error(resp.error);
      return;
    }
    const data = resp.data;
    data.temperature.forEach(room => room.data.forEach(p => p.y = p.y.toFixed(2)));
    data.humidity.forEach(room => room.data.forEach(p => p.y = p.y.toFixed(2)));
    data.pressure.forEach(room => room.data.forEach(p => p.y = p.y/100));
    data.pressure.forEach(room => room.data.forEach(p => p.y = p.y.toFixed(2)));

    setCurrent('temperature-current', data.temperature, v => `${v} ℃`);
    setCurrent('humidity-current', data.humidity, v => `${v} %`);
    setCurrent('pressure-current', data.pressure, v => `${v} hPa`);
    createChart('temperature-chart', 'Temperature', 10, 25, data.temperature);
    createChart('humidity-chart', 'Humidity', 55, 70, data.humidity);
    createChart('pressure-chart', 'Pressure', null, null, data.pressure);
  });
