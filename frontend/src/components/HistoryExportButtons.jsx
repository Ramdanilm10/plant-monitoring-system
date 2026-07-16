function escapeCSV(value) {
  const text =
    value === null ||
    value === undefined
      ? ""
      : String(value);

  return `"${text.replace(
    /"/g,
    '""',
  )}"`;
}

function formatDateTime(value) {
  const date = new Date(value);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "-";
  }

  return new Intl.DateTimeFormat(
    "id-ID",
    {
      dateStyle: "medium",
      timeStyle: "medium",
    },
  ).format(date);
}

function sanitizeFileName(value) {
  return String(
    value || "tanaman",
  )
    .trim()
    .toLowerCase()
    .replace(
      /[^a-z0-9]+/g,
      "-",
    )
    .replace(
      /^-+|-+$/g,
      "",
    );
}

function getStatistics(
  readings,
  dataKey,
) {
  const values = readings
    .map((reading) =>
      Number(
        reading[dataKey],
      ),
    )
    .filter(
      (value) =>
        Number.isFinite(value),
    );

  if (!values.length) {
    return {
      minimum: "-",
      average: "-",
      maximum: "-",
    };
  }

  const minimum =
    Math.min(...values);

  const maximum =
    Math.max(...values);

  const average =
    values.reduce(
      (total, value) =>
        total + value,
      0,
    ) / values.length;

  return {
    minimum,
    average:
      average.toFixed(2),
    maximum,
  };
}

function HistoryExportButtons({
  plant,
  rangeLabel,
  history,
  readings,
}) {
  const hasData =
    Array.isArray(readings) &&
    readings.length > 0;

  function handleDownloadCSV() {
    if (!hasData) {
      return;
    }

    const temperature =
      getStatistics(
        readings,
        "temperature",
      );

    const humidity =
      getStatistics(
        readings,
        "humidity",
      );

    const soilMoisture =
      getStatistics(
        readings,
        "soil_moisture",
      );

    const rows = [
      [
        "Laporan Monitoring Tanaman",
      ],

      [
        "Tanaman",
        plant?.name || "-",
      ],

      [
        "Jenis",
        plant?.type || "-",
      ],

      [
        "Rentang",
        rangeLabel,
      ],

      [
        "Mulai",
        formatDateTime(
          history?.start_at,
        ),
      ],

      [
        "Selesai",
        formatDateTime(
          history?.end_at,
        ),
      ],

      [],

      [
        "Ringkasan",
        "Minimum",
        "Rata-rata",
        "Maksimum",
      ],

      [
        "Suhu udara (°C)",
        temperature.minimum,
        temperature.average,
        temperature.maximum,
      ],

      [
        "Kelembapan udara (%)",
        humidity.minimum,
        humidity.average,
        humidity.maximum,
      ],

      [
        "Kelembapan tanah (%)",
        soilMoisture.minimum,
        soilMoisture.average,
        soilMoisture.maximum,
      ],

      [],

      [
        "Waktu",
        "Suhu (°C)",
        "Kelembapan Udara (%)",
        "Kelembapan Tanah (%)",
      ],

      ...readings.map(
        (reading) => [
          formatDateTime(
            reading.recorded_at,
          ),

          reading.temperature,

          reading.humidity,

          reading.soil_moisture,
        ],
      ),
    ];

    const csvContent = rows
      .map((row) =>
        row
          .map(escapeCSV)
          .join(";"),
      )
      .join("\r\n");

    const blob = new Blob(
      [
        "\uFEFF",
        csvContent,
      ],
      {
        type: "text/csv;charset=utf-8",
      },
    );

    const url =
      URL.createObjectURL(blob);

    const link =
      document.createElement("a");

    link.href = url;

    link.download =
      `laporan-${sanitizeFileName(
        plant?.name,
      )}-${sanitizeFileName(
        rangeLabel,
      )}.csv`;

    document.body.appendChild(
      link,
    );

    link.click();
    link.remove();

    URL.revokeObjectURL(url);
  }

  function handlePrintPDF() {
    if (!hasData) {
      return;
    }

    document.body.classList.add(
      "print-monitoring-report",
    );

    function cleanup() {
      document.body.classList.remove(
        "print-monitoring-report",
      );
    }

    window.addEventListener(
      "afterprint",
      cleanup,
      {
        once: true,
      },
    );

    window.setTimeout(() => {
      window.print();
    }, 100);
  }

  return (
    <>
      <div className="print-only mb-6">
        <h1 className="text-2xl font-bold text-slate-950">
          Laporan Monitoring Tanaman
        </h1>

        <p className="mt-3">
          Tanaman:{" "}
          <strong>
            {plant?.name || "-"}
          </strong>
        </p>

        <p>
          Jenis:{" "}
          {plant?.type || "-"}
        </p>

        <p>
          Periode: {rangeLabel}
        </p>

        <p>
          Dicetak:{" "}
          {formatDateTime(
            new Date(),
          )}
        </p>
      </div>

      <div className="no-print flex flex-wrap gap-2">
        <button
          type="button"
          disabled={!hasData}
          onClick={
            handleDownloadCSV
          }
          className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-xs font-semibold text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Unduh CSV
        </button>

        <button
          type="button"
          disabled={!hasData}
          onClick={handlePrintPDF}
          className="rounded-lg bg-emerald-700 px-4 py-2 text-xs font-semibold text-white transition hover:bg-emerald-800 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Cetak / Simpan PDF
        </button>
      </div>
    </>
  );
}

export default HistoryExportButtons;