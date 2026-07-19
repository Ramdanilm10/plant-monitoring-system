const W = 1000;
const H = 360;
const TZ = "Asia/Jakarta";
const P = { top: 42, right: 88, bottom: 78, left: 88 };
const COLORS = {
  temperature: "#059669",
  humidity: "#2563eb",
  soilMoisture: "#d97706",
};
const MINUTE = 60 * 1000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

function number(value) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function formatNumber(value) {
  const parsed = number(value);
  if (parsed === null) return "-";
  return new Intl.NumberFormat("id-ID", {
    maximumFractionDigits: 2,
  }).format(parsed);
}

function timestamp(value) {
  if (!value) return null;
  const parsed = new Date(value).getTime();
  return Number.isFinite(parsed) ? parsed : null;
}

function formatAxisTime(value, span) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";

  let options;
  if (span <= 6 * HOUR) {
    options = { hour: "2-digit", minute: "2-digit" };
  } else if (span <= DAY) {
    options = {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    };
  } else if (span <= WEEK) {
    options = {
      day: "2-digit",
      month: "short",
      hour: "2-digit",
    };
  } else {
    options = { day: "2-digit", month: "short" };
  }

  return new Intl.DateTimeFormat("id-ID", {
    ...options,
    timeZone: TZ,
    hour12: false,
  }).format(date);
}

function formatTooltipTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";

  return `${new Intl.DateTimeFormat("id-ID", {
    weekday: "short",
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZone: TZ,
  }).format(date)} WIB`;
}

function normalize(readings) {
  if (!Array.isArray(readings)) return [];

  const byTime = new Map();
  readings.forEach((reading) => {
    const time = timestamp(reading?.recorded_at);
    if (time === null) return;

    byTime.set(time, {
      time,
      recordedAt: reading.recorded_at,
      temperature: number(reading.temperature),
      humidity: number(reading.humidity),
      soil_moisture: number(reading.soil_moisture),
    });
  });

  return [...byTime.values()].sort((a, b) => a.time - b.time);
}

function series(readings, key) {
  return readings
    .map((reading) => ({
      time: reading.time,
      recordedAt: reading.recordedAt,
      value: reading[key],
    }))
    .filter((point) => Number.isFinite(point.value));
}

function timeScale(readings) {
  const times = readings.map((reading) => reading.time);

  if (!times.length) {
    const now = Date.now();
    return {
      times: [],
      dataMin: now - HOUR,
      dataMax: now,
      min: now - HOUR,
      max: now,
      span: HOUR,
    };
  }

  if (times.length === 1) {
    const pad = 30 * MINUTE;
    return {
      times,
      dataMin: times[0],
      dataMax: times[0],
      min: times[0] - pad,
      max: times[0] + pad,
      span: 0,
    };
  }

  const dataMin = times[0];
  const dataMax = times[times.length - 1];
  const span = dataMax - dataMin;
  const pad = Math.max(span * 0.035, MINUTE);

  return {
    times,
    dataMin,
    dataMax,
    min: dataMin - pad,
    max: dataMax + pad,
    span,
  };
}

function temperatureBounds(values) {
  if (!values.length) return { min: 0, max: 1 };

  let min = Math.min(...values);
  let max = Math.max(...values);

  if (min === max) {
    min -= 1;
    max += 1;
  } else {
    const pad = Math.max((max - min) * 0.12, 0.5);
    min -= pad;
    max += pad;
  }

  return {
    min: Math.floor(min * 2) / 2,
    max: Math.ceil(max * 2) / 2,
  };
}

function gridLines(min, max) {
  const drawableH = H - P.top - P.bottom;
  return Array.from({ length: 5 }, (_, index) => {
    const ratio = index / 4;
    return {
      ratio,
      y: P.top + ratio * drawableH,
      value: max - ratio * (max - min),
    };
  });
}

function pathFor(data, getX, getY) {
  if (data.length < 2) return "";
  return data
    .map((point, index) =>
      `${index === 0 ? "M" : "L"} ${getX(point.time)} ${getY(
        point.value,
      )}`,
    )
    .join(" ");
}

function markers(data, maximum = 24) {
  if (data.length <= maximum) return data;

  const indexes = new Set([0, data.length - 1]);
  const interval = (data.length - 1) / (maximum - 1);

  for (let index = 1; index < maximum - 1; index += 1) {
    indexes.add(Math.round(index * interval));
  }

  return [...indexes]
    .sort((a, b) => a - b)
    .map((index) => data[index]);
}

function timeTicks(scale) {
  if (!scale.times.length) return [];
  if (scale.times.length === 1) return [scale.times[0]];

  const count = scale.span <= 6 * HOUR ? 5 : 4;
  return Array.from({ length: count }, (_, index) => {
    const ratio = index / (count - 1);
    return scale.dataMin + ratio * scale.span;
  });
}

function SensorHistoryChart({
  title,
  description,
  readings,
  rangeLabel = "rentang terpilih",
  dataKey,
  unit,
  stroke,
}) {
  const normalized = normalize(readings);
  const data = series(normalized, dataKey);

  if (!data.length) return <EmptyChart title={title} />;

  const values = data.map((point) => point.value);
  const latest = data[data.length - 1].value;
  const minimum = Math.min(...values);
  const maximum = Math.max(...values);
  const average =
    values.reduce((total, value) => total + value, 0) / values.length;
  const scale = timeScale(normalized);
  const percent = dataKey === "humidity" || dataKey === "soil_moisture";
  const bounds = percent ? { min: 0, max: 100 } : temperatureBounds(values);
  const drawableW = W - P.left - P.right;
  const drawableH = H - P.top - P.bottom;
  const getX = (time) =>
    P.left + ((time - scale.min) / (scale.max - scale.min)) * drawableW;
  const getY = (value) =>
    P.top + ((bounds.max - value) / (bounds.max - bounds.min)) * drawableH;
  const path = pathFor(data, getX, getY);

  return (
    <article className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <header className="border-b border-slate-100 p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h3 className="font-semibold text-slate-950">{title}</h3>
            <p className="mt-1 text-sm text-slate-500">{description}</p>
            <p className="mt-2 text-xs font-medium text-slate-400">
              {data.length} titik pembacaan selama {rangeLabel} - waktu WIB
            </p>
          </div>

          <div className="text-left sm:text-right">
            <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">
              Terbaru
            </p>
            <p className="mt-1 text-2xl font-semibold text-slate-950">
              {formatNumber(latest)}{unit}
            </p>
          </div>
        </div>

        <p className="mt-5 text-xs leading-5 text-slate-500">
          Nilai minimum, rata-rata, dan maksimum dihitung dari seluruh pembacaan selama {rangeLabel} terakhir.
        </p>

        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          <Statistic
            label="Minimum"
            value={`${formatNumber(minimum)}${unit}`}
            periodLabel={rangeLabel}
          />
          <Statistic
            label="Rata-rata"
            value={`${formatNumber(average)}${unit}`}
            periodLabel={rangeLabel}
          />
          <Statistic
            label="Maksimum"
            value={`${formatNumber(maximum)}${unit}`}
            periodLabel={rangeLabel}
          />
        </div>
      </header>

      <div className="overflow-x-auto p-4 sm:p-6">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          role="img"
          aria-label={`Grafik ${title}`}
          className="min-w-[720px]"
        >
          <ChartGrid lines={gridLines(bounds.min, bounds.max)} />

          {path && (
            <path
              d={path}
              fill="none"
              stroke={stroke}
              strokeWidth="4"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          )}

          {markers(data).map((point, index) => (
            <circle
              key={`${point.recordedAt}-${index}`}
              cx={getX(point.time)}
              cy={getY(point.value)}
              r="5"
              fill="white"
              stroke={stroke}
              strokeWidth="3"
            >
              <title>{`Waktu pembacaan: ${formatTooltipTime(
                point.time,
              )} | ${title}: ${formatNumber(point.value)}${unit}`}</title>
            </circle>
          ))}

          <TimeLabels scale={scale} getX={getX} />
        </svg>
      </div>
    </article>
  );
}

function CombinedSensorHistoryChart({
  readings,
  rangeLabel = "rentang terpilih",
}) {
  const normalized = normalize(readings);
  const temperature = series(normalized, "temperature");
  const humidity = series(normalized, "humidity");
  const soil = series(normalized, "soil_moisture");

  if (!temperature.length && !humidity.length && !soil.length) {
    return <EmptyChart title="Grafik gabungan semua sensor" />;
  }

  const scale = timeScale(normalized);
  const tempBounds = temperatureBounds(temperature.map((point) => point.value));
  const drawableW = W - P.left - P.right;
  const drawableH = H - P.top - P.bottom;
  const getX = (time) =>
    P.left + ((time - scale.min) / (scale.max - scale.min)) * drawableW;
  const getTemperatureY = (value) =>
    P.top +
    ((tempBounds.max - value) / (tempBounds.max - tempBounds.min)) * drawableH;
  const getPercentageY = (value) =>
    P.top + ((100 - Math.min(Math.max(value, 0), 100)) / 100) * drawableH;
  const lines = gridLines(tempBounds.min, tempBounds.max);

  return (
    <article className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <header className="border-b border-slate-100 p-6">
        <p className="text-xs font-semibold uppercase tracking-wider text-emerald-700">
          Perbandingan
        </p>
        <h3 className="mt-2 text-lg font-semibold text-slate-950">
          Grafik gabungan semua sensor
        </h3>
        <p className="mt-1 text-sm text-slate-500">
          Sumbu kiri menunjukkan suhu. Sumbu kanan memakai rentang tetap 0-100% untuk kelembapan udara dan tanah.
        </p>
        <p className="mt-2 text-xs text-slate-400">
          {normalized.length} waktu pembacaan selama {rangeLabel} - waktu grafik menggunakan WIB.
        </p>

        <div className="mt-5 flex flex-wrap gap-3">
          <Legend color={COLORS.temperature} label={`Suhu udara (\u00B0C)`} />
          <Legend color={COLORS.humidity} label="Kelembapan udara (%)" />
          <Legend color={COLORS.soilMoisture} label="Kelembapan tanah (%)" />
        </div>
      </header>

      <div className="overflow-x-auto p-4 sm:p-6">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          role="img"
          aria-label="Grafik gabungan semua sensor"
          className="min-w-[740px]"
        >
          {lines.map((line) => (
            <g key={line.y}>
              <line
                x1={P.left}
                x2={W - P.right}
                y1={line.y}
                y2={line.y}
                stroke="#e2e8f0"
                strokeWidth="1"
              />
              <text
                x={P.left - 15}
                y={line.y + 7}
                textAnchor="end"
                fontSize="20"
                fill={COLORS.temperature}
              >
                {formatNumber(line.value)}{"\u00B0"}
              </text>
              <text
                x={W - P.right + 15}
                y={line.y + 7}
                textAnchor="start"
                fontSize="20"
                fill={COLORS.humidity}
              >
                {formatNumber(100 - line.ratio * 100)}%
              </text>
            </g>
          ))}

          <ChartAxes rightAxis />
          <ChartSeries
            data={temperature}
            getX={getX}
            getY={getTemperatureY}
            color={COLORS.temperature}
            label="Suhu udara"
            unit={`\u00B0C`}
          />
          <ChartSeries
            data={humidity}
            getX={getX}
            getY={getPercentageY}
            color={COLORS.humidity}
            label="Kelembapan udara"
            unit="%"
          />
          <ChartSeries
            data={soil}
            getX={getX}
            getY={getPercentageY}
            color={COLORS.soilMoisture}
            label="Kelembapan tanah"
            unit="%"
          />
          <TimeLabels scale={scale} getX={getX} />
        </svg>
      </div>
    </article>
  );
}

function ChartSeries({ data, getX, getY, color, label, unit }) {
  if (!data.length) return null;
  const path = pathFor(data, getX, getY);

  return (
    <g>
      {path && (
        <path
          d={path}
          fill="none"
          stroke={color}
          strokeWidth="4"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      )}
      {markers(data, 18).map((point, index) => (
        <circle
          key={`${label}-${point.recordedAt}-${index}`}
          cx={getX(point.time)}
          cy={getY(point.value)}
          r="4.5"
          fill="white"
          stroke={color}
          strokeWidth="3"
        >
          <title>{`Waktu pembacaan: ${formatTooltipTime(
            point.time,
          )} | ${label}: ${formatNumber(point.value)}${unit}`}</title>
        </circle>
      ))}
    </g>
  );
}

function ChartGrid({ lines }) {
  return (
    <>
      {lines.map((line) => (
        <g key={line.y}>
          <line
            x1={P.left}
            x2={W - P.right}
            y1={line.y}
            y2={line.y}
            stroke="#e2e8f0"
            strokeWidth="1"
          />
          <text
            x={P.left - 15}
            y={line.y + 7}
            textAnchor="end"
            fontSize="20"
            fill="#64748b"
          >
            {formatNumber(line.value)}
          </text>
        </g>
      ))}
      <ChartAxes />
    </>
  );
}

function ChartAxes({ rightAxis = false }) {
  return (
    <>
      <line
        x1={P.left}
        x2={P.left}
        y1={P.top}
        y2={H - P.bottom}
        stroke="#94a3b8"
        strokeWidth="1.5"
      />
      {rightAxis && (
        <line
          x1={W - P.right}
          x2={W - P.right}
          y1={P.top}
          y2={H - P.bottom}
          stroke="#94a3b8"
          strokeWidth="1.5"
        />
      )}
      <line
        x1={P.left}
        x2={W - P.right}
        y1={H - P.bottom}
        y2={H - P.bottom}
        stroke="#94a3b8"
        strokeWidth="1.5"
      />
    </>
  );
}

function TimeLabels({ scale, getX }) {
  const ticks = timeTicks(scale);
  if (!ticks.length) return null;

  return ticks.map((time, index) => {
    const anchor =
      index === 0 ? "start" : index === ticks.length - 1 ? "end" : "middle";

    return (
      <g key={`${time}-${index}`}>
        <line
          x1={getX(time)}
          x2={getX(time)}
          y1={H - P.bottom}
          y2={H - P.bottom + 8}
          stroke="#94a3b8"
          strokeWidth="1"
        />
        <text
          x={getX(time)}
          y={H - 32}
          textAnchor={anchor}
          fontSize="18"
          fill="#64748b"
        >
          {formatAxisTime(time, scale.span)}
        </text>
      </g>
    );
  });
}

function EmptyChart({ title }) {
  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
      <h3 className="font-semibold text-slate-950">{title}</h3>
      <p className="mt-2 text-sm text-slate-500">
        Belum ada data pada rentang waktu ini.
      </p>
    </article>
  );
}

function Legend({ color, label }) {
  return (
    <span className="inline-flex items-center gap-2 rounded-full bg-slate-50 px-3 py-2 text-xs font-semibold text-slate-600">
      <span
        className="h-2.5 w-2.5 rounded-full"
        style={{ backgroundColor: color }}
      />
      {label}
    </span>
  );
}

function Statistic({
  label,
  value,
  periodLabel,
}) {
  return (
    <div className="rounded-xl bg-slate-50 px-4 py-3">
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 font-semibold text-slate-900">{value}</p>
      <p className="mt-1 text-[10px] leading-4 text-slate-400">
        Periode {periodLabel}
      </p>
    </div>
  );
}

export { CombinedSensorHistoryChart };
export default SensorHistoryChart;
